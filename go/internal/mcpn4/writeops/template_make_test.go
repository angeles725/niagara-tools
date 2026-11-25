package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

const (
	mkService = "station:|slot:/Services/TemplateService"
	mkJob     = "station:|slot:/Services/JobService/MakeStationTemplateJob"
	mkFile    = "file:^temp/Station.ntpl"
)

// mkStation is a station with the TemplateService and a make-template job in state that produced
// file ("" = none); readLog answers log.
func mkStation(state, log, file string) *asks {
	job := []any{obj("h", "j7", "t", "template:MakeStationTemplateJob"),
		"jobState", obj("t", "baja:JobState", "v", state)}
	if file != "" {
		job = append(job, "templateFileOrd", obj("t", "baja:Ord", "v", file),
			"templateName", obj("t", "baja:String", "v", "Station"))
	}
	return &asks{loads: loads{nodes: obj(
		mkService, obj("", obj("h", "ts1", "t", "template:TemplateService")),
		mkJob, obj(append([]any{""}, job...)...),
	)}, online: obj("t", "baja:String", "v", log)}
}

// refuses is a station that answers every load with its own error reply (no such component).
type refuses struct{ loads }

func (*refuses) LoadTree(ord string, _ int, _ box.LoadOptions) (*pyjson.Object, error) {
	return nil, &box.Error{Kind: box.KindBox, Msg: "no such component: " + ord, Station: true}
}

// mkPlan plans with accept_runaway_risk=true (the guard, template_make_guard_test.go, refuses without it)
// unless args say otherwise.
func mkPlan(t *testing.T, c *asks, args *pyjson.Object) *plan.Planned {
	t.Helper()
	if _, given := args.Get("accept_runaway_risk"); !given {
		args.Set("accept_runaway_risk", true)
	}
	p, err := writeops.MakeStationTemplate.PlanWith(c, args, clock(), nil)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// BTemplateService.doMakeStationTemplate(BBoolean useMinorVersionOnDeployment, Context) takes a
// BBoolean (default true, BTemplateService.java:71-76) and returns the job's BOrd.
func TestMakeStationTemplateDryRunPlansTheActionAndNamesTheFile(t *testing.T) {
	p := mkPlan(t, mkStation("success", "", mkFile), obj())
	want := `{"ssc": "invokeAction", "arg": {"h": "ts1", "a": "makeStationTemplate", "b": {"nm": "p", "t": "baja:Boolean", "v": "true"}}}`
	if got := js(t, p.Ops[0]); got != want {
		t.Errorf("op = %s", got)
	}
	if len(p.Inverse) != 0 {
		t.Errorf("inverse = %v", p.Inverse)
	}
	notes := js(t, p.Notes)
	for _, part := range []string{"^temp", "writes a file", "no inverse", "super user", "n4_read_file"} {
		if !strings.Contains(notes, part) {
			t.Errorf("notes miss %q: %s", part, notes)
		}
	}
	if got := js(t, p.Targets); !strings.Contains(got, mkService) {
		t.Errorf("targets = %s", got)
	}
	off := mkPlan(t, mkStation("success", "", mkFile), obj("use_minor_version", false))
	if got := js(t, off.Ops[0]); !strings.Contains(got, `"v": "false"`) {
		t.Errorf("use_minor_version=false op = %s", got)
	}
}

// Like n4_save_station the tool is whole-station: it changes no component, so any write scope will do.
func TestMakeStationTemplateIsWholeStation(t *testing.T) {
	for _, a := range []*pyjson.Object{obj(), obj("service_ord", "station:|slot:/Services/Other")} {
		if got := writeops.MakeStationTemplate.ScopeOrds(a); got != nil {
			t.Errorf("scope ords = %v, want nil (whole-station)", got)
		}
	}
}

func TestMakeStationTemplateRefusals(t *testing.T) {
	none := &refuses{}
	if _, err := writeops.MakeStationTemplate.PlanWith(none, obj("accept_runaway_risk", true), clock(), nil); err == nil ||
		!strings.Contains(err.Error(), "no TemplateService") || !strings.Contains(err.Error(), mkService) {
		t.Errorf("no service: err = %v", err)
	}
	other := &asks{loads: loads{nodes: obj(mkService, obj("", obj("h", "x", "t", "baja:Folder")))}}
	if _, err := writeops.MakeStationTemplate.PlanWith(other, obj("accept_runaway_risk", true), clock(), nil); err == nil ||
		!strings.Contains(err.Error(), "template:TemplateService") {
		t.Errorf("wrong type: err = %v", err)
	}
	c := mkStation("success", "", mkFile)
	for _, tc := range []struct {
		args *pyjson.Object
		want string
	}{
		{obj("accept_runaway_risk", true, "use_minor_version", "yes"), "use_minor_version"},
		{obj("accept_runaway_risk", true, "wait_seconds", 0), "wait_seconds"},
		{obj("accept_runaway_risk", true, "wait_seconds", 601), "wait_seconds"},
	} {
		if _, err := writeops.MakeStationTemplate.PlanWith(c, tc.args, clock(), nil); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want one naming %s", js(t, tc.args), err, tc.want)
		}
	}
}

// PRESERVE: n4_invoke_action still refuses the hidden action; only the dedicated tool runs it.
func TestInvokeActionStillRefusesMakeStationTemplate(t *testing.T) {
	c := mkStation("success", "", mkFile)
	args := obj("ord", mkService, "action", "makeStationTemplate")
	if _, err := writeops.InvokeAction.PlanWith(c, args, clock(), nil); err == nil {
		t.Errorf("n4_invoke_action accepted makeStationTemplate")
	}
}
