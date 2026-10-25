package tiers

import (
	"encoding/json"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/golden"
)

// TestDifferentialPython compares TierOf, ProductVersion and WriteBlock with tiers.py on the
// target CPython, Unicode whitespace and word characters included.
func TestDifferentialPython(t *testing.T) {
	g := golden.Open(t, "tiers")
	versions := []string{"4.14", " 4.13.1 ", " 4.15　", "4.3\x1c", "04.03", "4.14.0.", "٤.14", "4.14a", "", "x", "4.15​"}
	abouts := []string{
		`<str name="productVersion" val="4.14"/>`, `<str_ name="productVersion" val="1"/>`, `<str½ name="productVersion" val="1"/>`,
		`<str` + "　" + `name="productVersion" val="2"/>`, `<str val="3"` + "\x1f" + `name="productVersion">`,
		`<str a_name="productVersion" val="4">`, `<str ½name="productVersion" val="5">`, `<str name="productVersion"` + " " + `=` + "\u0085" + `"6">`,
		`<str name="x val="7" name="productVersion">`, `<str name="productVersion" val="8"><str name="productVersion" val="9">`,
		`<STR name="productVersion" val="10">`, `<str name="productVersion" val="11" val="">`, "<str\nname=\"productVersion\"\nval=\"12\">",
	}
	script := `import json, sys
from mcp_n4 import tiers
data = json.load(sys.stdin)
opt = {"B": {"S"}}
print(json.dumps([[tiers.tier_of(v) for v in data["v"]], [tiers.product_version(a) or "" for a in data["a"]],
                  [tiers.write_block(tier, v or None, "S", opt) or "" for tier in "ABC" for v in data["v"]]]))`
	in, _ := json.Marshal(map[string][]string{"v": versions, "a": abouts})
	out := g.Run(script, in)
	var want [3][]string
	if err := json.Unmarshal(out, &want); err != nil {
		t.Fatal(err)
	}
	opt := map[string]map[string]bool{"B": {"S": true}}
	for i, v := range versions {
		if got := TierOf(v); got != want[0][i] {
			t.Errorf("TierOf(%q) = %s, Python %s", v, got, want[0][i])
		}
	}
	for i, a := range abouts {
		if got := ProductVersion(a); got != want[1][i] {
			t.Errorf("ProductVersion(%q) = %q, Python %q", a, got, want[1][i])
		}
	}
	for i, tier := range []string{"A", "B", "C"} {
		for j, v := range versions {
			if got := WriteBlock(tier, v, "S", opt); got != want[2][i*len(versions)+j] {
				t.Errorf("WriteBlock(%s, %q) = %q\nPython %q", tier, v, got, want[2][i*len(versions)+j])
			}
		}
	}
}
