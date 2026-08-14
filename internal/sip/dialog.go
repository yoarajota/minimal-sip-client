package sip

import (
	"context"
	"fmt"
	"time"
)

// Call is one established dialog (§12): a confirmed two-way RTP session.
// In-dialog requests (re-INVITE, BYE) are built per §12.2.1.1: Request-URI =
// the remote target (Contact of the 2xx), To carries the remote tag, From the
// local tag, and CSeq increments per request.
type Call struct {
	client       *Client
	callID       string
	fromTag      string
	toTag        string
	to           string // AOR; gains the remote tag for in-dialog requests
	remoteTarget string // bare Contact URI of the far end
	cseq         int
	stream       *Stream
	lastStats    MediaStats
	closed       bool
}

// ack generates the UAC-core ACK for a 2xx to an INVITE (§13.2.2.4): same
// CSeq number, method ACK, To with the remote tag, sent directly to the
// transaction's destination.
func (c *Call) ack(resp *Message) {
	m := NewRequest("ACK", c.remoteTarget, c.client.localIP, 5060, c.client.from, c.fromTag, c.to, c.toTag, c.callID, c.cseq, "")
	server := c.client.server
	c.client.conn.WriteToUDP([]byte(m.String()), server) //nolint:errcheck // ACK is best-effort (§13.2.2.4)
}

// request builds an in-dialog request (§12.2.1.1).
func (c *Call) request(method string, body, auth string) *Message {
	m := NewRequest(method, c.remoteTarget, c.client.localIP, 5060, c.client.from, c.fromTag, c.to, c.toTag, c.callID, c.cseq, "")
	m.Body = body
	if method == "INVITE" {
		m.Headers = append(m.Headers, Header{Name: "Contact", Value: c.client.contact()})
	}
	if auth != "" {
		m.Headers = append(m.Headers, Header{Name: "Authorization", Value: auth})
	}
	return m
}

// reinvite sends an in-dialog re-INVITE with the given SDP direction
// (RFC 3264 §5.1 — the hold/resume mechanism), handles a 401 challenge
// (§22.4, observed from Asterisk on out-of-dialog INVITEs and kept defensive
// here), and updates the RTP target from the answer.
func (c *Call) reinvite(ctx context.Context, dir Direction, step string) error {
	send := func(auth string) (*Message, error) {
		c.cseq++
		body := Offer(c.client.localIP, c.client.cfg.RTPPort, dir)
		req := c.request("INVITE", body, auth).String()
		return runTransaction(c.client.conn, c.client.server, []byte(req), true,
			transactionDeadline(ctx), nil)
	}
	resp, err := send("")
	if err != nil {
		return err
	}
	if resp.Code() == 401 || resp.Code() == 407 {
		challenge := resp.Get("WWW-Authenticate")
		if challenge == "" {
			challenge = resp.Get("Proxy-Authenticate")
		}
		if challenge == "" {
			return fmt.Errorf("sip: %d without a challenge", resp.Code())
		}
		auth := Authorization(challenge, "INVITE", c.remoteTarget, c.client.cfg.User, c.client.cfg.Password)
		resp, err = send(auth)
		if err != nil {
			return err
		}
	}
	if resp.Code() != 200 {
		return fmt.Errorf("sip: re-INVITE(%s) got %d", dir, resp.Code())
	}
	c.ack(resp)
	answer, err := ParseAnswer(resp.Body)
	if err != nil {
		return err
	}
	if err := c.stream.SetTarget(answer.IP, answer.Port); err != nil {
		return err
	}
	c.client.record(TraceEntry{Step: step, Method: "INVITE",
		RFCRefs: []string{"§13.2.1", "RFC 3264 §5.1"},
		Detail:  fmt.Sprintf("re-INVITE(%s) -> 200 (answer %s) -> ACK", dir, answer.Dir)})
	return nil
}

// Hold puts the far end on hold: a re-INVITE whose offer is sendonly
// (RFC 3264 §5.1). The client stops sending media while held.
func (c *Call) Hold(ctx context.Context) error {
	if c.closed {
		return fmt.Errorf("sip: call is closed")
	}
	return c.reinvite(ctx, SendOnly, "hold")
}

// Resume lifts the hold: a re-INVITE whose offer is sendrecv.
func (c *Call) Resume(ctx context.Context) error {
	if c.closed {
		return fmt.Errorf("sip: call is closed")
	}
	return c.reinvite(ctx, SendRecv, "resume")
}

// Hangup ends the session with a BYE (§15.1.1) and stops media. Per §15.1.1
// the session is terminated as soon as the BYE is passed to the transaction,
// even if the response is 481, 408 or absent.
func (c *Call) Hangup(ctx context.Context) error {
	if c.closed {
		return nil
	}
	c.cseq++
	req := c.request("BYE", "", "").String()
	resp, err := runTransaction(c.client.conn, c.client.server, []byte(req), false,
		transactionDeadline(ctx), nil)
	c.closed = true
	c.stream.Close() //nolint:errcheck
	if err != nil {
		return err
	}
	c.client.record(TraceEntry{Step: "bye", Method: "BYE",
		RFCRefs: []string{"§15.1.1"},
		Detail:  fmt.Sprintf("BYE -> %d", resp.Code())})
	return nil
}

// MediaPhase runs one media phase: sends a 440 Hz PCMU tone for `dur` when
// send is true, counts received packets (window dur + 1.5 s), and stores the
// counts as the call's last media stats.
func (c *Call) MediaPhase(dur time.Duration, send bool) MediaStats {
	c.lastStats = c.stream.Phase(dur, send)
	return c.lastStats
}

// Media returns the most recent phase's packet counts.
func (c *Call) Media() MediaStats { return c.lastStats }
