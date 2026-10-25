// Package golden freezes the mcp-n4 Python differentials into recorded goldens, so they keep
// comparing the Go port with the kit's Python after that Python is gone. A differential feeds
// its helper script the cases on stdin and compares the Go side with what the script prints.
// With pytarget.RecordEnv=1 the script runs on the target CPython and every call is appended to
// testdata/<name>.golden.jsonl; otherwise the file is replayed and no interpreter is needed.
// Replay is strict, like box/testdata/replay.jsonl: each call must send the recorded script
// and stdin byte for byte, in order, and no recorded call may be left over. A missing golden
// fails: a differential that silently stopped comparing would look like one that passed.
package golden

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pytarget"
)

// record is one recorded call: the script's digest, the cases sent and what Python printed.
// Cases over maxInline bytes are kept as their SHA-256 alone: the test regenerates them
// deterministically, and inlining them would make the golden several times larger.
type record struct {
	Script      string `json:"script"`
	Stdin       string `json:"stdin,omitempty"`
	StdinSHA256 string `json:"stdin_sha256,omitempty"`
	Stdout      string `json:"stdout"`
}

const maxInline = 64 << 10

// Golden is the recorded Python of one differential.
type Golden struct {
	t     testing.TB
	path  string
	py    string // set only when recording
	kit   string
	recs  []record
	alias []string // strings.Replacer pairs from the real text to its placeholder
	masks []func(string) string
	next  int
	hint  string // how to re-record, printed by every stale or missing golden
}

// Open returns the golden testdata/<name>.golden.jsonl of t's package. Recording needs the
// target CPython (pytarget.Python fails t without it); replaying needs only the file.
func Open(t testing.TB, name string) *Golden {
	t.Helper()
	g := &Golden{t: t, path: filepath.Join("testdata", name+".golden.jsonl"),
		hint: "re-record with " + pytarget.RecordEnv + "=1 and CPython " + pytarget.Minor}
	if os.Getenv(pytarget.RecordEnv) == "1" {
		g.py = pytarget.Python(t)
		kit, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "mcp-n4-kit"))
		if err != nil {
			t.Fatal(err)
		}
		g.kit = kit
		t.Cleanup(g.write)
		return g
	}
	data, err := os.ReadFile(g.path)
	if errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("golden %s missing: %s", g.path, g.hint)
	}
	if err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	for {
		var r record
		if err := dec.Decode(&r); err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("golden %s: record %d: %v", g.path, len(g.recs)+1, err)
		}
		g.recs = append(g.recs, r)
	}
	t.Cleanup(func() {
		if !t.Failed() && g.next != len(g.recs) {
			t.Errorf("golden %s is stale: %d recorded calls, the test made %d; %s", g.path, len(g.recs), g.next, g.hint)
		}
	})
	return g
}

// Alias records real, a path that changes between runs (a t.TempDir), as placeholder in both
// the cases and the output, and replays placeholder as the real of the run.
func (g *Golden) Alias(real, placeholder string) {
	g.alias = append(g.alias, real, placeholder)
}

// Mask rewrites each match of re in what Python prints to repl before it is recorded: for
// output that varies between runs and that the test ignores anyway (a timestamp), so a
// re-recording stays byte-identical.
func (g *Golden) Mask(re *regexp.Regexp, repl string) {
	g.masks = append(g.masks, func(s string) string { return re.ReplaceAllLiteralString(s, repl) })
}

// swap applies the aliases forward (real to placeholder) or back.
func (g *Golden) swap(text string, back bool) string {
	pairs := append([]string{}, g.alias...)
	for i := 0; back && i < len(pairs); i += 2 {
		pairs[i], pairs[i+1] = pairs[i+1], pairs[i]
	}
	return strings.NewReplacer(pairs...).Replace(text)
}

// Run is the script's stdout for stdin: run on the target when recording, else the recorded one.
// The script runs as the differentials always ran it, with the kit first on sys.path.
func (g *Golden) Run(script string, stdin []byte) []byte {
	g.t.Helper()
	digest := sha256Hex([]byte(script))[:16]
	masked := g.swap(string(stdin), false)
	if g.py != "" {
		cmd := exec.Command(g.py, "-I", "-c", "import sys; sys.path.insert(0, sys.argv[1]); exec(sys.argv[2])", g.kit, script)
		cmd.Stdin = bytes.NewReader(stdin)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			g.t.Fatalf("python: %v\n%s", err, stderr.String())
		}
		if !utf8.Valid(stdin) || !utf8.Valid(out) {
			g.t.Fatalf("golden %s: call %d is not UTF-8, a JSON golden cannot hold it byte for byte", g.path, len(g.recs)+1)
		}
		stdout := g.swap(string(out), false)
		for _, m := range g.masks {
			stdout = m(stdout)
		}
		r := record{Script: digest, Stdin: masked, Stdout: stdout}
		if len(masked) > maxInline {
			r.Stdin, r.StdinSHA256 = "", sha256Hex([]byte(masked))
		}
		g.recs = append(g.recs, r)
		return out
	}
	i := g.next
	g.next++
	if i >= len(g.recs) {
		g.t.Fatalf("golden %s is stale: call %d beyond the %d recorded; %s", g.path, i+1, len(g.recs), g.hint)
	}
	r := g.recs[i]
	if r.Script != digest {
		g.t.Fatalf("golden %s is stale: call %d runs a changed script; %s", g.path, i+1, g.hint)
	}
	if r.StdinSHA256 != "" {
		if r.StdinSHA256 != sha256Hex([]byte(masked)) {
			g.t.Fatalf("golden %s is stale: call %d sends other cases (SHA-256 differs); %s", g.path, i+1, g.hint)
		}
	} else if r.Stdin != masked {
		g.t.Fatalf("golden %s is stale: call %d sends other cases (first difference at byte %d); %s",
			g.path, i+1, firstDiff(r.Stdin, masked), g.hint)
	}
	return []byte(g.swap(r.Stdout, true))
}

// write saves the recorded calls, one JSON object per line, unless the test failed or skipped.
func (g *Golden) write() {
	if g.t.Failed() || g.t.Skipped() {
		return
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	for _, r := range g.recs {
		if err := enc.Encode(r); err != nil {
			g.t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(g.path), 0o755); err != nil {
		g.t.Fatal(err)
	}
	if err := os.WriteFile(g.path, buf.Bytes(), 0o644); err != nil {
		g.t.Fatal(err)
	}
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func firstDiff(a, b string) int {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	return i
}
