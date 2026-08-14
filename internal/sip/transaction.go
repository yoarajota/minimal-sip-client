package sip

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// Client transaction layer per RFC 3261 §17.1. Defaults:
//
//	T1 = 500 ms (RTT estimate)
//	T2 = 4 s   (non-INVITE retransmission cap)
//	T4 = 5 s   (network clearing time)
//	64×T1 = 32 s (transaction timeout: Timer B for INVITE, Timer F otherwise)
const (
	T1 = 500 * time.Millisecond
	T2 = 4 * time.Second
	T4 = 5 * time.Second
)

// TransactionError maps transaction-layer outcomes to SIP semantics
// (§8.1.3.1): a timeout is treated as 408, a transport failure as 503.
type TransactionError struct {
	Code int
	Msg  string
}

func (e *TransactionError) Error() string { return e.Msg }

// runTransaction executes one client transaction over UDP, bounded by stop.
//
// req is the raw request (already built). invite selects the INVITE state
// machine (§17.1.1: Timer A stops on 1xx; ACK generated for non-2xx finals)
// versus the non-INVITE one (§17.1.2: Timer E capped at T2). onProvisional
// receives every 1xx response. Returns the first final response.
//
// The caller (UAC core) generates the ACK for a 2xx to INVITE (§13.2.2.4);
// the ACK for any non-2xx final is generated here by the transaction layer
// (§17.1.1.3), with the same branch as the request.
func runTransaction(conn *net.UDPConn, server *net.UDPAddr, req []byte, invite bool, stop time.Time, onProvisional func(*Message)) (*Message, error) {
	branch := branchOf(req)
	interval := T1
	stopped := false // INVITE: retransmissions stop on 1xx (Calling→Proceeding)
	sent := 0
	for {
		if !stopped {
			if _, err := conn.WriteToUDP(req, server); err != nil {
				return nil, &TransactionError{Code: 503, Msg: "transport error: " + err.Error()}
			}
			sent++
		}
		// The read deadline is the earlier of the retransmission interval and
		// the transaction's stop time, so a stop (ctx deadline) is honoured
		// even while waiting for a final response after a 1xx.
		wait := interval
		if remaining := time.Until(stop); remaining < wait {
			wait = remaining
		}
		resp, err := readResponse(conn, branch, wait)
		if err != nil {
			return nil, err
		}
		if resp == nil { // timer fired (§17.1.1.2 Timer A/B, §17.1.2.2 Timer E/F)
			if time.Now().After(stop) {
				return nil, &TransactionError{Code: 408, Msg: fmt.Sprintf(
					"transaction timeout after %s (%d transmissions)", time.Until(stop).Round(time.Millisecond), sent)}
			}
			if stopped {
				continue
			}
			interval *= 2
			if interval > T2 {
				interval = T2
			}
			continue
		}
		if resp.IsProvisional() {
			if onProvisional != nil {
				onProvisional(resp)
			}
			if invite && !stopped {
				stopped = true
				interval = 32 * time.Second // wait for the final, no more retransmits
			}
			continue
		}
		// Final response.
		return handleFinal(conn, server, req, resp, invite), nil
	}
}

// handleFinal sends the transaction-layer ACK when the final response to an
// INVITE is 300–699 (§17.1.1.2, §17.1.1.3) and returns the response.
func handleFinal(conn *net.UDPConn, server *net.UDPAddr, req []byte, resp *Message, invite bool) *Message {
	if invite && resp.Code() >= 300 && resp.Code() <= 699 {
		// Same branch, same Call-ID/From/Request-URI, To from the response,
		// CSeq with method ACK.
		ack := ackForNon2xx(req, resp, branchOf(req))
		conn.WriteToUDP([]byte(ack), server) //nolint:errcheck // ACK is best-effort (§17.1.1.2)
	}
	return resp
}

// readResponse waits up to `wait` for a response matching the branch
// (§17.1.3). Returns nil, nil when the deadline fires without a match.
func readResponse(conn *net.UDPConn, branch string, wait time.Duration) (*Message, error) {
	conn.SetReadDeadline(time.Now().Add(wait)) //nolint:errcheck
	buf := make([]byte, 65535)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				return nil, nil
			}
			return nil, &TransactionError{Code: 503, Msg: "transport error: " + err.Error()}
		}
		resp, err := ParseMessage(string(buf[:n]))
		if err != nil || !resp.MatchesBranch(branch) {
			continue // not for this transaction (§17.1.3)
		}
		return resp, nil
	}
}

// branchOf extracts the branch parameter from a request's top Via (§17.1.3).
func branchOf(req []byte) string {
	m, err := ParseMessage(string(req))
	if err != nil {
		return ""
	}
	v := m.Get("Via")
	if i := strings.Index(v, "branch="); i >= 0 {
		return v[i+7:]
	}
	return ""
}

// ackForNon2xx builds the transaction-layer ACK for a 300–699 final response
// to an INVITE (§17.1.1.3).
func ackForNon2xx(original []byte, resp *Message, branch string) string {
	m, _ := ParseMessage(string(original))
	fields := strings.Fields(m.StartLine)
	ruri := ""
	if len(fields) >= 2 {
		ruri = fields[1]
	}
	cseqFields := strings.Fields(m.Get("CSeq"))
	seq := ""
	if len(cseqFields) >= 1 {
		seq = cseqFields[0]
	}
	ack := &Message{StartLine: "ACK " + ruri + " SIP/2.0"}
	ack.Headers = []Header{
		{Name: "Via", Value: m.Get("Via")},
		{Name: "Max-Forwards", Value: m.Get("Max-Forwards")},
		{Name: "From", Value: m.Get("From")},
		{Name: "To", Value: resp.Get("To")},
		{Name: "Call-ID", Value: m.Get("Call-ID")},
		{Name: "CSeq", Value: seq + " ACK"},
	}
	return ack.String()
}
