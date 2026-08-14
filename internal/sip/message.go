// Package sip implements the load-bearing subset of RFC 3261 a client needs
// to register, call, hold, resume and tear down against a mainstream PBX.
//
// The subset is deliberate: no proxy behaviour (§16), no S/MIME (§23), no
// presence, no session timers, no PRACK, no ICE. Every behaviour traces to
// the RFC section that forces it — see docs/matrix.md.
package sip

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// Message is a parsed SIP request or response (§7).
type Message struct {
	// StartLine is "METHOD ruri SIP/2.0" for requests and
	// "SIP/2.0 <code> <reason>" for responses.
	StartLine string
	// Headers preserves first-seen order and value per lowercased name.
	Headers []Header
	Body    string
}

// Header is one header field (§7.3.1).
type Header struct {
	Name  string // original case
	Value string
}

// Get returns the first value for a (case-insensitive) header name.
func (m *Message) Get(name string) string {
	low := strings.ToLower(name)
	for _, h := range m.Headers {
		if strings.ToLower(h.Name) == low {
			return h.Value
		}
	}
	return ""
}

// Code parses the response status code from the start line (§21). Returns 0
// for requests or malformed start lines.
func (m *Message) Code() int {
	parts := strings.Fields(m.StartLine)
	if len(parts) >= 2 {
		if c, err := strconv.Atoi(parts[1]); err == nil {
			return c
		}
	}
	return 0
}

// Method returns the request method from the start line, or "".
func (m *Message) Method() string {
	parts := strings.Fields(m.StartLine)
	if len(parts) >= 1 && !strings.HasPrefix(parts[0], "SIP/") {
		return parts[0]
	}
	return ""
}

// IsFinal reports whether this is a final response (>= 200).
func (m *Message) IsFinal() bool { return m.Code() >= 200 }

// IsProvisional reports whether this is a 1xx response.
func (m *Message) IsProvisional() bool { c := m.Code(); return c >= 100 && c < 200 }

// MatchesBranch reports whether the top Via carries the given branch
// parameter (§17.1.3 — responses are matched to transactions by branch).
func (m *Message) MatchesBranch(branch string) bool {
	return strings.Contains(m.Get("Via"), "branch="+branch)
}

// ParseMessage parses a raw SIP message (§7, §18.3 framing: CRLF headers,
// blank line, body).
func ParseMessage(raw string) (*Message, error) {
	parts := strings.SplitN(raw, "\r\n\r\n", 2)
	head := parts[0]
	m := &Message{}
	if len(parts) == 2 {
		m.Body = strings.TrimSuffix(parts[1], "\r\n")
	}
	lines := strings.Split(head, "\r\n")
	if len(lines) == 0 || lines[0] == "" {
		return nil, fmt.Errorf("empty message")
	}
	m.StartLine = lines[0]
	if !strings.Contains(m.StartLine, "SIP/2.0") {
		return nil, fmt.Errorf("not a SIP/2.0 message: %q", m.StartLine)
	}
	// §7.3.1: continuation lines start with SP/HT and fold into the previous
	// header's value.
	for i := 1; i < len(lines); i++ {
		ln := lines[i]
		if ln == "" {
			continue
		}
		if ln[0] == ' ' || ln[0] == '\t' {
			if len(m.Headers) > 0 {
				m.Headers[len(m.Headers)-1].Value += " " + strings.TrimSpace(ln)
			}
			continue
		}
		idx := strings.Index(ln, ":")
		if idx <= 0 {
			return nil, fmt.Errorf("malformed header line: %q", ln)
		}
		m.Headers = append(m.Headers, Header{
			Name:  strings.TrimSpace(ln[:idx]),
			Value: strings.TrimSpace(ln[idx+1:]),
		})
	}
	return m, nil
}

// String renders the message for transmission (§7.5).
func (m *Message) String() string {
	var b strings.Builder
	b.WriteString(m.StartLine + "\r\n")
	for _, h := range m.Headers {
		b.WriteString(h.Name + ": " + h.Value + "\r\n")
	}
	if m.Body != "" {
		b.WriteString("Content-Type: application/sdp\r\n")
		b.WriteString("Content-Length: " + strconv.Itoa(len(m.Body)) + "\r\n")
	} else {
		b.WriteString("Content-Length: 0\r\n")
	}
	b.WriteString("\r\n")
	b.WriteString(m.Body)
	return b.String()
}

// NewRequest builds a request with the six mandatory headers (§8.1.1):
// To, From, CSeq, Call-ID, Max-Forwards and Via.
func NewRequest(method, ruri, viaHost string, viaPort int, from, fromTag, to, toTag, callID string, cseq int, contact string) *Message {
	m := &Message{StartLine: fmt.Sprintf("%s %s SIP/2.0", method, ruri)}
	if toTag != "" {
		to = strings.TrimSuffix(to, ">") + ">;tag=" + toTag
	}
	m.Headers = []Header{
		{Name: "Via", Value: fmt.Sprintf("SIP/2.0/UDP %s:%d;branch=%s", viaHost, viaPort, NewBranch())},
		{Name: "Max-Forwards", Value: "70"},
		{Name: "From", Value: from + ";tag=" + fromTag},
		{Name: "To", Value: to},
		{Name: "Call-ID", Value: callID},
		{Name: "CSeq", Value: fmt.Sprintf("%d %s", cseq, method)},
	}
	if contact != "" {
		m.Headers = append(m.Headers, Header{Name: "Contact", Value: contact})
	}
	return m
}

// NewBranch returns a unique branch parameter value (§8.1.1.7).
func NewBranch() string { return "z9hG4bK" + randHex(10) }

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// BareURI extracts the bare SIP URI from a header value that may be a
// name-addr such as `"100" <sip:100@host>` (§19.1 — the Request-URI of
// in-dialog requests is the bare URI from the Contact of the 2xx).
func BareURI(v string) string {
	if i := strings.Index(v, "<"); i >= 0 {
		if j := strings.Index(v[i:], ">"); j >= 0 {
			return v[i+1 : i+j]
		}
	}
	return strings.TrimSpace(v)
}

// TagFrom returns the tag parameter of a To/From header value, if any (§19.3).
func TagFrom(v string) string {
	if i := strings.Index(v, "tag="); i >= 0 {
		t := v[i+4:]
		if j := strings.IndexAny(t, ">\";, "); j >= 0 {
			t = t[:j]
		}
		return t
	}
	return ""
}
