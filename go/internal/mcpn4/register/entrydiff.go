package register

import (
	"bytes"
	"encoding/json"
	"sort"
)

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
		if compactString(a[k]) != compactString(v) {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func compactString(raw []byte) string {
	var b bytes.Buffer
	if err := json.Compact(&b, raw); err != nil {
		return string(raw)
	}
	return b.String()
}
