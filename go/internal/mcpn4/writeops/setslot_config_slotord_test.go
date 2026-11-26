package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// #934: n4_apply re-points a copy's ConfigBinding from the golden's handle to the slot ORD of the
// copy's own component, so set_slot on that copy's templateConfig must follow a slot ORD inside
// the instance (and still refuse one outside it).
func TestSetSlotFollowsAConfigBoundBySlotOrdInsideTheInstance(t *testing.T) {
	c := cfgStation("20.0", cfgInst+"/Pt")
	p, err := writeops.SetSlot.Plan(c, cfgSet("setpoint", pyjson.FloatNumber(21), "baja:Double"))
	if err != nil {
		t.Fatal(err)
	}
	to, _ := p.Keys.Get("propagates_to")
	if got := js(t, to); !strings.Contains(got, `"ord": "station:|slot:/LabScratch/Inst/Pt"`) {
		t.Errorf("propagates_to = %s", got)
	}
	for _, outside := range []string{"station:|slot:/LabScratch/Other/Pt", "station:|slot:/LabScratch/InstX/Pt"} {
		_, err := writeops.SetSlot.Plan(cfgStation("20.0", outside), cfgSet("setpoint", pyjson.FloatNumber(21), "baja:Double"))
		if err == nil || !strings.Contains(err.Error(), "found in the template instance") {
			t.Errorf("%s: err = %v", outside, err)
		}
	}
}

// A ".." segment would leave the instance whatever its prefix (independent verify, #934).
func TestSetSlotRefusesADotDotSlotOrdTarget(t *testing.T) {
	for _, target := range []string{
		"station:|slot:/LabScratch/Inst/../Other/Pt",
		"station:|slot:/LabScratch/Inst/Pt/../../Other",
	} {
		_, err := writeops.SetSlot.Plan(cfgStation("20.0", target), cfgSet("setpoint", pyjson.FloatNumber(21), "baja:Double"))
		if err == nil || !strings.Contains(err.Error(), "found in the template instance") {
			t.Errorf("%s: err = %v", target, err)
		}
	}
}
