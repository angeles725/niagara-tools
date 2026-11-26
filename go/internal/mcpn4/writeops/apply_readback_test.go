package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// apWorld is the station after the batch: the parent holding Gold and each row, every instance
// loading by its own ORD (and its bound components by theirs).
func apWorld(gold apState, rows map[string]apState) *typed {
	parent := obj("", obj("h", "5", "t", "baja:Folder"), "Gold", obj("h", "10", "t", "baja:Folder"))
	nodes := obj()
	put := func(ord string, s apState) {
		nodes.Set(ord, apFlat(s.tree()))
		nodes.Set(ord+"/Pt", apFlat(s.pt()))
		nodes.Set(ord+"/Pt/Inner", apFlat(s.inner()))
	}
	put(apGold, gold)
	for n, s := range rows {
		parent.Set(n, obj("h", "20", "t", "baja:Folder"))
		put(apParent+"/"+n, s)
	}
	nodes.Set(apParent, parent)
	return &typed{loads: loads{nodes: nodes}, types: apTypes()}
}

// apApplied is an instance after a correct apply: bindings on its own components, the configs set
// and propagated to their targets.
func apApplied(row string, mod func(*apState)) apState {
	own := apParent + "/" + row
	s := apState{own + "/Pt", own + "/Pt", own + "/Pt/Inner", "20.0", "x", "1.0", "20.0", "", "1.0"}
	if mod != nil {
		mod(&s)
	}
	return s
}

func apReadback(t *testing.T, gold apState, rows map[string]apState) (*pyjson.Object, string) {
	t.Helper()
	p, err := writeops.Apply.Plan(apStation(apGolden()), apArgs(
		apRow("R1", "setpoint", pyjson.FloatNumber(25), "label", "Zone A"),
		apRow("R2", "gain", pyjson.FloatNumber(2.5))))
	if err != nil {
		t.Fatal(err)
	}
	inv, err := writeops.Apply.Inverse(p, []any{[]any{obj("nn", "R1")}, []any{obj("nn", "R2")}})
	if err != nil {
		t.Fatal(err)
	}
	_, _, observed, verdict, err := writeops.Apply.Readback(apWorld(gold, rows), apArgs(), p, nil, inv)
	if err != nil {
		t.Fatal(err)
	}
	return observed.(*pyjson.Object), verdict
}

func apGood() map[string]apState {
	return map[string]apState{
		"R1": apApplied("R1", func(s *apState) { s.setpoint, s.limit, s.label, s.tag = "25.0", "25.0", "Zone A", "Zone A" }),
		"R2": apApplied("R2", func(s *apState) { s.cfgGain, s.gain = "2.5", "2.5" }),
	}
}

func TestApplyReadbackVerifiesCopiesBindingsConfigsTargetsAndTheGolden(t *testing.T) {
	observed, verdict := apReadback(t, apGolden(), apGood())
	if verdict != "verified" || get(observed, "golden_unchanged") != true {
		t.Fatalf("verdict %s: %s", verdict, js(t, observed))
	}
	rows := get(observed, "rows").([]any)
	want := `{"name": "R1", "configs": {"setpoint": {"requested": 25.0, "observed": 25.0, "target_observed": 25.0}, ` +
		`"label": {"requested": "Zone A", "observed": "Zone A", "target_observed": "Zone A"}}, "verdict": "verified"}`
	if got := js(t, rows[0]); got != want {
		t.Errorf("row 1\n got  %s\n want %s", got, want)
	}
}

func TestApplyReadbackIsAMismatchWhenATargetDidNotReceiveTheValue(t *testing.T) {
	rows := apGood()
	rows["R1"] = apApplied("R1", func(s *apState) { s.setpoint, s.label, s.tag = "25.0", "Zone A", "Zone A" }) // limit stays 20
	observed, verdict := apReadback(t, apGolden(), rows)
	r := get(observed, "rows").([]any)
	if verdict != "mismatch" || get(r[0].(*pyjson.Object), "verdict") != "mismatch" || get(r[1].(*pyjson.Object), "verdict") != "verified" {
		t.Errorf("verdict %s: %s", verdict, js(t, observed))
	}
}

func TestApplyReadbackIsAMismatchWhenABindingStillNamesTheGolden(t *testing.T) {
	rows := apGood()
	s := rows["R2"]
	s.b1 = "h:13"
	rows["R2"] = s
	observed, verdict := apReadback(t, apGolden(), rows)
	text := js(t, observed)
	if verdict != "mismatch" || !strings.Contains(text, "B1") || !strings.Contains(text, "h:13") {
		t.Errorf("verdict %s: %s", verdict, text)
	}
}

func TestApplyReadbackReportsAChangedGolden(t *testing.T) {
	gold := apGolden()
	gold.limit = "25.0" // a copy's config reached the golden's target
	observed, verdict := apReadback(t, gold, apGood())
	if verdict != "mismatch" || get(observed, "golden_unchanged") != false {
		t.Errorf("verdict %s: %s", verdict, js(t, observed))
	}
}

func TestApplyReadbackIsAMismatchWhenACopyIsMissing(t *testing.T) {
	rows := apGood()
	delete(rows, "R2")
	observed, verdict := apReadback(t, apGolden(), rows)
	if verdict != "mismatch" || !strings.Contains(js(t, observed), "copy not found") {
		t.Errorf("verdict %s: %s", verdict, js(t, observed))
	}
}

var _ plan.Client = (*typed)(nil)
