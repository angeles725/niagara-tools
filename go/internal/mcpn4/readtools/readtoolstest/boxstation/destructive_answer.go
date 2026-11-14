package boxstation

// Answer is the reply the station gives to a BOX frame, computed with the station locked: a
// Hook that rewrites or replaces part of an answer (fault injection in black-box tests) calls it
// for what the station itself would send, then edits that before sending it.
func (st *Station) Answer(frame map[string]any) map[string]any {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.handleFrame(frame)
}
