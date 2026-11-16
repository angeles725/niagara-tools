package cisteps

import (
	"reflect"
	"testing"
)

// Review advisories of #701 (retro 2026-11-15 coordination-stretch T22).

// A `.`/`source` command anywhere in command position is a source line; its path resolves like a line-start
// one, and quoted or commented text is not a command.
func TestSourceRefsReadCompoundForms(t *testing.T) {
	for line, want := range map[string][]sourceRef{
		`[ -f "$R/lib/x.sh" ] && . "$R/lib/x.sh"`:     {{1, "lib/x.sh", false}},
		`if true; then . "$R/lib/x.sh"; fi`:           {{1, "lib/x.sh", false}},
		`{ . "$R/lib/x.sh"; }`:                        {{1, "lib/x.sh", false}},
		`true || source lib/x.sh`:                     {{1, "lib/x.sh", false}},
		`. "$R/a.sh"; . "$R/b.sh"`:                    {{1, "a.sh", false}, {1, "b.sh", false}},
		`[ -n "$x" ] && . "$x"`:                       {{1, "", false}},
		`. "$GO_DIR/go-tool.sh" || refuse "load . x"`: {{1, "go-tool.sh", false}},
		`echo "a; . b.sh" # ; . c.sh`:                 nil,
		`find . -name x && cd .. && ls ./y.sh`:        nil,
	} {
		if got := sourceRefs("s.sh", []byte(line+"\n")); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: refs %v, want %v", line, got, want)
		}
	}
}

// A `# shellcheck source=` directive holds over comment and blank lines up to its command.
func TestSourceDirectiveHoldsOverCommentLines(t *testing.T) {
	text := "# shellcheck source=SCRIPTDIR/lib/x.sh\n# shellcheck disable=SC1091\n\n. \"$x\"\n"
	if got, want := sourceRefs("scripts/s.sh", []byte(text)), []sourceRef{{4, "scripts/lib/x.sh", true}}; !reflect.DeepEqual(got, want) {
		t.Errorf("refs %v, want %v", got, want)
	}
}

// An unresolvable source line anywhere in the set may be the only sourcer of a changed lib, so it takes the
// whole set; a compound sourcer that resolves is followed upward like any other.
func TestShellcheckClosureFallsBackOnAnUnresolvedLineAnywhereInTheSet(t *testing.T) {
	files := map[string]string{
		"scripts/lib/x.sh":   "X=1\n",
		"scripts/guarded.sh": "[ -f \"$R/scripts/lib/x.sh\" ] && . \"$R/scripts/lib/x.sh\"\n",
		"scripts/other.sh":   "echo other\n",
		"scripts/dyn.sh":     "if [ -n \"$1\" ]; then . \"$1\"; fi\n",
	}
	read := func(p string) ([]byte, error) { return []byte(files[p]), nil }
	set := []string{"scripts/guarded.sh", "scripts/lib/x.sh", "scripts/other.sh"}
	got, reason, err := shellcheckClosure([]string{"scripts/lib/x.sh"}, set, read)
	if want := []string{"scripts/guarded.sh", "scripts/lib/x.sh"}; err != nil || reason != "" || !reflect.DeepEqual(got, want) {
		t.Errorf("compound sourcer: closure %v reason %q err %v, want %v", got, reason, err, want)
	}
	// Mutation: T22 -- reading only the closure's own lines for the fallback misses dyn.sh, the only file
	// that might source other.sh, and checks other.sh alone.
	set = append(set, "scripts/dyn.sh")
	if _, reason, _ := shellcheckClosure([]string{"scripts/other.sh"}, set, read); reason != "scripts/dyn.sh:1" {
		t.Errorf("unresolved line outside the closure: reason %q, want scripts/dyn.sh:1", reason)
	}
}

// A changed path that is not in the set at the change's end (deleted, or not a regular file) is not
// materialized: the closure holds members of the set only.
func TestShellcheckClosureLeavesOutAChangedPathTheSetDoesNotHold(t *testing.T) {
	read := func(p string) ([]byte, error) { return []byte("echo\n"), nil }
	got, reason, err := shellcheckClosure([]string{"scripts/gone.sh", "scripts/kept.sh"}, []string{"scripts/kept.sh"}, read)
	if want := []string{"scripts/kept.sh"}; err != nil || reason != "" || !reflect.DeepEqual(got, want) {
		t.Errorf("closure %v reason %q err %v, want %v", got, reason, err, want)
	}
}
