package modulemap

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: one case per check and mode (strict-clean, no-map, no-parts, the MM2 header and table shapes,
// missing-path, unlisted-strict, unlisted-legacy, dup, no-test-strict, java-glob, glob-segment).
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 15) }

// TestUnreadablePart: a part sub-directory the walk cannot read fails the scan closed (exit 3, the reason on
// stderr) instead of reading its files as absent (no MM4 row, never a clean pass).
func TestUnreadablePart(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	root := filepath.Join(t.TempDir(), "M")
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(lint.MapFile, "modularity: strict\n\n| Concern | Part | Files | Tests | Change here when |\n|---|---|---|---|---|\n"+
		"| control | rt | M-rt/src/A.java | M-rt/srcTest/ATest.java | the decision |\n")
	write("M-rt/src/A.java", "class A {}\n")
	write("M-rt/srcTest/ATest.java", "class ATest {}\n")
	locked := filepath.Join(root, "M-rt", "src", "locked")
	if err := os.Mkdir(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	var o, e bytes.Buffer
	code := lint.Command(Spec())([]string{ID, root}, cli.Stdio{Out: &o, Err: &e})
	if code != 3 || o.Len() != 0 || !bytes.Contains(e.Bytes(), []byte(locked)) {
		t.Errorf("exit %d out %q err %q, want 3, no row and the locked path on stderr", code, o.String(), e.String())
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
