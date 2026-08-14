package sip

import (
	"strings"
	"testing"
)

// Conformance tests for the message layer (§7).

func TestParseRequest(t *testing.T) {
	raw := "INVITE sip:100@asterisk SIP/2.0\r\n" +
		"Via: SIP/2.0/UDP 10.0.0.1:5060;branch=z9hG4bKabc\r\n" +
		"Max-Forwards: 70\r\n" +
		"From: Alice <sip:alice@asterisk>;tag=aaa\r\n" +
		"To: <sip:100@asterisk>\r\n" +
		"Call-ID: cid@10.0.0.1\r\n" +
		"CSeq: 1 INVITE\r\n" +
		"Content-Type: application/sdp\r\n" +
		"Content-Length: 4\r\n" +
		"\r\n" +
		"v=0\r\n"
	m, err := ParseMessage(raw)
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	if m.Method() != "INVITE" {
		t.Errorf("method = %q, want INVITE", m.Method())
	}
	if m.Get("Call-ID") != "cid@10.0.0.1" {
		t.Errorf("Call-ID = %q", m.Get("Call-ID"))
	}
	if m.Body != "v=0" {
		t.Errorf("body = %q", m.Body)
	}
}

func TestParseFoldedHeader(t *testing.T) {
	// §7.3.1: continuation lines fold into the previous value.
	raw := "REGISTER sip:asterisk SIP/2.0\r\n" +
		"Via: SIP/2.0/UDP 10.0.0.1:5060;branch=z9hG4bKx\r\n" +
		"Route: <sip:proxy1>\r\n" +
		" \t<sip:proxy2>\r\n" +
		"Max-Forwards: 70\r\n" +
		"From: <sip:a@b>;tag=t\r\n" +
		"To: <sip:a@b>\r\n" +
		"Call-ID: c@d\r\n" +
		"CSeq: 1 REGISTER\r\n" +
		"Content-Length: 0\r\n\r\n"
	m, err := ParseMessage(raw)
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	if got := m.Get("Route"); !strings.Contains(got, "<sip:proxy2>") {
		t.Errorf("folded Route = %q, want both values", got)
	}
}

func TestParseMalformed(t *testing.T) {
	for _, raw := range []string{
		"",
		"no sip version here\r\n\r\n",
		"INVITE sip:100@asterisk SIP/2.0\r\nGarbageHeaderNoColon\r\n\r\n",
	} {
		if _, err := ParseMessage(raw); err == nil {
			t.Errorf("ParseMessage(%q) succeeded, want error", raw)
		}
	}
}

func TestCodeAndClasses(t *testing.T) {
	m, _ := ParseMessage("SIP/2.0 401 Unauthorized\r\nVia: x;branch=y\r\nContent-Length: 0\r\n\r\n")
	if m.Code() != 401 || !m.IsFinal() || m.IsProvisional() {
		t.Errorf("401 classification wrong: code=%d", m.Code())
	}
	p, _ := ParseMessage("SIP/2.0 180 Ringing\r\nVia: x;branch=y\r\nContent-Length: 0\r\n\r\n")
	if !p.IsProvisional() || p.IsFinal() {
		t.Errorf("180 classification wrong")
	}
}

func TestMatchesBranch(t *testing.T) {
	m, _ := ParseMessage("SIP/2.0 200 OK\r\nVia: SIP/2.0/UDP h:5060;branch=z9hG4bKabc;received=1.2.3.4\r\nContent-Length: 0\r\n\r\n")
	if !m.MatchesBranch("z9hG4bKabc") {
		t.Error("branch should match")
	}
	if m.MatchesBranch("z9hG4bKxyz") {
		t.Error("branch should not match")
	}
}

func TestTagFromAndBareURI(t *testing.T) {
	if got := TagFrom("<sip:a@b>;tag=xyz>"); got != "xyz" {
		t.Errorf("TagFrom = %q", got)
	}
	if got := TagFrom("<sip:a@b>"); got != "" {
		t.Errorf("TagFrom without tag = %q", got)
	}
	if got := BareURI(`"100" <sip:100@h:5060>`); got != "sip:100@h:5060" {
		t.Errorf("BareURI = %q", got)
	}
}

func TestNewRequestMandatoryHeaders(t *testing.T) {
	m := NewRequest("REGISTER", "sip:asterisk", "10.0.0.1", 5060,
		"alice <sip:alice@asterisk>", "tag1", "<sip:alice@asterisk>", "", "cid@h", 1, "<sip:alice@10.0.0.1:5060>")
	raw := m.String()
	for _, h := range []string{"Via", "Max-Forwards", "From", "To", "Call-ID", "CSeq"} {
		if !strings.Contains(raw, h+":") {
			t.Errorf("missing mandatory header %s:\n%s", h, raw)
		}
	}
	if !strings.Contains(raw, "Content-Length:") {
		t.Errorf("missing Content-Length:\n%s", raw)
	}
}

// Property: parsing arbitrary input must never panic (§3 parser robustness
// is deliberately scoped out, but a panic would be a defect, not a scope).
func FuzzParseMessage(f *testing.F) {
	f.Add("INVITE sip:100@asterisk SIP/2.0\r\nVia: x;branch=y\r\n\r\n")
	f.Add("SIP/2.0 200 OK\r\nVia: x;branch=y\r\n\r\n")
	f.Add("garbage\r\n\r\n")
	f.Fuzz(func(t *testing.T, raw string) {
		_ = raw // ParseMessage must not panic
		m, err := ParseMessage(raw)
		if err == nil && m.Code() == 0 && m.Method() == "" && !strings.Contains(m.StartLine, "SIP/2.0") {
			t.Log("accepted non-SIP input without error")
		}
	})
}
