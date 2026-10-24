package bformatcall

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

// TestGolden: every tests/lint-bformat-call.bats run on a tree the format can hold and every edge tree, recorded
// from the bash lint (origin/main 45ddb3b5) on copies of the root.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 23) }

// TestUnreadable: the awk_lint_run contract (a walk error and an unreadable file, rows still printed, exit 3).
func TestUnreadable(t *testing.T) {
	golden.Unreadable(t, Spec(), "class A { String f = \"%user%\"; }\n")
}

// TestZippedBog: a zipped .bog (a station's config.bog) is read through xmlsrc.Open, so a pattern in its file.xml is
// found; the awk read the compressed bytes.
func TestZippedBog(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	w, err := z.Create("file.xml")
	if err != nil {
		t.Fatal(err)
	}
	// enough repetition that deflate compresses: the pattern never appears as plain bytes in the file
	body := "<bajaObjectGraph>\n" + strings.Repeat("<p n=\"pad\" v=\"x\"/>\n", 200) + "<p v=\"%user%\"/>\n</bajaObjectGraph>\n"
	if _, err := w.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(buf.Bytes(), []byte("%user%")) {
		t.Fatal("the zip holds the pattern as plain bytes: the case would not discriminate")
	}
	if err := os.WriteFile(filepath.Join(dir, "config.bog"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	code := lint.Command(Spec())([]string{ID, dir}, cli.Stdio{Out: &o, Err: &e})
	if want := "WARN  " + ID + "  " + dir + "/config.bog:202  " + bfc2 + "\n"; code != 0 || o.String() != want || e.Len() != 0 {
		t.Errorf("exit %d out %q err %q, want 0 and %q", code, o.String(), e.String(), want)
	}
}
