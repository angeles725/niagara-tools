package blackbox

// n4_read_slots (port of test_server.TestReadSlots).

import (
	"reflect"
	"testing"
)

// slots are the read_slots slots of the Folder child nn, by name.
func slots(r *Rig, nn string) map[string]map[string]any {
	out := map[string]map[string]any{}
	for _, s := range objects(r.Srv.OK("n4_read_slots", Args{"ord": Folder + "/" + nn})["slots"]) {
		out[s["name"].(string)] = s
	}
	return out
}

func TestStatusSlotIsRenderedWithValueStatusAndFlags(t *testing.T) {
	r := stationRig(t)
	c := r.AddWithOut("Calc", 5.5)
	want := map[string]any{"name": "out", "type": "baja:StatusNumeric", "value": "5.5", "status": "0",
		"status_ok": true, "status_null": false}
	if got := slots(r, c.Name)["out"]; !reflect.DeepEqual(got, want) {
		t.Errorf("out = %v", got)
	}
}

func TestOmittedValueTakesTheTypeDefault(t *testing.T) {
	r := stationRig(t)
	c := r.AddWithOut("Calc", 0) // the station omits value 0.0
	if got := slots(r, c.Name)["out"]["value"]; got != "0.0" {
		t.Errorf("value %v", got)
	}
}

func TestBooleanDefaultAndNullStatus(t *testing.T) {
	r := stationRig(t)
	c := r.Add("3", "Sw", "")
	r.Set(c.Handle, "out", StatusBoolean(false, "40"))
	out := slots(r, c.Name)["out"]
	if out["value"] != "false" || out["status"] != "40" || out["status_null"] != true || out["status_ok"] != false {
		t.Errorf("out = %v", out)
	}
}

func TestPlainSlotHasRawValueAndNoStatus(t *testing.T) {
	r := stationRig(t)
	c := r.Add("3", "Calc", "")
	r.Set(c.Handle, "label", map[string]any{"nm": "p", "t": "baja:String", "v": "hello"})
	want := map[string]any{"name": "label", "type": "baja:String", "value": "hello", "status": nil}
	if got := slots(r, c.Name)["label"]; !reflect.DeepEqual(got, want) {
		t.Errorf("label = %v", got)
	}
}

func TestUnsupportedStatusTypeFallsBackWithoutError(t *testing.T) {
	r := stationRig(t)
	c := r.Add("3", "Calc", "")
	r.Set(c.Handle, "txt", map[string]any{"nm": "p", "t": "baja:StatusString", "s": []any{
		map[string]any{"nm": "p", "n": "value", "t": "baja:String", "v": "abc"}}})
	if got := slots(r, c.Name)["txt"]; got["value"] != "abc" || got["status"] != "0" {
		t.Errorf("txt = %v", got)
	}
}

func TestOrdIsRequired(t *testing.T) {
	if code := stationRig(t).Srv.CallError("n4_read_slots", Args{}); code != -32602 {
		t.Errorf("code %d", code)
	}
}

func TestResponseNamesTheOrd(t *testing.T) {
	r := stationRig(t)
	ord := Folder + "/" + r.AddWithOut("Calc", 1).Name
	if got := r.Srv.OK("n4_read_slots", Args{"ord": ord})["ord"]; got != ord {
		t.Errorf("ord %v", got)
	}
}
