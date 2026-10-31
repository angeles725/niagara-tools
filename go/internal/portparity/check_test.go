package portparity

import (
	"reflect"
	"testing"
)

// fixture: the reference has one equipment view the port dropped, and a file the port renamed.
func fixture(t *testing.T) (Tree, Tree) {
	t.Helper()
	ref, port := t.TempDir(), t.TempDir()
	write(t, ref, map[string]string{
		"index.html":       `<div id="levels"></div><div id="equipmentView"></div>`,
		"ui.js":            "function renderLevels() {}\nfunction renderEquipment() {}\n",
		"update-banner.js": "function banner() {}\n",
	})
	write(t, port, map[string]string{
		"index.html": `<div id="levels"></div>`,
		"ui.js":      "function renderLevels() {}\n",
		"banner.js":  "function banner() {}\n",
	})
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

func rows(res Result) []string {
	var out []string
	for _, f := range res.Findings {
		out = append(out, f.Severity+" "+f.Site+" "+f.Message)
	}
	return out
}

const allWaived = "rename: update-banner.js -> banner.js\n" +
	"id equipmentView -- equipment view moved to its own page [decision: W5]\n" +
	"function renderEquipment -- same [decision: W5]\n"

func TestCheck(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		waivers string
		want    []string
		fail    bool
	}{
		{"every absence waived with a decision is clean", allWaived, nil, false},
		{"no waivers: every absence FAILs, sorted file, id, function", "", []string{
			"FAIL update-banner.js:1 missing file update-banner.js (in the reference, not in the port, no waiver)",
			"FAIL index.html:1 missing id equipmentView (in the reference, not in the port, no waiver)",
			"FAIL ui.js:2 missing function renderEquipment (in the reference, not in the port, no waiver)",
		}, true},
		{"removing one waiver line flips clean to FAIL", "rename: update-banner.js -> banner.js\n" +
			"id equipmentView -- moved [decision: W5]\n", []string{
			"FAIL ui.js:2 missing function renderEquipment (in the reference, not in the port, no waiver)",
		}, true},
		{"a waiver without decision: FAILs (dropping needs a recorded user decision)", "rename: update-banner.js -> banner.js\n" +
			"id equipmentView -- moved [decision: W5]\n" +
			"function renderEquipment -- not needed\n", []string{
			"FAIL .port-parity:3 waiver function renderEquipment has no [decision: <ref>] (dropping a reference feature needs a recorded user decision)",
		}, true},
		{"an empty [decision: ] is no decision", "rename: update-banner.js -> banner.js\n" +
			"id equipmentView -- moved [decision:  ]\nfunction renderEquipment -- moved [decision: W5]\n", []string{
			"FAIL .port-parity:2 waiver id equipmentView has no [decision: <ref>] (dropping a reference feature needs a recorded user decision)",
		}, true},
		{"a waiver matching nothing is a stale WARN, not a FAIL", allWaived + "function renderLevels -- gone [decision: W6]\nid nowhere -- x [decision: W6]\n", []string{
			"WARN .port-parity:4 stale waiver: function renderLevels matches no missing item (remove the line)",
			"WARN .port-parity:5 stale waiver: id nowhere matches no missing item (remove the line)",
		}, false},
		{"a rename to a file the port lacks FAILs", "rename: update-banner.js -> nope.js\n" +
			"id equipmentView -- moved [decision: W5]\nfunction renderEquipment -- moved [decision: W5]\n", []string{
			"FAIL .port-parity:1 rename target nope.js is not a file of the port",
		}, true},
		{"a malformed line FAILs", allWaived + "everything is fine\n", []string{
			`FAIL .port-parity:4 malformed waiver (want "<file|id|function> <name> -- <reason> [decision: <ref>]" or "rename: <ref> -> <port>"): everything is fine`,
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r, p := fixture(t)
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

func TestCheckCounts(t *testing.T) {
	t.Parallel()
	r, p := fixture(t)
	res := Check(r, p, ParseWaivers(allWaived), ".port-parity")
	want := Counts{RefFiles: 3, RefIDs: 2, RefFuncs: 3, Missing: 3, Waived: 3}
	if res.Counts != want {
		t.Errorf("counts = %+v, want %+v", res.Counts, want)
	}
}
