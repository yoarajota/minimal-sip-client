package sip

import (
	"encoding/binary"
	"math"
	"net"
	"strconv"
	"time"
)

// RTP send/receive (RFC 3550 §5.1). The minimal client sends PCMU (payload
// type 0) 20 ms packets and counts what it receives.

// RTPHeader is the 12-byte fixed RTP header (RFC 3550 §5.1).
type RTPHeader struct {
	Version        uint8 // 2
	Padding        bool
	Extension      bool
	CSRCCount      uint8
	Marker         bool
	PayloadType    uint8
	SequenceNumber uint16
	Timestamp      uint32
	SSRC           uint32
}

// Pack encodes the fixed header (§5.1 layout).
func (h *RTPHeader) Pack() []byte {
	b := make([]byte, 12)
	b[0] = h.Version<<6 | boolBit(h.Padding)<<5 | boolBit(h.Extension)<<4 | h.CSRCCount&0x0f
	b[1] = boolBit(h.Marker)<<7 | h.PayloadType&0x7f
	binary.BigEndian.PutUint16(b[2:4], h.SequenceNumber)
	binary.BigEndian.PutUint32(b[4:8], h.Timestamp)
	binary.BigEndian.PutUint32(b[8:12], h.SSRC)
	return b
}

func boolBit(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

// TonePacket builds one 20 ms PCMU packet of a 440 Hz tone: the 12-byte
// header plus 160 µ-law-encoded samples (§5.1, payload format per RFC 3551;
// the marker bit stays clear for steady audio).
func TonePacket(seq uint16, ts uint32, ssrc uint32) []byte {
	h := RTPHeader{Version: 2, PayloadType: 0, SequenceNumber: seq, Timestamp: ts, SSRC: ssrc}
	pkt := h.Pack()
	payload := make([]byte, 160)
	for i := range payload {
		s := int16(8000 * math.Sin(2*math.Pi*440*float64(i)/8000))
		payload[i] = Ulaw(s)
	}
	return append(pkt, payload...)
}

// Ulaw encodes a 16-bit PCM sample as G.711 µ-law (the canonical BIAS+132
// formulation: exponent from the position of the highest set bit of the biased
// sample, mantissa the next four bits, output is the bitwise complement).
func Ulaw(s int16) byte {
	sample := int(s)
	sign := 0
	if sample < 0 {
		sample = -sample
		sign = 0x80
	}
	const clip = 32635
	if sample > clip {
		sample = clip
	}
	sample += 132 // BIAS
	exponent := 0
	for m := sample >> 7; m > 1; m >>= 1 {
		exponent++
	}
	mantissa := (sample >> (exponent + 3)) & 0x0f
	return byte(^(sign | (exponent << 4) | mantissa) & 0xff)
}

// MediaStats counts RTP packets over one phase (sent, received).
type MediaStats struct {
	Sent int
	Recv int
}

// Stream sends tone packets to the negotiated RTP target while counting
// received packets. The receive window outlives the send window by 1.5 s so
// in-flight packets are counted (§5.1 sequence/timestamp semantics).
type Stream struct {
	conn *net.UDPConn
	dst  *net.UDPAddr
	seq  uint16
	ts   uint32
	ssrc uint32
}

// NewStream opens the RTP socket on the advertised port.
func NewStream(rtpPort int, ssrc uint32) (*Stream, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: rtpPort})
	if err != nil {
		return nil, err
	}
	return &Stream{conn: conn, ssrc: ssrc, seq: uint16(time.Now().UnixNano()), ts: uint32(time.Now().UnixNano())}, nil
}

// SetTarget points the stream at the peer's RTP address (the SDP answer).
func (s *Stream) SetTarget(ip string, port int) error {
	dst, err := net.ResolveUDPAddr("udp", net.JoinHostPort(ip, strconv.Itoa(port)))
	if err != nil {
		return err
	}
	s.dst = dst
	return nil
}

// Phase sends a 440 Hz tone for `dur` (when send) and returns packet counts.
// The receive counter runs for dur + 1.5 s regardless, so a "hold" phase
// (send=false) still observes what arrives.
func (s *Stream) Phase(dur time.Duration, send bool) MediaStats {
	end := time.Now().Add(dur)
	var stats MediaStats
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 2048)
		s.conn.SetReadDeadline(end.Add(1500 * time.Millisecond)) //nolint:errcheck
		for {
			n, _, err := s.conn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			if n >= 12 {
				stats.Recv++
			}
		}
	}()
	if send && s.dst != nil {
		ticker := time.NewTicker(20 * time.Millisecond) // 50 pkt/s
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if _, err := s.conn.WriteToUDP(TonePacket(s.seq, s.ts, s.ssrc), s.dst); err == nil {
					stats.Sent++
				}
				s.seq++
				s.ts += 160
			case <-time.After(time.Until(end) + 50*time.Millisecond):
				send = false
			}
			if !send {
				break
			}
		}
	}
	<-done
	return stats
}

// Close releases the socket.
func (s *Stream) Close() error { return s.conn.Close() }
