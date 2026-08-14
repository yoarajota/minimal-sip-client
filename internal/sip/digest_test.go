package sip

import (
	"testing"
)

// Digest conformance against the RFC 2617 §3.5 worked example (the vector
// SIP's digest scheme adopts via §22.4).
func TestDigestRFC2617Vector(t *testing.T) {
	challenge := `Digest realm="testrealm@host.com", qop="auth", nonce="dcd98b7102dd2f0e8b11d0f600bfb0c093", opaque="5ccc069c403ebaf9f0171e9517f40e41"`
	// The Authorization header from the RFC: method GET, uri /dir/index.html,
	// user Mufasa, password "Circle Of Life", nc 00000001, cnonce 0a4f113b.
	// Our Authorization() generates its own cnonce/nc, so verify the pieces:
	p := parseDigest(challenge)
	if p.Realm != "testrealm@host.com" || p.Nonce != "dcd98b7102dd2f0e8b11d0f600bfb0c093" {
		t.Fatalf("challenge parse: %+v", p)
	}
	ha1 := md5hex("Mufasa:testrealm@host.com:Circle Of Life")
	if ha1 != "939e7578ed9e3c518a452acee763bce9" {
		t.Errorf("HA1 = %s, want RFC value 939e7578ed9e3c518a452acee763bce9", ha1)
	}
	ha2 := md5hex("GET:/dir/index.html")
	if ha2 != "39aff3a2bab6126f332b942af96d3366" {
		t.Errorf("HA2 = %s, want RFC value 39aff3a2bab6126f332b942af96d3366", ha2)
	}
	response := md5hex("939e7578ed9e3c518a452acee763bce9:dcd98b7102dd2f0e8b11d0f600bfb0c093:00000001:0a4f113b:auth:39aff3a2bab6126f332b942af96d3366")
	if response != "6629fae49393a05397450978507c4ef1" {
		t.Errorf("response = %s, want RFC value 6629fae49393a05397450978507c4ef1", response)
	}
}

func TestAuthorizationFormat(t *testing.T) {
	auth := Authorization(`Digest realm="test", nonce="n1", qop="auth", opaque="o1"`, "REGISTER", "sip:asterisk", "alice", "secret")
	for _, want := range []string{"username=\"alice\"", "realm=\"test\"", "nonce=\"n1\"", "uri=\"sip:asterisk\"", "qop=auth", "response=\""} {
		if !contains(auth, want) {
			t.Errorf("Authorization missing %q: %s", want, auth)
		}
	}
	// No-qop legacy form must not contain qop.
	legacy := Authorization(`Digest realm="test", nonce="n1"`, "REGISTER", "sip:asterisk", "alice", "secret")
	if contains(legacy, "qop=") {
		t.Errorf("legacy Authorization should not carry qop: %s", legacy)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
