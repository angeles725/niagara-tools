package privateapiimport

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: the G65 contract (ledger.d/2026-10-23-r8-u5-c59-c62.md), one hand-written case per testdata/*.txt:
// the import, static-import and fully qualified forms WARN; the default allow-list (com.tridium.ndriver.**,
// com.tridium.driver.util.DrUtil and its members), com.tridiumx, comments and strings are clean; --default-allow off
// and --allow globs; --strict; test, build and non-part sources skipped; empty roots, usage and a bad option value.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 11) }

// TestAllowed: an exact entry allows the name and its members; a glob matches the whole name, "*" one segment.
func TestAllowed(t *testing.T) {
	for _, tc := range []struct {
		pattern, name string
		want          bool
	}{
		{"com.tridium.ndriver.**", "com.tridium.ndriver.comm.NMessage", true},
		{"com.tridium.ndriver.**", "com.tridium.ndriver.*", true},
		{"com.tridium.ndriver.**", "com.tridium.ndriverx.Foo", false},
		{"com.tridium.driver.util.DrUtil", "com.tridium.driver.util.DrUtil", true},
		{"com.tridium.driver.util.DrUtil", "com.tridium.driver.util.DrUtil.getParent", true},
		{"com.tridium.driver.util.DrUtil", "com.tridium.driver.util.DrUtilX", false},
		{"com.tridium.history.*", "com.tridium.history.BHistoryX", true},
		{"com.tridium.history.*", "com.tridium.history.db.BLocalDb", false},
	} {
		if got := compile(tc.pattern).allows(tc.name); got != tc.want {
			t.Errorf("%s allows %s = %v, want %v", tc.pattern, tc.name, got, tc.want)
		}
	}
}
