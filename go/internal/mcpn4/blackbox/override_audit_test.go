package blackbox

// n4_override_audit (retro R8 delta N7): the writables that hold an override, with the level
// read from the out text, the expiry (none = permanent) and the schedule feeding in16.

import (
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// The lab's out text holds the active level: "@ 16" is the base level (no override), "@ def" the
// fallback, "@ 8" a manual override, "@ 1" an emergency override.
const (
	oaHeader  = "Slot Path,Out,Override Expiration\n"
	oaNumeric = oaHeader +
		"slot:/Folder/Fan,75.0 {ok} @ 8,null\n" +
		"slot:/Folder/Pump,1.0 {ok} @ 1,null\n" +
		"slot:/Folder/Valve,20.0 {ok} @ 8,24-Nov-26 6:00 PM CST\n" +
		"slot:/Folder/Idle,\"0,0 {disabled,stale} @ 16\",null\n" +
		"slot:/Folder/Fb,5.0 {ok} @ def,null\n" +
		"slot:/Folder/Auto,50.0 {ok} @ 10,null\n"
	oaBoolean = oaHeader + "slot:/Folder/Lamp,true {ok} @ 8,null\n"
	oaNoRows  = oaHeader
)

// oaRig is a servlet rig (Lamp is boolean) with Fan linked from a schedule at in16, Pump linked from a plain
// numeric at in16 and Valve unlinked; routes answer the four writable types and the schedule list.
func oaRig(t *testing.T, numeric, boolean string, extra ...boxstation.OrdRoute) (*Rig, *boxstation.Servlets, Component) {
	t.Helper()
	r, sv := servletRig(t, nil)
	occ := r.Add("3", "Occ", "schedule:BooleanSchedule")
	src := r.Add("3", "Calc", "kitControl:NumericConst")
	fan, pump := r.Add("3", "Fan", "control:NumericWritable"), r.Add("3", "Pump", "control:NumericWritable")
	r.Add("3", "Valve", "control:NumericWritable")
	r.Add("3", "Lamp", "control:BooleanWritable")
	r.Link(occ, "out", fan, "in16")
	r.Link(src, "out", pump, "in16")
	routes := append([]boxstation.OrdRoute{}, extra...)
	routes = append(routes,
		boxstation.OrdRoute{Needle: "schedule:ControlSchedule", Body: []byte("Slot Path,Handle\nslot:/Folder/Occ," + occ.Handle + "\n")},
		boxstation.OrdRoute{Needle: "control:NumericWritable", Body: []byte(numeric)},
		boxstation.OrdRoute{Needle: "control:BooleanWritable", Body: []byte(boolean)},
		boxstation.OrdRoute{Needle: "control:EnumWritable", Body: []byte(oaNoRows)},
		boxstation.OrdRoute{Needle: "control:StringWritable", Body: []byte(oaNoRows)})
	routeOrds(sv, routes...)
	return r, sv, occ
}

func oaEntry(t *testing.T, out map[string]any, name string) map[string]any {
	t.Helper()
	for _, e := range objects(out["overrides"]) {
		if e["writable"] == "station:|slot:/Folder/"+name {
			return e
		}
	}
	t.Fatalf("no override entry for %s in %s", name, jsonOf(t, out["overrides"]))
	return nil
}

func TestOverrideAuditListsTheOverriddenWritablesPermanentFirstWithLevelAndExpiry(t *testing.T) {
	r, _, occ := oaRig(t, oaNumeric, oaBoolean)
	out := r.Srv.OK("n4_override_audit", Args{})
	var order []any
	for _, e := range objects(out["overrides"]) {
		order = append(order, e["writable"])
	}
	want := []any{"station:|slot:/Folder/Pump", "station:|slot:/Folder/Fan", "station:|slot:/Folder/Lamp", "station:|slot:/Folder/Valve"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("overrides = %v, want %v (permanent first, then by level; @ 16, @ def and @ 10 are no override)", order, want)
	}
	fan := `{"writable":"station:|slot:/Folder/Fan","type":"control:NumericWritable","level":8,"kind":"manual",` +
		`"out":"75.0 {ok} @ 8","expires":null,"permanent":true,` +
		`"in16":{"linked":true,"source_ord":"h:` + occ.Handle + `","source_slot":"out","schedule":"station:|slot:/Folder/Occ"}}`
	if got := jsonOf(t, oaEntry(t, out, "Fan")); got != canon(t, fan) {
		t.Errorf("Fan =\n%s\nwant\n%s", got, fan)
	}
	valve := oaEntry(t, out, "Valve")
	if valve["expires"] != "24-Nov-26 6:00 PM CST" || valve["permanent"] != false || valve["kind"] != "manual" ||
		Dig(valve, "in16", "linked") != false || Dig(valve, "in16", "schedule") != nil {
		t.Errorf("Valve = %v", valve)
	}
	if pump := oaEntry(t, out, "Pump"); pump["level"] != 1.0 || pump["kind"] != "emergency" {
		t.Errorf("Pump = %v", pump)
	}
	if lamp := oaEntry(t, out, "Lamp"); lamp["type"] != "control:BooleanWritable" {
		t.Errorf("Lamp = %v", lamp)
	}
	counts := jsonOf(t, out["counts"])
	wantCounts := `{"writables":7,"overridden":4,"permanent":3,"timed":1,"unreadable":0,` +
		`"by_level":{"1":1,"8":3,"10":1,"16":1,"def":1}}`
	if counts != canon(t, wantCounts) {
		t.Errorf("counts =\n%s\nwant\n%s", counts, wantCounts)
	}
	if out["truncated"] != false || !reflect.DeepEqual(out["levels"], []any{1.0, 8.0}) {
		t.Errorf("truncated %v levels %v", out["truncated"], out["levels"])
	}
}

func TestOverrideAuditNamesAnIn16SourceThatIsNotASchedule(t *testing.T) {
	r, _, _ := oaRig(t, oaNumeric, oaBoolean)
	pump := oaEntry(t, r.Srv.OK("n4_override_audit", Args{}), "Pump")
	in16, _ := pump["in16"].(map[string]any)
	if in16["linked"] != true || in16["schedule"] != nil || !strings.HasPrefix(in16["source_ord"].(string), "h:") {
		t.Errorf("Pump in16 = %v", in16)
	}
}

func TestOverrideAuditLevelsWidenTheListToALinkDrivenInput(t *testing.T) {
	r, _, _ := oaRig(t, oaNumeric, oaBoolean)
	out := r.Srv.OK("n4_override_audit", Args{"levels": []any{10}})
	e := oaEntry(t, out, "Auto")
	if e["level"] != 10.0 || e["kind"] != "input" || len(objects(out["overrides"])) != 1 {
		t.Errorf("overrides = %s", jsonOf(t, out["overrides"]))
	}
}

func TestOverrideAuditWithNoOverrideProvesNoPermanentOne(t *testing.T) {
	r, _, _ := oaRig(t, oaHeader+"slot:/Folder/Idle,\"0,0 {disabled,stale} @ 16\",null\n", oaNoRows)
	out := r.Srv.OK("n4_override_audit", Args{})
	if len(objects(out["overrides"])) != 0 || Dig(out, "counts", "permanent") != 0.0 || Dig(out, "counts", "writables") != 1.0 {
		t.Errorf("out = %v", out)
	}
}

func TestOverrideAuditKeepsReadingWhenOneTypeIsRefused(t *testing.T) {
	r, sv, _ := oaRig(t, oaNumeric, oaBoolean)
	routeOrds(sv, boxstation.OrdRoute{Needle: "control:NumericWritable", Body: []byte(oaNumeric)})
	out := r.Srv.OK("n4_override_audit", Args{})
	unreadable := objects(out["unreadable"])
	if len(unreadable) != 3 || Dig(out, "counts", "unreadable") != 3.0 || len(objects(out["overrides"])) != 3 {
		t.Errorf("unreadable %v, overrides %d", out["unreadable"], len(objects(out["overrides"])))
	}
	if unreadable[0]["type"] != "control:BooleanWritable" || unreadable[0]["error"] == nil {
		t.Errorf("unreadable[0] = %v", unreadable[0])
	}
}

func TestOverrideAuditSaysSoWhenTheScheduleHandlesCannotBeRead(t *testing.T) {
	r, sv, occ := oaRig(t, oaNumeric, oaBoolean)
	routeOrds(sv, boxstation.OrdRoute{Needle: "control:NumericWritable", Body: []byte(oaNumeric)})
	out := r.Srv.OK("n4_override_audit", Args{"levels": []any{8}})
	fan := oaEntry(t, out, "Fan")
	notes, _ := out["notes"].([]any)
	if Dig(fan, "in16", "schedule") != nil || Dig(fan, "in16", "source_ord") != "h:"+occ.Handle || len(notes) == 0 ||
		!strings.Contains(notes[0].(string), "schedule handles") {
		t.Errorf("Fan in16 = %v, notes %v", fan["in16"], notes)
	}
}

func TestOverrideAuditRefusesALevelOutsideTheInputs(t *testing.T) {
	r, _, _ := oaRig(t, oaNumeric, oaBoolean)
	for _, levels := range []any{[]any{16}, []any{0}, []any{}} {
		if text := r.Srv.Err("n4_override_audit", Args{"levels": levels}); !strings.Contains(text, "levels") {
			t.Errorf("levels %v: refusal = %s", levels, text)
		}
	}
}

func TestOverrideAuditReportsAnOverrideWhoseWritableCannotBeLoaded(t *testing.T) {
	r, _, _ := oaRig(t, oaHeader+"slot:/Folder/Ghost,3.0 {ok} @ 8,null\n", oaNoRows)
	e := oaEntry(t, r.Srv.OK("n4_override_audit", Args{}), "Ghost")
	if e["permanent"] != true || Dig(e, "in16", "linked") != nil || Dig(e, "in16", "error") == nil {
		t.Errorf("Ghost = %v", e)
	}
}
