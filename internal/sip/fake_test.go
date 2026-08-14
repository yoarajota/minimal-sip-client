package sip

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeUAS is a configurable minimal SIP server for tests. It answers
// REGISTER/INVITE/BYE the way a real PBX does (401 digest challenge,
// 200 with SDP answer and To tag, 200 to BYE), and can be told to drop
// requests, never respond, or answer with a wrong branch — the failure modes
// from docs/01-theory.md § 3.
type fakeUAS struct {
	t          *testing.T
	conn       *net.UDPConn
	Addr       *net.UDPAddr // client-facing address
	mu         sync.Mutex
	seen       []*Message
	dropN      int    // drop the first N requests (retransmission test)
	silent     bool   // never respond (timeout test)
	challenge  bool   // 401 before accepting REGISTER and INVITE
	rejectAuth bool   // 401 even with Authorization (wrong-password test)
	inviteCode string // status for INVITE finals, e.g. "404 Not Found"
	wrongBr    bool   // respond with a branch that does not match
}

func newFakeUAS(t *testing.T) *fakeUAS {
	t.Helper()
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: 0})
	if err != nil {
		t.Fatalf("fake UAS listen: %v", err)
	}
	f := &fakeUAS{t: t, conn: conn, Addr: conn.LocalAddr().(*net.UDPAddr)}
	go f.serve()
	t.Cleanup(func() { conn.Close() })
	return f
}

func (f *fakeUAS) requests() []*Message {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*Message, len(f.seen))
	copy(out, f.seen)
	return out
}

func (f *fakeUAS) countMethod(method string) int {
	n := 0
	for _, m := range f.requests() {
		if m.Method() == method {
			n++
		}
	}
	return n
}

// waitFor polls until the server has recorded at least n requests with the
// given method (responses are recorded by an async reader goroutine).
func (f *fakeUAS) waitFor(t *testing.T, method string, n int) []*Message {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if ms := f.requests(); len(ms) >= n {
			out := []*Message{}
			for _, m := range ms {
				if m.Method() == method {
					out = append(out, m)
				}
			}
			if len(out) >= n {
				return out
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

func (f *fakeUAS) serve() {
	buf := make([]byte, 65535)
	for {
		n, peer, err := f.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		req, err := ParseMessage(string(buf[:n]))
		if err != nil {
			continue
		}
		f.mu.Lock()
		f.seen = append(f.seen, req)
		drop := f.dropN > 0
		if drop {
			f.dropN--
		}
		silent := f.silent
		f.mu.Unlock()
		if drop || silent {
			continue
		}
		res := f.buildResponse(req)
		if res != "" {
			f.conn.WriteToUDP([]byte(res), peer) //nolint:errcheck
		}
	}
}

func (f *fakeUAS) authed(req *Message) bool {
	return req.Get("Authorization") != ""
}

func (f *fakeUAS) buildResponse(req *Message) string {
	branch := ""
	if v := req.Get("Via"); v != "" {
		if i := strings.Index(v, "branch="); i >= 0 {
			branch = v[i+7:]
		}
	}
	if f.wrongBr {
		branch = "z9hG4bK-bogus"
	}
	toTagged := addTag(req.Get("To"), "server-tag")
	switch req.Method() {
	case "REGISTER":
		if f.challenge && (!f.authed(req) || f.rejectAuth) {
			return response("401 Unauthorized", branch, toTagged, req,
				`WWW-Authenticate: Digest realm="test", nonce="nonce-1", qop="auth"`, "")
		}
		return response("200 OK", branch, toTagged, req, "", "")
	case "INVITE":
		if f.challenge && (!f.authed(req) || f.rejectAuth) {
			return response("401 Unauthorized", branch, toTagged, req,
				`WWW-Authenticate: Digest realm="test", nonce="nonce-1", qop="auth"`, "")
		}
		if f.inviteCode != "" {
			return response(f.inviteCode, branch, toTagged, req, "", "")
		}
		body := "v=0\r\no=- 1 1 IN IP4 127.0.0.1\r\ns=-\r\nc=IN IP4 127.0.0.1\r\nt=0 0\r\nm=audio 5004 RTP/AVP 0\r\na=rtpmap:0 PCMU/8000\r\na=sendrecv\r\n"
		return response("200 OK", branch, toTagged, req, "Contact: <sip:100@127.0.0.1:5060>", body)
	case "ACK":
		return "" // ACKs are not answered
	case "BYE", "CANCEL":
		return response("200 OK", branch, toTagged, req, "", "")
	default:
		return response("405 Method Not Allowed", branch, toTagged, req, "", "")
	}
}

func response(status, branch, to string, req *Message, extraHeaders, body string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "SIP/2.0 %s\r\n", status)
	fmt.Fprintf(&b, "Via: %s;received=127.0.0.1\r\n", branchVal(req, branch))
	fmt.Fprintf(&b, "From: %s\r\n", req.Get("From"))
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Call-ID: %s\r\n", req.Get("Call-ID"))
	fmt.Fprintf(&b, "CSeq: %s\r\n", req.Get("CSeq"))
	if extraHeaders != "" {
		b.WriteString(extraHeaders + "\r\n")
	}
	if body != "" {
		fmt.Fprintf(&b, "Content-Type: application/sdp\r\nContent-Length: %d\r\n\r\n%s", len(body), body)
	} else {
		b.WriteString("Content-Length: 0\r\n\r\n")
	}
	return b.String()
}

func branchVal(req *Message, branch string) string {
	v := req.Get("Via")
	if branch != "" {
		if i := strings.Index(v, "branch="); i >= 0 {
			return v[:i+7] + branch
		}
	}
	return v
}

func addTag(v, tag string) string {
	if strings.Contains(v, "tag=") {
		return v
	}
	return strings.TrimSuffix(v, ">") + ">;tag=" + tag
}
