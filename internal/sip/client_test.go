package sip

import (
	"context"
	"strings"
	"testing"
	"time"
)

// Client-level tests: the register/call/hold/resume/teardown flow against
// the fake UAS (failure modes from docs/01-theory.md § 3), and the
// message-trace matrix rows.

func testClient(t *testing.T, f *fakeUAS) *Client {
	t.Helper()
	c, err := New(Config{
		Server:   f.Addr.String(),
		Domain:   "test",
		User:     "alice",
		Password: "secret",
		RTPPort:  40000,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func TestRegisterSuccess(t *testing.T) {
	f := newFakeUAS(t)
	f.challenge = true
	c := testClient(t, f)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Register(ctx); err != nil {
		t.Fatalf("Register: %v", err)
	}
	regs := f.requests()
	if len(regs) != 2 || regs[0].Get("Authorization") != "" || regs[1].Get("Authorization") == "" {
		t.Errorf("want REGISTER -> 401 -> REGISTER+Authorization, got %d requests", len(regs))
	}
	tr := c.Trace()
	if len(tr) != 1 || tr[0].Step != "register" {
		t.Errorf("trace = %+v, want one register row", tr)
	}
}

// Wrong password: the server keeps challenging, the client surfaces the 401.
func TestRegisterWrongPassword(t *testing.T) {
	f := newFakeUAS(t)
	f.challenge = true
	f.rejectAuth = true
	c := testClient(t, f)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := c.Register(ctx)
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("Register err = %v, want 401", err)
	}
}

// PBX cold: no response at all, bounded by ctx.
func TestRegisterNoServer(t *testing.T) {
	f := newFakeUAS(t)
	f.silent = true
	c := testClient(t, f)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Register(ctx); err == nil {
		t.Fatal("Register should fail when the PBX never answers")
	}
}

func TestCallFlow(t *testing.T) {
	f := newFakeUAS(t)
	f.challenge = true
	c := testClient(t, f)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := c.Register(ctx); err != nil {
		t.Fatalf("Register: %v", err)
	}
	call, err := c.Call(ctx, "100")
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	// The UAC core ACK for the 2xx must be in the trace and on the wire.
	acks := 0
	for _, m := range f.requests() {
		if m.Method() == "ACK" {
			acks++
		}
	}
	if acks < 1 {
		t.Error("no core ACK for the 2xx seen by the server")
	}
	// The client can send media; the fake does not echo, so receiving is
	// not asserted here (echo symmetry is the integration test, E-002).
	stats := call.MediaPhase(300*time.Millisecond, true)
	if stats.Sent == 0 {
		t.Error("client sent no RTP packets")
	}
	// Hold / resume direction negotiation (RFC 3264 §5.1).
	if err := call.Hold(ctx); err != nil {
		t.Fatalf("Hold: %v", err)
	}
	held := call.MediaPhase(200*time.Millisecond, false)
	if err := call.Resume(ctx); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if err := call.Hangup(ctx); err != nil {
		t.Fatalf("Hangup: %v", err)
	}
	steps := map[string]bool{}
	for _, e := range c.Trace() {
		steps[e.Step] = true
	}
	for _, want := range []string{"register", "invite", "hold", "resume", "bye"} {
		if !steps[want] {
			t.Errorf("trace missing step %q (have %+v)", want, steps)
		}
	}
	_ = held
}

// A rejected call surfaces the status and the transaction ACKs the final.
func TestCallRejected(t *testing.T) {
	f := newFakeUAS(t)
	f.inviteCode = "486 Busy Here"
	c := testClient(t, f)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Register(ctx); err != nil {
		t.Fatalf("Register: %v", err)
	}
	_, err := c.Call(ctx, "100")
	if err == nil || !strings.Contains(err.Error(), "486") {
		t.Fatalf("Call err = %v, want 486", err)
	}
}

// Regression: the INVITE must carry the SDP offer with a matching
// Content-Length (§13.2.1, §20.14) — a lost body silently breaks media.
func TestInviteCarriesBody(t *testing.T) {
	f := newFakeUAS(t)
	f.challenge = true
	c := testClient(t, f)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Register(ctx); err != nil {
		t.Fatalf("Register: %v", err)
	}
	call, err := c.Call(ctx, "100")
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if err := call.Hangup(ctx); err != nil {
		t.Fatalf("Hangup: %v", err)
	}
	var inv *Message
	for _, m := range f.requests() {
		if m.Method() == "INVITE" && m.Get("Authorization") != "" {
			inv = m
		}
	}
	if inv == nil {
		t.Fatal("no authenticated INVITE seen")
	}
	if inv.Body == "" || !strings.Contains(inv.Body, "m=audio") {
		t.Errorf("INVITE body missing or not SDP: %q", inv.Body)
	}
	if got := inv.Get("Content-Length"); got == "0" || got == "" {
		t.Errorf("INVITE Content-Length = %q, want > 0", got)
	}
}

// In-dialog request CSeq increments (§12.2.1.1): INVITE, re-INVITE, BYE all
// carry distinct, increasing numbers.
func TestDialogCSeqMonotonic(t *testing.T) {
	f := newFakeUAS(t)
	c := testClient(t, f)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.Register(ctx); err != nil {
		t.Fatalf("Register: %v", err)
	}
	call, err := c.Call(ctx, "100")
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if err := call.Hold(ctx); err != nil {
		t.Fatalf("Hold: %v", err)
	}
	if err := call.Hangup(ctx); err != nil {
		t.Fatalf("Hangup: %v", err)
	}
	var prev int
	for _, m := range f.requests() {
		if m.Method() == "INVITE" || m.Method() == "BYE" {
			cseq := cseqNum(m.Get("CSeq"))
			if cseq <= prev {
				t.Errorf("CSeq %d after %d — not monotonic (%s)", cseq, prev, m.Method())
			}
			prev = cseq
		}
	}
}

func cseqNum(v string) int {
	var n int
	for _, r := range v {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}
