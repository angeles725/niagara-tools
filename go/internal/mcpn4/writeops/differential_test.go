package writeops_test

import (
	"bufio"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pytarget"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// pyPlans runs each case (one JSON line: tool, args, the nodes each ORD loads before and after
// the write) through the real tools_write plan and read-back of that tool, then prints the
// plan the token is bound to, its plan_hash, the targets and the read-back, or the error
// (class and message) where Python raises.
const pyPlans = `import hashlib, json, sys
from mcp_n4 import box, safety, tools_write
STATION = {"name": "FakeStation", "base_url": "http://fake.invalid", "session_id": "0" * 32}

def describe(e):
    return {"type": type(e).__name__, "message": str(e)}

class Client:
    def load_tree(self, ord_str, depth=2, **kw):
        if ord_str not in self.loads:
            raise box.BoxError("no such component: " + ord_str)
        return self.loads[ord_str]

for line in sys.stdin:
    c = json.loads(line)
    impl, client = tools_write._IMPLS[c["tool"]], Client()
    client.loads = c["before"]
    try:
        p = impl.plan(client, c["args"])
        inverse = tools_write._inverse_list(p.inverse)
        plan = {"tool": c["tool"], "ops": p.ops, "inverse": tools_write._hashed_inverse(inverse),
                "notes": p.notes, "station": STATION}
        out = {"plan": plan, "plan_hash": hashlib.sha256(safety.canonical(plan).encode()).hexdigest(),
               "targets": tools_write._data(p).get("targets", {})}
    except Exception as e:
        print(json.dumps({"error": describe(e)}))
        continue
    client.loads = c["after"]
    try:
        out["readback"] = list(impl.readback(client, c["args"], p, ["reply"], inverse))
    except Exception as e:
        out["readback_error"] = describe(e)
    print(json.dumps(out))
`

// goPlan is pyPlans for one case on the Go side.
func goPlan(c *pyjson.Object) *pyjson.Object {
	tool := get(c, "tool").(string)
	impl := writeops.SetSlot
	if tool == "n4_invoke_action" {
		impl = writeops.InvokeAction
	}
	args := get(c, "args").(*pyjson.Object)
	st := &loads{nodes: get(c, "before").(*pyjson.Object)}
	p, err := impl.Plan(st, args)
	if err != nil {
		return obj("error", describe(err))
	}
	sess := &session.Session{StationName: "FakeStation", BaseURL: "http://fake.invalid", SessionID: strings.Repeat("0", 32)}
	hashed, hash, err := plan.Hashed(tool, p, sess)
	if err != nil {
		return obj("error", describe(err))
	}
	targets := any(p.Targets)
	if p.Targets == nil {
		targets = pyjson.NewObject()
	}
	out := obj("plan", hashed, "plan_hash", hash, "targets", targets)
	st.nodes = get(c, "after").(*pyjson.Object)
	requested, accepted, observed, verdict, err := impl.Readback(st, args, p, []any{"reply"}, p.Inverse)
	if err != nil {
		out.Set("readback_error", describe(err))
	} else {
		out.Set("readback", []any{requested, accepted, observed, verdict})
	}
	return out
}

func describe(err error) *pyjson.Object {
	return obj("type", plan.PyType(err), "message", plan.PyStr(err))
}

// loads is a plan.Client whose loads answer nodes[ord]; it never sends.
type loads struct{ nodes *pyjson.Object }

func (l *loads) LoadTree(ord string, _ int, _ box.LoadOptions) (*pyjson.Object, error) {
	n, ok := l.nodes.Get(ord)
	if !ok {
		return nil, &box.Error{Kind: box.KindBox, Msg: "no such component: " + ord}
	}
	return n.(*pyjson.Object), nil
}
func (l *loads) SSC(string, any) (any, error) { panic("a plan never sends") }
func (l *loads) Sync(any) (any, error)        { panic("a plan never sends") }
func (l *loads) ClearHandles()                {}

func TestPlanAndReadbackDifferentialPython(t *testing.T) {
	py := pytarget.Python(t)
	kit, _ := filepath.Abs(filepath.Join("..", "..", "..", "..", "mcp-n4-kit"))
	cases := corpus()
	cmd := exec.Command(py, "-c", pyPlans)
	cmd.Dir = kit
	cmd.Env = append(cmd.Environ(), "PYTHONDONTWRITEBYTECODE=1")
	cmd.Stdin = strings.NewReader(strings.Join(cases, "\n") + "\n")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	sc.Buffer(nil, 1<<22)
	kinds := map[string]int{}
	for i := 0; sc.Scan(); i++ {
		c, err := pyjson.Unmarshal([]byte(cases[i]))
		if err != nil {
			t.Fatal(err)
		}
		got, err := pyjson.Marshal(goPlan(c.(*pyjson.Object)), pyjson.Options{})
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != sc.Text() {
			t.Errorf("case %d %s:\n go     %s\n python %s", i, cases[i], got, sc.Text())
		}
		kinds[outcome(sc.Text())]++
	}
	t.Logf("%d cases: %v", len(cases), kinds)
	for _, k := range []string{"refused", "internal", "verified", "mismatch", "unverified", "readback_error"} {
		if kinds[k] == 0 {
			t.Errorf("no case reaches %s", k)
		}
	}
}

// outcome classifies a Python result line, so the corpus provably reaches every branch class.
func outcome(line string) string {
	switch {
	case strings.Contains(line, `{"error": {"type": "ToolError"`):
		return "refused"
	case strings.HasPrefix(line, `{"error"`):
		return "internal"
	case strings.Contains(line, `"readback_error"`):
		return "readback_error"
	}
	for _, v := range []string{"verified", "mismatch", "unverified"} {
		if strings.HasSuffix(line, `, "`+v+`"]}`) {
			return v
		}
	}
	return "other"
}

func get(o *pyjson.Object, k string) any {
	v, _ := o.Get(k)
	return v
}

func obj(kv ...any) *pyjson.Object {
	o := pyjson.NewObject()
	for i := 0; i < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
	return o
}
