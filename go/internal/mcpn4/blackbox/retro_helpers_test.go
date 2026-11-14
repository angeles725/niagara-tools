package blackbox

// Frozen Python-server oracles and Python-written state files, for the retro draft and
// list_batches ports.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// oracleToday is the placeholder the recordings hold for today's UTC date.
const oracleToday = "<today-utc>"

// frozen is oracle.Frozen: the recorded calls of one Python test, in
// mcp-n4-kit/tests/fixtures/oracles/<name>.json (read in place). Each want consumes the next
// recorded call, whose args must be the ones the test sends; the test fails unless every
// recorded call is used. aliases are (real value, placeholder) pairs: a temporary state dir and
// today's date were recorded as placeholders.
type frozen struct {
	t       *testing.T
	key     string
	calls   []any // [args, answer] pairs, as pyjson values
	next    int
	aliases [][2]string
}

func newFrozen(t *testing.T, name, key string, aliases ...[2]string) *frozen {
	t.Helper()
	// pyjson: the recordings hold NaN, as json.dump writes it.
	doc, err := pyjson.Unmarshal(kitFixture(t, "oracles", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var calls []any
	if root, ok := doc.(*pyjson.Object); ok {
		if all, ok := root.Get("calls"); ok {
			if byKey, ok := all.(*pyjson.Object); ok {
				recorded, _ := byKey.Get(key)
				calls, _ = recorded.([]any)
			}
		}
	}
	if calls == nil {
		t.Fatalf("%s.json has no recording for %s", name, key)
	}
	f := &frozen{t: t, key: key, calls: calls, aliases: aliases}
	t.Cleanup(func() {
		if f.next != len(f.calls) {
			t.Errorf("%s made %d of the %d recorded calls", key, f.next, len(f.calls))
		}
	})
	return f
}

func (f *frozen) swap(text string, back bool) string {
	for _, a := range f.aliases {
		if back {
			text = strings.ReplaceAll(text, a[1], a[0])
		} else {
			text = strings.ReplaceAll(text, a[0], a[1])
		}
	}
	return text
}

// want is the recorded answer to args, as a pyjson value (dict order kept).
func (f *frozen) want(args Args) any {
	f.t.Helper()
	i := f.next
	f.next++
	if i >= len(f.calls) {
		f.t.Fatalf("%s makes more calls than the %d recorded", f.key, len(f.calls))
	}
	sent, err := json.Marshal(args)
	if err != nil {
		f.t.Fatal(err)
	}
	pair, _ := f.calls[i].([]any)
	if len(pair) != 2 {
		f.t.Fatalf("%s call %d: not an [args, answer] pair", f.key, i+1)
	}
	if got, recorded := decoded(f.t, f.swap(string(sent), false)), decoded(f.t, dumps(f.t, pair[0])); !reflect.DeepEqual(got, recorded) {
		f.t.Fatalf("%s call %d sends other args: %v, recorded %v", f.key, i+1, got, recorded)
	}
	answer, err := pyjson.Unmarshal([]byte(f.swap(dumps(f.t, pair[1]), true)))
	if err != nil {
		f.t.Fatal(err)
	}
	return answer
}

// wantText is want as json.dumps text: the bytes the Python server's reply text was.
func (f *frozen) wantText(args Args) string { f.t.Helper(); return dumps(f.t, f.want(args)) }

// dumps is json.dumps(v) of a pyjson value.
func dumps(t *testing.T, v any) string {
	t.Helper()
	out, err := pyjson.Marshal(v, pyjson.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// callRaw is tools/call whose reply line is decoded as json.loads would (NaN included); it
// returns the result object.
func callRaw(t *testing.T, s *Server, tool string, args Args) *pyjson.Object {
	t.Helper()
	reply, err := pyjson.Unmarshal([]byte(s.Line(RPC("tools/call", Args{"name": tool, "arguments": args}, 1))))
	if err != nil {
		t.Fatal(err)
	}
	result, _ := reply.(*pyjson.Object).Get("result")
	out, ok := result.(*pyjson.Object)
	if !ok {
		t.Fatalf("%s: not a tool result: %s", tool, dumps(t, reply))
	}
	return out
}

// resultText is the first content text of a raw result.
func resultText(r *pyjson.Object) string {
	content, _ := r.Get("content")
	list, _ := content.([]any)
	if len(list) == 0 {
		return ""
	}
	first, _ := list[0].(*pyjson.Object)
	if first == nil {
		return ""
	}
	text, _ := first.Get("text")
	s, _ := text.(string)
	return s
}

// decoded is text decoded as a tool result's structuredContent is.
func decoded(t *testing.T, text string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(text), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

// sameReply fails unless the reply is wantText (json.dumps of the Python answer) both as the
// content text and as structuredContent.
func sameReply(t *testing.T, r Result, wantText string) {
	t.Helper()
	if r.IsError {
		t.Fatalf("tool error: %s", r.Text)
	}
	if r.Text != wantText {
		t.Errorf("text differs from Python's:\n got %.300s\nwant %.300s", r.Text, wantText)
	}
	if !reflect.DeepEqual(any(r.Structured), decoded(t, wantText)) {
		t.Errorf("structuredContent differs from Python's: %.300v", r.Structured)
	}
}

// obj is an ordered JSON object (a Python dict) from key, value pairs.
func obj(kv ...any) *pyjson.Object {
	o := pyjson.NewObject()
	for i := 0; i < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
	return o
}

// with is dict(o, key=value): a copy of o with key replaced in place (or appended).
func with(o *pyjson.Object, kv ...any) *pyjson.Object {
	out := pyjson.NewObject()
	for _, k := range o.Keys() {
		v, _ := o.Get(k)
		out.Set(k, v)
	}
	for i := 0; i < len(kv); i += 2 {
		out.Set(kv[i].(string), kv[i+1])
	}
	return out
}

// appendLines is state_files._JsonlFile.append: the state dir is created 0700 when absent, the
// file 0600, and each entry is appended as one json.dumps line.
func appendLines(t *testing.T, dir, name string, entries ...any) {
	t.Helper()
	if _, err := os.Stat(dir); err != nil {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	f, err := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, e := range entries {
		if _, err := f.WriteString(dumps(t, e) + "\n"); err != nil {
			t.Fatal(err)
		}
	}
}

// copyTree copies the regular files of src into a new dir dst (0700), keeping their modes.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.Mkdir(dst, 0o700); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		info, err := e.Info()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), data, info.Mode().Perm()); err != nil {
			t.Fatal(err)
		}
	}
}

// pythonWrittenDir is fixtures/journal/python-written, a state dir the Python server wrote.
func pythonWrittenDir() string {
	return filepath.Join(kitDir, "tests", "fixtures", "journal", "python-written")
}

// candidateKeys are the keys of a draft's candidates, in order.
func candidateKeys(draft map[string]any) []any { return field(draft["candidates"], "key") }
