package bogaudit

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestScanSources pins the python walk and annotation reader: files before sub-directories, dot-directories
// skipped, a class's slots merged across files (first-seen order, last value), paren-joined annotations.
func TestScanSources(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"BA.java":        "@NiagaraProperty(name=\"x\", type=\"double\",\n facets=@Facet(\"BFacets.make(BFacets.MIN, BDouble.make(-2d))\"))\nclass BA extends BB { void w() { c.set( \"x\" ,1); } }\n",
		"sub/BA.java":    "@NiagaraProperty(name=\"y\", flags=Flags.TRANSIENT)\n@NiagaraProperty(name=\"x\", type=\"int\", facets=@Facet(\"BFacets.make(BFacets.MIN, BRelTime.makeSeconds(3))\"))\n",
		".git/BA.java":   "@NiagaraProperty(name=\"hidden\")\n",
		"sub/BB.java":    "@NiagaraAction(name=\"go\", flags=Flags.ASYNC)\n@NiagaraAction(name=\"stop\")\n@NiagaraAction(name=\"hide\",\n flags=Flags.HIDDEN)\n",
		"sub/notes.txt":  "@NiagaraProperty(name=\"txt\")\n",
		"sub/BC.java/BD": "",
	}
	for name, text := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s := scanSources(dir)
	a := s.slots["BA"]
	if a == nil || !reflect.DeepEqual(a.keys, []string{"x", "y"}) {
		t.Fatalf("BA slots = %+v", a)
	}
	if x, _ := a.get("x"); !x.hasMin || x.min != 3000 {
		t.Errorf("x = %+v, want the later file's MIN 3000 (makeSeconds)", x)
	}
	if y, _ := a.get("y"); y.flags != "Flags.TRANSIENT" || y.hasMin {
		t.Errorf("y = %+v", y)
	}
	acts := s.actions["BB"]
	for n, want := range map[string]string{"go": "a", "stop": "o", "hide": "h"} {
		if got, _ := acts.get(n); got != want {
			t.Errorf("action %s = %q, want %q", n, got, want)
		}
	}
	if s.extends["BA"] != "BB" || !s.writes["x"] || len(s.slots) != 1 {
		t.Errorf("extends %v writes %v classes %d", s.extends, s.writes, len(s.slots))
	}
}
