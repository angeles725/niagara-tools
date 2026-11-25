package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// A deployed template instance: Inst holds its templateConfig, whose exposed setpoint is bound
// (ConfigBinding, targetOrd h:9) to Inst/Pt.limit, and whose label is bound to #Name of Pt
// (#990 M3; BTemplateConfig.changed propagates a config slot to its binding's target).
const (
	cfgInst = "station:|slot:/LabScratch/Inst"
	cfgOrd  = cfgInst + "/templateConfig"
)

func cfgStation(fallback string, target string) *loads {
	binding := func(name, source, targetOrd, slot string) []any {
		return []any{name, obj("h", "b"+name, "t", "template:ConfigBinding"),
			name + "/sourceSlot", obj("t", "baja:String", "v", source),
			name + "/targetOrd", obj("t", "baja:Ord", "v", targetOrd),
			name + "/targetSlot", obj("t", "baja:String", "v", slot)}
	}
	cfg := obj("", obj("h", "8", "t", "template:TemplateConfig"),
		"setpoint", obj("t", "baja:Double", "v", "20.0"), "label", obj("t", "baja:String", "v", "Pt"),
		"note", obj("t", "baja:String", "v", "x"))
	for _, b := range [][]any{binding("B1", "setpoint", target, "limit"), binding("B2", "label", "h:9", "#Name")} {
		for i := 0; i < len(b); i += 2 {
			cfg.Set(b[i].(string), b[i+1])
		}
	}
	inst := obj("", obj("h", "7", "t", "baja:Folder"), "templateConfig", obj("h", "8", "t", "template:TemplateConfig"),
		"Pt", obj("h", "9", "t", "kitControl:Limiter"),
		"Pt/limit", obj("t", "baja:Double", "v", fallback))
	pt := obj("", obj("h", "9", "t", "kitControl:Limiter"), "limit", obj("t", "baja:Double", "v", fallback))
	return &loads{nodes: obj(cfgOrd, cfg, cfgInst, inst, cfgInst+"/Pt", pt)}
}

func cfgSet(slot string, value any, valueType string) *pyjson.Object {
	return obj("ord", cfgOrd, "slot", slot, "value", value, "value_type", valueType)
}

func TestSetSlotOnATemplateConfigNamesAndReadsBackThePropagatedTarget(t *testing.T) {
	c := cfgStation("20.0", "h:9")
	p, err := writeops.SetSlot.Plan(c, cfgSet("setpoint", pyjson.FloatNumber(21), "baja:Double"))
	if err != nil {
		t.Fatal(err)
	}
	to, _ := p.Keys.Get("propagates_to")
	if want := `{"ord": "station:|slot:/LabScratch/Inst/Pt", "slot": "limit"}`; pyjson.Repr(to) != want && js(t, to) != want {
		t.Errorf("propagates_to = %s, want %s", js(t, to), want)
	}
	if !strings.Contains(js(t, p.Notes), "station:|slot:/LabScratch/Inst/Pt") {
		t.Errorf("notes = %s", js(t, p.Notes))
	}
	readback := func(fallback string) string {
		after := cfgStation(fallback, "h:9")
		cfg, _ := after.nodes.Get(cfgOrd)
		cfg.(*pyjson.Object).Set("setpoint", obj("t", "baja:Double", "v", "21.0"))
		_, _, _, v, err := writeops.SetSlot.Readback(after, cfgSet("setpoint", pyjson.FloatNumber(21), "baja:Double"), p, []any{"r"}, p.Inverse)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	if v := readback("21.0"); v != "verified" {
		t.Errorf("target propagated: verdict %s", v)
	}
	if v := readback("20.0"); v != "mismatch" {
		t.Errorf("target not propagated: verdict %s", v)
	}
	// a slot no binding names plans as before
	if p, err := writeops.SetSlot.Plan(c, cfgSet("note", "y", "baja:String")); err != nil || hasKey(p.Keys, "propagates_to") {
		t.Errorf("unbound slot: %v %v", err, p)
	}
}

func TestSetSlotRefusesAConfigBoundToNameOrATargetOutsideTheInstance(t *testing.T) {
	if _, err := writeops.SetSlot.Plan(cfgStation("20.0", "h:9"), cfgSet("label", "Renamed", "baja:String")); err == nil ||
		!strings.Contains(err.Error(), "#Name") || !strings.Contains(err.Error(), "n4_rename_component") {
		t.Errorf("#Name binding: %v", err)
	}
	if _, err := writeops.SetSlot.Plan(cfgStation("20.0", "h:99"), cfgSet("setpoint", pyjson.FloatNumber(21), "baja:Double")); err == nil ||
		!strings.Contains(err.Error(), "h:99") {
		t.Errorf("target outside the instance: %v", err)
	}
}

func TestSetSlotRefusesAPropagatedTargetOutsideTheWriteScope(t *testing.T) {
	scope, err := ord.NewScope([]string{"station:|slot:/LabScratch/Inst/templateConfig"})
	if err != nil {
		t.Fatal(err)
	}
	w := &plan.State{Scope: scope}
	if _, err := writeops.SetSlot.PlanWith(cfgStation("20.0", "h:9"), cfgSet("setpoint", pyjson.FloatNumber(21), "baja:Double"), w, nil); err == nil ||
		!strings.Contains(err.Error(), "station:|slot:/LabScratch/Inst/Pt") {
		t.Errorf("target outside the write scope: %v", err)
	}
}

func hasKey(o *pyjson.Object, k string) bool {
	_, ok := o.Get(k)
	return ok
}
