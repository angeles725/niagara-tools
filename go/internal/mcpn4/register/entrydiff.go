package register

import (
	"encoding/json"
	"reflect"
	"sort"
)

// decode is raw as Go values (nil when it does not decode).
func decode(raw []byte) any {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return nil
	}
	return v
}

// sameValue is true when two JSON texts hold the same value, whatever their key order or layout
// (`claude mcp add` writes type, command, args, env; register writes them sorted).
func sameValue(a, b []byte) bool {
	va, vb := decode(a), decode(b)
	return va != nil && reflect.DeepEqual(va, vb)
}

// changedFields names the top-level fields of two JSON objects whose values differ (values are
// never shown: an entry's env may hold anything).
func changedFields(old, new []byte) []string {
	var a, b map[string]json.RawMessage
	if json.Unmarshal(old, &a) != nil || json.Unmarshal(new, &b) != nil {
		return []string{"(whole entry)"}
	}
	var out []string
	for k := range a {
		if _, ok := b[k]; !ok {
			out = append(out, k)
		}
	}
	for k, v := range b {
		if _, ok := a[k]; !ok || !sameValue(a[k], v) {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
