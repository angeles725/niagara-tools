package tiers_test

// Go twin of test_tiers.TestTierClassification.test_the_retro_classifies_tier_refusals
// (go_server.go_twin("tiers", "unit")).

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/retro"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/tiers"
)

func TestTheRetroClassifiesTierRefusals(t *testing.T) {
	for _, text := range []string{tiers.ReasonTier, tiers.WriteBlock("B", "4.15.3.28", "S", nil), tiers.WriteBlock("C", "", "S", nil)} {
		if got := retro.RefusalClass(text); got != "tier" {
			t.Errorf("%q is class %q", text, got)
		}
	}
}
