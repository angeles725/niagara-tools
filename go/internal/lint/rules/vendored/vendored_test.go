package vendored

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: one case per rule of the ledger check (vn-*): vn-defective (a listed file at a version in the
// known-defects table: WARN naming the defect), vn-fixed (listed at a fixed version: clean), vn-semver (range
// rows, a release above its pre-release), vn-unlisted (a vendor file in no row: WARN; licence and notes files
// are never vendored code), vn-no-ledger (no VENDORED.md: one WARN saying how to create it), vn-no-vendor and
// vn-only-licence (nothing to check), vn-strict, vn-incomparable (an r-number against a semver), vn-bad-row,
// vn-no-table, and the exit-3 env faults vn-no-option, vn-defects-missing and vn-defects-bad.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 14) }
