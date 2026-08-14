// Command minimal-sip-client drives the scenario suite: register, establish a
// two-way RTP call, hold, resume and tear down against a mainstream PBX,
// using the minimal RFC 3261 client in internal/sip. The message-trace
// matrix is printed on success.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/yoarajota/minimal-sip-client/internal/sip"
)

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	cfg := sip.Config{
		Server:   env("SIP_SERVER", "asterisk:5060"),
		Domain:   env("SIP_DOMAIN", "asterisk"),
		User:     env("SIP_USER", "alice"),
		Password: env("SIP_PASS", "secret"),
		RTPPort:  40000,
	}
	ext := env("SIP_EXT", "100")
	client, err := sip.New(cfg)
	if err != nil {
		fatalf("new client: %v", err)
	}
	defer client.Close() //nolint:errcheck

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	fmt.Printf("minimal SIP client (component) — suite against %s\n", cfg.Server)
	fmt.Println("Step 1/5  register")
	if err := client.Register(ctx); err != nil {
		fatalf("register: %v", err)
	}

	fmt.Println("Step 2/5  establish two-way RTP call")
	call, err := client.Call(ctx, ext)
	if err != nil {
		fatalf("call: %v", err)
	}

	fmt.Println("Step 3/5  media flows (send tone, expect echo)")
	active := call.MediaPhase(3*time.Second, true)
	fmt.Printf("  sent %d RTP packets, received %d — two-way media %s\n",
		active.Sent, active.Recv, ok(active.Recv > 0))
	if active.Recv == 0 {
		fatalf("no RTP received during active call — media path broken")
	}

	fmt.Println("Step 4/5  hold / resume")
	if err := call.Hold(ctx); err != nil {
		fatalf("hold: %v", err)
	}
	held := call.MediaPhase(2*time.Second, false)
	if err := call.Resume(ctx); err != nil {
		fatalf("resume: %v", err)
	}
	resumed := call.MediaPhase(2*time.Second, true)
	last := call.Media()
	fmt.Printf("  held: received %d (expected ~0); resumed: sent %d, received %d — %s\n",
		held.Recv, last.Sent, last.Recv, ok(resumed.Recv > 0))
	if resumed.Recv == 0 {
		fatalf("no RTP received after resume")
	}

	fmt.Println("Step 5/5  teardown")
	if err := call.Hangup(ctx); err != nil {
		fatalf("hangup: %v", err)
	}

	fmt.Println()
	fmt.Println("SUITE PASSED: register -> call -> hold -> resume -> teardown against a real PBX")
	fmt.Println()
	fmt.Println("Message-trace matrix (rows map to docs/matrix.md):")
	fmt.Println("| step    | method   | RFC refs")
	fmt.Println("| :---    | :---     | :---")
	for _, e := range client.Trace() {
		fmt.Printf("| %-8s | %-8s | %s\n", e.Step, e.Method, join(e.RFCRefs))
	}
}

func join(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}

func ok(b bool) string {
	if b {
		return "OK"
	}
	return "MISSING"
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FAIL: "+format+"\n", args...)
	os.Exit(1)
}
