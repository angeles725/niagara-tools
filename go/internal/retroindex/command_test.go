package retroindex

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/gen"
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

func TestMainCheckThenWrite(t *testing.T) {
	t.Parallel()
	root := kit(t, map[string]string{"2026-09-01-a.md": okRetro})
	index := root + "/retros/INDEX.md"
	// --check validates the headers only: the index is not committed, so a missing one is fine
	code, out, errs := run("", "--check", root)
	if code != ExitOK || out != "gen-retro-index: the retro headers under "+root+"/retros are valid\n" || errs != "" {
		t.Errorf("--check: got (%d, %q, %q)", code, out, errs)
	}
	if _, err := os.Stat(index); !os.IsNotExist(err) {
		t.Errorf("--check wrote INDEX.md (stat err %v)", err)
	}
	// The default kit root comes from the wrapper's environment when no <kit-root> is given.
	code, out, errs = run(root)
	if code != ExitOK || out != "gen-retro-index: wrote "+index+"\n" || errs != "" {
		t.Errorf("write: got (%d, %q, %q)", code, out, errs)
	}
	if b, _ := os.ReadFile(index); !strings.HasSuffix(string(b), "| 2026-09-01-a.md | kit | 2026-09-01 | pending | 1 |\n") {
		t.Errorf("written index:\n%s", b)
	}
}

func TestMainViolationWritesNothing(t *testing.T) {
	t.Parallel()
	// Mutation: GRI-required -- defaulting a missing header to an empty cell writes a row nobody can fold
	root := kit(t, map[string]string{"2026-09-01-a.md": okRetro, "2026-09-02-b.md": "<!-- review-status: folded -->\n", "plan.md": okRetro})
	code, out, errs := run("", root)
	want := "gen-retro-index: 2026-09-02-b.md: " + errModule + "\n" +
		"gen-retro-index: plan.md: " + errDate + "\n" +
		"gen-retro-index: retro-header violation; index not generated\n"
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
		!strings.HasSuffix(errs, "; index not generated\n") {
		t.Errorf("got (%d, %q, %q)", code, out, errs)
	}
}

func TestGenerate(t *testing.T) {
	t.Parallel()
	root := kit(t, map[string]string{"2026-10-01-a.md": okRetro})
	var diag bytes.Buffer
	got, code := Generate(root, &diag)
	if code != ExitOK || !strings.HasPrefix(got, gen.GeneratedLine+"\n") ||
		!strings.HasSuffix(got, "| 2026-10-01-a.md | kit | 2026-10-01 | pending | 1 |\n") || diag.String() != "" {
		t.Errorf("Generate = (%q, %d, %q)", got, code, diag.String())
	}
	if _, err := os.Stat(root + "/retros/INDEX.md"); !os.IsNotExist(err) {
		t.Errorf("Generate wrote INDEX.md (stat err %v)", err)
	}
}

// TestMainWriteReadOnlyTarget: the written index ends in exactly one newline, and a read-only index
// left by an earlier run is replaced atomically (its directory is writable) and keeps its mode.
func TestMainWriteReadOnlyTarget(t *testing.T) {
	t.Parallel()
	root := kit(t, map[string]string{"2026-09-01-a.md": okRetro})
	index := root + "/retros/INDEX.md"
	if err := os.WriteFile(index, []byte("stale\n"), 0o444); err != nil {
		t.Fatal(err)
	}
	if code, out, errs := run("", root); code != ExitOK || out != "gen-retro-index: wrote "+index+"\n" || errs != "" {
		t.Errorf("read-only write: got (%d, %q, %q)", code, out, errs)
	}
	st, err := os.Stat(index)
	b, _ := os.ReadFile(index)
	if err != nil || !strings.HasSuffix(string(b), "|\n") || strings.HasSuffix(string(b), "\n\n") || st.Mode().Perm() != 0o444 {
		t.Errorf("read-only write: target %q, want one final newline and mode 0444", b)
	}
}
