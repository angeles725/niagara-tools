package portparity

import (
	"reflect"
	"testing"
)

// cascadeTrees: the reference ships a simulator file the port drops; simTick and simPanel live only there,
// shared() also lives in ui.js, which the port keeps but without shared().
func cascadeTrees(t *testing.T) (Tree, Tree) {
	t.Helper()
	ref, port := t.TempDir(), t.TempDir()
	write(t, ref, map[string]string{
		"js/ui.js":  "function render() {}\nfunction shared() {}\n",
		"js/sim.js": "function simTick() {}\nfunction shared() {}\nconst p = '<div id=\"simPanel\">';\n",
	})
	write(t, port, map[string]string{"js/ui.js": "function render() {}\n"})
	r, err := Scan(ref)
	if err != nil {
		t.Fatal(err)
	}
	p, err := Scan(port)
	if err != nil {
		t.Fatal(err)
	}
	return r, p
}

func TestFileWaiverCascade(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		waivers string
		want    []string
		fail    bool
	}{
		{"a file waiver waives what only that file defines (INFO); a name also defined elsewhere still FAILs",
			"file sim.js -- simulator not shipped [decision: W7]\n", []string{
				"INFO js/sim.js:3 id simPanel waived via file sim.js (defined only in that reference file)",
				"FAIL js/sim.js:2 missing function shared (in the reference, not in the port, no waiver)",
				"INFO js/sim.js:1 function simTick waived via file sim.js (defined only in that reference file)",
			}, true},
		{"with the shared name waived too, the module is clean and nothing is stale",
			"file sim.js -- simulator not shipped [decision: W7]\nfunction shared -- folded [decision: W7]\n", []string{
				"INFO js/sim.js:3 id simPanel waived via file sim.js (defined only in that reference file)",
				"INFO js/sim.js:1 function simTick waived via file sim.js (defined only in that reference file)",
			}, false},
		{"removing the file waiver FAILs the file and everything it defined",
			"function shared -- folded [decision: W7]\n", []string{
				"FAIL js/sim.js:1 missing file sim.js (in the reference, not in the port, no waiver)",
				"FAIL js/sim.js:3 missing id simPanel (in the reference, not in the port, no waiver)",
				"FAIL js/sim.js:1 missing function simTick (in the reference, not in the port, no waiver)",
			}, true},
		{"a file waiver without decision: FAILs and cascades nothing",
			"file sim.js -- simulator not shipped\nfunction shared -- folded [decision: W7]\n", []string{
				"FAIL js/sim.js:3 missing id simPanel (in the reference, not in the port, no waiver)",
				"FAIL js/sim.js:1 missing function simTick (in the reference, not in the port, no waiver)",
				"FAIL .port-parity:1 waiver file sim.js has no [decision: <ref>] (dropping a reference feature needs a recorded user decision)",
			}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r, p := cascadeTrees(t)
			res := Check(r, p, ParseWaivers(tt.waivers), ".port-parity")
			if got := rows(res); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("rows =\n%q\nwant\n%q", got, tt.want)
			}
			if res.Failed() != tt.fail {
				t.Errorf("Failed() = %v, want %v", res.Failed(), tt.fail)
			}
		})
	}
}

func TestFileWaiverCascadeCounts(t *testing.T) {
	t.Parallel()
	r, p := cascadeTrees(t)
	res := Check(r, p, ParseWaivers("file sim.js -- x [decision: W7]\nfunction shared -- y [decision: W7]\n"), ".port-parity")
	want := Counts{RefFiles: 2, RefIDs: 1, RefFuncs: 3, Missing: 4, Waived: 4, Cascaded: 2, Info: 2}
	if res.Counts != want {
		t.Errorf("counts = %+v, want %+v", res.Counts, want)
	}
}
