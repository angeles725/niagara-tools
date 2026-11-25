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
		args := []string{"--type", typ, "--vendor", "Angeles"}
		if typ == "logic" {
			// The fixture carries the opt-in offload component: the round trip runs with the flag and compares to
			// the fixture with its marker lines removed (TestRealKitWithoutOffload pins the run without it).
			args = append(args, "--with-offload")
		}
		r := run(t, kit+"/toolbelt", append([]string{fixture, out}, args...)...)
		if r.code != 0 || r.err != "" {
			t.Fatalf("%s: code=%d err=%q", typ, r.code, r.err)
		}
		want, got := tree(t, filepath.Join(kit, "fixtures", fixture)), tree(t, filepath.Join(out, fixture))
		if typ == "logic" {
			for rel, content := range want {
				want[rel] = filterOffload(content, true)
			}
		}
		if len(want) == 0 {
			t.Fatalf("%s: no bundled fixture at %s", typ, kit)
		}
		for rel, content := range got {
			base := filepath.Base(rel)
			// The overlay files come from templates/station-test (pinned by TestStationTestOverlay and the
			// bats goldens); the fixture's own copies need not be byte-equal, only present.
			overlay := base == "moduleTest-include.xml" || strings.HasSuffix(base, "StationTestBase.java") || strings.HasSuffix(base, "LinkTest.java")
			_, inFixture := want[rel]
			if typ == "dashboard" {
				// The login overlay (templates/dashboard-login, pinned by TestLoginOverlay and
				// tests/scaffold-module-login.bats) adds files to -rt and inserts lines into five fixture files:
				// a new file is the overlay's, an inserted-into file keeps every fixture line in its order.
				if !inFixture && isLoginFile(rel) || inFixture && keepsEveryLine(want[rel], content) {
					delete(want, rel)
					continue
				}
			}
			if !inFixture || !overlay && want[rel] != content {
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

// isLoginFile: a file of the login overlay in the dashboard's -rt part (the module's own name, MinimalDash).
func isLoginFile(rel string) bool {
	return strings.Contains(rel, "/login/") || strings.Contains(rel, "-login/") || strings.HasSuffix(rel, "/BMinimalDashService.java")
}

// keepsEveryLine: every line of want is in got, in the same order (got may hold more).
func keepsEveryLine(want, got string) bool {
	rest := strings.Split(got, "\n")
	for _, line := range strings.Split(want, "\n") {
		i := 0
		for i < len(rest) && rest[i] != line {
			i++
		}
		if i == len(rest) {
			return false
		}
		rest = rest[i+1:]
	}
	return true
}
