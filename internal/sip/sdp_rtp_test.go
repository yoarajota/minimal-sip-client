package sip

import (
	"testing"
)

func TestSDPOfferAndParse(t *testing.T) {
	offer := Offer("10.0.0.1", 40000, SendRecv)
	ans, err := ParseAnswer(offer)
	if err != nil {
		t.Fatalf("ParseAnswer(offer): %v", err)
	}
	if ans.IP != "10.0.0.1" || ans.Port != 40000 {
		t.Errorf("answer = %+v, want ip 10.0.0.1 port 40000", ans)
	}
	if ans.Dir != SendRecv {
		t.Errorf("direction = %s, want sendrecv (the default)", ans.Dir)
	}
}

func TestParseAnswerDirections(t *testing.T) {
	// RFC 3264 §5.1: a sendonly offer is answered recvonly or inactive.
	for _, tc := range []struct {
		body string
		want Direction
	}{
		{"c=IN IP4 10.0.0.2\r\nm=audio 5004 RTP/AVP 0\r\na=recvonly\r\n", RecvOnly},
		{"c=IN IP4 10.0.0.2\r\nm=audio 5004 RTP/AVP 0\r\na=sendrecv\r\n", SendRecv},
		{"c=IN IP4 10.0.0.2\r\nm=audio 5004 RTP/AVP 0\r\n", SendRecv}, // default
	} {
		ans, err := ParseAnswer(tc.body)
		if err != nil {
			t.Fatalf("ParseAnswer: %v", err)
		}
		if ans.Dir != tc.want {
			t.Errorf("direction = %s, want %s", ans.Dir, tc.want)
		}
	}
}

func TestParseAnswerMissingMedia(t *testing.T) {
	if _, err := ParseAnswer("c=IN IP4 10.0.0.2\r\n"); err == nil {
		t.Error("missing m= line should error")
	}
}

func TestRTPHeaderPack(t *testing.T) {
	h := RTPHeader{Version: 2, PayloadType: 0, SequenceNumber: 0x1234, Timestamp: 0xdeadbeef, SSRC: 0x01020304}
	b := h.Pack()
	// PT=0, marker unset: byte 1 is 0x00 (§5.1 layout).
	want := []byte{0x80, 0x00, 0x12, 0x34, 0xde, 0xad, 0xbe, 0xef, 0x01, 0x02, 0x03, 0x04}
	for i := range want {
		if b[i] != want[i] {
			t.Fatalf("byte %d = %#x, want %#x (packed %x)", i, b[i], want[i], b)
		}
	}
	if len(TonePacket(0, 0, 1)) != 12+160 {
		t.Errorf("tone packet length = %d, want 172", len(TonePacket(0, 0, 1)))
	}
}

func TestUlawKnownValues(t *testing.T) {
	// Known G.711 µ-law anchor: 0 → 0xFF (the BIAS+132 formulation).
	if Ulaw(0) != 0xff {
		t.Errorf("Ulaw(0) = %#x, want 0xff", Ulaw(0))
	}
	// Property: over the full int16 input space the encoder must cover
	// essentially the whole 255-code µ-law range (it is a bijection onto
	// 8-bit codes for in-range samples).
	seen := map[byte]bool{}
	for i := int16(-32768); ; i++ {
		seen[Ulaw(i)] = true
		if i == 32767 {
			break
		}
	}
	if len(seen) < 250 {
		t.Errorf("µ-law encoder covers only %d of 255 codes", len(seen))
	}
}
