package jsonrpc

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
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

// A refusal echoes at most 64 characters of the caller's value or argument name, then "…", inside
// its quotes (retro row M11b: the clip kept the closing quote off).
func TestValidateClipsAHugeEchoedValue(t *testing.T) {
	t.Parallel()
	huge := strings.Repeat("x", 1<<20)
	refusedWith(t, `{"properties": {"p": {"type": "string", "enum": ["a"]}}}`, `{"p": "`+huge+`"}`,
		"argument p must be one of a; got '"+strings.Repeat("x", 64)+"…'")
	refusedWith(t, `{"properties": {"p": {}}}`, `{"`+huge+`": 1}`,
		"unknown argument '"+strings.Repeat("x", 64)+"…' for tool")
	refusedWith(t, `{"properties": {"p": {"enum": ["a", ""]}}}`, `{"p": "b"}`, "argument p must be one of a, ''; got 'b'")
}

// Two declared keys equally close to the unknown one: no guess (the tie branch, retro row M11b).
func TestValidateSuggestsNothingOnATie(t *testing.T) {
	t.Parallel()
	s := `{"properties": {"mode": {}, "node": {}}}`
	refusedWith(t, s, `{"code": 1}`, "unknown argument 'code' for tool")
	refusedWith(t, s, `{"mod": 1}`, "unknown argument 'mod' for tool; did you mean 'mode'?") // one is closer
}

// A huge unknown argument name is refused without an edit distance over it: the did-you-mean
// costs a bounded number of allocations whatever the caller sends (retro row M11b).
func TestValidateRefusesAHugeUnknownKeyWithBoundedAllocations(t *testing.T) {
	huge := strings.Repeat("x", 1<<20)
	schema := decodeObj(t, `{"properties": {"capacity": {}, "depth": {}, "dry_run": {}, "full_policy": {}}}`)
	args := pyjson.NewObject()
	args.Set(huge, pyjson.IntLiteral("1"))
	allocs := testing.AllocsPerRun(3, func() {
		if Validate("tool", schema, args) == nil {
			t.Fatal("a huge unknown argument was accepted")
		}
	})
	if allocs > 64 {
		t.Errorf("refusing a 1 MB unknown argument took %.0f allocations, want at most 64", allocs)
	}
}
