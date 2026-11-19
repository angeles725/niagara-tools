package blackbox

// A Status value's status is part of its configuration (delta P2, live near-miss 2026-11-19): a
// schedule whose in and defaultOutput held {null} was copied with {ok} on every copy, because the
// status child is declared readonly and was left out; the copies' out became a live false / 0.0
// that would have switched units OFF once linked. A copy carries the status whole with its value
// when it is off its declared default, and the read-back compares it.

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// statusValueContract declares a Status value type's value and its readonly status (default ok).
func statusValueContract(valueType string) map[string]any {
	return map[string]any{"c": []any{frozenSlot("value", valueType, ""),
		map[string]any{"st": "p", "n": "status", "f": "r", "v": map[string]any{"nm": "p", "t": "baja:Status", "v": "0"}}},
		"p": "baja:StatusValue"}
}

func scheduleContract(valueType string) map[string]any {
	return map[string]any{"c": []any{frozenSlot("in", valueType, ""), frozenSlot("defaultOutput", valueType, ""),
		frozenSlot("out", valueType, "rtso")}, "p": "schedule:WeeklySchedule"}
}

var statusContracts = map[string]any{
	"schedule:BooleanSchedule": scheduleContract("baja:StatusBoolean"),
	"schedule:NumericSchedule": scheduleContract("baja:StatusNumeric"),
	"kitControl:NumericConst":  constContracts["kitControl:NumericConst"],
	"baja:StatusBoolean":       statusValueContract("baja:Boolean"),
	"baja:StatusNumeric":       statusValueContract("baja:Double"),
}

// nullSchedule adds a schedule of typ whose in and defaultOutput hold status null (40).
func (d *dRig) nullSchedule(name, typ string) Component {
	d.T.Helper()
	src := d.Add("3", name, typ)
	whole := StatusNumeric(0.0, "40")
	if typ == "schedule:BooleanSchedule" {
		whole = StatusBoolean(false, "40")
	}
	d.Set(src.Handle, "in", whole)
	d.Set(src.Handle, "defaultOutput", whole)
	return src
}

// statusOf is the status child's "v" of slot in a copy op's body; nil when it carries none.
func statusOf(op any, slot string) any {
	s := byName(Dig(byName(Dig(op, "b", "s"))[slot], "s"))["status"]
	if s == nil {
		return nil
	}
	return s["v"]
}

func TestACopyOfANullScheduleCarriesTheNullStatus(t *testing.T) {
	for _, typ := range []string{"schedule:BooleanSchedule", "schedule:NumericSchedule"} {
		t.Run(typ, func(t *testing.T) {
			d := destructiveRig(t, nil)
			d.withContractsOf(statusContracts)
			src := d.nullSchedule("S", typ)
			dry := d.Srv.Dry("n4_copy_component", copyConstArgs(src.Name, "S2"))
			ops := objects(planOf(dry)["ops"])
			if len(ops) != 1 {
				t.Fatalf("plan ops = %v", planOf(dry)["ops"])
			}
			for _, slot := range []string{"in", "defaultOutput"} {
				if got := statusOf(ops[0], slot); got != "40" {
					t.Errorf("planned %s status = %v, want 40 (op %v)", slot, got, ops[0])
				}
			}
			done := d.Srv.OK("n4_copy_component", Execute(copyConstArgs(src.Name, "S2"), dry["confirmation_token"]))
			if done["verdict"] != "verified" {
				t.Errorf("verdict = %v (%v)", done["verdict"], done)
			}
			if got := d.value(d.node("S2").Handle, "in", "status"); got != "40" {
				t.Errorf("S2.in status = %v, want 40", got)
			}
		})
	}
}

func TestACopyWhoseStatusDiffersFromTheSourceIsAMismatch(t *testing.T) {
	d := destructiveRig(t, nil)
	d.withContractsOf(statusContracts)
	src := d.nullSchedule("S", "schedule:BooleanSchedule")
	d.afterAdd(func(_ *boxstation.Station, n *boxstation.Node) { // the station re-arms it {ok}
		if s := n.Child("in").Child("status"); s != nil {
			s.Value = "0"
		}
	})
	done := d.Srv.RunWrite("n4_copy_component", copyConstArgs(src.Name, "S2"))
	if done["verdict"] != "mismatch" {
		t.Errorf("verdict = %v (%v)", done["verdict"], done)
	}
}

func TestAPlainCopyStillPlansTheSameOps(t *testing.T) {
	d := destructiveRig(t, nil)
	d.withContractsOf(statusContracts)
	src := d.AddWithOut("K", 10.0) // status ok: the declared default, left out as before
	dry := d.Srv.Dry("n4_copy_component", copyConstArgs(src.Name, "K2"))
	ops := objects(planOf(dry)["ops"])
	want := map[string]any{"nm": "a", "h": "3", "n": "K2", "b": map[string]any{"nm": "p",
		"t": "kitControl:NumericConst", "s": []any{map[string]any{"nm": "p", "t": "baja:StatusNumeric", "n": "out",
			"s": []any{map[string]any{"nm": "p", "t": "baja:Double", "n": "value", "v": "10.0"}}}}}}
	if len(ops) != 1 || !same(ops[0], want) {
		t.Errorf("plan ops = %v, want [%v]", planOf(dry)["ops"], want)
	}
}
