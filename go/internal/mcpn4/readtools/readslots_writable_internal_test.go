package readtools

// Retro 2026-11-02 row 13: a BooleanWritable's in1..in16, facets and minActiveTime are frozen
// properties the load omits at their default; n4_read_slots lists each from the type contract
// (default: true), so "are the facets empty?" and "is minActiveTime 0?" can be answered, and a
// slots filter answers each named one.

import (
	"fmt"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// bwContract is the contract of control:BooleanWritable: in1..in16, facets and minActiveTime.
func bwContract() *pyjson.Object {
	props := []any{}
	for i := 1; i <= 16; i++ {
		props = append(props, obj("st", "p", "n", fmt.Sprintf("in%d", i),
			"v", obj("nm", "p", "t", "baja:StatusBoolean", "d", "false {null}", "s", []any{
				obj("nm", "p", "n", "value", "t", "baja:Boolean", "v", "false"),
				obj("nm", "p", "n", "status", "t", "baja:Status", "v", "40")})))
	}
	props = append(props,
		obj("st", "p", "n", "facets", "v", obj("nm", "p", "t", "baja:Facets", "v", "", "d", "")),
		obj("st", "p", "n", "minActiveTime", "v", obj("nm", "p", "t", "baja:RelTime", "v", "0", "d", "0 s")))
	return obj("control:BooleanWritable", obj("p", "baja:Component", "c", props), "baja:Component", obj())
}

func bwList(t *testing.T, loaded ...any) []any {
	t.Helper()
	f := &ctfLoader{answer: ctfAnswer(bwContract())}
	nodes := obj("", obj("t", "control:BooleanWritable"))
	for i := 0; i < len(loaded); i += 2 {
		nodes.Set(loaded[i].(string), loaded[i+1])
	}
	var gone undescribed
	d := fillDefaults(f, f, &gone, nodes, []string{""}, func(s string) string { return s })
	list, err := listSlots(nodes, d, false)
	if err != nil {
		t.Fatal(err)
	}
	return list
}

func bwSlots(t *testing.T, loaded ...any) map[string]*pyjson.Object {
	t.Helper()
	list := bwList(t, loaded...)
	out := map[string]*pyjson.Object{}
	for _, e := range list {
		name, _ := e.(*pyjson.Object).Get("name")
		out[name.(string)] = e.(*pyjson.Object)
	}
	return out
}

func TestAFreshBooleanWritableListsInOneToSixteenFacetsAndMinActiveTimeAtTheirDefault(t *testing.T) {
	got := bwSlots(t)
	if len(got) != 18 {
		t.Errorf("%d slots, want in1..in16, facets and minActiveTime", len(got))
	}
	for i := 1; i <= 16; i++ {
		e := got[fmt.Sprintf("in%d", i)]
		if e == nil {
			t.Fatalf("in%d is not listed", i)
		}
		for k, v := range map[string]any{"type": "baja:StatusBoolean", "default": true, "status": "40", "status_null": true} {
			if g, _ := e.Get(k); g != v {
				t.Errorf("in%d %s = %v, want %v", i, k, g, v)
			}
		}
	}
	for name, want := range map[string][2]string{"facets": {"baja:Facets", ""}, "minActiveTime": {"baja:RelTime", "0"}} {
		e := got[name]
		if e == nil {
			t.Fatalf("%s is not listed", name)
		}
		if typ, _ := e.Get("type"); typ != want[0] {
			t.Errorf("%s type = %v", name, typ)
		}
		if v, _ := e.Get("value"); v != want[1] {
			t.Errorf("%s value = %v, want %q", name, v, want[1])
		}
		if d, _ := e.Get("default"); d != true {
			t.Errorf("%s default = %v", name, d)
		}
	}
}

func TestALoadedFacetsIsNotListedASecondTimeAsADefault(t *testing.T) {
	got := bwSlots(t, "facets", obj("t", "baja:Facets", "v", "precision=i:1"))
	if d, has := got["facets"].Get("default"); has {
		t.Errorf("a loaded facets slot is marked default = %v", d)
	}
	if v, _ := got["facets"].Get("value"); v != "precision=i:1" {
		t.Errorf("facets value = %v", v)
	}
}

// A slots filter answers each named slot, a default-valued one included, and names the rest.
func TestASlotsFilterAnswersDefaultValuedSlotsToo(t *testing.T) {
	kept, notFound := keepSlots(bwList(t), []string{"facets", "minActiveTime", "in16", "nope"})
	var names []string
	for _, e := range kept {
		n, _ := e.(*pyjson.Object).Get("name")
		names = append(names, n.(string))
	}
	if fmt.Sprint(names) != "[in16 facets minActiveTime]" || fmt.Sprint(notFound) != "[nope]" {
		t.Errorf("kept %v, not found %v", names, notFound)
	}
}
