package scaffoldmodule

import (
	"path/filepath"
	"strings"
	"testing"
)

// With the fixture's own name and the defaults, the emitted tree is the bundled fixture byte for byte.
func TestRealKitRoundTrip(t *testing.T) {
	kit, err := filepath.Abs("../../../build-n4-module-kit")
	if err != nil {
		t.Fatal(err)
	}
	for typ, fixture := range map[string]string{"logic": "MinimalPan", "dashboard": "MinimalDash", "driver": "MinimalDriver"} {
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
