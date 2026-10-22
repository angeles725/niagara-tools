package retroindex

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const usageLine = "usage: gen-retro-index.sh [--check] [<kit-root>]\n"

// kit makes a kit root holding retros/ with the given retro files.
func kit(t *testing.T, retros map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "retros"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, text := range retros {
		if err := os.WriteFile(filepath.Join(root, "retros", name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// run calls Main with buffers; defaultRoot is what the wrapper would export.
func run(defaultRoot string, args ...string) (code int, out, errs string) {
	var o, e bytes.Buffer
	s := cli.Stdio{Out: &o, Err: &e, Getenv: func(name string) string {
		if name == DefaultRootEnv {
			return defaultRoot
		}
		return ""
	}}
	code = Main(args, s)
	return code, o.String(), e.String()
}

const okRetro = "<!-- review-status: pending -->\n<!-- module: kit -->\n<!-- deltas: 1 -->\n# x\n"

func TestMainUsage(t *testing.T) {
	t.Parallel()
	root := kit(t, nil)
	for _, args := range [][]string{{"--help"}, {"-x"}, {"--check", "--bogus"}, {root, root}, {"--"}} {
		code, out, errs := run("", args...)
		if code != ExitEnv || out != "" || errs != usageLine {
			t.Errorf("%q: got (%d, %q, %q), want (3, \"\", usage)", args, code, out, errs)
		}
	}
}

func TestMainNoRetrosDir(t *testing.T) {
	t.Parallel()
	missing := filepath.Join(t.TempDir(), "nope")
	code, _, errs := run("", missing)
	if code != ExitEnv || errs != "gen-retro-index: no retros/ under "+missing+"\n" {
		t.Errorf("got (%d, %q)", code, errs)
	}
}

func TestMainWriteThenCheck(t *testing.T) {
	t.Parallel()
	root := kit(t, map[string]string{"2026-09-01-a.md": okRetro})
	index := root + "/retros/INDEX.md"
	code, out, errs := run("", "--check", root)
	if code != ExitStale || errs != "gen-retro-index: "+index+" is missing; run gen-retro-index.sh\n" {
		t.Errorf("--check before write: got (%d, %q)", code, errs)
	}
	// The default kit root comes from the wrapper's environment when no <kit-root> is given.
	code, out, errs = run(root)
	if code != ExitOK || out != "gen-retro-index: wrote "+index+"\n" || errs != "" {
		t.Errorf("write: got (%d, %q, %q)", code, out, errs)
	}
	code, out, _ = run("", root, "--check")
	if code != ExitOK || out != "gen-retro-index: "+index+" is fresh\n" {
		t.Errorf("--check fresh: got (%d, %q)", code, out)
	}
	// Mutation: GRI-stale -- dropping the generated-vs-committed compare makes --check vacuously clean
	if err := os.WriteFile(filepath.Join(root, "retros", "2026-09-01-a.md"), []byte(strings.Replace(okRetro, "pending", "folded", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errs = run("", "--check", root)
	want := "gen-retro-index: " + index + " is stale; run gen-retro-index.sh and commit the result\n"
	if code != ExitStale || out != "" || !strings.HasPrefix(errs, want) || !strings.Contains(errs, "> | 2026-09-01-a.md | kit | 2026-09-01 | pending | 1 |") {
		t.Errorf("--check stale: got (%d, %q, %q)", code, out, errs)
	}
}

func TestMainViolationWritesNothing(t *testing.T) {
	t.Parallel()
	// Mutation: GRI-required -- defaulting a missing header to an empty cell writes a row nobody can fold
	root := kit(t, map[string]string{"2026-09-01-a.md": okRetro, "2026-09-02-b.md": "<!-- review-status: folded -->\n", "plan.md": okRetro})
	code, out, errs := run("", root)
	want := "gen-retro-index: 2026-09-02-b.md: " + errModule + "\n" +
		"gen-retro-index: plan.md: " + errDate + "\n" +
		"gen-retro-index: retro-header violation; index not written\n"
	if code != ExitStale || out != "" || errs != want {
		t.Errorf("got (%d, %q, %q)", code, out, errs)
	}
	if _, err := os.Stat(filepath.Join(root, "retros", "INDEX.md")); !os.IsNotExist(err) {
		t.Errorf("INDEX.md was written despite a violation (stat err %v)", err)
	}
}

func TestMainUnlistableRetrosDirExits3(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	root := kit(t, map[string]string{"2026-09-01-a.md": okRetro})
	dir := filepath.Join(root, "retros")
	if err := os.Chmod(dir, 0o300); err != nil { // a directory, but not readable
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	code, out, errs := run("", root)
	if code != ExitEnv || out != "" || !strings.HasPrefix(errs, "gen-retro-index: cannot list "+dir+": ") ||
		!strings.HasSuffix(errs, "; index not written\n") {
		t.Errorf("got (%d, %q, %q)", code, out, errs)
	}
}
