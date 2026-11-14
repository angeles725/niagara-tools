package livesmoke

import "github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"

// BuildReport is the apply-mode report: ok only when there are steps, every required one
// verified, and the run ended without an error (nil, or its message).
func BuildReport(station string, steps []*pyjson.Object, runErr any, serverStderr []string) *pyjson.Object {
	ok := len(steps) > 0 && runErr == nil
	stepList := make([]any, len(steps))
	for i, s := range steps {
		stepList[i] = s
		if required, _ := s.Get("required"); required == true {
			if verdict, _ := s.Get("verdict"); verdict != "verified" {
				ok = false
			}
		}
	}
	stderr := make([]any, len(serverStderr))
	for i, line := range serverStderr {
		stderr[i] = line
	}
	probe := make([]any, len(ProbeOnly))
	for i, p := range ProbeOnly {
		probe[i] = p
	}
	return kwargs("ok", ok, "station", station, "mode", "apply", "error", runErr,
		"probe_only", probe, "steps", stepList, "server_stderr", stderr)
}
