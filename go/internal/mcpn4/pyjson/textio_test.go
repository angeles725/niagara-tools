package pyjson

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pytarget"
)

func TestReadLinesSplitsUniversalNewlines(t *testing.T) {
	lines, err := ReadLines([]byte("a\r\nb\rc\n\nd\r\r\ne"))
	if err != nil || strings.Join(lines, "|") != "a\n|b\n|c\n|\n|d\n|\n|e" {
		t.Fatalf("%q %v", lines, err)
	}
	if lines, err := ReadLines(nil); err != nil || len(lines) != 0 {
		t.Fatalf("empty: %q %v", lines, err)
	}
}

func TestReadLinesCountsErrorPositionsPerChunk(t *testing.T) {
	ascii := func(n int) []byte { return bytes.Repeat([]byte("a"), n) }
	for _, c := range []struct {
		data []byte
		want string
	}{
		{append(ascii(100), 0xff), "byte 0xff in position 100: invalid start byte"},
		{append(ascii(8192), 0xff), "byte 0xff in position 0: invalid start byte"},
		{append(ascii(20000), 0xff), "byte 0xff in position 3616: invalid start byte"},
		{append(ascii(8191), 0xe2, 0x82, 'x'), "bytes in position 0-1: invalid continuation byte"},
		{append(ascii(8190), 0xe2, 0x82, 'x'), "bytes in position 0-1: invalid continuation byte"},
		{append(ascii(8189), 0xe2, 0x82, 'x'), "bytes in position 8189-8190: invalid continuation byte"},
		{append(ascii(10), 0xe2, 0x82), "bytes in position 0-1: unexpected end of data"},
		{append(ascii(8192), 0xe2), "byte 0xe2 in position 0: unexpected end of data"},
		{append(ascii(5), 0xed, 0xa0, 0x80), "byte 0xed in position 5: invalid continuation byte"},
		{append(ascii(8190), 0xed, 0xa0, 0x80), "byte 0xed in position 0: invalid continuation byte"},
		{append(ascii(5), 0xed, 0xa0), "byte 0xed in position 0: invalid continuation byte"},
		{append(ascii(5), 0xed, 0x80), "bytes in position 0-1: unexpected end of data"},
	} {
		_, err := ReadLines(c.data)
		if ve, ok := err.(*ValueError); !ok || ve.Msg != "'utf-8' codec can't decode "+c.want {
			t.Errorf("%d bytes: got %v, want %s", len(c.data), err, c.want)
		}
	}
	// A character split across a chunk boundary is decoded whole.
	if lines, err := ReadLines(append(ascii(8191), "€\n"...)); err != nil || !strings.HasSuffix(lines[0], "€\n") {
		t.Fatalf("split character: %v", err)
	}
}

// TestReadLinesDifferentialPython reads files with open(path).readlines() on CPython 3.15 and
// with ReadLines, around the 8192-byte chunk boundaries.
func TestReadLinesDifferentialPython(t *testing.T) {
	py := pytarget.Python(t)
	dir := t.TempDir()
	var paths []string
	for i, tail := range [][]byte{{0xff}, {0xe2, 0x82}, {0xe2, 0x82, 'x'}, {0xe2}, {0xf0, 0x9f, 0x98}, {0xed, 0xa0, 0x80},
		[]byte("€\r\n\ry"), {0xc0, 0x80}, {0xf4, 0x90}, []byte("\r"), {0xed, 0xa0}, {0xed, 0x80},
		{0xed, 0xbf, 'a'}, {0xe0, 0x80}, {0xf0, 0x8f, 0x80}, []byte("\xef\xbb\xbf{}")} {
		for _, n := range []int{0, 1, 8189, 8190, 8191, 8192, 16383, 16384, 30000} {
			p := filepath.Join(dir, fmt.Sprintf("t%d-%d.txt", i, n))
			data := append(bytes.Repeat([]byte("ab\r\n"), n/4), bytes.Repeat([]byte("z"), n%4)...)
			if err := os.WriteFile(p, append(data, tail...), 0o600); err != nil {
				t.Fatal(err)
			}
			paths = append(paths, p)
		}
	}
	script := `import json, sys
out = []
for p in sys.argv[1:]:
    try:
        with open(p) as fh:
            out.append(fh.readlines())
    except ValueError as exc:
        out.append(str(exc))
print(json.dumps(out))`
	cmd := exec.Command(py, append([]string{"-I", "-c", script}, paths...)...)
	cmd.Env = append(os.Environ(), "LC_ALL=C", "PYTHONUTF8=")
	want, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var got []any
	for _, p := range paths {
		data, _ := os.ReadFile(p)
		lines, err := ReadLines(data)
		if err != nil {
			got = append(got, err.Error())
			continue
		}
		got = append(got, lines)
	}
	pyOut, err := Loads(want)
	if err != nil || len(pyOut.([]any)) != len(got) {
		t.Fatalf("python answered %d results for %d files (%v)", len(pyOut.([]any)), len(got), err)
	}
	for i, w := range pyOut.([]any) {
		g, _ := Marshal(got[i], Options{})
		wb, _ := Marshal(w, Options{})
		if string(g) != string(wb) {
			t.Errorf("%s: Go and Python differ\ngo:     %.300s\npython: %.300s", filepath.Base(paths[i]), g, wb)
		}
	}
}
