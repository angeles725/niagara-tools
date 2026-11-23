package scaffoldmodule

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// With the fixture's own name and the defaults, the emitted tree is the bundled fixture byte for byte.
func TestRealKitRoundTrip(t *testing.T) {
	kit, err := filepath.Abs("../../../build-n4-module-kit")
	if err != nil {
		t.Fatal(err)
	}
	for typ, fixture := range map[string]string{"logic": "MinimalPan", "dashboard": "MinimalDash", "driver": "MinimalDriver", "library": "MinimalLibrary"} {
		out := t.TempDir()
		r := run(t, kit+"/toolbelt", fixture, out, "--type", typ, "--vendor", "Angeles")
		if r.code != 0 || r.err != "" {
			t.Fatalf("%s: code=%d err=%q", typ, r.code, r.err)
		}
		want, got := tree(t, filepath.Join(kit, "fixtures", fixture)), tree(t, filepath.Join(out, fixture))
		if len(want) == 0 {
			t.Fatalf("%s: no bundled fixture at %s", typ, kit)
		}
		for rel, content := range got {
			base := filepath.Base(rel)
			// The overlay files come from templates/station-test (pinned by TestStationTestOverlay and the
			// bats goldens); the fixture's own copies need not be byte-equal, only present.
			overlay := base == "moduleTest-include.xml" || strings.HasSuffix(base, "StationTestBase.java") || strings.HasSuffix(base, "LinkTest.java")
			if _, inFixture := want[rel]; !inFixture || !overlay && want[rel] != content {
				t.Errorf("%s: %s differs from the fixture (or is not in it)", typ, rel)
			}
			delete(want, rel)
		}
		if len(want) != 0 {
			t.Errorf("%s: not emitted: %v", typ, keys(want))
		}
	}
}

// TestEmittedMapDeclaresTheTypesKind: the module map a scaffold emits declares the kind of its type, read with
// lint.ReadMapHeader (the reader lint-structure and lint-module-map use): a type that needs a declaration (a
// library, whose -rt part has no palette) carries its `type:` line, and a type of component modules carries none.
func TestEmittedMapDeclaresTheTypesKind(t *testing.T) {
	kit, err := filepath.Abs("../../../build-n4-module-kit")
	if err != nil {
		t.Fatal(err)
	}
	for _, typ := range moduleTypes {
		out := t.TempDir()
		if r := run(t, kit+"/toolbelt", "ModuleA", out, "--type", typ.name, "--vendor", "Example"); r.code != 0 || r.err != "" {
			t.Fatalf("%s: code=%d err=%q", typ.name, r.code, r.err)
		}
		b, err := os.ReadFile(filepath.Join(out, "ModuleA", "ModuleA", lint.MapFile))
		if err != nil {
			t.Fatalf("%s: %v", typ.name, err)
		}
		h := lint.ReadMapHeader(strings.Split(string(b), "\n"))
		if h.Kind != typ.kind || h.Mode != lint.ModularityStrict {
			t.Errorf("%s: map declares kind %q mode %v, want kind %q and strict", typ.name, h.Kind, h.Mode, typ.kind)
		}
	}
}
