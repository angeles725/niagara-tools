package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// Field retro 2026-11-16, row U18: a writable point's fallback is its output when no priority
// input is set, so a copy with a non-null fallback commands every target device on first
// communication. The dry run notes it; it never refuses. Live 4.15: a null fallback is absent
// from the copy body, a set one carries its value (status ok left out).

// setpointFallback is the note for gwTree's Setpoint (fallback 200.0).
const setpointFallback = `"Gw/points/Setpoint: writable point with a non-null fallback (200.0): every copy commands ` +
	`its device with it on first communication (the fallback is the output when no priority input is set); ` +
	`confirm that value is safe on each target, or set fallback to null on the copies"`

func TestACopyNotesAWritablePointWithANonNullFallback(t *testing.T) {
	_, _, notes := copyPlanOf(t, gwTree())
	if !strings.Contains(notes, setpointFallback) {
		t.Errorf("notes %s\n lack %s", notes, setpointFallback)
	}
}

// withFallbackStatus is gwTree whose Setpoint fallback carries status bits.
func withFallbackStatus(bits string) *pyjson.Object {
	tree := gwTree()
	points, _ := tree.Get("s")
	pts := points.([]any)[3].(*pyjson.Object)
	kids, _ := pts.Get("s")
	setpoint := kids.([]any)[0].(*pyjson.Object)
	sk, _ := setpoint.Get("s")
	fb := sk.([]any)[1].(*pyjson.Object)
	fk, _ := fb.Get("s")
	fb.Set("s", append(fk.([]any), obj("nm", "p", "n", "status", "v", bits)))
	return tree
}

func TestACopyDoesNotNoteANullFallback(t *testing.T) {
	_, _, notes := copyPlanOf(t, withFallbackStatus("40"))
	if strings.Contains(notes, "fallback") {
		t.Errorf("a null fallback is noted: %s", notes)
	}
	_, _, notes = copyPlanOf(t, withFallbackStatus("0"))
	if !strings.Contains(notes, "non-null fallback (200.0)") {
		t.Errorf("an ok fallback is not noted: %s", notes)
	}
}

func TestAReplicationNotesANonNullFallbackWithoutRefusing(t *testing.T) {
	c := replicaClient(t, pointTree("control:NumericWritable", "40002"), "T1")
	src, _ := c.loads.nodes.Get(net + "/Gw")
	root, _ := src.(*pyjson.Object).Get("")
	kids, _ := root.(*pyjson.Object).Get("s")
	root.(*pyjson.Object).Set("s", append(kids.([]any), obj("nm", "p", "n", "fallback", "t", "baja:StatusNumeric",
		"s", []any{obj("nm", "p", "n", "value", "v", "0.0")})))
	p, err := writeops.ReplicatePoints.PlanWith(c, replicaArgs("T1"), budget(200), nil)
	if err != nil {
		t.Fatalf("a fallback refused the plan: %v", err)
	}
	if notes := js(t, p.Notes); !strings.Contains(notes, "Gw: writable point with a non-null fallback (0.0)") {
		t.Errorf("notes %s lack the fallback warning", notes)
	}
}
