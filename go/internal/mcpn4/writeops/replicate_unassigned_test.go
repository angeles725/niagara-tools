package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// -1 is the address the station resets a duplicate to ("unassigned"), not a register: a point already on
// the target at -1 does not hold one, so a copy at -1 is not noted as sharing it (verifier probe G2 of #798).
func TestAnUnassignedAddressDoesNotHoldARegister(t *testing.T) {
	c := replicaClient(t, pointTree("control:NumericWritable", "-1"), "T1")
	c.loads.nodes.Set(net+"/T1", existingPoint("50", "Old", "control:NumericWritable", "-1"))
	p, err := writeops.ReplicatePoints.PlanWith(c, replicaArgs("T1"), budget(200), nil)
	if err != nil {
		t.Fatal(err)
	}
	if notes := js(t, p.Notes); strings.Contains(notes, "Duplicate Writable") {
		t.Errorf("an unassigned (-1) address is noted as a shared register: %s", notes)
	}
}
