package cisteps

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

func runMain(args ...string) (int, string, string) {
	var o, e bytes.Buffer
	code := Main(args, cli.Stdio{Out: &o, Err: &e})
	return code, o.String(), e.String()
}

func TestMainWritesNULRecords(t *testing.T) {
	t.Parallel()
	p := filepath.Join(t.TempDir(), "w.yml")
	yml := head + "      - uses: a/b@v1\n      - name: r\n        run: |\n          x\n          y\n      - name: e\n        env: {}\n        run: z\n"
	if err := os.WriteFile(p, []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errs := runMain(p)
	want := "uses\x00a/b@v1\x00a/b@v1\x00run\x00r\x00x\ny\x00bad\x00e\x00step-level env: not replayed\x00"
	if code != 0 || out != want || errs != "" {
		t.Errorf("Main = %d, %q, %q; want 0, %q, \"\"", code, out, errs, want)
	}
}

func TestMainParseErrorExits2WithNoRecords(t *testing.T) {
	t.Parallel()
	p := filepath.Join(t.TempDir(), "w.yml")
	if err := os.WriteFile(p, []byte(head+"      - run: ok\n      - run: >\n          x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errs := runMain(p)
	if want := p + ":8: run: > block style not supported (use | or |-)\n"; code != 2 || out != "" || errs != want {
		t.Errorf("Main = %d, %q, %q; want 2, \"\", %q", code, out, errs, want)
	}
}

func TestMainUnreadableFileExits2LikePython(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	missing := filepath.Join(dir, "nope.yml")
	if code, out, errs := runMain(missing); code != 2 || out != "" || errs != "[Errno 2] No such file or directory: '"+missing+"'\n" {
		t.Errorf("missing file: %d, %q, %q", code, out, errs)
	}
	if code, _, errs := runMain(dir); code != 2 || errs != "[Errno 21] Is a directory: '"+dir+"'\n" {
		t.Errorf("directory: %d, %q", code, errs)
	}
}

func TestMainUsage(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{nil, {"a", "b"}} {
		if code, out, errs := runMain(args...); code != 2 || out != "" || errs != "usage: ci-steps <workflow.yml>\n" {
			t.Errorf("Main(%q) = %d, %q, %q", args, code, out, errs)
		}
	}
}
