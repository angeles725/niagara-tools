package readtools_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/golden"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// A loaded subtree: components with Status outs, a link by handle, one by slot path and one
// with an unresolved source, handles of several kinds, a non-ASCII name.
const stationTree = `{"": {"t": "baja:Folder", "h": "3"},
	"A": {"n": "A", "t": "kitControl:NumericConst", "h": "10"},
	"A/out": {"n": "out", "t": "baja:StatusNumeric"}, "A/out/value": {"n": "value", "v": "5.5"},
	"B": {"n": "B", "t": "kitControl:Add", "h": "11"}, "B/out": {"n": "out", "t": "baja:StatusNumeric"},
	"B/in10": {"n": "in10", "t": "baja:StatusNumeric"}, "B/Link": {"n": "Link", "t": "baja:Link"},
	"B/Link/sourceOrd": {"v": "h:10"}, "B/Link/sourceSlotName": {"v": "out"},
	"B/Link/targetSlotName": {"v": "in10"}, "B/L2": {"n": "L2", "t": "baja:ConversionLink"},
	"B/L2/sourceOrd": {"v": "station:|slot:/Folder/C/"}, "B/L2/sourceSlotName": {"v": "out"},
	"C": {"n": "C", "t": "baja:Folder", "h": 12}, "C/Ünï": {"n": "Ünï", "t": "control:NumericWritable", "h": "10"},
	"C/Ünï/out": {"t": "baja:StatusNumeric"}, "C/Ünï/L": {"t": "baja:Link"},
	"C/Ünï/L/sourceOrd": {"v": "local:|foxs:|bql:select x"}, "C/Ünï/L/sourceSlotName": {},
	"D": {"n": "D", "t": "", "v": "x", "d": "x!"}, "E": {"n": "E", "d": "Disp"}}`

// TestDifferentialPython runs each tool on the same loaded nodes in tools_read.py on the target
// CPython and in Go, and requires the same reply text byte for byte: the JSON of a result, or
// the error words (an internal error by its Python type), for well-formed and odd node shapes.
func TestDifferentialPython(t *testing.T) {
	g := golden.Open(t, "read-tools")
	odd := func(k, v string) string { return `{"": {"t": "a:B"}, ` + `"` + k + `": ` + v + `}` }
	cases := []struct{ tool, ord, args, tree string }{
		{"n4_navigate", "station:|slot:/Folder", `{"depth": 3}`, stationTree},
		{"n4_navigate", "station:|slot:/Folder", `{"depth": 2, "types": ["control:NumericWritable", "baja:Link"]}`, stationTree},
		{"n4_navigate", "station:|slot:/", `{"types": ["x", 5]}`, stationTree},
		{"n4_navigate", "station:|slot:/", `{}`, `{"": {}, "a": {"t": "x:Y"}, "a/b": {"t": 5}}`},
		{"n4_navigate", "station:|slot:/", `{}`, `{"": {}, "a": {"t": "x:Y"}, "a/b": {"t": "x:Y"}, "a/c": {"t": null}}`},
		{"n4_navigate", "station:|slot:/", `{"types": ["x"]}`, odd("a", `{"t": ["x"]}`)},
		{"n4_navigate", "slot:/", `{}`, stationTree},
		{"n4_read_slots", "station:|slot:/Folder/B", `{}`, stationTree},
		{"n4_read_slots", "station:|slot:/Folder/B", `{}`, `{"": {}, "s": {"t": "baja:StatusBoolean"},
			"s/status": {"v": "40;activeLevel=e"}, "t": {"t": "baja:StatusString"}, "t/value": {"v": "abc"},
			"u": {"t": "baja:StatusEnum", "d": "On"}, "v": {"t": 0, "d": 1.5}, "w": {"t": "baja:StatusNumeric", "v": 2}}`},
		{"n4_read_slots", "station:|slot:/X", `{}`, odd("s", `{"t": "baja:StatusNumeric", "s": [1]}`)},
		{"n4_read_slots", "station:|slot:/X", `{}`, `{"": {}, "s": {"t": "baja:StatusNumeric"}, "s/status": {"v": "0x_zz"}}`},
		{"n4_read_slots", "station:|slot:/X", `{}`, `{"": {}, "s": {"t": "baja:StatusNumeric"}, "s/status": {"v": 7}}`},
		{"n4_read_slots", "station:|slot:/X", `{}`, odd("s", `{"t": ["baja:Status"]}`)},
		{"n4_list_links", "station:|slot:/Folder/", `{"depth": 1}`, stationTree},
		{"n4_list_links", "station:|slot:/Folder", `{"depth": 4}`, stationTree},
		{"n4_list_links", "station:", `{}`, stationTree},
		{"n4_find_dangling_outputs", "station:|slot:/Folder", `{}`, stationTree},
		{"n4_find_dangling_outputs", "station:|slot:/Folder", `{"depth": 1}`, strings.Replace(stationTree, `"v": "local:|foxs:|bql:select x"`, `"v": "h:404"`, 1)},
		{"n4_find_dangling_outputs", "station:|slot:/Folder", `{}`, strings.Replace(stationTree, `"C/Ünï/L/sourceSlotName": {}`, `"C/Ünï/L/sourceSlotName": {"v": ["out"]}`, 1)},
		{"n4_list_links", "station:|slot:/Folder", `{}`, strings.Replace(stationTree, `"C/Ünï/L/sourceSlotName": {}`, `"C/Ünï/L/sourceSlotName": {"v": ["out"]}`, 1)},
		{"n4_list_links", "station:|slot:/Folder", `{}`, strings.Replace(stationTree, `"h:10"`, `10`, 1)},
		{"n4_list_links", "station:|slot:/Folder", `{}`, strings.Replace(stationTree, `"h": 12`, `"h": [12]`, 1)},
		{"n4_list_links", "station:|slot:/Folder", `{}`, strings.Replace(stationTree, `"h": 12`, `"h": []`, 1)},
	}
	script := `import json, sys
from mcp_n4 import box, tools_read
class Client:
    def __init__(self, nodes):
        self.nodes = nodes
    def load_tree(self, ord_str, depth=1, **kw):
        return self.nodes
class Ctx:
    def clock(self):
        return 1000.0
out = []
for case in json.load(sys.stdin):
    ctx = Ctx()
    ctx.observations, ctx.session = [], type("S", (), {"client": Client(json.loads(case["tree"]))})
    args = dict(json.loads(case["args"]), ord=case["ord"])
    try:
        out.append(json.dumps(getattr(tools_read, case["tool"])(ctx, args)))
    except (tools_read.ToolError, box.BoxError, ValueError) as exc:
        out.append("error: %s" % exc)
    except Exception as exc:
        out.append("error: internal error (%s)" % type(exc).__name__)
print(json.dumps(out))`
	var rows []map[string]string
	for _, c := range cases {
		rows = append(rows, map[string]string{"tool": c.tool, "ord": c.ord, "args": c.args, "tree": c.tree})
	}
	in, _ := json.Marshal(rows)
	out := g.Run(script, in)
	var want []string
	if err := json.Unmarshal(out, &want); err != nil {
		t.Fatal(err)
	}
	for i, c := range cases {
		s := serve(t, c.ord, c.tree)
		args, err := pyjson.Unmarshal([]byte(c.args))
		if err != nil {
			t.Fatal(err)
		}
		kv := []any{"ord", c.ord}
		for _, k := range args.(*pyjson.Object).Keys() {
			v, _ := args.(*pyjson.Object).Get(k)
			kv = append(kv, k, v)
		}
		text, isErr, _ := s.Call(t, c.tool, kv...)
		if isErr {
			text = "error: " + text
		}
		if text != want[i] {
			t.Errorf("case %d %s %s:\nGo     %s\nPython %s", i, c.tool, c.args, text, want[i])
		}
	}
}

// bqlScript runs one n4_bql_query or n4_inventory case in tools_read.py with a client whose
// get_ord answers the case's body (cut at cap, when set) and prints the reply text and the
// result file, if one was written: its bytes and mode.
const bqlScript = `import json, os, sys
from mcp_n4 import box, tools_read
case = json.load(sys.stdin)
class Client:
    def get_ord(self, ord_text, timeout=None, max_bytes=None):
        body = case["body"]
        if case["cap"] and len(body) > case["cap"]:
            return body[:case["cap"]], True
        return body, False
class Ctx:
    observations, write, progress_path, state_dir = [], None, None, case["state"]
    session = type("S", (), {"client": Client()})
    def clock(self):
        return 1000.0
    def scrub(self, text):
        return text
try:
    out = json.dumps(getattr(tools_read, case["tool"])(Ctx(), json.loads(case["args"])))
except (tools_read.ToolError, box.BoxError, ValueError) as exc:
    out = "error: %s" % exc
except Exception as exc:
    out = "error: internal error (%s)" % type(exc).__name__
print(json.dumps([out, bool(case["file"]) and os.path.lexists(case["file"]) and
                  [open(case["file"], "rb").read().decode("latin-1"), oct(os.stat(case["file"]).st_mode)]]))`

// TestBqlDifferentialPython runs n4_bql_query and n4_inventory on the same station answers in
// tools_read.py on the target CPython and in Go, one after the other on the same state dir, and
// requires the same reply text and the same result file bytes and mode: result shaping
// (localized and repeated headers, the name column by position, a byte cap inside a quoted cell,
// the row cap), the refusals, csv.writer's quoting and line ends, json.dump's bytes and a lone
// surrogate that UTF-8 cannot encode. Progress timestamps are the only masked bytes.
func TestBqlDifferentialPython(t *testing.T) {
	g := golden.Open(t, "bql-tools")
	devices := "\ufeffSlot Path,Name,Type\nslot:/Drivers/N$201/D,D$2cx,a:Dev\nslot:/Drivers/M/localDevice,localDevice,a:Loc\n"
	odd := "Name,B\n\" lead\",\"q\"\"uote\"\nA$0aB$00,\"\"\nx$ud800y$udc00,$ud800\nCaf$u00e9,\"line\r\nnext\"\n"
	q := `{"base": "/Drivers", "query": "select name, b from x:Y"`
	cases := []struct {
		tool, args, body string
		cap              int
		file             string
	}{
		{"n4_bql_query", `{"query": " select slotPath, name, type from a:B "}`, devices, 0, ""},
		{"n4_bql_query", `{"query": "SELECT slotPath, NAME from a:B", "max_rows": 1}`, "Ruta,Nombre\nslot:/A$20B,A$20B\nslot:/C,C\n", 0, ""},
		{"n4_bql_query", `{"query": "select * from a:B"}`, "Type,Type,Name\nA,B,C$20D\n", 0, ""},
		{"n4_bql_query", `{"query": "select slotPath, name from a:B"}`, "Slot Path,Name\nslot:/Z,\"ok\"\nslot:/A,\"line one\nline two\"\n", 28, ""},
		{"n4_bql_query", `{"query": "select a from b"}`, "A,B\nx\n", 0, ""},
		{"n4_bql_query", `{"query": "select a from b"}`, "", 0, ""},
		{"n4_bql_query", `{"query": "select a from b"}`, "A\n" + strings.Repeat("x", 131073) + "\n", 0, ""},
		{"n4_bql_query", `{"query": "select a from b", "base": "station:|h:1"}`, "A\n", 0, ""},
		{"n4_bql_query", `{"query": "delete a"}`, "A\n", 0, ""},
		{"n4_bql_query", `{"query": "select a from b", "output_file": "../x.csv"}`, "A\n", 0, ""},
		{"n4_bql_query", q + `, "output_file": "r.csv"}`, strings.Replace(odd, "x$ud800y$udc00,$ud800\n", "", 1), 0, "r.csv"},
		{"n4_bql_query", q + `, "output_file": "r.json"}`, odd, 0, "r.json"},
		{"n4_bql_query", `{"query": "select name from x:Y", "output_file": "s.csv"}`, "Name\nok\nA$ud800$udfffB\n", 0, "s.csv"},
		{"n4_bql_query", `{"query": "select name from x:Y", "output_file": "t.csv"}`, "Name\n\n\"\"\nok\nC$0dR\n", 0, "t.csv"},
		{"n4_bql_query", `{"query": "select name from x:Y", "output_file": "e.csv"}`, "Name\n", 0, "e.csv"},
		{"n4_inventory", `{}`, devices, 0, ""},
		{"n4_inventory", `{"base": "/Drivers/N$201", "include_points": true, "max_rows": 1}`, devices, 0, ""},
		{"n4_inventory", `{}`, "A,B\nx,y\n", 0, ""},
		{"n4_inventory", `{}`, "Slot Path,Name,Type\n", 0, ""},
		{"n4_inventory", `{"base": "Drivers"}`, devices, 0, ""},
	}
	root := t.TempDir()
	g.Alias(root, "$STATE_ROOT") // the cases and replies name the state dirs under it
	// The progress stamps were masked to "T" when the golden was recorded, and are masked below.
	ts := regexp.MustCompile(`"ts": "[^"]*"`)
	for i, c := range cases {
		state := filepath.Join(root, fmt.Sprint(i), "state")
		file := ""
		if c.file != "" {
			file = filepath.Join(state, "bql", c.file)
		}
		in, _ := json.Marshal(map[string]any{"tool": c.tool, "args": c.args, "body": c.body, "cap": c.cap, "state": state, "file": file})
		out := g.Run(bqlScript, in)
		var want []any
		if err := json.Unmarshal(out, &want); err != nil {
			t.Fatal(err)
		}
		os.RemoveAll(state)
		s := readtoolstest.NewServer(readtoolstest.NewStation(), session.Config{StateDir: state})
		s.Connect(t)
		s.Station.OrdRoutes = []readtoolstest.OrdRoute{{Needle: "", Body: c.body}}
		s.Station.OrdCap = c.cap
		args, _ := pyjson.Unmarshal([]byte(c.args))
		var kv []any
		for _, k := range args.(*pyjson.Object).Keys() {
			v, _ := args.(*pyjson.Object).Get(k)
			kv = append(kv, k, v)
		}
		text, isErr, _ := s.Call(t, c.tool, kv...)
		if isErr {
			text = "error: " + text
		}
		var written any = false
		if _, err := os.Lstat(file); file != "" && err == nil {
			data, _ := os.ReadFile(file)
			info, _ := os.Stat(file)
			latin := make([]rune, len(data))
			for j, b := range data {
				latin[j] = rune(b)
			}
			written = []any{string(latin), fmt.Sprintf("0o%o", uint32(info.Mode().Perm())|0o100000)}
		}
		if got, py := ts.ReplaceAllString(text, `"ts": "T"`), ts.ReplaceAllString(want[0].(string), `"ts": "T"`); got != py {
			t.Errorf("case %d %s %s:\nGo     %.400s\nPython %.400s", i, c.tool, c.args, got, py)
		}
		if g, w := jsonText(written), jsonText(want[1]); g != w {
			t.Errorf("case %d file:\nGo     %s\nPython %s", i, g, w)
		}
	}
}

func jsonText(v any) string { b, _ := json.Marshal(v); return string(b) }
