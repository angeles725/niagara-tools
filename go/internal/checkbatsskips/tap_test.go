package checkbatsskips

import (
	"reflect"
	"testing"
)

// TestSkips pins how a TAP line becomes a skip: the split at the last valid marker and the field
// split of the bash tool's `IFS=$'\t' read -r test reason` (a tab ends the test name, tabs around
// the reason are dropped).
func TestSkips(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, tap string
		want      []Skip
	}{
		{"no skip", "1..2\nok 1 a\nnot ok 2 b\n", nil},
		{"reason", "ok 1 a # skip why\n", []Skip{{"ok 1 a", "why"}}},
		{"no reason", "ok 1 a # skip", []Skip{{"ok 1 a", ""}}},
		{"only one leading blank is dropped", "ok 1 a # skip  two\n", []Skip{{"ok 1 a", " two"}}},
		{"# skipped is not a marker", "ok 1 a # skipped\n", nil},
		{"the last valid marker wins", "ok 1 a # skip x # skip y # skipped\n", []Skip{{"ok 1 a # skip x", "y # skipped"}}},
		{"a number is required", "ok a # skip why\n", nil},
		{"a carriage return stays in the reason", "ok 1 a # skip why\r\n", []Skip{{"ok 1 a", "why\r"}}},
		{"a tab ends the test name", "ok 1 a\tb # skip why\n", []Skip{{"ok 1 a", "b\twhy"}}},
		{"tabs around the reason are dropped", "ok 1 a # skip \t\twhy\t\n", []Skip{{"ok 1 a", "why"}}},
	}
	for _, c := range cases {
		if got := Skips(c.tap); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: Skips = %q, want %q", c.name, got, c.want)
		}
	}
}
