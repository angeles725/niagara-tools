package bql_test

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
)

// The active level is the number after the last "@" of an out text, "def" for the fallback.
func TestOutLevelReadsTheLevelAfterTheAt(t *testing.T) {
	for _, c := range []struct {
		text  string
		level int
		name  string
	}{
		{"0,0 {disabled,stale} @ 16", 16, "16"},
		{"75.0 {ok} @ 8", 8, "8"},
		{"1.0 {ok} @ 1", 1, "1"},
		{"5.0 {ok} @ def", 0, "def"},
		{"5.0 {ok} @ DEF ", 0, "def"},
		{"a @ b @ 3", 3, "3"},
		{"5.0 {ok}", 0, "unknown"},
		{"5.0 {ok} @ 17", 0, "unknown"},
		{"5.0 {ok} @ 0", 0, "unknown"},
		{"5.0 {ok} @", 0, "unknown"},
		{"", 0, "unknown"},
	} {
		if level, name := bql.OutLevel(c.text); level != c.level || name != c.name {
			t.Errorf("OutLevel(%q) = %d, %q; want %d, %q", c.text, level, name, c.level, c.name)
		}
	}
}
