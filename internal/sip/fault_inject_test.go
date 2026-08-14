//go:build integration

// Fault injection at the real seam (bench/run.sh fault leg): kill the Asterisk
// container mid-call and assert the documented behaviour — media stops, the
// hangup BYE times out with a 408-class TransactionError, the client exits
// cleanly without crashing (R-003 mitigation from docs/04-tradeoffs.md).
package sip

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// TestKilledPBXMidCall is orchestrated by bench/run.sh: the runner starts this
// test, watches for the READY_FOR_KILL marker, then kills the asterisk
// container. The test waits until the media stream goes silent (the kill), then
// hangs up and expects the 64xT1 timeout on the BYE transaction.
func TestKilledPBXMidCall(t *testing.T) {
	c := suiteClient(t)
	ctx := context.Background()
	if err := c.Register(ctx); err != nil {
		t.Fatalf("register: %v", err)
	}
	call, err := c.Call(ctx, getenv("SIP_EXT", "100"))
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	// Let media flow, then tell the runner to kill the PBX.
	active := call.MediaPhase(2*time.Second, true)
	if active.Recv == 0 {
		t.Fatalf("no media before kill (recv=%d)", active.Recv)
	}
	fmt.Println("READY_FOR_KILL")
	// Wait for the media stream to go silent — that is the kill landing.
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		sample := call.MediaPhase(2*time.Second, true)
		if sample.Recv == 0 && sample.Sent == 0 {
			break
		}
	}
	// Hang up against the dead PBX: the BYE transaction must time out
	// (408-class) rather than hang or crash.
	start := time.Now()
	err = call.Hangup(ctx)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatalf("hangup succeeded against a dead PBX (elapsed %s)", elapsed)
	}
	var te *TransactionError
	if !errors.As(err, &te) || te.Code != 408 {
		t.Fatalf("expected 408-class TransactionError, got %v (elapsed %s)", err, elapsed)
	}
	if elapsed < 20*time.Second {
		t.Fatalf("bye timeout elapsed %s — shorter than the 64xT1 retransmission window", elapsed)
	}
	fmt.Printf("PASS killed-pbx: media stopped, BYE -> %d after %s, clean exit\n", te.Code, elapsed.Round(time.Second))
}
