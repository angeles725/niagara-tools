package bogaudit

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// parseTime is the one line whose bytes differ run to run.
var parseTime = regexp.MustCompile(`(?m)^# bog-audit: parse time [0-9]+\.[0-9]{3}s$`)

// writeTree writes a case's files under dir: "<path> (zip)" is a .bog holding the text as file.xml,
// "<path> (b64)" a binary file given as base64.
func writeTree(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, text := range files {
		data := []byte(text)
		switch {
		case strings.HasSuffix(name, " (zip)"):
			name = strings.TrimSuffix(name, " (zip)")
			var buf bytes.Buffer
			zw := zip.NewWriter(&buf)
			w, _ := zw.Create("file.xml")
			w.Write(data)
			zw.Close()
			data = buf.Bytes()
		case strings.HasSuffix(name, " (b64)"):
			name = strings.TrimSuffix(name, " (b64)")
			var err error
			if data, err = base64.StdEncoding.DecodeString(strings.ReplaceAll(text, "\n", "")); err != nil {
				t.Fatal(err)
			}
		}
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestGolden runs every testdata case (recorded from the python bog-audit.sh, or a hand-written truth-*
// case) and compares exit code, stdout and stderr byte for byte, the parse-time line aside.
func TestGolden(t *testing.T) {
	paths, _ := filepath.Glob("testdata/*.txt")
	if len(paths) < 23 {
		t.Fatalf("found %d golden cases, want at least 23", len(paths))
	}
	for _, p := range paths {
		t.Run(strings.TrimSuffix(filepath.Base(p), ".txt"), func(t *testing.T) {
			c, err := golden.Read(p)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			writeTree(t, dir, c.Files)
			args := make([]string, len(c.Args))
			for i, a := range c.Args {
				args[i] = strings.ReplaceAll(a, "@DIR@", dir)
			}
			cwd, _ := os.Getwd()
			if err := os.Chdir(dir); err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(cwd)
			var out, errb bytes.Buffer
			code := Main(args, cli.Stdio{Out: &out, Err: &errb})
			got := "exit " + strconv.Itoa(code) + "\n" + out.String()
			want := strings.ReplaceAll(c.Want, "@DIR@", dir)
			norm := func(s string) string { return parseTime.ReplaceAllString(s, "# bog-audit: parse time T") }
			if norm(got) != norm(want) {
				t.Errorf("stdout:\n got: %q\nwant: %q", got, want)
			}
			if gotErr, wantErr := errb.String(), strings.ReplaceAll(c.Stderr, "@DIR@", dir); gotErr != wantErr {
				t.Errorf("stderr:\n got: %q\nwant: %q", gotErr, wantErr)
			}
		})
	}
}
