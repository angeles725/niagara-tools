package readtools_test

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Field retro 2026-11-15 (alarm tools, row 7): `slots: ["alarmInhibit"]` on an alarm extension
// must answer that slot alone, not the whole component.

func slotNames(t *testing.T, out *pyjson.Object) []string {
	t.Helper()
	list, _ := out.Get("slots")
	var names []string
	for _, e := range list.([]any) {
		n, _ := e.(*pyjson.Object).Get("name")
		names = append(names, n.(string))
	}
	return names
}

func TestASlotsListReturnsOnlyThoseSlotsInStationOrder(t *testing.T) {
	s := serve(t, ord, rfxStatusTree)
	text, isErr, out := s.Call(t, "n4_read_slots", "ord", ord, "slots", []any{"nulled", "faulted"})
	if isErr {
		t.Fatal(text)
	}
	if got := slotNames(t, out); len(got) != 2 || got[0] != "faulted" || got[1] != "nulled" {
		t.Errorf("slots = %v, want [faulted nulled] in station order", got)
	}
	list, _ := out.Get("slots")
	want(t, list.([]any)[0].(*pyjson.Object), "status_ok", false, "value_display", "{fault}") // each entry is complete
	if v, ok := out.Get("slots_not_found"); ok {
		t.Errorf("slots_not_found = %v with every slot present", v)
	}
}

func TestASlotsListNamesTheSlotsTheComponentDoesNotHave(t *testing.T) {
	s := serve(t, ord, rfxStatusTree)
	_, isErr, out := s.Call(t, "n4_read_slots", "ord", ord, "slots", []any{"healthy", "nope"})
	if isErr {
		t.Fatal("refused")
	}
	if got := slotNames(t, out); len(got) != 1 || got[0] != "healthy" {
		t.Errorf("slots = %v", got)
	}
	if v, _ := out.Get("slots_not_found"); pyjson.Repr(v) != "['nope']" {
		t.Errorf("slots_not_found = %s", pyjson.Repr(v))
	}
}

func TestWithoutASlotsListEverySlotIsStillRead(t *testing.T) {
	s := serve(t, ord, rfxStatusTree)
	_, _, out := s.Call(t, "n4_read_slots", "ord", ord)
	if got := slotNames(t, out); len(got) != 6 {
		t.Errorf("slots = %v, want all 6", got)
	}
	if _, ok := out.Get("slots_not_found"); ok {
		t.Error("slots_not_found without a filter")
	}
}
