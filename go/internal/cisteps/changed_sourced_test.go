package cisteps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A sourced file is read with its sourcer: a lone ci-local.sh read CACHE as unused without its libs. The
// change's shell files are checked with the files they source and the files that source them, never CI's
// whole set unless a source path cannot be resolved. (retro 2026-11-15 coordination-stretch C3, T14)
func TestChangedShellcheckTakesTheChangedScriptWithWhatItSourcesNotTheWholeSet(t *testing.T) {
	dir, git := gitRepo(t)
	write(t, dir, "scripts/lib.sh", "LIBVAR=1\n")      // sourced by the changed script, unchanged by the range
	write(t, dir, "scripts/other.sh", "echo other\n")  // in the set, unrelated: left out
	write(t, dir, ".shellcheckrc", "disable=SC2002\n") // the project config CI's shellcheck reads
	git("add", "-A")
	git("commit", "-q", "-m", "lib")
	base := git("rev-parse", "HEAD")
	write(t, dir, "scripts/old.sh", "# shellcheck disable=SC1091\n. \"$ROOT/scripts/lib.sh\"\n")
	write(t, dir, "tests/t.bats", "@test \"a\" { :; }\n")
	git("add", "-A")
	git("commit", "-q", "-m", "change")
	head := git("rev-parse", "HEAD")
	tree := filepath.Join(t.TempDir(), "tree")
	code, out, errs := runChanged(t, dir, "--tree", tree, "--range", base+".."+head)
	if want := "shellcheck\tscripts/lib.sh\nshellcheck\tscripts/old.sh\nshellcheck\ttests/t.bats\nbats\ttests/t.bats\n"; code != 0 || out != want {
		t.Fatalf("exit %d, plan %q, want %q\n%s", code, out, want, errs)
	}
	if b, _ := os.ReadFile(filepath.Join(tree, "scripts/lib.sh")); string(b) != "LIBVAR=1\n" {
		t.Errorf("the sourced file was not materialized: %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(tree, ".shellcheckrc")); string(b) != "disable=SC2002\n" {
		t.Errorf("the project's .shellcheckrc was not materialized beside the set: %q", b)
	}
	// only a bats file changed, outside the shellcheck globs of this workflow: no shellcheck at all
	write(t, dir, ".github/workflows/ci.yml", strings.Replace(workflow, " tests/*.bats", "", 1))
	git("add", "-A")
	git("commit", "-q", "-m", "narrow")
	mid := git("rev-parse", "HEAD")
	write(t, dir, "tests/t.bats", "@test \"b\" { :; }\n")
	git("add", "-A")
	git("commit", "-q", "-m", "bats only")
	code, out, _ = runChanged(t, dir, "--tree", filepath.Join(t.TempDir(), "t2"), "--range", mid+"..HEAD")
	if code != 0 || out != "bats\ttests/t.bats\n" {
		t.Errorf("exit %d, plan %q", code, out)
	}
}

func TestChangedShellcheckTakesTheWholeSetWithANoteWhenASourceCannotBeResolved(t *testing.T) {
	dir, git := gitRepo(t)
	write(t, dir, "scripts/other.sh", "echo other\n")
	git("add", "-A")
	git("commit", "-q", "-m", "other")
	base := git("rev-parse", "HEAD")
	write(t, dir, "scripts/dyn.sh", ". \"$1\"\n")
	git("add", "-A")
	git("commit", "-q", "-m", "dynamic source")
	code, out, errs := runChanged(t, dir, "--tree", filepath.Join(t.TempDir(), "tree"), "--range", base+"..HEAD")
	if want := "shellcheck\tscripts/dyn.sh\nshellcheck\tscripts/old.sh\nshellcheck\tscripts/other.sh\n"; code != 0 || out != want {
		t.Fatalf("exit %d, plan %q, want %q\n%s", code, out, want, errs)
	}
	if !strings.Contains(errs, "scripts/dyn.sh:1: cannot resolve the sourced path; shellchecking CI's whole file set") {
		t.Errorf("no fallback note on stderr: %q", errs)
	}
}
