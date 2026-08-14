package sip

// Trace records one row per observed behaviour — the raw material of the
// message-trace matrix (docs/matrix.md). Each entry names the RFC section
// that forces the behaviour (S-003: analyzability).
type TraceEntry struct {
	Step    string   // "register", "invite", "ack-2xx", "hold", "resume", "bye"
	Method  string   // "REGISTER", "INVITE", "ACK", "BYE"
	RFCRefs []string // RFC 3261 sections forcing this behaviour
	Detail  string   // observed outcome, e.g. "401 -> 200"
}

// Trace returns a copy of the rows recorded so far.
func (c *Client) Trace() []TraceEntry {
	c.traceMu.Lock()
	defer c.traceMu.Unlock()
	out := make([]TraceEntry, len(c.trace))
	copy(out, c.trace)
	return out
}

func (c *Client) record(e TraceEntry) {
	c.traceMu.Lock()
	c.trace = append(c.trace, e)
	c.traceMu.Unlock()
}
