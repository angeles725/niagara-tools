package structure

import "testing"

// LacksSigningBlock: only a root that applies the plugin and has no niagaraSigning { in its code is refused;
// a comment or a string holding the text is no block.
func TestLacksSigningBlock(t *testing.T) {
	apply := "plugins { id(\"com.tridium.niagara-signing\") }\n"
	for _, c := range []struct {
		name, src string
		want, err bool
	}{
		{"wizard root", apply, true, false},
		{"block", apply + "niagaraSigning {\n}\n", false, false},
		{"block in a comment", apply + "// niagaraSigning {\n", true, false},
		{"block in a string", apply + "val s = \"niagaraSigning {\"\n", true, false},
		{"plugin not applied", "plugins { id(\"java\") }\n", false, false},
		{"unscannable", "val s = \"x\n", false, true},
	} {
		got, err := LacksSigningBlock(c.src)
		if got != c.want || (err != nil) != c.err {
			t.Errorf("%s: got %v, %v; want %v (error %v)", c.name, got, err, c.want, c.err)
		}
	}
}
