package buildstate

import (
	"strings"
	"testing"
)

// state builds a BUILD-STATE.md with a module envelope and a kit envelope whose generated parts are given.
func state(lastChange, items, log string) string {
	return "# BUILD-STATE\n\n<!-- build-state.v1 -->\nmodule: DemoPan\nlast_change: 2020-01-01\nopen_issues:\n  - demo item\n" +
		"retro_required: false\nretro_pending: false\nlast_session: 2020-01-01 · demo\n<!-- /build-state.v1 -->\n\n## kit\n\n" +
		"<!-- build-state.v1 -->\nmodule: kit                          # the kit\nversion: 1.0.0\n" + lastChange + "\nopen_issues:\n" + items +
		"retro_required: true\nretro_pending: false\nlast_session: 2026-10-16 · hand-kept\n" + log + "<!-- /build-state.v1 -->\n"
}

var archive = Fragment{Name: "0000-archive-pre-2026-10-23.md", Date: "2026-10-22", Archive: true, Note: "old note",
	Logs: []string{"2026-10-23 · late line", "2026-09-01 · early line"}, Issues: []string{"OLD A", "OLD B"}}

func TestRenderArchiveOnlyReproducesTheLedger(t *testing.T) {
	t.Parallel()
	in := state("last_change: 2026-10-22              # old note", "  - OLD A\n  - OLD B\n",
		"2026-10-23 · late line\n2026-09-01 · early line\n")
	got, err := Render(in, []Fragment{archive})
	if err != nil || got != in {
		t.Errorf("Render = (%q, %v), want the input unchanged", got, err)
	}
}

func TestRenderOrder(t *testing.T) {
	t.Parallel()
	a := Fragment{Name: "2026-10-23-a.md", Date: "2026-10-23", Note: "A note", Logs: []string{"2026-10-23 · A"}, Issues: []string{"A1", "A2"}}
	b := Fragment{Name: "2026-10-23-b.md", Date: "2026-10-23", Logs: []string{"2026-10-23 · B"}, Issues: []string{"B1"}}
	c := Fragment{Name: "2026-10-24-c.md", Date: "2026-10-24", Note: "C note", Logs: []string{"2026-10-24 · C"}}
	// Scan hands fragments over in name order; Render must not depend on it.
	got, err := Render(state("last_change: x", "  - stale\n", "2026-01-01 · stale\n"), []Fragment{c, a, archive, b})
	if err != nil {
		t.Fatal(err)
	}
	// Mutation: GBS-order -- issues oldest-first or the log newest-first breaks the conflict-free append
	want := state("last_change: 2026-10-24              # C note",
		"  - B1\n  - A1\n  - A2\n  - OLD A\n  - OLD B\n",
		"2026-10-23 · late line\n2026-09-01 · early line\n2026-10-23 · A\n2026-10-23 · B\n2026-10-24 · C\n")
	if got != want {
		t.Errorf("Render =\n%s\nwant\n%s", got, want)
	}
}

func TestLastChange(t *testing.T) {
	t.Parallel()
	same := Fragment{Name: "2026-10-22-z.md", Date: "2026-10-22", Note: "same day, later name"}
	older := Fragment{Name: "2026-10-21-y.md", Date: "2026-10-21", Note: "older"}
	bare := Fragment{Name: "2026-10-25-w.md", Date: "2026-10-25"}
	cases := []struct {
		name  string
		frags []Fragment
		want  string
	}{
		{"archive alone", []Fragment{archive}, "last_change: 2026-10-22              # old note"},
		// Mutation: GBS-last -- taking the last fragment by name instead of the newest date
		{"an older fragment never wins", []Fragment{archive, older}, "last_change: 2026-10-22              # old note"},
		{"a date tie goes to the later name", []Fragment{archive, same}, "last_change: 2026-10-22              # same day, later name"},
		{"no note, no comment", []Fragment{archive, bare}, "last_change: 2026-10-25"},
	}
	for _, c := range cases {
		if got := LastChange(c.frags); got != c.want {
			t.Errorf("%s: LastChange = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestRenderStructureViolations(t *testing.T) {
	t.Parallel()
	ok := state("last_change: x", "", "")
	cases := []struct{ name, in, want string }{
		{"no kit envelope", strings.Replace(ok, "module: kit ", "module: kat ", 1), errNoKit},
		{"two kit envelopes", ok + "<!-- build-state.v1 -->\nmodule: kit\n<!-- /build-state.v1 -->\n", errNoKit},
		{"unclosed kit envelope", strings.TrimSuffix(ok, "<!-- /build-state.v1 -->\n"), errNoKit},
		{"no last_change", strings.Replace(ok, "last_change: x\n", "", 1), errFields},
		{"no open_issues", strings.Replace(ok, "open_issues:\nretro", "retro", 1), errFields},
		{"no last_session", strings.Replace(ok, "last_session: 2026-10-16 · hand-kept\n", "", 1), errFields},
		{"last_session before open_issues", strings.Replace(strings.Replace(ok, "last_session: 2026-10-16 · hand-kept\n", "", 1),
			"open_issues:\n", "last_session: x\nopen_issues:\n", 1), errFields},
		// Mutation: GBS-tail -- a hand-added field after last_session would be dropped silently on regeneration
		{"a field after the session log", strings.Replace(ok, "<!-- /build-state.v1 -->\n", "last_commit: abc\n<!-- /build-state.v1 -->\n", 2),
			"BUILD-STATE.md line 24 (`last_commit: abc`): " + errLogTail},
	}
	for _, c := range cases {
		_, err := Render(c.in, []Fragment{archive})
		if err == nil || err.Error() != c.want {
			t.Errorf("%s: err = %v, want %q", c.name, err, c.want)
		}
	}
}
