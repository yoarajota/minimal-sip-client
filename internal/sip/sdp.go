package sip

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SDP offer/answer (RFC 4566) as used by the offer/answer model (RFC 3264).
// The minimal client speaks one media line: audio, RTP/AVP, PCMU (payload
// type 0, static, so no rtpmap is strictly required).

// Direction is the media direction attribute (RFC 4566 §6).
type Direction string

const (
	SendRecv Direction = "sendrecv"
	SendOnly Direction = "sendonly"
	RecvOnly Direction = "recvonly"
	Inactive Direction = "inactive"
)

// Offer builds an SDP offer advertising the local RTP port (§13.2.1 — the
// initial offer is in the INVITE; hold/resume re-INVITEs change direction).
func Offer(ip string, rtpPort int, dir Direction) string {
	return fmt.Sprintf("v=0\r\no=- %d %d IN IP4 %s\r\ns=-\r\nc=IN IP4 %s\r\nt=0 0\r\nm=audio %d RTP/AVP 0\r\na=rtpmap:0 PCMU/8000\r\na=%s\r\n",
		int(time.Now().UnixNano()%1000000), int(time.Now().UnixNano()%1000000), ip, ip, rtpPort, dir)
}

// Answer is the peer's SDP answer: where to send RTP and the negotiated
// direction (RFC 3264 §5.1 — a sendonly offer MUST be answered recvonly or
// inactive, and vice versa).
type Answer struct {
	IP   string
	Port int
	Dir  Direction
}

// ParseAnswer extracts the media target and direction from an SDP body.
func ParseAnswer(body string) (Answer, error) {
	a := Answer{Dir: SendRecv} // sendrecv is the default when no direction attribute
	for _, ln := range strings.Split(body, "\r\n") {
		switch {
		case strings.HasPrefix(ln, "c=IN IP4 "):
			a.IP = strings.TrimPrefix(ln, "c=IN IP4 ")
		case strings.HasPrefix(ln, "m=audio "):
			f := strings.Fields(strings.TrimPrefix(ln, "m=audio "))
			if len(f) >= 1 {
				a.Port, _ = strconv.Atoi(f[0])
			}
		case ln == "a=sendonly":
			a.Dir = SendOnly
		case ln == "a=recvonly":
			a.Dir = RecvOnly
		case ln == "a=inactive":
			a.Dir = Inactive
		}
	}
	if a.IP == "" || a.Port == 0 {
		return a, fmt.Errorf("SDP answer missing c= or m= line")
	}
	return a, nil
}
