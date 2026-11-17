package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// The fixture deploys as one add of the template component's tree: the given settings written
// into their bound slots (Setpoint's fallback value, the root's location), FanAuto's template
// value written into a fallback the bog leaves at its default (created from the declared
// default), and the template configuration, the link, the handle ORD, facets and categories
// left out with a note each.
func TestADeployAddsTheTemplateTreeWithItsSettings(t *testing.T) {
	p, err := writeops.DeployTemplate.Plan(tplClient(tplTypes()), deployArgs(tplFile(t, nil, nil)))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"nm": "a", "h": "10", "n": "FC1", "b": {"nm": "p", "t": "baja:Folder", "s": [` +
		`{"nm": "p", "t": "baja:String", "n": "location", "v": "Room 101"}, ` +
		`{"nm": "p", "t": "control:NumericWritable", "n": "Setpoint", "s": [` +
		`{"nm": "p", "n": "facets", "v": "units=u:fahrenheit;\u00b0F;(K);*0.5555555555555556+255.37222222222223;|precision=i:1"}, ` +
		`{"nm": "p", "t": "baja:StatusNumeric", "n": "fallback", "s": [{"nm": "p", "n": "value", "v": "68.0"}]}]}, ` +
		`{"nm": "p", "t": "control:BooleanWritable", "n": "FanCmd", "s": [` +
		`{"nm": "p", "t": "baja:StatusBoolean", "n": "fallback", "s": [{"nm": "p", "n": "value", "v": "true"}, {"nm": "p", "n": "status", "v": "40"}]}]}, ` +
		`{"nm": "p", "t": "baja:Folder", "n": "Control", "s": [` +
		`{"nm": "p", "t": "baja:Double", "n": "deadband", "v": "2.0"}, ` +
		`{"nm": "p", "t": "control:NumericWritable", "n": "Limit"}]}, ` +
		`{"nm": "p", "t": "baja:String", "n": "ntpl$3atitle", "v": "FanCoil", "f": "rhm"}, ` +
		`{"nm": "p", "t": "baja:String", "n": "ntpl$3aid", "v": "5f0c2a8e-3b7d-4c1e-9a62-1d8e4b7c9f30", "f": "rhm"}, ` +
		`{"nm": "p", "t": "baja:Marker", "n": "ntpl$3atemplate", "v": "M", "f": "rhm"}, ` +
		`{"nm": "p", "t": "baja:String", "n": "ntpl$3aversion", "v": "1.2", "f": "rhm"}]}}`
	if len(p.Ops) != 1 {
		t.Fatalf("ops %d, want 1", len(p.Ops))
	}
	if got := js(t, p.Ops[0]); got != want {
		t.Errorf("op\n got  %s\n want %s", got, want)
	}
	if got := js(t, p.Inverse); got != `[{"nm": "v", "h": "10", "n": "FC1"}]` {
		t.Errorf("inverse = %s", got)
	}
	notes := js(t, p.Notes)
	for _, s := range []string{
		`Template 'FanCoil' version 1.2 (vendor Lab, id 5f0c2a8e-3b7d-4c1e-9a62-1d8e4b7c9f30): deploys 'FanCoil' (baja:Folder) as 'FC1' under ` + lab,
		`Dependencies (manifest): baja 4.15 minimum, control-rt 4.15 minimum, template-rt 4.15 minimum`,
		`Settings: Setpoint=68.0 (given; degrees Fahrenheit; 50.0..90.0) -> Setpoint/fallback, ` +
			`Location=Room 101 (given) -> location, FanAuto=true (template) -> FanCmd/fallback [status null (40): clear it with n4_set_slot for the value to apply], ` +
			`Deadband=2.0 (template; 0.0..10.0) -> Control/deadband`,
		`Not deployed: templateConfig (template:TemplateConfig)`,
		`1 link(s) inside the template are not deployed: Setpoint.out -> Control/Limit.in10; create them on the deployed component with n4_create_link`,
		`ORD values naming a template handle are left out (they would name unrelated station components): Control/peer (h:4)`,
		`Slot facets (x) are not carried: Control/deadband`,
		`Categories (c) are not carried: (root)`,
	} {
		if !strings.Contains(notes, s) {
			t.Errorf("notes lack %q:\n%s", s, notes)
		}
	}
}

func TestADeployRefusesBadSettings(t *testing.T) {
	path := tplFile(t, nil, nil)
	for _, tc := range []struct {
		settings *pyjson.Object
		want     string
	}{
		{obj("Setpoint", pyjson.IntLiteral("70")), "setting 'Location' is required and the template holds no value for it; pass it in settings"},
		{obj("Location", "A", "Setpoint", pyjson.IntLiteral("95")), "setting 'Setpoint' must be between 50.0 and 90.0, got 95"},
		{obj("Location", "A", "Deadband", pyjson.FloatNumber(-1)), "setting 'Deadband' must be between 0.0 and 10.0, got -1.0"},
		{obj("Location", "A", "Speed", pyjson.IntLiteral("3")), "unknown setting 'Speed'; the template's settings are: Setpoint, Location, FanAuto, Deadband"},
		{obj("Location", "A", "uID", "x"), "setting 'uID' is the template's own metadata, not a deploy-time setting"},
		{obj("Location", "A", "Setpoint", "70"), "setting 'Setpoint' must be a number, got '70'"},
		{obj("Location", pyjson.IntLiteral("5")), "setting 'Location' must be a string, got 5"},
		{obj("Location", "A", "FanAuto", "yes"), "setting 'FanAuto' must be true or false, got 'yes'"},
	} {
		_, err := writeops.DeployTemplate.Plan(tplClient(tplTypes()), deployArgs(path, "settings", tc.settings))
		if err == nil || err.Error() != tc.want {
			t.Errorf("%s: err = %v, want %q", js(t, tc.settings), err, tc.want)
		}
	}
	if _, err := writeops.DeployTemplate.Plan(tplClient(tplTypes()), deployArgs(path, "settings", "x")); err == nil ||
		err.Error() != "settings must be an object of setting name -> value" {
		t.Errorf("settings not an object: err = %v", err)
	}
}

// A station or application template is not a component: deploying it is refused by its kind.
func TestADeployRefusesAStationOrApplicationTemplate(t *testing.T) {
	for attr, want := range map[string]string{
		`isStation="true"`:     `template 'FanCoil' is a station template (isStation="true"): it creates a new station (Workbench New Station wizard), not a component`,
		`isApplication="true"`: `template 'FanCoil' is an application template (isApplication="true"): it is installed on the station's Config root with Workbench, not deployed`,
	} {
		path := tplFile(t, []string{`signature="4a1b2c3d"`, `signature="4a1b2c3d" ` + attr}, nil)
		if _, err := writeops.DeployTemplate.Plan(tplClient(tplTypes()), deployArgs(path)); err == nil || err.Error() != want {
			t.Errorf("%s: err = %v, want %q", attr, err, want)
		}
	}
}

// A type the station's registry does not describe means its module is missing: refused before a
// token, naming the module, the type, where the template uses it and the manifest dependency.
func TestADeployRefusesAMissingModule(t *testing.T) {
	types := tplTypes()
	types.Set("control:BooleanWritable", obj("isErr", true, "m", "control:BooleanWritable"))
	_, err := writeops.DeployTemplate.Plan(tplClient(types), deployArgs(tplFile(t, nil, nil)))
	want := "the station does not have module 'control' installed: type control:BooleanWritable of FanCmd is " +
		"unknown to its type registry (manifest dependency control-rt 4.15 minimum); install the module, then deploy again"
	if err == nil || err.Error() != want {
		t.Errorf("err = %v\nwant %q", err, want)
	}
	types = without(tplTypes(), "control:NumericWritable") // an older station leaves an unknown type out
	if _, err := writeops.DeployTemplate.Plan(tplClient(types), deployArgs(tplFile(t, nil, nil))); err == nil ||
		!strings.Contains(err.Error(), "type control:NumericWritable of Setpoint is unknown") {
		t.Errorf("type left out: err = %v", err)
	}
}

func TestADeployRefusesABadFileOrName(t *testing.T) {
	good := tplFile(t, nil, nil)
	noBog := writeZip(t, "X.ntpl", map[string]string{"template-manifest.xml": tplSource(t, "template-manifest.xml")})
	noManifest := writeZip(t, "Y.ntpl", map[string]string{"template.bog": tplSource(t, "template.bog")})
	for _, tc := range []struct {
		args *pyjson.Object
		want string
	}{
		{deployArgs(good, "name", "Existing"), "a child named 'Existing' already exists under " + lab},
		{deployArgs(good, "name", "a/b"), "name must be a plain slot name (letters, digits, underscore), got 'a/b'"},
		{deployArgs("/nonexistent/T.ntpl"), "template_file /nonexistent/T.ntpl: cannot be read (no such file or directory)"},
		{deployArgs(strings.TrimSuffix(good, ".ntpl") + ".napl"), "template_file " + strings.TrimSuffix(good, ".ntpl") +
			".napl: not a component template (.ntpl); an application template (.napl) is installed with Workbench"},
		{deployArgs(noBog), "template_file " + noBog + ": no template.bog entry: the template holds no component tree"},
		{deployArgs(noManifest), "template_file " + noManifest + ": no template-manifest.xml entry: not a template"},
	} {
		if _, err := writeops.DeployTemplate.Plan(tplClient(tplTypes()), tc.args); err == nil || err.Error() != tc.want {
			t.Errorf("%s: err = %v\nwant %q", js(t, tc.args), err, tc.want)
		}
	}
	args := without(deployArgs(good), "name") // the template component's own name
	p, err := writeops.DeployTemplate.Plan(tplClient(tplTypes()), args)
	if err != nil || !strings.Contains(js(t, p.Ops[0]), `"n": "FanCoil", "b"`) {
		t.Errorf("default name: err %v", err)
	}
}
