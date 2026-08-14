package sip

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// Client is a minimal RFC 3261 user agent client (D-001). Synchronous and
// per-dialog: one registration, one call at a time.
type Client struct {
	cfg     Config
	localIP string
	conn    *net.UDPConn
	server  *net.UDPAddr

	// Registration identity (§8.1.1): From with local tag, To = address of
	// record, REGISTER Call-ID and CSeq.
	from    string
	fromTag string
	to      string
	regCall string
	cseq    int

	trace   []TraceEntry
	traceMu sync.Mutex
}

// Config is the minimal set of facts a client needs to operate.
type Config struct {
	Server   string // PBX outbound address, "host:port"
	Domain   string // AOR domain for To/From and the REGISTER Request-URI
	User     string // AOR user part
	Password string // digest password
	RTPPort  int    // local RTP port, advertised in the SDP offer
}

// New opens the SIP socket and resolves the server. It does not touch the
// network beyond binding the local socket.
func New(cfg Config) (*Client, error) {
	if cfg.Server == "" || cfg.Domain == "" || cfg.User == "" {
		return nil, fmt.Errorf("sip: Server, Domain and User are required")
	}
	server, err := net.ResolveUDPAddr("udp", cfg.Server)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: 5060})
	if err != nil {
		return nil, err
	}
	probe, err := net.DialUDP("udp", nil, server)
	if err != nil {
		conn.Close() //nolint:errcheck // best-effort cleanup on the error path
		return nil, err
	}
	localIP := probe.LocalAddr().(*net.UDPAddr).IP.String()
	probe.Close() //nolint:errcheck

	c := &Client{
		cfg:     cfg,
		localIP: localIP,
		conn:    conn,
		server:  server,
		from:    cfg.User + " <sip:" + cfg.User + "@" + cfg.Domain + ">",
		to:      "<sip:" + cfg.User + "@" + cfg.Domain + ">",
	}
	c.fromTag = randHex(8)
	c.regCall = randHex(16) + "@" + localIP
	return c, nil
}

// Close releases the SIP socket.
func (c *Client) Close() error { return c.conn.Close() }

func (c *Client) nextCSeq() int { c.cseq++; return c.cseq }

// regURI is the REGISTER Request-URI: the domain only, no userinfo (§10.2).
func (c *Client) regURI() string { return "sip:" + c.cfg.Domain }

// contact is the client's Contact URI: its own address (§8.1.1.8).
func (c *Client) contact() string {
	return "<sip:" + c.cfg.User + "@" + c.localIP + ":5060>"
}

// request builds a request with the six mandatory headers (§8.1.1), optional
// Contact (INVITE/REGISTER), Authorization, and an SDP body.
func (c *Client) request(callID, method, ruri string, cseq int, toTag, contact, body, auth string) *Message {
	m := NewRequest(method, ruri, c.localIP, 5060, c.from, c.fromTag, c.to, toTag, callID, cseq, contact)
	m.Body = body
	if auth != "" {
		m.Headers = append(m.Headers, Header{Name: "Authorization", Value: auth})
	}
	if os.Getenv("SIP_DEBUG") != "" {
		fmt.Printf("--- raw %s ---\n%s\n---\n", method, m.String())
	}
	return m
}

// send runs one transaction and handles a 401/407 challenge by retrying with
// an Authorization header (§22.4, §8.1.3.5). Each transaction consumes a fresh
// CSeq from the client's sequence (§8.1.1.5). Returns the final response and
// the CSeq of the transaction that produced it.
func (c *Client) send(ctx context.Context, callID, method, ruri string, toTag, contact, body string) (*Message, int, error) {
	stop := transactionDeadline(ctx)
	cseq := c.nextCSeq()
	req := c.request(callID, method, ruri, cseq, toTag, contact, body, "").String()
	resp, err := runTransaction(c.conn, c.server, []byte(req), method == "INVITE", stop, nil)
	if err != nil {
		return nil, 0, err
	}
	if resp.Code() != 401 && resp.Code() != 407 {
		return resp, cseq, nil
	}
	challenge := resp.Get("WWW-Authenticate")
	if challenge == "" {
		challenge = resp.Get("Proxy-Authenticate")
	}
	if challenge == "" {
		return nil, 0, fmt.Errorf("sip: %d without a challenge", resp.Code())
	}
	auth := Authorization(challenge, method, ruri, c.cfg.User, c.cfg.Password)
	cseq = c.nextCSeq()
	req = c.request(callID, method, ruri, cseq, toTag, contact, body, auth).String()
	resp, err = runTransaction(c.conn, c.server, []byte(req), method == "INVITE", stop, nil)
	return resp, cseq, err
}

// transactionDeadline bounds a transaction by 64×T1 (§17) and any context
// deadline, whichever is sooner.
func transactionDeadline(ctx context.Context) time.Time {
	stop := time.Now().Add(64 * T1)
	if d, ok := ctx.Deadline(); ok && d.Before(stop) {
		stop = d
	}
	return stop
}

// Register performs REGISTER → (401 challenge) → REGISTER+Authorization →
// 200 (§10.2, §22.4). While the PBX has not answered at all (a cold
// container), it retries until ctx expires; a definitive non-200 final (for
// example a failed authentication) is an error.
func (c *Client) Register(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		resp, _, err := c.send(ctx, c.regCall, "REGISTER", c.regURI(), "", c.contact(), "")
		if err == nil && resp.Code() == 200 {
			c.record(TraceEntry{Step: "register", Method: "REGISTER",
				RFCRefs: []string{"§10.2", "§8.1.1", "§22.4"},
				Detail:  "401 challenge -> REGISTER+Authorization -> 200"})
			return nil
		}
		if err == nil {
			return fmt.Errorf("sip: REGISTER got %d", resp.Code())
		}
		select {
		case <-time.After(2 * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Call establishes a two-way RTP call to target (an extension such as "100")
// and returns the live call (§13.1, §13.2). The offer is in the INVITE; the
// answer in the 2xx; the ACK is generated by the UAC core (§13.2.2.4).
func (c *Client) Call(ctx context.Context, target string) (*Call, error) {
	ruri := "sip:" + target + "@" + c.cfg.Domain
	body := Offer(c.localIP, c.cfg.RTPPort, SendRecv)
	callID := randHex(16) + "@" + c.localIP
	resp, cseq, err := c.send(ctx, callID, "INVITE", ruri, "", c.contact(), body)
	if err != nil {
		return nil, err
	}
	if resp.Code() != 200 {
		return nil, fmt.Errorf("sip: INVITE got %d (%s)", resp.Code(), reason(resp))
	}
	// Dialog confirmed (§12.1.2): To tag from the response, remote target
	// from its Contact.
	call := &Call{
		client:       c,
		callID:       callID,
		fromTag:      c.fromTag,
		toTag:        TagFrom(resp.Get("To")),
		to:           c.to,
		remoteTarget: BareURI(resp.Get("Contact")),
		cseq:         cseq,
	}
	call.ack(resp)
	answer, err := ParseAnswer(resp.Body)
	if err != nil {
		return nil, err
	}
	stream, err := NewStream(c.cfg.RTPPort, randSSRC())
	if err != nil {
		return nil, err
	}
	if err := stream.SetTarget(answer.IP, answer.Port); err != nil {
		stream.Close() //nolint:errcheck
		return nil, err
	}
	call.stream = stream
	c.record(TraceEntry{Step: "invite", Method: "INVITE",
		RFCRefs: []string{"§13.2.1", "§13.2.2.4", "§8.1.1.8", "§12.1.2"},
		Detail:  fmt.Sprintf("180 -> 200 (SDP answer %s:%d %s) -> ACK", answer.IP, answer.Port, answer.Dir)})
	return call, nil
}

func reason(resp *Message) string {
	parts := strings.Fields(resp.StartLine)
	if len(parts) >= 3 {
		return strings.Join(parts[2:], " ")
	}
	return ""
}

func randSSRC() uint32 { return uint32(time.Now().UnixNano()) & 0x7fffffff }
