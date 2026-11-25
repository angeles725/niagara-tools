package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// Confirmed on a 4.14 station: the job zips the station home including ^temp, where it writes its own
// output, so the .ntpl grows without end. The dry run refuses by default and says cause, risk, recovery.
func TestMakeStationTemplateRefusesByDefault(t *testing.T) {
	for _, args := range []*pyjson.Object{obj(), obj("accept_runaway_risk", false), obj("use_minor_version", false)} {
		c := mkStation("success", "", mkFile)
		_, err := writeops.MakeStationTemplate.PlanWith(c, args, clock(), nil)
		if err == nil {
			t.Fatalf("%s: planned without accept_runaway_risk", js(t, args))
		}
		for _, part := range []string{"^temp", "zips", "own output", "without end", "disk", "n4_save_station",
			"stop the station process", "delete the partial", "^temp/*.ntpl", "restart", "cancel does not stop it",
			"accept_runaway_risk=true"} {
			if !strings.Contains(err.Error(), part) {
				t.Errorf("%s: refusal misses %q: %v", js(t, args), part, err)
			}
		}
		if len(c.sent) != 0 {
			t.Errorf("a refused plan touched the station: %v", c.sent)
		}
	}
}

func TestMakeStationTemplateAcceptedRiskPlansWithTheSameWarning(t *testing.T) {
	p := mkPlan(t, mkStation("success", "", mkFile), obj("accept_runaway_risk", true))
	notes := js(t, p.Notes)
	for _, part := range []string{"RUNAWAY RISK", "without end", "disk", "stop the station process", "delete the partial"} {
		if !strings.Contains(notes, part) {
			t.Errorf("notes miss %q: %s", part, notes)
		}
	}
	if got := js(t, p.Ops[0]); !strings.Contains(got, `"a": "makeStationTemplate"`) {
		t.Errorf("op = %s", got)
	}
}

func TestMakeStationTemplateAcceptRunawayRiskMustBeABoolean(t *testing.T) {
	_, err := writeops.MakeStationTemplate.PlanWith(mkStation("success", "", mkFile),
		obj("accept_runaway_risk", "yes"), clock(), nil)
	if err == nil || !strings.Contains(err.Error(), "accept_runaway_risk must be true or false") {
		t.Errorf("err = %v", err)
	}
}
