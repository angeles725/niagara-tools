package jsonrpc

import (
	"strings"
	"testing"
)

// A did-you-mean never turns a short common word into a safety override (retro row M11 verifier):
// "all" is not "allow_ad_hoc", "ack" is not "acknowledge_links".
func TestValidateNeverSuggestsAnOverrideForAShortWord(t *testing.T) {
	t.Parallel()
	tags := `{"properties": {"ords": {"type": "array"}, "allow_ad_hoc": {"type": "boolean"}, "dry_run": {}}}`
	refusedWith(t, tags, `{"all": true}`, "unknown argument 'all' for tool")
	remove := `{"properties": {"parent_ord": {}, "name": {}, "acknowledge_links": {"type": "boolean"}}}`
	refusedWith(t, remove, `{"ack": true}`, "unknown argument 'ack' for tool")
	// the suggestions that help stay: a declared key that starts the unknown one, and a typo
	refusedWith(t, `{"properties": {"capacity": {}}}`, `{"capacity_records": 5}`,
		"unknown argument 'capacity_records' for tool; did you mean 'capacity'?")
	refusedWith(t, `{"properties": {"allow_expired": {}}}`, `{"allowexpired": true}`,
		"unknown argument 'allowexpired' for tool; did you mean 'allow_expired'?")
}

// A refusal echoes at most 64 characters of the caller's value or argument name, then "…".
func TestValidateClipsAHugeEchoedValue(t *testing.T) {
	t.Parallel()
	huge := strings.Repeat("x", 1<<20)
	refusedWith(t, `{"properties": {"p": {"type": "string", "enum": ["a"]}}}`, `{"p": "`+huge+`"}`,
		"argument p must be one of a; got '"+strings.Repeat("x", 63)+"…")
	refusedWith(t, `{"properties": {"p": {}}}`, `{"`+huge+`": 1}`,
		"unknown argument '"+strings.Repeat("x", 63)+"… for tool")
	refusedWith(t, `{"properties": {"p": {"enum": ["a", ""]}}}`, `{"p": "b"}`, "argument p must be one of a, ''; got 'b'")
}
