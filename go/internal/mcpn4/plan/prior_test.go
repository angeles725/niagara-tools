package plan_test

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pytarget"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// pyPrior prints, per inverse, tools_write._hashed_inverse and _link_input_values (or its
// error).
const pyPrior = `import json, sys
from mcp_n4 import tools_write
for line in sys.stdin:
    inv = json.loads(line)
    out = {"hashed": tools_write._hashed_inverse(inv)}
    try:
        out["preview"] = tools_write._link_input_values(inv)
    except Exception as e:
        out["error"] = {"type": type(e).__name__, "message": str(e)}
    print(json.dumps(out))
`

var inverses = []string{
	`[]`, `[{"nm": "v", "h": "3", "n": "A"}]`, `[1, "x", null, [1], {"prior": 1}]`,
	`[{"relink": {"source_path": "A", "target_path": "B", "target_slot": "in1", "prior": 1.5, "z": 0}}]`,
	`[{"x": 1, "unlinked_input": {"target_slot": "in2", "prior": null, "target_path": "C"}, "y": 2}]`,
	`[{"relink": {"target_path": "B"}, "unlinked_input": {"target_path": "D", "target_slot": "s", "prior": "v"}}]`,
	`[{"relink": {"prior": {"value": 1, "status": "0"}, "target_path": "B", "target_slot": "t"}}, {"nm": "v"}]`,
	`[{"relink": "flat", "unlinked_input": [1]}, {"relink": {"target_slot": "t", "prior": 2}}]`,
	`[{"relink": {"target_path": "B", "prior": 2}}]`, `[{"unlinked_input": {"prior": 2}}]`,
	`[{"relink": {"target_path": "A", "target_slot": "t", "prior": 1}, "unlinked_input": {"target_path": "B", "target_slot": "u", "prior": 2}}]`,
}

func TestHashedInverseAndLinkInputValuesDifferentialPython(t *testing.T) {
	py := pytarget.Python(t)
	kit, _ := filepath.Abs(filepath.Join("..", "..", "..", "..", "mcp-n4-kit"))
	cmd := exec.Command(py, "-c", pyPrior)
	cmd.Dir = kit
	cmd.Env = append(cmd.Environ(), "PYTHONDONTWRITEBYTECODE=1")
	cmd.Stdin = strings.NewReader(strings.Join(inverses, "\n") + "\n")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	n := 0
	for i := 0; sc.Scan(); i++ {
		v, err := pyjson.Unmarshal([]byte(inverses[i]))
		if err != nil {
			t.Fatal(err)
		}
		inv := v.([]any)
		got := node("hashed", plan.HashedInverse(inv))
		if preview, err := plan.LinkInputValues(inv); err != nil {
			got.Set("error", node("type", plan.PyType(err), "message", plan.PyStr(err)))
		} else {
			got.Set("preview", preview)
		}
		b, _ := pyjson.Marshal(got, pyjson.Options{})
		if string(b) != sc.Text() {
			t.Errorf("case %d %s:\n go     %s\n python %s", i, inverses[i], b, sc.Text())
		}
		n++
	}
	if n != len(inverses) {
		t.Fatalf("python answered %d of %d", n, len(inverses))
	}
}

// keysRig serves n4_set_slot whose plan carries keys and an inverse entry whose prior value
// is *prior when planned.
func keysRig(t *testing.T, prior *any, keys *pyjson.Object) *rig {
	r := newRig(t)
	impl := writeops.SetSlot
	plan0 := impl.Plan
	impl.Plan = func(c plan.Client, args *pyjson.Object) (*plan.Planned, error) {
		p, err := plan0(c, args)
		if err == nil {
			p.Keys = keys
			p.Inverse = append(p.Inverse, node("relink", node("target_path", "B", "target_slot", "in1", "prior", *prior)))
		}
		return p, err
	}
	r.srv.Handlers["n4_set_slot"] = (&plan.Engine{Ctx: r.ctx, State: r.state}).Handler("n4_set_slot", impl)
	return r
}

// The token covers the plan keys and the inverse without its prior values: a prior that moved
// between the dry run and the execution still executes, and the journal keeps the prior of the
// re-plan (_process).
func TestPlanKeysAndPriorValuesInTheTokenAndTheIntent(t *testing.T) {
	prior := any(1)
	keys := node("snapshot_truncated", []any{"x"}, "components", []any{}, "relinks", []any{"r"},
		"rollback_of", "b1", "outgoing_links_broken", nil, "unlinked_inputs", []any{"u"})
	r := keysRig(t, &prior, keys)
	_, isErr, out := r.call(t, "n4_set_slot", setArgs())
	p := get(out, "plan").(*pyjson.Object)
	canon, _ := pyjson.Marshal(p, pyjson.Canonical)
	sum := sha256.Sum256(canon)
	if isErr || hex.EncodeToString(sum[:]) != get(out, "plan_hash") {
		t.Fatal(pyjson.Repr(out))
	}
	if got := strings.Join(p.Keys(), ","); got != "tool,ops,inverse,notes,station,relinks,unlinked_inputs,snapshot_truncated" {
		t.Errorf("plan keys %s", got)
	}
	if got := pyjson.Repr(get(p, "inverse").([]any)[1]); got != "{'relink': {'target_path': 'B', 'target_slot': 'in1'}}" {
		t.Errorf("hashed inverse %s", got)
	}
	if got := pyjson.Repr(get(out, "link_input_values")); got != "[{'target_path': 'B', 'target_slot': 'in1', 'value': 1}]" {
		t.Errorf("preview %s", got)
	}
	prior = 2
	text, isErr, _ := r.execute(t, "n4_set_slot", setArgs(), get(out, "confirmation_token").(string))
	intent := r.lines(t, "journal.jsonl")[0]
	if isErr {
		t.Fatal(text)
	}
	if got := strings.Join(intent.Keys(), ","); got != "batch_id,ts,tool,ops,inverse_plan,station_name,phase,targets,rollback_of,relinks" {
		t.Errorf("intent keys %s", got)
	}
	if got := pyjson.Repr(get(intent, "inverse_plan").([]any)[1]); !strings.Contains(got, "'prior': 2") {
		t.Errorf("journaled inverse %s", got)
	}
}

func TestAPreviewThatCannotBeBuiltIsAnAuditedInternalError(t *testing.T) {
	r := newRig(t)
	impl := writeops.SetSlot
	plan0 := impl.Plan
	impl.Plan = func(c plan.Client, args *pyjson.Object) (*plan.Planned, error) {
		p, err := plan0(c, args)
		if err == nil {
			p.Inverse = append(p.Inverse, node("unlinked_input", node("target_slot", "in1", "prior", 1)))
		}
		return p, err
	}
	r.srv.Handlers["n4_set_slot"] = (&plan.Engine{Ctx: r.ctx, State: r.state}).Handler("n4_set_slot", impl)
	text, isErr, _ := r.call(t, "n4_set_slot", setArgs())
	audit := r.lines(t, "audit.jsonl")
	if !isErr || text != "internal error (KeyError)" || get(audit[0], "reason") != "KeyError: 'target_path'" {
		t.Fatal(text, pyjson.Repr(audit))
	}
}
