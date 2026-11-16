package blackbox

// n4_read_slots on a struct slot path (live retro row M3): a BOX load answers with the nearest
// component, so a path such as .../OutOfRangeAlarmExt/toOffnormalTimes read the extension's own
// slots. The reply now lists the struct's fields and decodes alarm:AlarmTimestamps.

import (
	"reflect"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

const offnormalAt = "2026-11-16T13:24:12.641-06:00"

// absTime is a baja:AbsTime value node; the null time (epoch 0) as the lab station encodes it.
func absTime(name, v string) *boxstation.Node {
	return &boxstation.Node{Name: name, Type: "baja:AbsTime", Value: v}
}

const nullTime = "1969-12-31T18:00:00.000-06:00"

// stsContracts describe alarm:AlarmSourceExt and alarm:AlarmTimestamps as the lab 4.15 station's
// registry does (trimmed to the properties these tests read).
func stsContracts() map[string]any {
	field := func(name, typ, v string) map[string]any {
		return dfxProp(name, map[string]any{"nm": "p", "t": typ, "d": v, "v": v})
	}
	times := map[string]any{"nm": "p", "t": "alarm:AlarmTimestamps", "d": "Alarm Timestamps", "s": []any{}}
	return map[string]any{
		"alarm:AlarmTimestamps": map[string]any{"p": "baja:Struct", "c": []any{
			field("alarmTime", "baja:AbsTime", nullTime), field("ackTime", "baja:AbsTime", nullTime),
			field("normalTime", "baja:AbsTime", nullTime), field("count", "baja:Integer", "0")}},
		"alarm:AlarmSourceExt": map[string]any{"p": "baja:Component", "c": []any{
			field("alarmState", "alarm:AlarmState", "normal"),
			dfxProp("toOffnormalTimes", times), dfxProp("toFaultTimes", times)}},
		"baja:Struct": map[string]any{}, "baja:Component": map[string]any{},
	}
}

// stsRig is a station whose Folder holds Ext, an alarm extension in high limit whose
// toOffnormalTimes has an alarm time and count 1 (ackTime and normalTime at their default, so
// the load omits them) and whose toFaultTimes is at its default (omitted whole).
func stsRig(t *testing.T) (*Rig, string) {
	r := stationRig(t)
	r.Station.SetRegistry(stsContracts())
	c := r.Add("3", "Ext", "alarm:AlarmSourceExt")
	r.Station.Do(func(st *boxstation.Station) {
		ext := st.Root().Child("Folder").Child(c.Name)
		ext.Children = append(ext.Children,
			&boxstation.Node{Name: "alarmState", Type: "alarm:AlarmState", Value: "highLimit"},
			&boxstation.Node{Name: "toOffnormalTimes", Type: "alarm:AlarmTimestamps", Children: []*boxstation.Node{
				absTime("alarmTime", offnormalAt), {Name: "count", Type: "baja:Integer", Value: "1"}}},
			&boxstation.Node{Name: "toFaultTimes", Type: "alarm:AlarmTimestamps", AtDefault: true})
	})
	return r, Folder + "/" + c.Name
}

func slotsByName(out map[string]any) (names []string, by map[string]map[string]any) {
	by = map[string]map[string]any{}
	for _, s := range objects(out["slots"]) {
		names = append(names, s["name"].(string))
		by[s["name"].(string)] = s
	}
	return names, by
}

func TestReadSlotsOnAStructPathListsTheStructsOwnFields(t *testing.T) {
	r, ext := stsRig(t)
	out := r.Srv.OK("n4_read_slots", Args{"ord": ext + "/toOffnormalTimes"})
	names, by := slotsByName(out)
	wantStrings(t, "slots", names, []string{"alarmTime", "count", "ackTime", "normalTime"})
	if by["alarmTime"]["value"] != offnormalAt || by["alarmTime"]["default"] != nil {
		t.Errorf("alarmTime = %v", by["alarmTime"])
	}
	if by["ackTime"]["value"] != nullTime || by["ackTime"]["default"] != true {
		t.Errorf("ackTime = %v", by["ackTime"])
	}
	for k, want := range map[string]any{"ord": ext + "/toOffnormalTimes", "component_ord": ext,
		"property": "toOffnormalTimes", "property_type": "alarm:AlarmTimestamps"} {
		if out[k] != want {
			t.Errorf("%s = %v, want %v", k, out[k], want)
		}
	}
	want := map[string]any{"alarm_time": offnormalAt, "ack_time": nil, "normal_time": nil, "count": float64(1)}
	if got := out["alarm_timestamps"]; !reflect.DeepEqual(got, want) {
		t.Errorf("alarm_timestamps = %v\nwant %v", got, want)
	}
}

func TestReadSlotsOnAStructAtItsDefaultListsTheContractFields(t *testing.T) {
	r, ext := stsRig(t)
	out := r.Srv.OK("n4_read_slots", Args{"ord": ext + "/toFaultTimes"})
	names, by := slotsByName(out)
	wantStrings(t, "slots", names, []string{"alarmTime", "ackTime", "normalTime", "count"})
	for _, n := range names {
		if by[n]["default"] != true {
			t.Errorf("%s not marked default: %v", n, by[n])
		}
	}
	want := map[string]any{"alarm_time": nil, "ack_time": nil, "normal_time": nil, "count": float64(0)}
	if got := out["alarm_timestamps"]; !reflect.DeepEqual(got, want) {
		t.Errorf("alarm_timestamps = %v\nwant %v", got, want)
	}
}

func TestReadSlotsOnAStatusPropertyListsItsValueAndStatus(t *testing.T) {
	r := stationRig(t)
	c := r.AddWithOut("Calc", 5.5)
	out := r.Srv.OK("n4_read_slots", Args{"ord": Folder + "/" + c.Name + "/out"})
	names, by := slotsByName(out)
	wantStrings(t, "slots", names, []string{"value"})
	if by["value"]["value"] != "5.5" || out["property_type"] != "baja:StatusNumeric" || out["alarm_timestamps"] != nil {
		t.Errorf("out = %v", out)
	}
}

func TestReadSlotsOnASimplePropertyListsThePropertyItself(t *testing.T) {
	r := stationRig(t)
	c := r.Add("3", "Calc", "")
	r.Set(c.Handle, "label", map[string]any{"nm": "p", "t": "baja:String", "v": "hello"})
	out := r.Srv.OK("n4_read_slots", Args{"ord": Folder + "/" + c.Name + "/label"})
	want := []any{map[string]any{"name": "label", "type": "baja:String", "value": "hello", "status": nil}}
	if !reflect.DeepEqual(out["slots"], want) || out["component_ord"] != Folder+"/"+c.Name {
		t.Errorf("out = %v", out)
	}
}

func TestReadSlotsOnAComponentIsUnchanged(t *testing.T) {
	r, ext := stsRig(t)
	out := r.Srv.OK("n4_read_slots", Args{"ord": ext})
	names, _ := slotsByName(out)
	wantStrings(t, "slots", names, []string{"alarmState", "toOffnormalTimes", "toFaultTimes"})
	for _, k := range []string{"component_ord", "property", "property_type", "alarm_timestamps"} {
		if _, ok := out[k]; ok {
			t.Errorf("a component read carries %s: %v", k, out)
		}
	}
}

// A component with a struct slot named like the component itself is still read as the component:
// the extra load of the would-be parent answers another handle.
func TestAComponentWithAStructNamedLikeItselfIsReadAsTheComponent(t *testing.T) {
	r := stationRig(t)
	c := r.Add("3", "Same", "")
	r.Station.Do(func(st *boxstation.Station) {
		n := st.Root().Child("Folder").Child(c.Name)
		n.Children = append(n.Children, &boxstation.Node{Name: c.Name, Type: "alarm:AlarmTimestamps",
			Children: []*boxstation.Node{{Name: "count", Type: "baja:Integer", Value: "2"}}})
	})
	out := r.Srv.OK("n4_read_slots", Args{"ord": Folder + "/" + c.Name})
	names, _ := slotsByName(out)
	wantStrings(t, "slots", names, []string{c.Name})
	if _, ok := out["component_ord"]; ok {
		t.Errorf("read as a property: %v", out)
	}
}
