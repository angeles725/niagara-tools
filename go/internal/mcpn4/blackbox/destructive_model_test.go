package blackbox

// The fake station refuses what a live N4.14 station refuses, so the rollback tests above cannot
// pass on a body the live station would reject (port of the test_the_station_model_* tests of
// test_tools_destructive.TestRollback).

import (
	"strings"
	"testing"
)

// wantRefused fails unless the add of body under the Folder is refused with want in its error.
func wantRefused(t *testing.T, d *dRig, name string, body map[string]any, want string) {
	t.Helper()
	_, err := d.sync(map[string]any{"nm": "a", "h": "3", "n": name, "b": body})
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("add of %v: err = %v, want %q", body, err, want)
	}
}

func TestTheStationModelRejectsNullTypesAndStatusFacets(t *testing.T) {
	d := destructiveRig(t, nil)
	for _, body := range []map[string]any{
		{"nm": "p", "t": "baja:Folder", "s": []any{map[string]any{"nm": "p", "t": nil, "n": "v"}}},
		{"nm": "p", "t": "baja:Folder", "s": []any{map[string]any{"nm": "p", "t": "baja:StatusNumeric", "n": "x",
			"s": []any{map[string]any{"nm": "p", "n": "status", "v": "0;activeLevel=e:17"}}}}},
	} {
		wantRefused(t, d, "Bad", body, "Unable to process request")
	}
}

// Regression for the live failure of 2026-10-01 (N4.14 station LLM).
func TestTheStationModelRejectsAnAddThatNestsComponents(t *testing.T) {
	d := destructiveRig(t, nil)
	wantRefused(t, d, "Grp", map[string]any{"nm": "p", "t": "baja:Folder", "s": []any{
		map[string]any{"nm": "p", "n": "Src", "t": "kitControl:NumericConst"}}}, "Unable to process request")
	plain := map[string]any{"nm": "p", "t": "baja:Folder", "s": []any{wsAnnotation(1, 2, 3)}}
	if _, err := d.sync(map[string]any{"nm": "a", "h": "3", "n": "Ok", "b": plain}); err != nil { // plain slots are fine
		t.Errorf("plain add: %v", err)
	}
}

func TestTheStationModelRejectsDoubleSlashesLikeTheLiveStation(t *testing.T) {
	d := destructiveRig(t, nil)
	if err := d.loadSlots("station:|slot://Folder"); err == nil || !strings.Contains(err.Error(), "Illegal double slashes") {
		t.Errorf("err = %v", err)
	}
}

func TestTheStationModelGivesWritablesAFrozenProxyExtAndRejectsAnother(t *testing.T) {
	d := destructiveRig(t, nil)
	nn := d.writableGroup()
	sp := d.node(nn, "Sp")
	if ext := d.node(nn, "Sp", "proxyExt"); ext == nil || ext.Type != "control:NullProxyExt" {
		t.Fatalf("Sp proxyExt = %v", ext)
	}
	_, err := d.sync(map[string]any{"nm": "a", "h": sp.Handle, "n": "proxyExt", "b": map[string]any{"nm": "p", "t": "control:NullProxyExt", "s": []any{}}})
	want := `Illegal child "control:NullProxyExt" for parent "control:NumericWritable".`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("err = %v", err)
	}
}
