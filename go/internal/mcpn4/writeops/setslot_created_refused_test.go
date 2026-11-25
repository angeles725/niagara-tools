package writeops_test

// The probe of row 18 for a registry that does not describe the created type (the one case
// the same-session set cannot plan; createdRig is in setslot_created_test.go).

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/blackbox"
)

func TestASetOnACreatedPointWhoseTypeTheRegistryDoesNotDescribeIsRefusedWithTheCause(t *testing.T) {
	r := createdRig(t, map[string]any{"baja:Component": map[string]any{}})
	text := r.Srv.Err("n4_set_slot", blackbox.Args{"dry_run": true, "ord": blackbox.Folder + "/Pt", "slot": "fallback",
		"value": 3.0, "value_type": "baja:StatusNumeric"})
	if !strings.Contains(text, "not found") || !strings.Contains(text, "no type contract") {
		t.Errorf("refusal = %s", text)
	}
}
