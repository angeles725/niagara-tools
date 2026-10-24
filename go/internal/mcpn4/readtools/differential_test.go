package readtools_test

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pytarget"
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
	py := pytarget.Python(t)
	kit, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "mcp-n4-kit"))
	if err != nil {
		t.Fatal(err)
	}
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
	cmd := exec.Command(py, "-I", "-c", "import sys; sys.path.insert(0, sys.argv[1]); exec(sys.argv[2])", kit, script)
	cmd.Stdin = strings.NewReader(string(in))
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("python: %v\n%s", err, stderr.String())
	}
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
