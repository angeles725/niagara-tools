package wbusability

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

// TestGolden: one hand-written case per testdata/*.txt (the narrow part of the lint candidate in
// types/wb-widgets.md, UX rule). A class that extends BWbComponentView and holds a BOrd member, in a file with no
// BComponentChooser and no targetType facet, WARNs at the class; a chooser or facet in code, a waiver comment with a
// reason, another base class, a static constant, a local variable and a comment or string that only names the
// chooser are judged as the cases say; --strict; the empty root and usage.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 20) }

// TestUnreadable: an unreadable sub-directory is exit 3 with nothing on stdout (the engine's fail-closed walk): a
// WARN row from the files that were read would pass for a complete scan.
func TestUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads every directory")
	}
	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	if err := os.WriteFile(filepath.Join(dir, "A.java"), []byte("class A extends BWbComponentView {\n  BOrd t;\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	var out, errw bytes.Buffer
	code := lint.Command(Spec())([]string{ID, dir}, cli.Stdio{Out: &out, Err: &errw})
	if code != 3 || out.Len() != 0 || !strings.Contains(errw.String(), "cannot list every file under "+dir) {
		t.Errorf("exit %d out %q err %q, want 3, no stdout and the listing error", code, out.String(), errw.String())
	}
}
