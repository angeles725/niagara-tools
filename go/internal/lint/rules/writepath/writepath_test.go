package writepath

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-write-path.bats and tests/write-path-drift.bats run, the report-module
// fixtures and the edge trees, recorded from the bash lint.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 77) }

// TestZippedBog: a saved station config.bog is file.xml zipped; it gives the rows of the same bog read
// as plain XML (the edge-bog-module-root tree, which has bog-linked FAIL rows).
func TestZippedBog(t *testing.T) {
	c, err := golden.Read("testdata/edge-bog-module-root.txt")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for p, text := range c.Files {
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var zipped bytes.Buffer
	zw := zip.NewWriter(&zipped)
	w, _ := zw.Create("file.xml")
	if _, err := w.Write([]byte(c.Files["bog/config.bog"])); err != nil || zw.Close() != nil {
		t.Fatal("cannot zip the bog")
	}
	zbog := filepath.Join(dir, "bog", "zipped.bog")
	if err := os.WriteFile(zbog, zipped.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	run := func(bog string) (int, string) {
		var o, e bytes.Buffer
		code := lint.Command(Spec())([]string{"lint-write-path", filepath.Join(dir, "bog/ColdRoomPan"), "--bog", bog}, cli.Stdio{Out: &o, Err: &e})
		return code, o.String() + e.String()
	}
	pc, plain := run(filepath.Join(dir, "bog", "config.bog"))
	zc, got := run(zbog)
	if zc != pc || got != plain || !strings.Contains(got, "slot panelSlot") {
		t.Errorf("zipped: exit %d\n%s\nplain: exit %d\n%s", zc, got, pc, plain)
	}
}
