package portparity

import (
	"reflect"
	"testing"
)

// renameTrees: the port renamed drawLevels and merged drawEq and drawEqAlt into renderEq; id oldPanel became
// newPanel; keep exists in both trees.
func renameTrees(t *testing.T) (Tree, Tree) {
	t.Helper()
	ref, port := t.TempDir(), t.TempDir()
	write(t, ref, map[string]string{
		"ui.js":      "function drawLevels() {}\nfunction drawEq() {}\nfunction drawEqAlt() {}\nfunction keep() {}\n",
		"index.html": `<div id="oldPanel"></div>`,
	})
	write(t, port, map[string]string{
		"ui.js":      "function renderLevels() {}\nfunction renderEq() {}\nfunction keep() {}\n",
		"index.html": `<div id="newPanel"></div>`,
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

const (
	rLevels = "rename: function drawLevels -> renderLevels -- renamed in the port\n"
	rEq     = "rename: function drawEq -> renderEq -- merged\n"
	rEqAlt  = "rename: function drawEqAlt -> renderEq -- merged\n"
	rPanel  = "rename: id oldPanel -> newPanel -- renamed in the port\n"
	renames = rLevels + rEq + rEqAlt + rPanel
)

func TestKindRenames(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		waivers string
		want    []string
		fail    bool
	}{
		{"renames to targets of the same kind pass as INFO, several renames may merge into one target", renames, []string{
			"INFO index.html:1 id oldPanel renamed to newPanel (renamed in the port)",
			"INFO ui.js:2 function drawEq renamed to renderEq (merged)",
			"INFO ui.js:3 function drawEqAlt renamed to renderEq (merged)",
			"INFO ui.js:1 function drawLevels renamed to renderLevels (renamed in the port)",
		}, false},
		{"a function or id rename without -- <reason> FAILs", "rename: function drawLevels -> renderLevels\n" + rEq + rEqAlt + rPanel, []string{
			"INFO index.html:1 id oldPanel renamed to newPanel (renamed in the port)",
			"INFO ui.js:2 function drawEq renamed to renderEq (merged)",
			"INFO ui.js:3 function drawEqAlt renamed to renderEq (merged)",
			"FAIL .port-parity:1 rename function drawLevels -> renderLevels has no -- <reason> (say why the port renamed it)",
		}, true},
		{"a rename whose target the port lacks FAILs", "rename: function drawLevels -> paintLevels -- renamed\n" + rEq + rEqAlt + rPanel, []string{
			"INFO index.html:1 id oldPanel renamed to newPanel (renamed in the port)",
			"INFO ui.js:2 function drawEq renamed to renderEq (merged)",
			"INFO ui.js:3 function drawEqAlt renamed to renderEq (merged)",
			"FAIL .port-parity:1 rename target function paintLevels is not in the port",
		}, true},
		{"a target of another kind is not a target", rLevels + rEq + rEqAlt + "rename: id oldPanel -> renderEq -- wrong kind\n", []string{
			"INFO ui.js:2 function drawEq renamed to renderEq (merged)",
			"INFO ui.js:3 function drawEqAlt renamed to renderEq (merged)",
			"INFO ui.js:1 function drawLevels renamed to renderLevels (renamed in the port)",
			"FAIL .port-parity:4 rename target id renderEq is not in the port",
		}, true},
		{"a rename of a name the port still has is a stale WARN", renames + "rename: function keep -> renderLevels -- x\n", []string{
			"INFO index.html:1 id oldPanel renamed to newPanel (renamed in the port)",
			"INFO ui.js:2 function drawEq renamed to renderEq (merged)",
			"INFO ui.js:3 function drawEqAlt renamed to renderEq (merged)",
			"INFO ui.js:1 function drawLevels renamed to renderLevels (renamed in the port)",
			"WARN .port-parity:5 stale rename: function keep is still in the port (remove the line)",
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r, p := renameTrees(t)
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
