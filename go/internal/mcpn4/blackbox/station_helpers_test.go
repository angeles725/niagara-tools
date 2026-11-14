package blackbox

import (
	"sort"
	"testing"
)

// stationRig is StationTestCase: a rig whose server is connected to the fake station.
func stationRig(t *testing.T) *Rig {
	r := NewRig(t, nil)
	r.Connect(nil)
	return r
}

// objects is a JSON array of objects.
func objects(v any) []map[string]any {
	list, _ := v.([]any)
	out := []map[string]any{}
	for _, item := range list {
		m, _ := item.(map[string]any)
		out = append(out, m)
	}
	return out
}

// field is key of every object of a JSON array, in order.
func field(v any, key string) []any {
	out := []any{}
	for _, m := range objects(v) {
		out = append(out, m[key])
	}
	return out
}

// sortedStrings is field for string values, sorted.
func sortedStrings(v any, key string) []string {
	out := []string{}
	for _, x := range field(v, key) {
		s, _ := x.(string)
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// named is the object of a JSON array whose name is name, nil when absent.
func named(v any, name string) map[string]any {
	for _, m := range objects(v) {
		if m["name"] == name {
			return m
		}
	}
	return nil
}
