package buildstatefields

import "testing"

const okEnv = "<!-- build-state.v1 -->\nmodule: kit\nretro_required: true\nretro_pending: true\n<!-- /build-state.v1 -->\n"

func TestCheckEnvelopes(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, text, want string }{
		{"one well-formed envelope", okEnv, ""},
		{"no envelope at all", "# prose only\n", ""},
		{"unbalanced markers", "<!-- build-state.v1 -->\nmodule: kit\n", "unbalanced build-state.v1 markers (1 open / 0 close)"},
		{"missing module", "<!-- build-state.v1 -->\nretro_required: true\nretro_pending: true\n<!-- /build-state.v1 -->\n", "envelope missing required field: module"},
		{"an empty module (comment only) is missing", "<!-- build-state.v1 -->\nmodule:   # c\nretro_required: true\nretro_pending: true\n<!-- /build-state.v1 -->\n", "envelope missing required field: module"},
		{"a later empty module keeps the earlier one", "<!-- build-state.v1 -->\nmodule: x\nmodule:\nretro_required: true\nretro_pending: true\n<!-- /build-state.v1 -->\n", ""},
		{"missing retro_required", "<!-- build-state.v1 -->\nmodule: kit\nretro_pending: true\n<!-- /build-state.v1 -->\n", "envelope (kit) missing required field: retro_required"},
		{"missing retro_pending", "<!-- build-state.v1 -->\nmodule: kit\nretro_required: true\n<!-- /build-state.v1 -->\n", "envelope (kit) missing required field: retro_pending"},
		{"retro_required not boolean", "<!-- build-state.v1 -->\nmodule: kit\nretro_required: yes\nretro_pending: true\n<!-- /build-state.v1 -->\n", "envelope (kit) retro_required not boolean: 'yes'"},
		{"retro_pending not boolean", "<!-- build-state.v1 -->\nmodule: kit\nretro_required: true\nretro_pending: TRUE\n<!-- /build-state.v1 -->\n", "envelope (kit) retro_pending not boolean: 'TRUE'"},
		{"comments, tabs, list items and blank lines are tolerated",
			"<!-- build-state.v1 -->\nmodule: kit\nretro_required:   true   # why\nretro_pending:\tfalse\t\nopen_issues:\n  - one: a\n  - two\n  foo - bar: x\n\n   \n<!-- /build-state.v1 -->\n", ""},
		{"an indented line holding '- ' is a list item, even a key", "<!-- build-state.v1 -->\nmodule: kit\nretro_required: true\n  retro_pending - x: true\n<!-- /build-state.v1 -->\n", "envelope (kit) missing required field: retro_pending"},
		{"CRLF markers are not envelopes", "<!-- build-state.v1 -->\r\nmodule: kit\r\n<!-- /build-state.v1 -->\r\n", ""},
		{"CRLF field values are trimmed", "<!-- build-state.v1 -->\nmodule: kit\r\nretro_required: true\r\nretro_pending: true\r\n<!-- /build-state.v1 -->\n", ""},
		{"quoted or indented markers are prose", "prose `<!-- build-state.v1 -->`\n  <!-- build-state.v1 -->\n" + okEnv, ""},
		{"the second envelope is checked too", okEnv + "<!-- build-state.v1 -->\nmodule: m2\nretro_required: true\nretro_pending: maybe\n<!-- /build-state.v1 -->\n", "envelope (m2) retro_pending not boolean: 'maybe'"},
		{"a close before any open has no fields", "<!-- /build-state.v1 -->\n<!-- build-state.v1 -->\nmodule: kit\nretro_required: true\nretro_pending: true\n", "envelope missing required field: module"},
		{"a stray close after an envelope re-checks its fields", okEnv + "<!-- /build-state.v1 -->\n<!-- build-state.v1 -->\n", ""},
		{"an unterminated last close marker still closes the envelope", "<!-- build-state.v1 -->\nmodule: kit\nretro_required: true\n<!-- /build-state.v1 -->", "envelope (kit) missing required field: retro_pending"},
		{"key is up to the first colon, value after it, # cuts the value",
			"<!-- build-state.v1 -->\nmodule: a:b\nretro_required:true\nretro_pending: true#x\nnocolon\n<!-- /build-state.v1 -->\n", ""},
		{"empty file", "", ""},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := CheckEnvelopes(tt.text); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseIndexRow(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, line string
		want       IndexRow
		ok         bool
	}{
		{"a data row", "| 2026-09-04-demo.md | pending |", IndexRow{"2026-09-04-demo.md", "pending", ""}, true},
		{"header and separator are not data rows", "| Retro file | review-status |", IndexRow{}, false},
		{"separator", "|---|---|", IndexRow{}, false},
		{"not a table line", " | 2026-09-04-demo.md | pending |", IndexRow{}, false},
		{"no valid status", "| 2026-09-04-demo.md | maybe |", IndexRow{"2026-09-04-demo.md", "", ""}, true},
		{"the deltas cell follows the status, padding spaces trimmed", "| 2026-01-01-a.md | pending | 1 0 |", IndexRow{"2026-01-01-a.md", "pending", "1 0"}, true},
		{"status in the last cell has an empty deltas cell", "|2026-07-08-i.md|pending", IndexRow{"2026-07-08-i.md", "pending", ""}, true},
		{"INDEX.md is never the file; the first other .md is", "| INDEX.md | see 2026-09-04-demo.md and other.md | folded pending |", IndexRow{"2026-09-04-demo.md", "", ""}, true},
		{"first .md name, dated or not", "| x.md | 2026-09-04-demo.md | pending |", IndexRow{"x.md", "pending", ""}, true},
		{"tabs are not trimmed from a cell", "|\t2026-09-04-demo.md\t|\tpending\t|", IndexRow{"2026-09-04-demo.md", "", ""}, true},
		{"a status cell trims spaces only", "|2026-09-04-demo.md|pending |", IndexRow{"2026-09-04-demo.md", "pending", ""}, true},
		{"the date must be in one cell with .md", "| 2026-09-04 | demo.md |", IndexRow{}, false},
	}
	for _, tt := range cases {
		got, ok := ParseIndexRow(tt.line)
		if ok != tt.ok || got != tt.want {
			t.Errorf("%s: got (%+v, %v), want (%+v, %v)", tt.name, got, ok, tt.want, tt.ok)
		}
	}
}

func TestMarker(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, text, want string }{
		{"pending", "<!-- review-status: pending -->\n# x\n", "pending"},
		{"absent", "# x\n", ""},
		{"line 6 is past the header", "1\n2\n3\n4\n5\n<!-- review-status: folded -->\n", ""},
		{"line 5 is read", "1\n2\n3\n4\n<!-- review-status: folded -->\n", "folded"},
		{"tight spacing keeps the comment close", "<!--review-status:pending-->\n", "pending-->"},
		{"the first blank-delimited word", "<!-- review-status:   pending\tx -->\n", "pending"},
		{"CR is not an awk field separator", "<!-- review-status: pending\r\n", "pending\r"},
		{"CR after the closing comment stays outside the word", "<!-- review-status: pending -->\r\n", "pending"},
		{"the first column-0 marker wins", "  <!-- review-status: folded -->\n<!-- review-status: pending -->\n<!-- review-status: folded -->\n", "pending"},
		{"an empty value", "<!-- review-status: -->\n", "-->"},
		{"nothing after the colon", "<!-- review-status:\n", ""},
	}
	for _, tt := range cases {
		if got := Marker(tt.text); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestDispositionSkips(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, line string
		want       bool
	}{
		{"tracked-by-issue on a 0-delta row", "| 2026-08-01-b.md | pending | 0 | <!-- disposition: tracked-by-issue #12 -->", true},
		{"worklist-complete on a 0-delta row", "| 2026-08-03-d.md | pending | 0 | <!-- disposition: worklist-complete x -->", true},
		{"D8: a row that enumerates deltas still ages", "| 2026-08-02-c.md | pending | 2 | <!-- disposition: tracked-by-issue #12 -->", false},
		{"D9: any other disposition still ages", "| 2026-08-04-e.md | pending | 0 | <!-- disposition: other -->", false},
		{"tracked-by-issue needs a number", "| 2026-08-05-f.md | pending | 0 | <!-- disposition: tracked-by-issue #x -->", false},
		{"no disposition", "| 2026-01-01-a.md | pending | 0 |", false},
		{"R3-002: worklist-complete must be a whole token", "| 2026-08-03-d.md | pending | 0 | <!-- disposition: worklist-completed later -->", false},
		{"R3-002: worklist-complete- is not the token", "| 2026-08-03-d.md | pending | 0 | <!-- disposition: worklist-complete-ish -->", false},
		{"R3-002: tracked-by-issue needs a whole number", "| 2026-08-01-b.md | pending | 0 | <!-- disposition: tracked-by-issue #12abc -->", false},
		{"worklist-complete closing the comment", "| 2026-08-03-d.md | pending | 0 | <!-- disposition: worklist-complete -->", true},
		{"worklist-complete followed by a dash phrase", "| 2026-08-02-done.md | pending | 0 | <!-- disposition: worklist-complete — all applied -->", true},
		{"tracked-by-issue #n, then the comment end", "| 2026-08-01-b.md | pending | 0 | <!-- disposition: tracked-by-issue #7-->", true},
		{"an unreadable deltas cell (inner space) fails closed: the row ages", "| 2026-07-07-h.md | pending |0 0| <!-- disposition: worklist-complete -->", false},
		{"retrocells reads the deltas cell as decimal: 00 is zero", "| 2026-07-07-h.md | pending | 00 | <!-- disposition: worklist-complete -->", true},
		{"a non-numeric deltas cell fails closed", "| 2026-07-07-h.md | pending | none | <!-- disposition: worklist-complete -->", false},
		{"no deltas cell", "|2026-07-08-i.md|pending", false},
	}
	for _, tt := range cases {
		if got := DispositionSkips(tt.line); got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}
