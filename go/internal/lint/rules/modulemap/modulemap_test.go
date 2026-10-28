package modulemap

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: one case per check and mode (strict-clean, no-map, no-parts, the MM2 header and table shapes,
// missing-path, unlisted-strict, unlisted-legacy, dup, no-test-strict, java-glob, glob-segment).
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 15) }

// strictMap is a one-concern strict map over M-rt/src/A.java and its test.
const strictMap = "modularity: strict\n\n| Concern | Part | Files | Tests | Change here when |\n|---|---|---|---|---|\n" +
	"| control | rt | M-rt/src/A.java | M-rt/srcTest/ATest.java | the decision |\n"

// tree writes a module root M under a temp dir: the strict map, A.java and ATest.java plus the extra files.
func tree(t *testing.T, extra map[string]string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "M")
	files := map[string]string{lint.MapFile: strictMap, "M-rt/src/A.java": "class A {}\n", "M-rt/srcTest/ATest.java": "class ATest {}\n"}
	for k, v := range extra {
		files[k] = v
	}
	for rel, body := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// run is the wrapper's command on root: exit code, stdout, stderr.
func run(root string) (int, string, string) {
	var o, e bytes.Buffer
	code := lint.Command(Spec())([]string{ID, root}, cli.Stdio{Out: &o, Err: &e})
	return code, o.String(), e.String()
}

// TestUnreadablePart: a part sub-directory the walk cannot read fails the scan closed (exit 3, the reason on
// stderr) instead of reading its files as absent (no MM4 row, never a clean pass).
func TestUnreadablePart(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	root := tree(t, nil)
	locked := filepath.Join(root, "M-rt", "src", "locked")
	if err := os.Mkdir(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	code, out, errs := run(root)
	if code != 3 || out != "" || !strings.Contains(errs, locked) {
		t.Errorf("exit %d out %q err %q, want 3, no row and the locked path on stderr", code, out, errs)
	}
}

// TestUnreadableMap: an existing MODULE-MAP.md the lint cannot read fails closed (exit 3, the map path on
// stderr), never an MM1 "no map" WARN or a clean pass.
func TestUnreadableMap(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	root := tree(t, nil)
	m := filepath.Join(root, lint.MapFile)
	if err := os.Chmod(m, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(m, 0o644) })
	code, out, errs := run(root)
	if code != 3 || out != "" || !strings.Contains(errs, m) {
		t.Errorf("exit %d out %q err %q, want 3, no row and the map path on stderr", code, out, errs)
	}
}

// TestCyclicDirSymlink: a directory symlink that points back up the part is not descended (no ELOOP exit 3,
// no exponential walk): the module stays clean.
func TestCyclicDirSymlink(t *testing.T) {
	root := tree(t, nil)
	src := filepath.Join(root, "M-rt", "src")
	for _, n := range []string{"loop", "loop2"} { // two links: the old walk grew exponentially
		if err := os.Symlink("..", filepath.Join(src, n)); err != nil {
			t.Fatal(err)
		}
	}
	if code, out, errs := run(root); code != 0 || out != "" || errs != "" {
		t.Errorf("exit %d out %q err %q, want a clean exit 0", code, out, errs)
	}
}

// TestSiblingDirAlias: a symlinked sibling directory does not list its target's files a second time (no
// spurious MM4 unlisted or duplicate row): only the real path counts.
func TestSiblingDirAlias(t *testing.T) {
	root := tree(t, map[string]string{
		lint.MapFile:          strictMap + "| codec | rt | M-rt/src/pkg/B.java | M-rt/srcTest/ATest.java | the frame |\n",
		"M-rt/src/pkg/B.java": "class B {}\n",
	})
	if err := os.Symlink("pkg", filepath.Join(root, "M-rt", "src", "alias")); err != nil {
		t.Fatal(err)
	}
	if code, out, errs := run(root); code != 0 || out != "" || errs != "" {
		t.Errorf("exit %d out %q err %q, want a clean exit 0", code, out, errs)
	}
}

// TestMatchGlob: ** spans any number of segments (none included), * and ? stay inside one segment.
func TestMatchGlob(t *testing.T) {
	for _, tc := range []struct {
		pat, rel string
		want     bool
	}{
		{"a/src/rc/**", "a/src/rc/index.html", true},
		{"a/src/rc/**", "a/src/rc/js/x/y.js", true},
		{"a/src/rc/**", "a/src/rcx/y.js", false},
		{"a/src/rc/js/*.js", "a/src/rc/js/y.js", true},
		{"a/src/rc/js/*.js", "a/src/rc/js/p/y.js", false},
		{"a/**/spec/*.js", "a/srcTest/rc/spec/S.js", true},
		{"a/**/spec/*.js", "a/spec/S.js", true},
		{"a/src/rc/**/*.css", "a/src/rc/s.css", true},
		{"a/src/rc/[", "a/src/rc/[", false},
	} {
		if got := matchGlob(tc.pat, tc.rel); got != tc.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", tc.pat, tc.rel, got, tc.want)
		}
	}
}

// TestSummarize (modularity-gaps Δ8): the counts report-module's modularity row prints: the mode, the map's
// concern rows and the part source files in no concern; a module with no valid map has neither count (-1).
func TestSummarize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		extra map[string]string
		want  Summary
	}{
		{"strict, every file listed", nil, Summary{Mode: lint.ModularityStrict, Concerns: 1, Unlisted: 0}},
		{"two unlisted sources", map[string]string{"M-rt/src/B.java": "class B {}\n", "M-ux/src/rc/app.js": "x\n"},
			Summary{Mode: lint.ModularityStrict, Concerns: 1, Unlisted: 2}},
		{"no map", map[string]string{lint.MapFile: ""}, Summary{Mode: lint.ModularityNone, Concerns: -1, Unlisted: -1}},
		{"no table", map[string]string{lint.MapFile: "modularity: strict\n"}, Summary{Mode: lint.ModularityNone, Concerns: -1, Unlisted: -1}},
	}
	for _, tt := range tests {
		root := tree(t, tt.extra)
		if tt.extra != nil && tt.extra[lint.MapFile] == "" && len(tt.extra) == 1 {
			if err := os.Remove(filepath.Join(root, lint.MapFile)); err != nil {
				t.Fatal(err)
			}
		}
		got, err := Summarize(root)
		if err != nil || got != tt.want {
			t.Errorf("%s: Summarize = %+v, %v; want %+v", tt.name, got, err, tt.want)
		}
	}
}
