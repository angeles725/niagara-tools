package modulemap

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/gitcmd"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

const (
	legacyHead = "modularity: legacy\nlegacy-reason: predates the map\n\n"
	strictHead = "modularity: strict\n\n"
	tableHead  = "| Concern | Part | Files | Tests | Change here when |\n|---|---|---|---|---|\n"
	newTest    = "M-ux/webtest/new.test.mjs"
)

// mapOf is a one-concern map under head whose Tests cell is ATest plus extra.
func mapOf(head, extra string) string {
	return head + tableHead + "| control | rt | M-rt/src/A.java | M-rt/srcTest/ATest.java" + extra + " | the decision |\n"
}

// repoTree is a module root M inside a git repository, committed and tagged "base": the map, A.java, ATest.java and
// an unlisted old SPA test. The tree is kept off the host's git configuration.
func repoTree(t *testing.T, head string) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	root := tree(t, map[string]string{lint.MapFile: mapOf(head, ""), "M-ux/webtest/old.test.mjs": "x\n"})
	commit(t, filepath.Dir(root), "base", "init", "-q", "-b", "main")
	return root
}

// commit runs git in dir (the first call of repoTree inits it) then commits everything and tags it.
func commit(t *testing.T, dir, tag string, init ...string) {
	t.Helper()
	g := func(args ...string) {
		if out, ok := gitcmd.Run(dir, nil, args...); !ok {
			t.Fatalf("git %v failed: %s", args, out)
		}
	}
	if len(init) > 0 {
		g(init...)
	}
	g("add", "-A")
	g("-c", "user.name=t", "-c", "user.email=t@example.invalid", "-c", "commit.gpgsign=false", "commit", "-q", "-m", tag)
	g("tag", "-f", tag)
}

// addFiles writes the files under root, then commits them as "next".
func addFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	commit(t, filepath.Dir(root), "next")
}

func runArgs(args ...string) (int, string, string) {
	var o, e bytes.Buffer
	code := lint.Command(Spec())(append([]string{ID}, args...), cli.Stdio{Out: &o, Err: &e})
	return code, o.String(), e.String()
}

const mm8Msg = "MM8: new file in no concern (list it in the Files or Tests of the concern that owns it, in the same change)"

// TestMM8NewUnlistedFileInLegacyWarns: an SPA test added since the base and in no row is one WARN row (exit 0).
func TestMM8NewUnlistedFileInLegacyWarns(t *testing.T) {
	root := repoTree(t, legacyHead)
	addFiles(t, root, map[string]string{newTest: "y\n"})
	code, out, errs := runArgs("--base", "base", root)
	want := "WARN  lint-module-map  " + root + "/" + newTest + "  " + mm8Msg + "\n"
	if code != 0 || out != want || errs != "" {
		t.Errorf("exit %d out %q err %q, want 0, %q and no stderr", code, out, errs, want)
	}
}

// TestMM8NewUnlistedFileInStrictFails: the same file in a strict module is a FAIL row (exit 1).
func TestMM8NewUnlistedFileInStrictFails(t *testing.T) {
	root := repoTree(t, strictHead)
	addFiles(t, root, map[string]string{newTest: "y\n"})
	code, out, errs := runArgs("--base", "base", root)
	want := "FAIL  lint-module-map  " + root + "/" + newTest + "  " + mm8Msg + "\n"
	if code != 1 || out != want || errs != "" {
		t.Errorf("exit %d out %q err %q, want 1, %q and no stderr", code, out, errs, want)
	}
}

// TestMM8NewListedFileIsClean: a new file a row lists (exactly or by an src/rc glob) is no row at all.
func TestMM8NewListedFileIsClean(t *testing.T) {
	root := repoTree(t, strictHead)
	addFiles(t, root, map[string]string{newTest: "y\n", "M-ux/src/rc/vendor/lib.js": "z\n"})
	m := mapOf(strictHead, "; "+newTest+"; M-ux/src/rc/vendor/*.js")
	if err := os.WriteFile(filepath.Join(root, lint.MapFile), []byte(m), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errs := runArgs("--base", "base", root)
	if code != 0 || out != "" || errs != "" {
		t.Errorf("exit %d out %q err %q, want a clean run", code, out, errs)
	}
}

// TestMM8OldUnlistedFileIsNotReported: a file already at the base (old.test.mjs) is not new, so a legacy module
// whose only change is an edit of a listed file gets no MM8 row for it.
func TestMM8OldUnlistedFileIsNotReported(t *testing.T) {
	root := repoTree(t, legacyHead)
	addFiles(t, root, map[string]string{"M-rt/src/A.java": "class A { int x; }\n"})
	code, out, errs := runArgs("--base", "base", root)
	if code != 0 || out != "" || errs != "" {
		t.Errorf("exit %d out %q err %q, want a clean run", code, out, errs)
	}
}

// TestMM8NeedsABaseRef: with no --base the check does not run (no MM8 row, the lint's other output unchanged);
// with a --base that does not resolve it is one INFO row saying so, and still no MM8 row.
func TestMM8NeedsABaseRef(t *testing.T) {
	root := repoTree(t, legacyHead)
	addFiles(t, root, map[string]string{newTest: "y\n"})
	if code, out, errs := runArgs(root); code != 0 || out != "" || errs != "" {
		t.Errorf("no --base: exit %d out %q err %q, want a clean run", code, out, errs)
	}
	code, out, errs := runArgs("--base", "no-such-ref", root)
	want := "INFO  lint-module-map  " + root + "  MM8: new-file check skipped: base ref no-such-ref does not resolve " +
		"here (needs a base ref in the module's repository, e.g. --base origin/main)\n"
	if code != 0 || out != want || errs != "" {
		t.Errorf("unresolved --base: exit %d out %q err %q, want 0, %q and no stderr", code, out, errs, want)
	}
}

// TestMM8LeavesMM4ToMM4: a new source file in no row stays MM4's row alone; a new srcTest Java file and a new
// vendored file are MM8's, whatever the module's mode.
func TestMM8LeavesMM4ToMM4(t *testing.T) {
	root := repoTree(t, legacyHead)
	addFiles(t, root, map[string]string{"M-rt/src/B.java": "class B {}\n", "M-rt/srcTest/BTest.java": "class BTest {}\n",
		"M-ux/src/rc/vendor/lib.js": "z\n"})
	_, out, _ := runArgs("--base", "base", root)
	var mm8 []string
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "  MM8: ") {
			mm8 = append(mm8, strings.Fields(l)[2])
		}
	}
	want := []string{root + "/M-rt/srcTest/BTest.java", root + "/M-ux/src/rc/vendor/lib.js"}
	if strings.Join(mm8, "|") != strings.Join(want, "|") || !strings.Contains(out, "B.java  MM4: in no concern") {
		t.Errorf("out %q: want MM8 rows for %v only and B.java left to MM4", out, want)
	}
}
