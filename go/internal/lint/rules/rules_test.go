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
		if id == "" || seen[id] || sp.Usage == "" || (len(sp.Select.Names) == 0) != (sp.Root != nil) {
			t.Errorf("bad or duplicate spec %q: %+v", id, sp)
		}
		seen[id] = true
	}
}

// TestAllRuleOrder: one walk over a file several registered rules report on prints each rule's rows
// contiguously, in registry order under --all and in the requested order for a rule list.
func TestAllRuleOrder(t *testing.T) {
	root := t.TempDir()
	src := "class A {\n  void a() { parent.set(p, v, null); Clock.schedule(this, null, x, null); }\n}\n"
	if err := os.WriteFile(filepath.Join(root, "A.java"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	a := filepath.Join(root, "A.java")
	ncw := "WARN  lint-null-context-write  " + a + ":2  null-Context write: void a() { parent.set(p, v, null); Clock.schedule(this, null, x, null); }\n"
	timers := "FAIL  discarded-ticket  " + a + ":2  Clock.schedule result is discarded — assign it to a Clock.Ticket field\n" +
		"FAIL  timer-ticket  " + a + ":2  schedules a Clock ticket but has no stopped() override to cancel it\n"
	delays := "FAIL  lint-delays     " + a + ":2  unfloored  cannot verify floor: null\n"
	main := lint.Command(All()...)
	for args, want := range map[string]string{
		"--all":                               ncw + timers + delays,
		"lint-timers,lint-null-context-write": timers + ncw,
	} {
		var o, e bytes.Buffer
		if code := main([]string{args, root}, cli.Stdio{Out: &o, Err: &e}); code != 1 || o.String() != want || e.Len() != 0 {
			t.Errorf("%s: exit %d err %q out\n%s\nwant\n%s", args, code, e.String(), o.String(), want)
		}
	}
}

// TestAllWalkError: under --all an unreadable sub-directory stops lint-timers (scan_files_or_exit: no
// rows) while lint-null-context-write, which opts out (awk_lint_run), still reports the listed files.
func TestAllWalkError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	root := t.TempDir()
	src := "class A {\n  void a() { parent.set(p, v, null); Clock.schedule(this, null, x, null); }\n}\n"
	locked := filepath.Join(root, "locked")
	if err := os.WriteFile(filepath.Join(root, "A.java"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	a := filepath.Join(root, "A.java")
	ncw := "WARN  lint-null-context-write  " + a + ":2  null-Context write: void a() { parent.set(p, v, null); Clock.schedule(this, null, x, null); }\n"
	env := "n4lint: cannot list every file under " + root + ": find: '" + locked + "': Permission denied\n"
	var o, e bytes.Buffer
	if code := lint.Command(All()...)([]string{"--all", root}, cli.Stdio{Out: &o, Err: &e}); code != 3 || o.String() != ncw || e.String() != env {
		t.Errorf("exit %d out %q err %q, want 3, %q and %q", code, o.String(), e.String(), ncw, env)
	}
}

// TestEveryRuleFailsAnEmptyRoot: every registered rule, alone and under --all, reports a root with no
// file it selects as an ERROR row and exit 3, never a clean pass (truth audit 2026-10-22 T6), in its
// Spec.EmptyError shape when it overrides the default. A RootRule (lint-structure) is out of scope: it
// selects no file, --all leaves it out, and its empty-root contract is its own (TestRootRulesOnAnEmptyRoot).
func TestEveryRuleFailsAnEmptyRoot(t *testing.T) {
	root := t.TempDir()
	main := lint.Command(All()...)
	var all string
	for _, sp := range All() {
		if sp.Root != nil {
			continue
		}
		if sp.EmptyOK { // LA5 (pinned by tests/lint-empty-root.bats ER5): an empty root is clean, exit 0, no row in --all
			var o, e bytes.Buffer
			if code := main([]string{sp.ID(), root}, cli.Stdio{Out: &o, Err: &e}); code != 0 || o.Len() != 0 || e.Len() != 0 {
				t.Errorf("%s (EmptyOK): exit %d out %q err %q, want 0 and nothing", sp.ID(), code, o.String(), e.String())
			}
			continue
		}
		// a rule may keep its bash lint's own K20 row (Spec.EmptyError override); every other rule gets the default
		check, msg := sp.ID(), "no *.java sources found (wrong path or empty source tree)"
		if sp.EmptyError.Check != "" {
			check = sp.EmptyError.Check
		}
		if sp.EmptyError.Msg != "" {
			msg = sp.EmptyError.Msg
		}
		row := "ERROR  " + check + "  " + root + "  " + msg + "\n"
		all += row
		var o, e bytes.Buffer
		if code := main([]string{sp.ID(), root}, cli.Stdio{Out: &o, Err: &e}); code != 3 || o.String() != row || e.Len() != 0 {
			t.Errorf("%s: exit %d out %q err %q, want 3 and %q", sp.ID(), code, o.String(), e.String(), row)
		}
	}
	var o, e bytes.Buffer
	if code := main([]string{"--all", root}, cli.Stdio{Out: &o, Err: &e}); code != 3 || o.String() != all || e.Len() != 0 {
		t.Errorf("--all: exit %d out %q err %q, want 3 and\n%s", code, o.String(), e.String(), all)
	}
}

// TestRootRulesOnAnEmptyRoot: a RootRule fails a root that holds nothing it checks like every other rule
// fails an empty root (#351, truth audit 2026-10-22 T6): an ERROR row, no other row, exit 3. For
// lint-structure that is a root without profiles, a wrong path that the bash lint passed clean.
func TestRootRulesOnAnEmptyRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil { // stop the upward walk here
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "gradle.properties"), []byte("niagara_home=C:\\N\n"), 0o644); err != nil {
		t.Fatal(err) // a module-level finding the ERROR row must not be mixed with
	}
	main := lint.Command(All()...)
	var roots []string
	for _, sp := range All() {
		if sp.Root == nil {
			continue
		}
		roots = append(roots, sp.ID())
		var o, e bytes.Buffer
		want := "ERROR  " + sp.ID() + "  " + root + "  " + sp.EmptyError.Msg + "\n"
		if code := main([]string{sp.ID(), root}, cli.Stdio{Out: &o, Err: &e}); code != 3 || sp.EmptyError.Msg == "" || o.String() != want || e.Len() != 0 {
			t.Errorf("%s: exit %d out %q err %q, want 3 and %q", sp.ID(), code, o.String(), e.String(), want)
		}
	}
	if len(roots) != 1 || roots[0] != "lint-structure" {
		t.Errorf("RootRules %q, want exactly lint-structure", roots)
	}
}
