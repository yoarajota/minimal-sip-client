//go:build integration

// Integration tests against a real PBX. Run with:
//
//	docker compose up --build --abort-on-container-exit --exit-code-from client
//
// which starts the pinned Asterisk 20 container and this test as the client
// service (SIP_SERVER=asterisk:5060). S-001's verification path.
package sip

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func suiteClient(t *testing.T) *Client {
	t.Helper()
	server := getenv("SIP_SERVER", "asterisk:5060")
	domain := getenv("SIP_DOMAIN", "asterisk")
	user := getenv("SIP_USER", "alice")
	pass := getenv("SIP_PASS", "secret")
	rtpPort := 40000
	if v := os.Getenv("SIP_RTP_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &rtpPort)
	}
	c, err := New(Config{Server: server, Domain: domain, User: user, Password: pass, RTPPort: rtpPort})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

// TestSuiteIntegration is the scenario suite against the real PBX: register,
// two-way RTP call, hold/resume, teardown. The media path is through
// Asterisk's Echo() dialplan application (poc/asterisk/extensions.conf), so
// received packet counts mirror sent counts while the call is active.
func TestSuiteIntegration(t *testing.T) {
	c := suiteClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	if err := c.Register(ctx); err != nil {
		t.Fatalf("Register: %v", err)
	}

	call, err := c.Call(ctx, getenv("SIP_EXT", "100"))
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	for _, e := range c.Trace() {
		t.Logf("trace so far: %-8s %-8s %s", e.Step, e.Method, e.Detail)
	}

	active := call.MediaPhase(3*time.Second, true)
	if active.Recv == 0 {
		t.Fatalf("no RTP received during active call (sent %d, received %d)", active.Sent, active.Recv)
	}
	t.Logf("active: sent %d, received %d", active.Sent, active.Recv)

	if err := call.Hold(ctx); err != nil {
		t.Fatalf("Hold: %v", err)
	}
	held := call.MediaPhase(2*time.Second, false)
	if err := call.Resume(ctx); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	resumed := call.MediaPhase(2*time.Second, true)
	if resumed.Recv == 0 {
		t.Fatalf("no RTP received after resume (sent %d, received %d)", resumed.Sent, resumed.Recv)
	}
	t.Logf("held: received %d (expected ~0); resumed: sent %d, received %d", held.Recv, resumed.Sent, resumed.Recv)

	if err := call.Hangup(ctx); err != nil {
		t.Fatalf("Hangup: %v", err)
	}

	for _, e := range c.Trace() {
		t.Logf("trace: %-8s %-8s %s", e.Step, e.Method, e.Detail)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
