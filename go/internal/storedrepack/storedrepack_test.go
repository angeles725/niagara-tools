package storedrepack

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestEntryOrder pins what the goldens cannot: their listing sorts the entries after the signature head (the
// bash wrote those in its extraction dir's walk order), while the port keeps them in input order. The jar has
// its rest out of name order, directory entries, two aliases (one with all three blocks), an orphan block and
// a nested .SF; the output is the manifest, each top-level .SF in byte order with its .RSA, .DSA and .EC
// blocks, then every other file entry in input order.
func TestEntryOrder(t *testing.T) {
	jar, err := golden.BuildJar("--- META-INF/ ---\n--- z/last.txt ---\nz\n--- META-INF/B.SF ---\nb\n--- a/ ---\n" +
		"--- a/first.txt ---\na\n--- META-INF/MANIFEST.MF ---\nM\n--- META-INF/C.RSA ---\no\n--- META-INF/A.EC ---\ne\n" +
		"--- META-INF/sub/N.SF ---\nn\n--- META-INF/B.RSA ---\nr\n--- META-INF/A.SF ---\na\n--- m/mid.txt (stored) ---\nm\n" +
		"--- META-INF/A.DSA ---\nd\n--- META-INF/A.RSA ---\nr\n")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	in, out := filepath.Join(dir, "in.jar"), filepath.Join(dir, "out.jar")
	if err := os.WriteFile(in, []byte(jar), 0o644); err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	if code := Main([]string{in, out}, cli.Stdio{Out: &o, Err: &e}); code != 0 {
		t.Fatalf("exit %d: %s", code, e.String())
	}
	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	var got []string
	for _, f := range zr.File {
		got = append(got, f.Name)
	}
	want := []string{
		"META-INF/MANIFEST.MF",
		"META-INF/A.SF", "META-INF/A.RSA", "META-INF/A.DSA", "META-INF/A.EC",
		"META-INF/B.SF", "META-INF/B.RSA",
		"z/last.txt", "a/first.txt", "META-INF/C.RSA", "META-INF/sub/N.SF", "m/mid.txt",
	}
	if !slices.Equal(got, want) {
		t.Errorf("entries\n got %q\nwant %q", got, want)
	}
}

// TestUsageIsTheScriptHeader pins usage to lines 2-15 of the wrapper with their "# " taken off: the bash
// printed `sed -n '2,15p' "$0"`, and the wrapper keeps that header.
func TestUsageIsTheScriptHeader(t *testing.T) {
	b, err := os.ReadFile("../../../build-n4-module-kit/toolbelt/stored-repack.sh")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.SplitAfter(string(b), "\n")
	if len(lines) < 15 {
		t.Fatalf("the wrapper has %d lines, want its 15-line header", len(lines))
	}
	var want strings.Builder
	for _, ln := range lines[1:15] {
		if rest, ok := strings.CutPrefix(ln, "#"); ok {
			ln = strings.TrimPrefix(rest, " ")
		}
		want.WriteString(ln)
	}
	if usage != want.String() {
		t.Errorf("usage\n%s\nwant (the script header)\n%s", usage, want.String())
	}
}
