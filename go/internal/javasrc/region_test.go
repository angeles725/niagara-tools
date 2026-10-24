package javasrc

import "testing"

func TestMarker(t *testing.T) {
	raw := []string{
		"//region /*+ ------------ BEGIN BAJA AUTO GENERATED CODE ------------ +*/",
		"//endregion /*+ ------------ END BAJA AUTO GENERATED CODE -------------- +*/",
		"/*+ ------------ BEGIN BAJA AUTO GENERATED CODE ------------ +*/",
		`  String s = "BEGIN BAJA AUTO GENERATED CODE";`,
		`  String t = """`,
		"END BAJA AUTO GENERATED CODE",
		`  """;`,
		"  int x; // no marker here",
	}
	f := NewFile("A.java", raw)
	for i, want := range []struct{ begin, end bool }{
		{true, false}, {false, true}, {true, false}, {false, false},
		{false, false}, {false, false}, {false, false}, {false, false},
	} {
		if b, e := f.Marker(i, RegionBegin), f.Marker(i, RegionEnd); b != want.begin || e != want.end {
			t.Errorf("line %d %q: begin %v end %v, want %v %v", i+1, raw[i], b, e, want.begin, want.end)
		}
	}
}
