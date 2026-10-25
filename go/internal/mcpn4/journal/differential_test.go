package journal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/golden"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The helper runs safety.Journal, n4_list_batches and the journal checks of _rollback_plan on
// each state dir. Its session matches any station, so the station check between the journal
// checks always passes; a refusal that is not a journal check counts as passing them.
const helperScript = `import json, re, sys
from types import SimpleNamespace
from mcp_n4 import safety, tools_read, tools_write
class AnyStation:
    def __eq__(self, other): return True
    def __ne__(self, other): return False
GATE = re.compile(r"batch_id must be|journal line unreadable|unknown batch|batch \S+ is itself|batch \S+ is in-doubt|a rollback of batch|batch \S+ was already")
def run(fn):
    try:
        return fn()
    except ValueError as exc:
        return "ValueError: %s" % exc
    except Exception as exc:
        return type(exc).__name__
def gate(d, bid):
    ctx = SimpleNamespace(write=SimpleNamespace(journal=safety.Journal(d), scope=safety.WriteScope([])),
                          session=SimpleNamespace(station_name=AnyStation()))
    try:
        tools_write._rollback_plan(None, {"batch_id": bid}, ctx)
    except tools_write.ToolError as exc:
        if GATE.match(str(exc)):
            return "refused: %s" % exc
    except ValueError as exc:
        return "ValueError: %s" % exc
    except Exception as exc:
        if not isinstance(exc, (safety.SafetyError, AttributeError)):
            return type(exc).__name__
    return "passed"
cases = json.load(sys.stdin)
out = []
for c in cases:
    d, j = c["dir"], safety.Journal(c["dir"])
    out.append(run(lambda: j.views()))
    for limit in (1, 50):
        out.append(run(lambda: tools_read.n4_list_batches(SimpleNamespace(state_dir=d), {"limit": limit})))
    out.append(run(lambda: tools_read.n4_list_batches(SimpleNamespace(state_dir=d), {})))
    for bid in c["ids"]:
        if isinstance(bid, str):  # the Go API takes a string id: S13 checks it first
            out.append(run(lambda: j.read(bid)))
            out.append(run(lambda: j.rollbacks_of(bid)))
        out.append(gate(d, bid))
print(json.dumps(out))`

const (
	idA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	idB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	idC = "cccccccccccccccccccccccccccccccc"
	idD = "dddddddddddddddddddddddddddddddd"
)

// edgeJournals are journal.jsonl bodies Python could meet: torn and non-object lines, CR and
// CRLF line ends, a BOM, duplicate intents and results, results before intents, odd ops,
// numeric, NaN, null and unhashable batch ids, odd rollback_of values, earlier rollbacks in
// each state, and bytes that are not UTF-8.
var edgeJournals = []string{
	"",
	"{torn\n[1]\n\"s\"\n\n   \n{\"batch_id\": \"" + idA + "\", \"phase\": \"intent\", \"tool\": \"t\", \"ops\": [1]}\r\n",
	"\xef\xbb\xbf{\"batch_id\": \"" + idA + "\", \"phase\": \"intent\"}\n{\"batch_id\": \"" + idB + "\", \"phase\": \"intent\"}\r" +
		"{\"batch_id\": \"" + idB + "\", \"phase\": \"result\", \"verdict\": \"verified\", \"inverse\": []}",
	"{\"batch_id\": \"" + idA + "\", \"phase\": \"result\", \"verdict\": \"early\"}\n" +
		"{\"batch_id\": \"" + idA + "\", \"phase\": \"intent\", \"tool\": \"first\", \"station_name\": \"S\"}\n" +
		"{\"batch_id\": \"" + idA + "\", \"phase\": \"intent\", \"tool\": \"second\"}\n" +
		"{\"batch_id\": \"" + idA + "\", \"phase\": \"result\", \"verdict\": \"late\", \"in_doubt\": [], \"ts\": 5, \"phase2\": 1}\n",
	"{\"batch_id\": \"" + idA + "\", \"phase\": \"intent\"}\n{\"batch_id\": \"" + idA + "\", \"phase\": \"relink-intent\", \"ops\": \"xy\"}\n" +
		"{\"batch_id\": \"" + idA + "\", \"phase\": \"relink-intent\", \"ops\": [2]}\n{\"batch_id\": \"" + idA + "\", \"phase\": \"component-intent\", \"ops\": \"é\\ud800\"}\n" +
		"{\"batch_id\": \"" + idA + "\", \"phase\": \"component-intent\", \"ops\": {\"k\": 1, \"j\": 2}}\n{\"batch_id\": \"" + idA + "\", \"phase\": \"component-intent\", \"ops\": null}\n",
	"{\"batch_id\": 1, \"phase\": \"intent\"}\n{\"batch_id\": true, \"phase\": \"intent\"}\n{\"batch_id\": 1.0, \"phase\": \"result\"}\n" +
		"{\"batch_id\": NaN, \"phase\": \"intent\"}\n{\"batch_id\": NaN, \"phase\": \"result\", \"in_doubt\": NaN}\n{\"batch_id\": -0.0, \"phase\": \"intent\"}\n" +
		"{\"batch_id\": false, \"phase\": \"result\"}\n{\"batch_id\": 1e300, \"phase\": \"intent\"}\n{\"batch_id\": 1" + strings.Repeat("0", 300) + ", \"phase\": \"result\"}\n" +
		"{\"phase\": \"intent\", \"rollback_of\": 1}\n{\"batch_id\": [1], \"phase\": \"note\"}\n{\"batch_id\": 0.5, \"phase\": \"intent\", \"rollback_of\": true}\n",
	"{\"batch_id\": [1], \"phase\": \"intent\"}\n",
	"{\"batch_id\": \"" + idA + "\", \"phase\": \"intent\"}\n{\"batch_id\": {}, \"phase\": \"result\"}\n",
	"{\"batch_id\": \"" + idA + "\", \"phase\": \"component-intent\", \"ops\": 7}\n",
	"{\"batch_id\": \"" + idA + "\", \"phase\": \"intent\"}\n{\"batch_id\": \"" + idB + "\", \"phase\": \"intent\", \"rollback_of\": [\"" + idA + "\"]}\n",
	// Earlier rollbacks of idA: in-doubt, done, failed with and without an accepted read-back.
	"{\"batch_id\": \"" + idA + "\", \"phase\": \"intent\", \"station_name\": \"S\"}\n{\"batch_id\": \"" + idA + "\", \"phase\": \"result\", \"verdict\": \"verified\", \"inverse\": [{\"nm\": \"v\", \"h\": \"3\", \"n\": \"A\"}]}\n" +
		"{\"batch_id\": \"" + idB + "\", \"phase\": \"intent\", \"rollback_of\": \"" + idA + "\"}\n{\"batch_id\": \"" + idB + "\", \"phase\": \"result\", \"verdict\": \"failed\", \"readback_failed\": true, \"accepted\": [{\"z\": 1, \"a\": \"é\"}]}\n" +
		"{\"batch_id\": \"" + idC + "\", \"phase\": \"intent\", \"rollback_of\": \"" + idA + "\"}\n{\"batch_id\": \"" + idC + "\", \"phase\": \"result\", \"verdict\": \"failed\"}\n" +
		"{\"batch_id\": \"" + idD + "\", \"phase\": \"intent\", \"rollback_of\": \"" + idC + "\"}\n",
	"{\"batch_id\": \"" + idA + "\", \"phase\": \"intent\"}\n{\"batch_id\": \"" + idA + "\", \"phase\": \"result\", \"verdict\": \"verified\"}\n" +
		"{\"batch_id\": \"" + idB + "\", \"phase\": \"intent\", \"rollback_of\": \"" + idA + "\"}\n{\"batch_id\": \"" + idB + "\", \"phase\": \"result\", \"verdict\": 0}\n",
	"{\"batch_id\": \"" + idA + "\", \"phase\": \"intent\"}\n{\"batch_id\": \"" + idA + "\", \"phase\": \"result\"}\n{\"phase\": \"intent\", \"rollback_of\": \"" + idA + "\"}\n",
	"{\"batch_id\": \"" + idA + "\", \"phase\": \"intent\"}\n{\"batch_id\": \"" + idA + "\", \"phase\": \"result\"}\n{\"batch_id\": \"" + idB + "\", \"phase\": \"intent\", \"rollback_of\": \"" + idA + "\"}\n",
	"{\"batch_id\": \"" + idA + "\", \"phase\": \"intent\"}\n\xff\n",
	"{\"batch_id\": \"" + idA + "\", \"phase\": \"intent\", \"ops\": \"" + strings.Repeat("x", 9000) + "\"}\n\xe2\x82\n",
	// Issue #448: one bad line in the middle of a multi-batch journal. Not UTF-8 (its batch is
	// unknown: every rollback is refused), across CR, CRLF and an 8192-byte chunk boundary.
	twoBatches("{\"batch_id\": \"\xff\", \"phase\": \"result\"}\r{\"batch_id\": \"" + idC + "\", \"phase\": \"intent\", \"x\": \"" +
		strings.Repeat("y", 8200) + "\xe2\"}\r\n\xfe\n"),
	// Known batches: an unhashable id naming idB in rollback_of, idC's ops that are a number,
	// an unhashable rollback_of on a result: each blocks only the batches it names.
	twoBatches("{\"batch_id\": [1], \"phase\": \"intent\", \"rollback_of\": \"" + idB + "\"}\n" +
		"{\"batch_id\": \"" + idC + "\", \"phase\": \"component-intent\", \"ops\": 7}\n" +
		"{\"batch_id\": \"" + idD + "\", \"phase\": \"intent\", \"station_name\": \"S\"}\n" +
		"{\"batch_id\": \"" + idD + "\", \"phase\": \"result\", \"rollback_of\": {\"k\": 1}}\n" +
		"{\"batch_id\": \"" + idC + "\", \"phase\": \"intent\", \"rollback_of\": \"" + idA + "\"}\n"),
}

// twoBatches is a journal of two completed batches, idA and idB, with middle between them.
func twoBatches(middle string) string {
	batch := func(id string) string {
		return "{\"batch_id\": \"" + id + "\", \"phase\": \"intent\", \"station_name\": \"S\", \"tool\": \"t\"}\n" +
			"{\"batch_id\": \"" + id + "\", \"phase\": \"result\", \"verdict\": \"verified\", \"inverse\": [{\"nm\": \"v\", \"h\": \"3\", \"n\": \"A\"}]}\n"
	}
	return batch(idA) + middle + batch(idB)
}

// gateKinds are the outcomes of the rollback journal checks the corpus must reach.
var gateKinds = []string{"must be a 32-character", "unknown batch", "is itself a rollback", "is in-doubt (the station",
	") is in-doubt: inspect", "was already rolled back", "read-back failed", "passed", "journal line unreadable", "UnicodeDecodeError", "TypeError", "KeyError"}

// ids are the batch ids every state dir is asked about.
var ids = []any{idA, idB, idC, idD, idA + "\n", idA + "\n\n", "x", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", nil, 5}

func TestJournalDifferentialPython(t *testing.T) {
	g := golden.Open(t, "journal")
	dirs := []string{fixtureDir}
	for i, body := range edgeJournals {
		dir := filepath.Join(t.TempDir(), "s")
		if i == 0 { // every t.TempDir of the test shares this parent
			g.Alias(filepath.Dir(filepath.Dir(dir)), "$TMP")
		}
		os.MkdirAll(dir, 0o700)
		if i > 0 { // the first edge case is a state dir without a journal
			os.WriteFile(filepath.Join(dir, "journal.jsonl"), []byte(body), 0o600)
		}
		dirs = append(dirs, dir)
	}
	fixtureIDs := fixtureBatchIDs(t)
	var cases, got []any
	for i, dir := range dirs {
		these := ids
		if i == 0 {
			these = append(fixtureIDs, ids...)
		}
		c := pyjson.NewObject()
		c.Set("dir", dir)
		c.Set("ids", these)
		cases = append(cases, c)
		got = append(got, goSide(dir, these)...)
	}
	data, _ := pyjson.Marshal(cases, pyjson.Options{})
	want, err := pyjson.Loads(g.Run(helperScript, data))
	if err != nil || len(want.([]any)) != len(got) {
		t.Fatalf("python answered %v for %d Go results (%v)", len(want.([]any)), len(got), err)
	}
	seen := map[string]int{}
	for _, g := range got {
		if s, ok := g.(string); ok {
			for _, kind := range gateKinds {
				if strings.Contains(s, kind) {
					seen[kind]++
				}
			}
		}
	}
	for _, kind := range gateKinds {
		if seen[kind] == 0 {
			t.Errorf("no case reaches the %q branch", kind)
		}
	}
	for i, w := range want.([]any) {
		g, _ := pyjson.Marshal(got[i], pyjson.Options{})
		wb, _ := pyjson.Marshal(w, pyjson.Options{})
		if string(g) != string(wb) {
			t.Errorf("result %d: Go and Python differ\ngo:     %.600s\npython: %.600s", i, g, wb)
		}
	}
}

// goSide is what the helper prints for one state dir, computed in Go.
func goSide(dir string, ids []any) []any {
	j := New(dir)
	run := func(v any, err error) any {
		if ve, ok := err.(*pyjson.ValueError); ok {
			return "ValueError: " + ve.Msg
		}
		if ex, ok := err.(*pyjson.Exception); ok {
			return ex.Type
		}
		return v
	}
	views, err := j.Views()
	out := []any{run(objects(views), err)}
	for _, limit := range []int{1, 50, 50} {
		out = append(out, run(listBatches(j, limit)))
	}
	for _, id := range ids {
		if bid, ok := id.(string); ok {
			v, err := j.Read(bid)
			if v == nil && err == nil {
				out = append(out, nil)
			} else {
				out = append(out, run(v, err))
			}
			rb, err := j.RollbacksOf(bid)
			out = append(out, run(objects(rb), err))
		}
		out = append(out, gate(j, id))
	}
	return out
}

func listBatches(j *Journal, limit int) (any, error) { return j.ListBatches(limit) }

func gate(j *Journal, id any) string {
	view, err := j.RollbackTarget(id)
	if err == nil {
		err = j.EarlierRollbacks(id.(string))
	}
	switch e := err.(type) {
	case nil:
		_ = view
		return "passed"
	case *pyjson.Exception:
		return e.Type
	case *pyjson.ValueError:
		return "ValueError: " + e.Msg
	}
	return "refused: " + err.Error()
}

func objects(views []*pyjson.Object) []any {
	out := []any{}
	for _, v := range views {
		out = append(out, v)
	}
	return out
}
