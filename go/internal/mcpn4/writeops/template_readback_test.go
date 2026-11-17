package writeops_test

import (
	"archive/zip"
	"io"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// deployed is FC1 as the station loads it after the deploy: what was planned, plus slots the
// station adds (status, out), without FanCmd's fallback value (the station lists a frozen slot
// only off its declared default; FanAuto=false equals it) and with a status value carrying facets.
func deployed(setpoint string) *pyjson.Object {
	return obj("h", "20", "t", "baja:Folder", "s", []any{
		obj("n", "location", "t", "baja:String", "v", "Room 101"),
		obj("n", "Setpoint", "h", "21", "t", "control:NumericWritable", "s", []any{
			obj("n", "status", "t", "baja:Status", "v", "0;activeLevel=e:17@control:PriorityLevel"),
			obj("n", "facets", "v", "units=u:fahrenheit;°F;(K);*0.5555555555555556+255.37222222222223;|precision=i:1"),
			obj("n", "fallback", "t", "baja:StatusNumeric", "s", []any{obj("n", "value", "v", setpoint)})}),
		obj("n", "FanCmd", "h", "22", "t", "control:BooleanWritable"),
		obj("n", "Control", "h", "23", "t", "baja:Folder", "s", []any{
			obj("n", "deadband", "t", "baja:Double", "v", "2.0"),
			obj("n", "Limit", "h", "24", "t", "control:NumericWritable")}),
		obj("n", "ntpl$3atitle", "t", "baja:String", "v", "FanCoil", "f", "rhm"),
		obj("n", "ntpl$3aid", "t", "baja:String", "v", "5f0c2a8e-3b7d-4c1e-9a62-1d8e4b7c9f30", "f", "rhm"),
		obj("n", "ntpl$3atemplate", "t", "baja:Marker", "v", "M", "f", "rhm"),
		obj("n", "ntpl$3aversion", "t", "baja:String", "v", "1.2", "f", "rhm")})
}

func TestADeployReadsBackTheDeployedTree(t *testing.T) {
	args := deployArgs(tplFile(t, nil, nil))
	args.Set("settings", obj("Setpoint", pyjson.IntLiteral("68"), "Location", "Room 101", "FanAuto", false))
	for setpoint, want := range map[string]string{
		"68.0": `"verified" []`,
		"72.0": `"mismatch" ["Setpoint/fallback/value"]`,
	} {
		c := tplClient(tplTypes(), lab+"/FC1", obj("", deployed(setpoint)))
		p, err := writeops.DeployTemplate.Plan(c, args)
		if err != nil {
			t.Fatal(err)
		}
		replies := []any{[]any{obj("nn", "FC1")}}
		inverse, err := writeops.DeployTemplate.Inverse(p, replies)
		if err != nil {
			t.Fatal(err)
		}
		_, _, observed, verdict, err := writeops.DeployTemplate.Readback(c, args, p, replies, inverse)
		if err != nil {
			t.Fatal(err)
		}
		diffs, _ := observed.(*pyjson.Object).Get("differences")
		if got := js(t, verdict) + " " + js(t, diffs); got != want {
			t.Errorf("setpoint %s: %s, want %s", setpoint, got, want)
		}
	}
}

// The station renames on collision: the inverse removes the name the reply assigned, and a
// component that is not there reads back as a mismatch.
func TestADeployInverseUsesTheAssignedName(t *testing.T) {
	args := deployArgs(tplFile(t, nil, nil))
	c := tplClient(tplTypes())
	p, err := writeops.DeployTemplate.Plan(c, args)
	if err != nil {
		t.Fatal(err)
	}
	replies := []any{[]any{obj("nn", "FC11")}}
	inverse, _ := writeops.DeployTemplate.Inverse(p, replies)
	if got := js(t, inverse); got != `[{"nm": "v", "h": "10", "n": "FC11"}]` {
		t.Errorf("inverse = %s", got)
	}
	_, _, observed, verdict, err := writeops.DeployTemplate.Readback(c, args, p, replies, inverse)
	if err != nil || verdict != "mismatch" || js(t, observed) != `{"name": "FC11", "type": null, "differences": ["(component not found)"]}` {
		t.Errorf("readback of a missing component: %v %s %s", err, verdict, js(t, observed))
	}
	if unapplied, err := writeops.DeployTemplate.Unapplied(c, args, p); err != nil || !unapplied {
		t.Errorf("unapplied = %v, %v; want true (the parent gained no child)", unapplied, err)
	}
}

// The committed FanCoil.ntpl (the fixture the live proof deploys) holds exactly the sources.
func TestTheCommittedFixtureTemplateMatchesItsSources(t *testing.T) {
	z, err := zip.OpenReader("testdata/template/FanCoil.ntpl")
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	seen := 0
	for _, f := range z.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != tplSource(t, f.Name) {
			t.Errorf("%s differs from testdata/template/FanCoil/%s: rebuild the .ntpl", f.Name, f.Name)
		}
		seen++
	}
	if seen != 2 {
		t.Errorf("FanCoil.ntpl holds %d entries, want 2", seen)
	}
}
