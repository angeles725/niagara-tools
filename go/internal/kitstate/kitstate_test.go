package kitstate

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/buildstate"
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/gen"
)

const (
	lint   = "#!/usr/bin/env bash\n# lint-demo.sh — flags demo things.\n#\n# Usage:  lint-demo.sh <src-root>\n#   Exit: 0 clean · 1 any FAIL\nexit 0\n"
	retro  = "<!-- review-status: pending -->\n<!-- module: kit -->\n<!-- deltas: 1 -->\n# x\n"
	frag   = "<!-- ledger: kit -->\n<!-- date: 2026-10-24 -->\nlog: 2026-10-24 · demo change\nopen_issue: OWED demo\n"
	source = "<!-- build-state.v1 -->\nmodule: kit\n%LC%\nopen_issues:\nretro_required: true\nretro_pending: true\n" +
		"last_session: below\n<!-- /build-state.v1 -->\n"
)

// kit makes a kit root with a BUILD-STATE.src.md, one ledger fragment and one retro.
func kit(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for path, text := range map[string]string{
		buildstate.SourceName:         strings.Replace(source, "%LC%", buildstate.LastChange(nil), 1),
		"ledger.d/2026-10-24-demo.md": frag,
		"retros/2026-10-24-demo-a.md": retro,
		"toolbelt/lint-demo.sh":       lint,
	} {
		p := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

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

func TestUsage(t *testing.T) {
	t.Parallel()
	root := kit(t)
	for _, args := range [][]string{nil, {root}, {"--bogus", "build-state"}, {"build-state", "retro-index"},
		{"build-state", root, root}, {"--write", "build-state", "build-state"}} {
		if code, out, errs := run(root, args...); code != gen.ExitEnv || out != "" || errs != usage {
			t.Errorf("%q: got (%d, %q, %q), want usage", args, code, out, errs)
		}
	}
	if code, _, errs := run("", "build-state"); code != gen.ExitEnv || !strings.Contains(errs, DefaultRootEnv) {
		t.Errorf("no root: got (%d, %q)", code, errs)
	}
}

// TestPrintIsTheGeneratorOutput: kit-state renders nothing itself; it prints what the generators make,
// and writes no file.
func TestPrintIsTheGeneratorOutput(t *testing.T) {
	t.Parallel()
	root := kit(t)
	code, out, errs := run(root, "build-state")
	if code != gen.ExitOK || errs != "" || !strings.HasPrefix(out, gen.GeneratedLine+"\n") ||
		!strings.Contains(out, "last_change: 2026-10-24\nopen_issues:\n  - OWED demo\n") ||
		!strings.HasSuffix(out, "last_session: below\n2026-10-24 · demo change\n<!-- /build-state.v1 -->\n") {
		t.Errorf("build-state: got (%d, %q, %q)", code, out, errs)
	}
	code, out, errs = run("", "retro-index", root)
	if code != gen.ExitOK || errs != "" || !strings.HasPrefix(out, gen.GeneratedLine+"\n") ||
		!strings.HasSuffix(out, "| 2026-10-24-demo-a.md | kit | 2026-10-24 | pending | 1 |\n") {
		t.Errorf("retro-index: got (%d, %q, %q)", code, out, errs)
	}
	for _, f := range []string{root + "/BUILD-STATE.md", root + "/retros/INDEX.md"} {
		if _, err := os.Stat(f); err == nil {
			t.Errorf("printing wrote %s", f)
		}
	}
}

func TestWriteBothAndOne(t *testing.T) {
	t.Parallel()
	root := kit(t)
	state, index, lints := root+"/BUILD-STATE.md", root+"/retros/INDEX.md", root+"/toolbelt/INDEX.md"
	code, out, errs := run(root, "--write")
	if code != gen.ExitOK || errs != "" || out != "kit-state: wrote "+state+"\nkit-state: wrote "+index+"\nkit-state: wrote "+lints+"\n" {
		t.Errorf("--write: got (%d, %q, %q)", code, out, errs)
	}
	if b, err := os.ReadFile(lints); err != nil || !strings.Contains(string(b), "| `toolbelt/lint-demo.sh` | flags demo things. |") {
		t.Errorf("lint index: %v %q", err, b)
	}
	for _, f := range []string{state, index} {
		if b, err := os.ReadFile(f); err != nil || !strings.HasPrefix(string(b), gen.GeneratedLine+"\n") {
			t.Errorf("%s: %v %q", f, err, b)
		}
	}
	if code, out, _ = run("", "retro-index", "--write", root); code != gen.ExitOK || out != "kit-state: wrote "+index+"\n" {
		t.Errorf("--write retro-index: got (%d, %q)", code, out)
	}
}

// TestViolationWritesNothing: every selected file is generated before any is written, so a bad retro
// header leaves no fresh BUILD-STATE.md beside a missing index.
func TestViolationWritesNothing(t *testing.T) {
	t.Parallel()
	root := kit(t)
	if err := os.WriteFile(root+"/retros/2026-10-25-bad-b.md", []byte("<!-- review-status: pending -->\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errs := run(root, "--write")
	if code != gen.ExitStale || out != "" || !strings.HasSuffix(errs, "gen-retro-index: retro-header violation; index not generated\n") {
		t.Errorf("got (%d, %q, %q)", code, out, errs)
	}
	if _, err := os.Stat(root + "/BUILD-STATE.md"); err == nil {
		t.Error("BUILD-STATE.md was written although the index could not be generated")
	}
}

// TestRealKit generates every file of the real kit (the reader path every doc points at); each starts
// with its GENERATED comment.
func TestRealKit(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "..", "build-n4-module-kit")
	for view, first := range map[string]string{
		"build-state": gen.GeneratedLine,
		"retro-index": gen.GeneratedLine,
		"lint-index":  "<!-- GENERATED by toolbelt/gen-lint-index.sh",
	} {
		if code, out, errs := run("", view, root); code != gen.ExitOK || errs != "" || !strings.HasPrefix(out, first) {
			t.Errorf("%s: exit %d, stderr %q", view, code, errs)
		}
	}
}

// TestWriteFailureLeavesNoMixedPair: the pair is written as two files; when the second write fails
// (an unwritable retros/), the BUILD-STATE.md already written is removed, so no reader sees a fresh
// ledger beside a stale or missing index.
func TestWriteFailureLeavesNoMixedPair(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root writes into a read-only directory")
	}
	root := kit(t)
	if err := os.WriteFile(root+"/retros/INDEX.md", []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root+"/retros", 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root+"/retros", 0o755) })
	code, _, errs := run(root, "--write")
	if code != gen.ExitEnv || !strings.Contains(errs, "kit-state: cannot write "+root+"/retros/INDEX.md") {
		t.Errorf("got (%d, %q)", code, errs)
	}
	// Mutation: KS-pair -- keeping the first file written leaves a fresh BUILD-STATE.md beside a stale index
	if _, err := os.Stat(root + "/BUILD-STATE.md"); err == nil {
		t.Error("BUILD-STATE.md left behind although retros/INDEX.md could not be written")
	}
	if !strings.Contains(errs, "kit-state: removed "+root+"/BUILD-STATE.md") {
		t.Errorf("the removal is not reported: %q", errs)
	}
}
