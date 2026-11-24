package blackbox

// n4_list_templates (retro R13 ΔM1): one read-only call lists the deployed template instances
// below a base: the station's TemplateInfo row of each (vendor, version, counts, default
// passwords, signatures) and each exposed config of its templateConfig with the slot it is bound
// to and its current value, on a fake station.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

const (
	ltConfigList = "Slot Path\nslot:/Folder/Ahu1/templateConfig\nslot:/Folder/Ahu2/templateConfig\n"
	ltInfoHeader = "Location Ord,Template Name,Vendor,Version,Inputs,Outputs,Relations,Default Passwords," +
		"Ntpl Signature,Local Signature\n"
)

// ltInstance adds a template root named name under the Folder with a templateConfig holding
// the configs (slot name to value text, all baja:Double) and one ConfigBinding per entry
// (bound to <name>Target/in<n>); it returns the templateConfig's handle, which is what the
// TemplateInfo row's locationOrd names (BTemplateInfo.make).
func ltInstance(r *Rig, name string, configs [][2]string) string {
	root := r.Add("3", name, "baja:Component")
	cfg := r.Add(root.Handle, "templateConfig", "template:TemplateConfig")
	for i, c := range configs {
		r.Set(cfg.Handle, c[0], map[string]any{"nm": "p", "n": c[0], "t": "baja:Double", "v": c[1]})
		b := r.Add(cfg.Handle, "binding"+string(rune('A'+i)), "template:ConfigBinding")
		for _, kv := range [][2]string{{"sourceSlot", c[0]}, {"targetOrd", "slot:/Folder/" + name + "Target"},
			{"targetSlot", "in" + string(rune('1'+i))}} {
			r.Set(b.Handle, kv[0], map[string]any{"nm": "p", "n": kv[0], "t": "baja:String", "v": kv[1]})
		}
	}
	return cfg.Handle
}

// ltRig is a servlet rig with two instances: Ahu1 (two configs) and Ahu2 (one), answering the
// config list and the TemplateInfo rows of both.
func ltRig(t *testing.T) (*Rig, *boxstation.Servlets) {
	t.Helper()
	r, sv := servletRig(t, nil)
	h1 := ltInstance(r, "Ahu1", [][2]string{{"setpoint", "21.5"}, {"deadband", "1.0"}})
	h2 := ltInstance(r, "Ahu2", [][2]string{{"setpoint", "19.0"}})
	info := ltInfoHeader +
		"station:|h:" + h1 + ",Ahu,Acme,1.2,4,2,1,2,abc123,abc123\n" +
		"station:|h:" + h2 + ",Ahu,Acme,1.1,4,2,0,0,abc123,ffff00\n"
	routeOrds(sv, boxstation.OrdRoute{Needle: "template:TemplateInfo", Body: []byte(info)},
		boxstation.OrdRoute{Needle: "template:TemplateConfig", Body: []byte(ltConfigList)})
	return r, sv
}

// ltEntry is the instances entry whose root ends in name.
func ltEntry(t *testing.T, out map[string]any, name string) map[string]any {
	t.Helper()
	for _, e := range objects(out["instances"]) {
		if e["root"] == "station:|slot:/Folder/"+name {
			return e
		}
	}
	t.Fatalf("no instance %s in %s", name, jsonOf(t, out["instances"]))
	return nil
}

func TestListTemplatesReportsAnInstanceWithItsInfoAndBothConfigs(t *testing.T) {
	r, sv := ltRig(t)
	out := r.Srv.OK("n4_list_templates", Args{})
	e := ltEntry(t, out, "Ahu1")
	got := jsonOf(t, map[string]any{"name": e["template_name"], "vendor": e["vendor"], "version": e["version"],
		"inputs": e["inputs"], "outputs": e["outputs"], "relations": e["relations"],
		"default_passwords": e["default_passwords"], "signature": e["signature"]})
	want := `{"name":"Ahu","vendor":"Acme","version":"1.2","inputs":4,"outputs":2,"relations":1,` +
		`"default_passwords":2,"signature":{"template":"abc123","local":"abc123","modified":false}}`
	if got != canon(t, want) {
		t.Errorf("Ahu1 =\n%s\nwant\n%s", got, want)
	}
	wantCfg := `[{"name":"setpoint","value":"21.5","type":"baja:Double",` +
		`"bound_to":{"ord":"slot:/Folder/Ahu1Target","slot":"in1"}},` +
		`{"name":"deadband","value":"1.0","type":"baja:Double",` +
		`"bound_to":{"ord":"slot:/Folder/Ahu1Target","slot":"in2"}}]`
	if got := jsonOf(t, e["configs"]); got != canon(t, wantCfg) {
		t.Errorf("configs =\n%s\nwant\n%s", got, wantCfg)
	}
	if e["config_ord"] != "station:|slot:/Folder/Ahu1/templateConfig" || e["has_default_passwords"] != true {
		t.Errorf("config_ord %v has_default_passwords %v", e["config_ord"], e["has_default_passwords"])
	}
	if got := jsonOf(t, out["counts"]); got != canon(t, `{"instances":2,"configs":3,"with_default_passwords":1,`+
		`"signature_modified":1,"unreadable":0}`) {
		t.Errorf("counts = %s", got)
	}
	if out["truncated"] != false || out["base"] != "station:|slot:/" {
		t.Errorf("truncated %v base %v", out["truncated"], out["base"])
	}
	queries := 0
	for _, q := range ordRequests(sv) {
		if strings.Contains(q, "bql:") {
			queries++
		}
	}
	if queries != 2 {
		t.Errorf("%d BQL reads, want 2 (the instances, the TemplateInfo rows): %v", queries, ordRequests(sv))
	}
}

func TestListTemplatesFlagsAModifiedSignatureAndLeavesNoPasswordFlagOff(t *testing.T) {
	r, _ := ltRig(t)
	e := ltEntry(t, r.Srv.OK("n4_list_templates", Args{}), "Ahu2")
	sig := e["signature"].(map[string]any)
	if sig["modified"] != true || e["has_default_passwords"] != false || e["default_passwords"] != float64(0) {
		t.Errorf("Ahu2 signature %v passwords %v/%v", sig, e["has_default_passwords"], e["default_passwords"])
	}
}

func TestListTemplatesOnAStationWithoutInstancesIsEmptyNotAnError(t *testing.T) {
	r, sv := servletRig(t, nil)
	routeOrds(sv, boxstation.OrdRoute{Needle: "template:TemplateConfig", Body: []byte("Slot Path\n")})
	out := r.Srv.OK("n4_list_templates", Args{})
	if got := jsonOf(t, out["instances"]); got != "[]" {
		t.Errorf("instances = %s, want []", got)
	}
	if got := jsonOf(t, out["counts"]); got != canon(t, `{"instances":0,"configs":0,"with_default_passwords":0,`+
		`"signature_modified":0,"unreadable":0}`) || out["truncated"] != false {
		t.Errorf("counts %s truncated %v", got, out["truncated"])
	}
	for _, q := range ordRequests(sv) {
		if strings.Contains(q, "TemplateInfo") {
			t.Errorf("TemplateInfo read for zero instances: %s", q)
		}
	}
}

func TestListTemplatesNamesAStationThatRefusesTheInstanceQuery(t *testing.T) {
	r, sv := servletRig(t, nil)
	routeOrds(sv) // every /ord/ GET is a 404
	msg := r.Srv.Err("n4_list_templates", Args{})
	if !strings.Contains(msg, "template:TemplateConfig") || !strings.Contains(msg, "no instance list") {
		t.Errorf("error = %s", msg)
	}
}

func TestListTemplatesWithoutTemplateInfoLeavesItsFieldsNullWithANote(t *testing.T) {
	r, sv := ltRig(t)
	routeOrds(sv, boxstation.OrdRoute{Needle: "template:TemplateConfig", Body: []byte(ltConfigList)})
	out := r.Srv.OK("n4_list_templates", Args{})
	e := ltEntry(t, out, "Ahu1")
	if e["vendor"] != nil || e["version"] != nil || e["inputs"] != nil || e["signature"] != nil ||
		e["has_default_passwords"] != nil || len(objects(e["configs"])) != 2 {
		t.Errorf("Ahu1 without info = %s", jsonOf(t, e))
	}
	if !strings.Contains(jsonOf(t, out["notes"]), "TemplateInfo") {
		t.Errorf("notes = %s", jsonOf(t, out["notes"]))
	}
}

func TestListTemplatesLeavesAConfigTheStationDoesNotAnswerNullWithANote(t *testing.T) {
	r, _ := ltRig(t)
	r.Set(handleAt(r, "Ahu2", "templateConfig/bindingA"), "sourceSlot",
		map[string]any{"nm": "p", "n": "sourceSlot", "t": "baja:String", "v": "gone"})
	c := ltEntry(t, r.Srv.OK("n4_list_templates", Args{}), "Ahu2")["configs"].([]any)[0].(map[string]any)
	if c["name"] != "gone" || c["value"] != nil || !strings.Contains(c["note"].(string), "not in the station's answer") {
		t.Errorf("config = %s", jsonOf(t, c))
	}
}

func TestListTemplatesCapsTheInstancesLoaded(t *testing.T) {
	r, _ := ltRig(t)
	out := r.Srv.OK("n4_list_templates", Args{"max_instances": 1})
	if len(objects(out["instances"])) != 1 || out["truncated"] != true ||
		!strings.Contains(jsonOf(t, out["notes"]), "max_instances") {
		t.Errorf("instances %d truncated %v notes %s", len(objects(out["instances"])), out["truncated"],
			jsonOf(t, out["notes"]))
	}
}

func TestListTemplatesListsAnUnloadableInstanceAsUnreadable(t *testing.T) {
	r, sv := ltRig(t)
	list := ltConfigList + "slot:/Folder/Gone/templateConfig\n"
	routeOrds(sv, boxstation.OrdRoute{Needle: "template:TemplateConfig", Body: []byte(list)})
	out := r.Srv.OK("n4_list_templates", Args{})
	un := objects(out["unreadable"])
	if len(un) != 1 || un[0]["root"] != "station:|slot:/Folder/Gone" || un[0]["error"] == "" ||
		len(objects(out["instances"])) != 2 {
		t.Errorf("unreadable %s", jsonOf(t, out["unreadable"]))
	}
}

func TestListTemplatesRefusesABaseThatIsNotASlotORD(t *testing.T) {
	r, _ := ltRig(t)
	if msg := r.Srv.Err("n4_list_templates", Args{"base": "h:5"}); !strings.Contains(msg, "base must be") {
		t.Errorf("error = %s", msg)
	}
}
