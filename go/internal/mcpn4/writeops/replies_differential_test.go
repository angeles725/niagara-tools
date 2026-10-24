package writeops_test

import (
	"bufio"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pytarget"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// pyReplies is pyPlans for the tools whose inverse is built from the station's replies: after
// the plan, the inverse of each case's replies (or its error, then the recover hook's
// observed and inverse) or the read-back.
const pyReplies = `import hashlib, json, sys
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
        plan = {"tool": c["tool"], "ops": p.ops, "inverse": tools_write._hashed_inverse(p.inverse),
                "notes": p.notes, "station": STATION}
        out = {"plan": plan, "plan_hash": hashlib.sha256(safety.canonical(plan).encode()).hexdigest(),
               "targets": p.data["targets"]}
    except Exception as e:
        print(json.dumps({"error": describe(e)}))
        continue
    client.loads = c["after"]
    try:
        inverse = tools_write._inverse_list(impl.inverse(p, c["replies"]))
    except Exception as e:
        out["inverse_error"] = describe(e)
        if impl.recover is not None:
            try:
                out["recovered"] = list(impl.recover(client, c["args"], p, c["replies"]))
            except Exception as e:
                out["recover_error"] = describe(e)
        print(json.dumps(out))
        continue
    out["inverse"] = inverse
    try:
        out["readback"] = list(impl.readback(client, c["args"], p, c["replies"], inverse))
    except Exception as e:
        out["readback_error"] = describe(e)
    print(json.dumps(out))
`

func goReplies(c *pyjson.Object) *pyjson.Object {
	tool := get(c, "tool").(string)
	impl := writeops.CreateComponent
	if tool == "n4_create_link" {
		impl = writeops.CreateLink
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
	out := obj("plan", hashed, "plan_hash", hash, "targets", p.Targets)
	st.nodes = get(c, "after").(*pyjson.Object)
	replies := get(c, "replies").([]any)
	inverse, err := impl.Inverse(p, replies)
	if err != nil {
		out.Set("inverse_error", describe(err))
		if impl.Recover != nil {
			observed, inv, err := impl.Recover(st, args, p, replies)
			if err != nil {
				out.Set("recover_error", describe(err))
			} else {
				out.Set("recovered", []any{observed, inv})
			}
		}
		return out
	}
	out.Set("inverse", inverse)
	requested, accepted, observed, verdict, err := impl.Readback(st, args, p, replies, inverse)
	if err != nil {
		out.Set("readback_error", describe(err))
	} else {
		out.Set("readback", []any{requested, accepted, observed, verdict})
	}
	return out
}

func TestInverseRecoverAndReadbackDifferentialPython(t *testing.T) {
	py := pytarget.Python(t)
	kit, _ := filepath.Abs(filepath.Join("..", "..", "..", "..", "mcp-n4-kit"))
	cases := append(createCases(), linkCases()...)
	cmd := exec.Command(py, "-c", pyReplies)
	cmd.Dir = kit
	cmd.Env = append(cmd.Environ(), "PYTHONDONTWRITEBYTECODE=1")
	cmd.Stdin = strings.NewReader(strings.Join(cases, "\n") + "\n")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	sc.Buffer(nil, 1<<22)
	kinds, n := map[string]int{}, 0
	for i := 0; sc.Scan(); i++ {
		c, err := pyjson.Unmarshal([]byte(cases[i]))
		if err != nil {
			t.Fatal(err)
		}
		got, _ := pyjson.Marshal(goReplies(c.(*pyjson.Object)), pyjson.Options{})
		if string(got) != sc.Text() {
			t.Errorf("case %d %s:\n go     %s\n python %s", i, cases[i], got, sc.Text())
		}
		kinds[replyOutcome(sc.Text())]++
		n++
	}
	if n != len(cases) {
		t.Fatalf("python answered %d of %d cases", n, len(cases))
	}
	t.Logf("%d cases: %v", n, kinds)
	for _, k := range []string{"refused", "internal", "verified", "mismatch", "failed", "readback_error",
		"recovered", "recovered_none", "recover_error"} { // each an inverse error first
		if kinds[k] == 0 {
			t.Errorf("no case reaches %s", k)
		}
	}
}

// replyOutcome classifies a Python result line, so the corpus provably reaches every branch.
func replyOutcome(line string) string {
	switch {
	case strings.Contains(line, `"recover_error"`):
		return "recover_error"
	case strings.Contains(line, `"recovered": [null, []]`):
		return "recovered_none"
	case strings.Contains(line, `"recovered"`):
		return "recovered"
	case strings.Contains(line, `"inverse_error"`):
		return "inverse_error"
	}
	if o := outcome(line); o != "other" {
		return o
	}
	if strings.HasSuffix(line, `, "failed"]}`) {
		return "failed"
	}
	return "other"
}
