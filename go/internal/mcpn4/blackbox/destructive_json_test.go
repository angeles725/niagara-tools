package blackbox

// JSON helpers of the destructive tests: compare and pick apart tool replies.

import (
	"encoding/json"
	"strings"
)

// same compares two JSON values by their canonical JSON (sorted keys).
func same(got, want any) bool {
	a, errA := json.Marshal(got)
	b, errB := json.Marshal(want)
	return errA == nil && errB == nil && string(a) == string(b)
}

// strs are the strings of a JSON array.
func strs(v any) []string {
	list, _ := v.([]any)
	out := []string{}
	for _, x := range list {
		s, _ := x.(string)
		out = append(out, s)
	}
	return out
}

// noteWith are the notes that contain every one of subs.
func noteWith(notes any, subs ...string) []string {
	out := []string{}
	for _, n := range strs(notes) {
		all := true
		for _, s := range subs {
			all = all && strings.Contains(n, s)
		}
		if all {
			out = append(out, n)
		}
	}
	return out
}

// planOf is the "plan" object of a dry-run reply.
func planOf(reply map[string]any) map[string]any {
	p, _ := reply["plan"].(map[string]any)
	return p
}

// byName indexes a JSON array of BOX nodes by "n".
func byName(v any) map[string]map[string]any {
	out := map[string]map[string]any{}
	for _, m := range objects(v) {
		n, _ := m["n"].(string)
		out[n] = m
	}
	return out
}
