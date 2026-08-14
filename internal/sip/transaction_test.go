package sip

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

// Failure-mode and property tests for the transaction layer (§17), driven by
// the fake UAS. Each row of docs/01-theory.md § 3 maps to at least one test.

func testConn(t *testing.T) *net.UDPConn {
	t.Helper()
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func testRequest(method string) []byte {
	m := NewRequest(method, "sip:100@test", "127.0.0.1", 5060,
		"alice <sip:alice@test>", "tag1", "<sip:alice@test>", "", "cid@h", 1, "")
	return []byte(m.String())
}

// Transaction timeout → treated as 408 (§8.1.3.1, §17.1.1.2 Timer B).
func TestTransactionTimeout(t *testing.T) {
	f := newFakeUAS(t)
	f.silent = true
	_, err := runTransaction(testConn(t), f.Addr, testRequest("REGISTER"), false,
		time.Now().Add(1500*time.Millisecond), nil)
	te, ok := err.(*TransactionError)
	if !ok || te.Code != 408 {
		t.Fatalf("err = %v, want TransactionError 408", err)
	}
}

// A dropped first transmission is recovered by Timer E retransmission (§17.1.2.2).
func TestTransactionRetransmission(t *testing.T) {
	f := newFakeUAS(t)
	f.dropN = 1
	_, err := runTransaction(testConn(t), f.Addr, testRequest("REGISTER"), false,
		time.Now().Add(5*time.Second), nil)
	if err != nil {
		t.Fatalf("runTransaction: %v", err)
	}
	if n := f.countMethod("REGISTER"); n < 2 {
		t.Errorf("server saw %d REGISTERs, want >= 2 (retransmission)", n)
	}
}

// A response with a foreign branch is ignored (§17.1.3), so the transaction
// times out instead of accepting it.
func TestTransactionWrongBranchIgnored(t *testing.T) {
	f := newFakeUAS(t)
	f.wrongBr = true
	_, err := runTransaction(testConn(t), f.Addr, testRequest("REGISTER"), false,
		time.Now().Add(1200*time.Millisecond), nil)
	te, ok := err.(*TransactionError)
	if !ok || te.Code != 408 {
		t.Fatalf("err = %v, want TransactionError 408 (wrong-branch response ignored)", err)
	}
}

// A provisional response stops INVITE retransmission (§17.1.1.2 Calling→Proceeding).
func TestInviteProvisionalStopsRetransmission(t *testing.T) {
	f := newFakeUAS(t)
	f.inviteCode = "180 Ringing"
	// The fake answers every INVITE with 180, which is provisional; the
	// transaction must stop retransmitting and keep waiting until the stop
	// time, then time out — with exactly ONE transmission after the 1xx.
	_, err := runTransaction(testConn(t), f.Addr, testRequest("INVITE"), true,
		time.Now().Add(1500*time.Millisecond), nil)
	if err == nil {
		t.Fatal("want timeout after a lone 180")
	}
	// First send + retransmissions before the 180 arrived. The 180 arrives on
	// the first exchange, so only one INVITE should be transmitted.
	if n := f.countMethod("INVITE"); n != 1 {
		t.Errorf("server saw %d INVITEs after 180, want 1 (retransmission stopped)", n)
	}
}

// A non-2xx final to an INVITE gets a transaction-layer ACK (§17.1.1.2,
// §17.1.1.3): same branch, CSeq method ACK, To from the response.
func TestInviteNon2xxGeneratesACK(t *testing.T) {
	f := newFakeUAS(t)
	f.inviteCode = "404 Not Found"
	if _, err := runTransaction(testConn(t), f.Addr, testRequest("INVITE"), true,
		time.Now().Add(3*time.Second), nil); err != nil {
		t.Fatalf("runTransaction: %v", err)
	}
	acks := f.waitFor(t, "ACK", 1)
	if len(acks) == 0 {
		t.Fatal("no ACK for the non-2xx final response")
	}
	last := acks[len(acks)-1]
	if got := last.Get("CSeq"); !strings.HasSuffix(got, " ACK") {
		t.Errorf("ACK CSeq = %q, want method ACK", got)
	}
	inv := f.requests()[0]
	wantBranch := strings.TrimPrefix(branchOf([]byte(inv.String())), "branch=")
	if !strings.Contains(last.Get("Via"), "branch="+wantBranch) {
		t.Errorf("ACK Via %q does not carry the INVITE branch", last.Get("Via"))
	}
	if !strings.Contains(last.Get("To"), "server-tag") {
		t.Errorf("ACK To %q does not carry the response tag", last.Get("To"))
	}
}

// Context cancellation bounds the transaction (client-side).
func TestTransactionContextDeadline(t *testing.T) {
	f := newFakeUAS(t)
	f.silent = true
	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()
	_, err := runTransaction(testConn(t), f.Addr, testRequest("REGISTER"), false,
		transactionDeadline(ctx), nil)
	if err == nil {
		t.Fatal("want error on silent server")
	}
}
