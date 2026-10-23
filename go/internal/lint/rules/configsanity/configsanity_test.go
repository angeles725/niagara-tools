package configsanity

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-config-sanity.bats run (LCS-finderr aside: its locked directory is not a
// golden tree) and every edge tree, recorded from the bash lint-config-sanity.sh.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 46) }

// TestAwkNum pins gawk's string-to-number reading of the CS2 min facet argument.
func TestAwkNum(t *testing.T) {
	for in, want := range map[string]float64{"2.5": 2.5, "0": 0, "0.0": 0, "5,x": 5, "1e3": 1000, "1e": 1, "0x5": 0, "3.": 3, "": 0, "x": 0} {
		if got := awkNum(in); got != want {
			t.Errorf("awkNum(%q) = %v, want %v", in, got, want)
		}
	}
}
