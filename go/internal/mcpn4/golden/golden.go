// Package golden holds the mcp-n4 Python differentials as recorded goldens, so they keep
// comparing the Go port with the kit's Python now that Python is gone (removed in S17). A
// differential feeds its helper script the cases on stdin and compares the Go side with what the
// script printed when it was recorded: testdata/<name>.golden.jsonl. The goldens are frozen: the
// Python they were recorded from no longer exists, so none can be re-recorded or extended.
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
	"path/filepath"
	"strings"
	"testing"
)

// record is one recorded call: the script's digest, the cases sent and what Python printed.
// Cases over maxInline bytes were kept as their SHA-256 alone: the test regenerates them
// deterministically.
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
	recs  []record
	alias []string // strings.Replacer pairs from the real text to its placeholder
	next  int
}

// hint follows every stale or missing golden: there is no Python left to record from.
const hint = "the goldens are frozen (the Python implementation was removed in S17): restore the " +
	"recorded file, or change the Go side only where the recorded Python behaviour allows"

// Open returns the golden testdata/<name>.golden.jsonl of t's package, replayed.
func Open(t testing.TB, name string) *Golden {
	t.Helper()
	g := &Golden{t: t, path: filepath.Join("testdata", name+".golden.jsonl")}
	data, err := os.ReadFile(g.path)
	if errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("golden %s missing: %s", g.path, hint)
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
			t.Errorf("golden %s is stale: %d recorded calls, the test made %d; %s", g.path, len(g.recs), g.next, hint)
		}
	})
	return g
}

// Alias records real, a path that changes between runs (a t.TempDir), as placeholder in both
// the cases and the output, and replays placeholder as the real of the run.
func (g *Golden) Alias(real, placeholder string) {
	g.alias = append(g.alias, real, placeholder)
}

// swap applies the aliases forward (real to placeholder) or back.
func (g *Golden) swap(text string, back bool) string {
	pairs := append([]string{}, g.alias...)
	for i := 0; back && i < len(pairs); i += 2 {
		pairs[i], pairs[i+1] = pairs[i+1], pairs[i]
	}
	return strings.NewReplacer(pairs...).Replace(text)
}

// Run is the script's recorded stdout for stdin.
func (g *Golden) Run(script string, stdin []byte) []byte {
	g.t.Helper()
	digest := sha256Hex([]byte(script))[:16]
	masked := g.swap(string(stdin), false)
	i := g.next
	g.next++
	if i >= len(g.recs) {
		g.t.Fatalf("golden %s is stale: call %d beyond the %d recorded; %s", g.path, i+1, len(g.recs), hint)
	}
	r := g.recs[i]
	if r.Script != digest {
		g.t.Fatalf("golden %s is stale: call %d runs a changed script; %s", g.path, i+1, hint)
	}
	if r.StdinSHA256 != "" {
		if r.StdinSHA256 != sha256Hex([]byte(masked)) {
			g.t.Fatalf("golden %s is stale: call %d sends other cases (SHA-256 differs); %s", g.path, i+1, hint)
		}
	} else if r.Stdin != masked {
		g.t.Fatalf("golden %s is stale: call %d sends other cases (first difference at byte %d); %s",
			g.path, i+1, firstDiff(r.Stdin, masked), hint)
	}
	return []byte(g.swap(r.Stdout, true))
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
