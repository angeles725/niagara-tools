package boxstation

import (
	"reflect"
	"testing"
)

// flagsStation is a station whose Folder holds note (a String "x") and zero (a Double at its
// type default, which a load omits).
func flagsStation() *Station {
	st := New("u", "p", "S")
	folder := st.ByHandle("3")
	folder.Children = append(folder.Children, &Node{Name: "note", Type: "baja:String", Value: "x"},
		&Node{Name: "zero", Type: "baja:Double", Value: "0.0"})
	return st
}

// loaded are the children of the Folder as a load to depth 1 serializes them.
func loaded(st *Station) any { return serialize(st.ByHandle("3"), 1, true)["s"] }

// The set-flags op (javax.baja.sync.SetFlagsOp, id 'f') replaces the whole flag set of a slot of
// the component; the station keeps the letters it knows in its own order, answers the op id, and
// a load carries "f" only for a slot that has flags.
func TestTheSetFlagsOpReplacesTheFlagsOfASlotAndALoadCarriesThem(t *testing.T) {
	st := flagsStation()
	if got := loaded(st); !reflect.DeepEqual(got, []any{map[string]any{"nm": "p", "t": "baja:String", "n": "note", "v": "x"}}) {
		t.Fatalf("before: %v", got)
	}
	out, err := st.sync(map[string]any{"nm": "f", "h": "3", "n": "note", "f": "ozh"})
	if err != nil || !reflect.DeepEqual(out, []any{map[string]any{"id": "f"}}) {
		t.Fatalf("reply %v, err %v", out, err)
	}
	if got := st.ByHandle("3").Child("note").Flags; got != "ho" { // 'z' is ignored, as Flags.decodeFromString does
		t.Errorf("flags = %q", got)
	}
	want := []any{map[string]any{"nm": "p", "t": "baja:String", "n": "note", "v": "x", "f": "ho"}}
	if got := loaded(st); !reflect.DeepEqual(got, want) {
		t.Errorf("after: %v", got)
	}
	if _, err := st.sync(map[string]any{"nm": "f", "h": "3", "n": "note", "f": ""}); err != nil {
		t.Fatal(err)
	}
	if got := loaded(st); !reflect.DeepEqual(got, []any{map[string]any{"nm": "p", "t": "baja:String", "n": "note", "v": "x"}}) {
		t.Errorf("cleared: %v", got)
	}
}

// A slot at its type default is omitted from a load unless it has flags.
func TestASlotAtItsDefaultIsLoadedOnceItHasFlags(t *testing.T) {
	st := flagsStation()
	if _, err := st.sync(map[string]any{"nm": "f", "h": "3", "n": "zero", "f": "h"}); err != nil {
		t.Fatal(err)
	}
	kids, _ := loaded(st).([]any)
	if len(kids) != 2 || !reflect.DeepEqual(kids[1], map[string]any{"nm": "p", "t": "baja:Double", "n": "zero", "v": "0.0", "f": "h"}) {
		t.Errorf("loaded = %v", kids)
	}
}

func TestTheSetFlagsOpOnAMissingSlotOrWithoutFlagsIsAnError(t *testing.T) {
	st := flagsStation()
	if _, err := st.sync(map[string]any{"nm": "f", "h": "3", "n": "nope", "f": "h"}); err == nil {
		t.Error("a missing slot was accepted")
	}
	if _, err := st.sync(map[string]any{"nm": "f", "h": "3", "n": "note"}); err == nil {
		t.Error("an op without flags was accepted")
	}
	if got := st.ByHandle("3").Child("note").Flags; got != "" {
		t.Errorf("flags = %q", got)
	}
}
