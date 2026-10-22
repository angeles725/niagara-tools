package rules

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

func TestAllUniqueIDs(t *testing.T) {
	seen := map[string]bool{}
	for _, sp := range All() {
		id := sp.ID()
		if id == "" || seen[id] || sp.Usage == "" || len(sp.Select.Names) == 0 {
			t.Errorf("bad or duplicate spec %q: %+v", id, sp)
		}
		seen[id] = true
	}
}

// TestAllRuleOrder: one walk over a file both registered rules report on prints each rule's rows
// contiguously, in registry order under --all and in the requested order for a rule list.
func TestAllRuleOrder(t *testing.T) {
	root := t.TempDir()
	src := "class A {\n  void a() { parent.set(p, v, null); Clock.schedule(this, null, x, null); }\n}\n"
	if err := os.WriteFile(filepath.Join(root, "A.java"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	a := filepath.Join(root, "A.java")
	ncw := "WARN  lint-null-context-write  " + a + ":2  null-Context write: void a() { parent.set(p, v, null); Clock.schedule(this, null, x, null); }\n"
	timers := "FAIL  discarded-ticket  " + a + ": Clock.schedule result is discarded — assign it to a Clock.Ticket field\n" +
		"FAIL  timer-ticket  " + a + ": schedules a Clock ticket but has no stopped() override to cancel it\n"
	main := lint.Command(All()...)
	for args, want := range map[string]string{
		"--all":                               ncw + timers,
		"lint-timers,lint-null-context-write": timers + ncw,
	} {
		var o, e bytes.Buffer
		if code := main([]string{args, root}, cli.Stdio{Out: &o, Err: &e}); code != 1 || o.String() != want || e.Len() != 0 {
			t.Errorf("%s: exit %d err %q out\n%s\nwant\n%s", args, code, e.String(), o.String(), want)
		}
	}
}
