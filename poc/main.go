// Minimal SIP client — P2 proof of concept.
//
// Demonstrates the critical function: a from-scratch RFC 3261 client registers,
// establishes a two-way RTP call, holds, resumes, and tears down against a real
// Asterisk PBX, and prints the message-trace matrix mapping each behaviour to
// the RFC section that forces it.
//
// Throwaway by design (workflow 04 § PoC): hardcoded flow, no error handling
// beyond fatal prints, no tests. The P3 component rewrite lives in src/.
package main

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Tiny SIP message model
// ---------------------------------------------------------------------------

type sipMsg struct {
	start   string
	headers map[string]string // lowercased name -> value (first occurrence)
	body    string
}

func parseMsg(raw string) *sipMsg {
	m := &sipMsg{headers: map[string]string{}}
	parts := strings.SplitN(raw, "\r\n\r\n", 2)
	head := parts[0]
	if len(parts) == 2 {
		m.body = strings.TrimSuffix(parts[1], "\r\n")
	}
	lines := strings.Split(head, "\r\n")
	if len(lines) == 0 {
		return nil
	}
	m.start = lines[0]
	for _, ln := range lines[1:] {
		idx := strings.Index(ln, ":")
		if idx <= 0 {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(ln[:idx]))
		val := strings.TrimSpace(ln[idx+1:])
		if _, ok := m.headers[name]; !ok {
			m.headers[name] = val
		}
	}
	return m
}

func (m *sipMsg) h(name string) string { return m.headers[strings.ToLower(name)] }

func (m *sipMsg) code() int {
	// "SIP/2.0 401 Unauthorized" -> 401
	parts := strings.Fields(m.start)
	if len(parts) >= 2 {
		if c, err := strconv.Atoi(parts[1]); err == nil {
			return c
		}
	}
	return 0
}

func (m *sipMsg) matchesBranch(branch string) bool {
	return strings.Contains(m.h("via"), "branch="+branch)
}

func randHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// bareURI extracts the bare sip URI from a Contact header value (which may be
// a name-addr like `"100" <sip:100@host>`). The Request-URI must be the bare URI.
func bareURI(contact string) string {
	if i := strings.Index(contact, "<"); i >= 0 {
		if j := strings.Index(contact[i:], ">"); j >= 0 {
			return contact[i+1 : i+j]
		}
	}
	return strings.TrimSpace(contact)
}

func md5hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

// ---------------------------------------------------------------------------
// UA state
// ---------------------------------------------------------------------------

type ua struct {
	server string // host:port of the PBX
	domain string // host of the PBX, without port (the AOR domain)
	user   string
	pass   string
	ext    string // extension to call

	localIP string
	sipConn *net.UDPConn
	rtpConn *net.UDPConn
	rtpDst  *net.UDPAddr

	aor, from, contact, callID string
	fromTag                    string
	to                         string // To header value, gains tag at 2xx
	toTag                      string
	remoteTarget               string // Contact URI of the far end (in-dialog requests)
	cseq                       int
	ssrc                       uint32
	seq                        uint16
	ts                         uint32
}

func newUA(server, user, pass, ext string) (*ua, error) {
	u := &ua{server: server, user: user, pass: pass, ext: ext, cseq: 0}
	if h, _, err := net.SplitHostPort(server); err == nil {
		u.domain = h
	} else {
		u.domain = server
	}
	saddr, err := net.ResolveUDPAddr("udp", server)
	if err != nil {
		return nil, err
	}
	// SIP socket, fixed port so the Contact is stable.
	u.sipConn, err = net.ListenUDP("udp", &net.UDPAddr{Port: 5060})
	if err != nil {
		return nil, err
	}
	// Discover our own address the way packets to the server will go.
	probe, err := net.DialUDP("udp", nil, saddr)
	if err != nil {
		return nil, err
	}
	u.localIP = probe.LocalAddr().(*net.UDPAddr).IP.String()
	probe.Close()
	// RTP socket, fixed port, advertised in the SDP offer.
	u.rtpConn, err = net.ListenUDP("udp", &net.UDPAddr{Port: 40000})
	if err != nil {
		return nil, err
	}
	u.aor = "sip:" + user + "@" + u.domain
	u.from = "Alice <sip:" + user + "@" + u.domain + ">"
	u.contact = "<sip:" + user + "@" + u.localIP + ":5060>"
	u.callID = randHex(16) + "@" + u.localIP
	u.fromTag = randHex(8)
	u.to = "<sip:" + user + "@" + u.domain + ">"
	u.ssrc = uint32(time.Now().UnixNano()) & 0x7fffffff
	u.seq = uint16(time.Now().UnixNano())
	u.ts = uint32(time.Now().UnixNano())
	return u, nil
}

func (u *ua) nextCSeq() int { u.cseq++; return u.cseq }

// buildRequest assembles a SIP request with the six mandatory headers.
func (u *ua) buildRequest(method, ruri, branch string, extra ...string) string {
	to := u.to
	if u.toTag != "" {
		// Tag belongs after the closing bracket: <sip:alice@host>;tag=xxx (§12.1.2).
		to = strings.TrimSuffix(to, ">") + ">;tag=" + u.toTag
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s SIP/2.0\r\n", method, ruri)
	fmt.Fprintf(&b, "Via: SIP/2.0/UDP %s:%d;branch=%s\r\n", u.localIP, 5060, branch)
	fmt.Fprintf(&b, "Max-Forwards: 70\r\n")
	fmt.Fprintf(&b, "From: %s;tag=%s\r\n", u.from, u.fromTag)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Call-ID: %s\r\n", u.callID)
	fmt.Fprintf(&b, "CSeq: %d %s\r\n", u.cseq, method)
	for _, e := range extra {
		b.WriteString(e + "\r\n")
	}
	return b.String()
}

// transaction sends a request and returns the first final response, retransmitting
// per RFC 3261 §17: Timer A at T1 doubling for INVITE until a 1xx; Timer E capped
// at T2 for non-INVITE; both bounded by 64*T1 = 32 s.
func (u *ua) transaction(method, ruri, body string, extra ...string) (*sipMsg, error) {
	branch := "z9hG4bK" + randHex(10)
	var hdrs []string
	hdrs = append(hdrs, extra...)
	if body != "" {
		hdrs = append(hdrs, "Content-Type: application/sdp", "Content-Length: "+strconv.Itoa(len(body)))
	} else {
		hdrs = append(hdrs, "Content-Length: 0")
	}
	if method == "INVITE" || method == "REGISTER" {
		hdrs = append(hdrs, "Contact: "+u.contact)
	}
	req := u.buildRequest(method, ruri, branch, hdrs...) + "\r\n" + body
	if os.Getenv("SIP_DEBUG") != "" {
		fmt.Printf("--- raw %s ---\n%s\n---\n", method, req)
	}
	server, _ := net.ResolveUDPAddr("udp", u.server)

	t1 := 500 * time.Millisecond
	interval := t1
	stop := time.Now().Add(64 * t1)
	stopped := false
	for {
		if !stopped {
			u.sipConn.WriteToUDP([]byte(req), server)
		}
		u.sipConn.SetReadDeadline(time.Now().Add(interval))
		buf := make([]byte, 65535)
		for {
			n, err := u.sipConn.Read(buf)
			if err != nil {
				break
			}
			resp := parseMsg(string(buf[:n]))
			if resp == nil || !resp.matchesBranch(branch) {
				continue
			}
			if resp.code() >= 200 {
				return resp, nil
			}
			// Provisional. INVITE stops retransmitting here (§17.1.1.2).
			if method == "INVITE" && !stopped {
				stopped = true
				interval = 32 * time.Second
			}
		}
		if time.Now().After(stop) {
			return nil, fmt.Errorf("transaction timeout on %s (64*T1 = 32s)", method)
		}
		if stopped {
			continue
		}
		interval *= 2
		if interval > 4*time.Second {
			interval = 4 * time.Second
		}
	}
}

// ---------------------------------------------------------------------------
// Digest authentication (§22.4)
// ---------------------------------------------------------------------------

func digestAuth(challenge, method, uri, user, pass string) string {
	params := map[string]string{}
	for _, kv := range strings.Split(strings.TrimPrefix(challenge, "Digest "), ",") {
		kv = strings.TrimSpace(kv)
		idx := strings.Index(kv, "=")
		if idx < 0 {
			continue
		}
		params[strings.TrimSpace(kv[:idx])] = strings.Trim(strings.TrimSpace(kv[idx+1:]), `"`)
	}
	realm, nonce := params["realm"], params["nonce"]
	ha1 := md5hex(user + ":" + realm + ":" + pass)
	ha2 := md5hex(method + ":" + uri)
	opaque := ""
	if params["opaque"] != "" {
		opaque = fmt.Sprintf(`, opaque="%s"`, params["opaque"])
	}
	if strings.Contains(params["qop"], "auth") {
		nc := "00000001"
		cnonce := randHex(8)
		response := md5hex(strings.Join([]string{ha1, nonce, nc, cnonce, "auth", ha2}, ":"))
		return fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", qop=auth, nc=%s, cnonce="%s", response="%s"%s`,
			user, realm, nonce, uri, nc, cnonce, response, opaque)
	}
	response := md5hex(ha1 + ":" + nonce + ":" + ha2)
	return fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", response="%s"%s`,
		user, realm, nonce, uri, response, opaque)
}

// ---------------------------------------------------------------------------
// SDP (RFC 4566) and RTP (RFC 3550)
// ---------------------------------------------------------------------------

func sdpOffer(ip string, port int, dir string) string {
	return fmt.Sprintf("v=0\r\no=alice %d %d IN IP4 %s\r\ns=-\r\nc=IN IP4 %s\r\nt=0 0\r\nm=audio %d RTP/AVP 0\r\na=rtpmap:0 PCMU/8000\r\na=%s\r\n",
		time.Now().UnixNano()%1000000, time.Now().UnixNano()%1000000, ip, ip, port, dir)
}

type sdpAnswer struct {
	ip   string
	port int
	dir  string
}

func parseSDPAnswer(body string) (sdpAnswer, error) {
	a := sdpAnswer{dir: "sendrecv"}
	for _, ln := range strings.Split(body, "\r\n") {
		switch {
		case strings.HasPrefix(ln, "c=IN IP4 "):
			a.ip = strings.TrimPrefix(ln, "c=IN IP4 ")
		case strings.HasPrefix(ln, "m=audio "):
			parts := strings.Fields(strings.TrimPrefix(ln, "m=audio "))
			if len(parts) >= 1 {
				a.port, _ = strconv.Atoi(parts[0])
			}
		case ln == "a=sendonly":
			a.dir = "sendonly"
		case ln == "a=recvonly":
			a.dir = "recvonly"
		case ln == "a=inactive":
			a.dir = "inactive"
		}
	}
	if a.ip == "" || a.port == 0 {
		return a, fmt.Errorf("SDP answer missing c= or m= line")
	}
	return a, nil
}

// ulaw encodes one 16-bit signed sample as μ-law (G.711).
func ulaw(s int16) byte {
	u := int(math.Abs(float64(s)))
	const clip = 32635
	if u > clip {
		u = clip
	}
	mag := (u + 132) >> 7 // 0..255
	seg := 0
	for m := mag; m > 1; m >>= 1 {
		seg++
	}
	val := (seg << 4) | ((mag >> (seg + 1)) & 0x0f)
	if s < 0 {
		return byte(^val & 0x7f)
	}
	return byte(val | 0x80)
}

func tonePkt(seq uint16, ts uint32, ssrc uint32) []byte {
	hdr := make([]byte, 12)
	hdr[0] = 0x80 // V=2, no padding/extension/CSRC
	hdr[1] = 0x80 // marker=0, PT=0 (PCMU)
	hdr[2], hdr[3] = byte(seq>>8), byte(seq)
	hdr[4], hdr[5], hdr[6], hdr[7] = byte(ts>>24), byte(ts>>16), byte(ts>>8), byte(ts)
	hdr[8], hdr[9], hdr[10], hdr[11] = byte(ssrc>>24), byte(ssrc>>16), byte(ssrc>>8), byte(ssrc)
	payload := make([]byte, 160) // 20 ms of 8 kHz
	for i := range payload {
		s := int16(8000 * math.Sin(2*math.Pi*440*float64(i)/8000))
		payload[i] = ulaw(s)
	}
	return append(hdr, payload...)
}

// ---------------------------------------------------------------------------
// Scenario steps
// ---------------------------------------------------------------------------

func (u *ua) regURI() string { return "sip:" + u.domain }

func (u *ua) waitReady(maxWait time.Duration) error {
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		u.nextCSeq()
		_, err := u.transaction("REGISTER", u.regURI(), "")
		if err == nil {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("PBX not reachable within %s", maxWait)
}

func (u *ua) register() error {
	// First pass: expect 401 challenge.
	u.nextCSeq()
	resp, err := u.transaction("REGISTER", u.regURI(), "")
	if err != nil {
		return err
	}
	code := resp.code()
	if code == 200 {
		return nil // PBX does not require auth
	}
	if code != 401 {
		return fmt.Errorf("REGISTER got %d, expected 401 or 200", code)
	}
	challenge := resp.h("www-authenticate")
	auth := digestAuth(challenge, "REGISTER", u.regURI(), u.user, u.pass)
	u.nextCSeq()
	resp2, err := u.transaction("REGISTER", u.regURI(), "", "Authorization: "+auth)
	if err != nil {
		return err
	}
	if resp2.code() != 200 {
		return fmt.Errorf("authenticated REGISTER got %d, expected 200", resp2.code())
	}
	fmt.Printf("  REGISTER  -> 401 challenge -> REGISTER+Authorization -> 200 OK  [RFC 3261 §10.2, §22.4]\n")
	return nil
}

func (u *ua) invite() (*sdpAnswer, error) {
	u.nextCSeq()
	ruri := "sip:" + u.ext + "@" + u.domain
	body := sdpOffer(u.localIP, 40000, "sendrecv")
	resp, err := u.transaction("INVITE", ruri, body)
	if err != nil {
		return nil, err
	}
	code := resp.code()
	if code == 401 || code == 407 {
		challenge := resp.h("www-authenticate")
		if challenge == "" {
			challenge = resp.h("proxy-authenticate")
		}
		auth := digestAuth(challenge, "INVITE", ruri, u.user, u.pass)
		u.nextCSeq()
		body = sdpOffer(u.localIP, 40000, "sendrecv")
		resp, err = u.transaction("INVITE", ruri, body, "Authorization: "+auth)
		if err != nil {
			return nil, err
		}
		code = resp.code()
	}
	if code != 200 {
		return nil, fmt.Errorf("INVITE got %d (%s), expected 200", code, strings.Join(strings.Fields(resp.start)[2:], " "))
	}
	// 2xx: dialog confirmed — capture To tag and remote target (§12.1.2, §13.2.2.4).
	toVal := resp.h("to")
	if i := strings.Index(toVal, "tag="); i >= 0 {
		u.toTag = strings.TrimPrefix(toVal[i:], "tag=")
		u.toTag = strings.Trim(u.toTag, ">\"")
	}
	if c := resp.h("contact"); c != "" {
		u.remoteTarget = bareURI(c)
	}
	answer, err := parseSDPAnswer(resp.body)
	if err != nil {
		return nil, err
	}
	// ACK for the 2xx is generated by the UAC core (§13.2.2.4): same CSeq number, method ACK.
	u.ack(200)
	fmt.Printf("  INVITE -> 180 -> 200 OK (SDP answer %s:%d) -> ACK  [RFC 3261 §13.2, §8.1.1.8]\n", answer.ip, answer.port)
	return &answer, nil
}

func (u *ua) ack(finalCode int) {
	// ACK reuses the INVITE CSeq number with method ACK (§13.2.2.4).
	method := "ACK"
	branch := "z9hG4bK" + randHex(10)
	to := strings.TrimSuffix(u.to, ">") + ">;tag=" + u.toTag
	var b strings.Builder
	fmt.Fprintf(&b, "ACK %s SIP/2.0\r\n", u.remoteTarget)
	fmt.Fprintf(&b, "Via: SIP/2.0/UDP %s:%d;branch=%s\r\n", u.localIP, 5060, branch)
	fmt.Fprintf(&b, "Max-Forwards: 70\r\n")
	fmt.Fprintf(&b, "From: %s;tag=%s\r\n", u.from, u.fromTag)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Call-ID: %s\r\n", u.callID)
	fmt.Fprintf(&b, "CSeq: %d %s\r\n", u.cseq, method)
	if finalCode == 200 {
		b.WriteString("Content-Length: 0\r\n")
	}
	b.WriteString("\r\n")
	server, _ := net.ResolveUDPAddr("udp", u.server)
	u.sipConn.WriteToUDP([]byte(b.String()), server)
}

func (u *ua) reinvite(dir string) (*sdpAnswer, error) {
	ruri := u.remoteTarget
	sendInvite := func() (*sipMsg, error) {
		u.nextCSeq()
		return u.transaction("INVITE", ruri, sdpOffer(u.localIP, 40000, dir))
	}
	resp, err := sendInvite()
	if err != nil {
		return nil, err
	}
	if resp.code() == 401 || resp.code() == 407 {
		challenge := resp.h("www-authenticate")
		if challenge == "" {
			challenge = resp.h("proxy-authenticate")
		}
		auth := digestAuth(challenge, "INVITE", ruri, u.user, u.pass)
		u.nextCSeq()
		resp, err = u.transaction("INVITE", ruri, sdpOffer(u.localIP, 40000, dir), "Authorization: "+auth)
		if err != nil {
			return nil, err
		}
	}
	if resp.code() != 200 {
		return nil, fmt.Errorf("re-INVITE(%s) got %d, expected 200", dir, resp.code())
	}
	u.ack(200)
	answer, err := parseSDPAnswer(resp.body)
	if err != nil {
		return nil, err
	}
	fmt.Printf("  re-INVITE(%s) -> 200 OK (answer %s) -> ACK  [RFC 3261 §13, RFC 3264 §5.1]\n", dir, answer.dir)
	return &answer, nil
}

func (u *ua) bye() error {
	u.nextCSeq()
	ruri := u.remoteTarget
	resp, err := u.transaction("BYE", ruri, "")
	if err != nil {
		return err
	}
	if resp.code() != 200 {
		return fmt.Errorf("BYE got %d, expected 200", resp.code())
	}
	fmt.Printf("  BYE -> 200 OK  [RFC 3261 §15.1]\n")
	return nil
}

// mediaPhase sends a 440 Hz PCMU tone for `dur` and counts received RTP packets.
// The reader goroutine has a fixed deadline (phase end + 1.5 s) so it always exits.
func (u *ua) mediaPhase(dur time.Duration, send bool) (sent, recv int) {
	end := time.Now().Add(dur)
	ticker := time.NewTicker(20 * time.Millisecond) // 50 pkt/s, 160 samples each
	defer ticker.Stop()
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 2048)
		u.rtpConn.SetReadDeadline(end.Add(1500 * time.Millisecond))
		for {
			n, _, err := u.rtpConn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			if n >= 12 {
				recv++
			}
		}
	}()
	for send {
		select {
		case <-ticker.C:
			pkt := tonePkt(u.seq, u.ts, u.ssrc)
			if _, err := u.rtpConn.WriteToUDP(pkt, u.rtpDst); err == nil {
				sent++
			}
			u.seq++
			u.ts += 160
		case <-time.After(time.Until(end) + 50*time.Millisecond):
			send = false
		}
	}
	<-done // reader exits when its deadline passes
	return sent, recv
}

// ---------------------------------------------------------------------------

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FAIL: "+format+"\n", args...)
	os.Exit(1)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	server := os.Getenv("SIP_SERVER")
	if server == "" {
		server = "asterisk:5060"
	}
	user := getenv("SIP_USER", "alice")
	pass := getenv("SIP_PASS", "secret")
	ext := getenv("SIP_EXT", "100")

	u, err := newUA(server, user, pass, ext)
	if err != nil {
		fail("%v", err)
	}
	defer u.sipConn.Close()
	defer u.rtpConn.Close()

	fmt.Println("minimal SIP client PoC — scenario suite against", server)
	fmt.Printf("  UA: %s  Contact: %s  RTP: %s:40000\n", u.aor, u.contact, u.localIP)
	fmt.Println()

	fmt.Println("Step 1/5  register")
	if err := u.waitReady(60 * time.Second); err != nil {
		fail("%v", err)
	}
	if err := u.register(); err != nil {
		fail("%v", err)
	}

	fmt.Println("Step 2/5  establish two-way RTP call")
	ans, err := u.invite()
	if err != nil {
		fail("%v", err)
	}
	u.rtpDst, _ = net.ResolveUDPAddr("udp", net.JoinHostPort(ans.ip, strconv.Itoa(ans.port)))

	fmt.Println("Step 3/5  media flows (send tone, expect echo)")
	s1, r1 := u.mediaPhase(3*time.Second, true)
	fmt.Printf("  sent %d RTP packets, received %d — two-way media %s  [RFC 3550 §5.1]\n",
		s1, r1, map[bool]string{true: "OK", false: "MISSING"}[r1 > 0])
	if r1 == 0 {
		fail("no RTP received during active call — media path broken")
	}

	fmt.Println("Step 4/5  hold / resume")
	holdAns, err := u.reinvite("sendonly")
	if err != nil {
		fail("%v", err)
	}
	_, rHold := u.mediaPhase(2*time.Second, false) // held: we send nothing
	resAns, err := u.reinvite("sendrecv")
	if err != nil {
		fail("%v", err)
	}
	u.rtpDst, _ = net.ResolveUDPAddr("udp", net.JoinHostPort(resAns.ip, strconv.Itoa(resAns.port)))
	s2, r2 := u.mediaPhase(2*time.Second, true)
	fmt.Printf("  hold: answer %s, received during hold %d (expected ~0); resume: answer %s, sent %d, received %d — %s\n",
		holdAns.dir, rHold, resAns.dir, s2, r2, map[bool]string{true: "resumed OK", false: "NO MEDIA AFTER RESUME"}[r2 > 0])
	if r2 == 0 {
		fail("no RTP received after resume")
	}

	fmt.Println("Step 5/5  teardown")
	if err := u.bye(); err != nil {
		fail("%v", err)
	}

	fmt.Println()
	fmt.Println("SUITE PASSED: register -> call -> hold -> resume -> teardown against a real PBX")
}
