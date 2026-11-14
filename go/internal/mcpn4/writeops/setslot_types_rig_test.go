package writeops_test

import (
	"os"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

const ext = "station:|slot:/Drivers/Net/Dev/Points/RA_T/NumericInterval"

// typed is loads plus the station's registry channel: LoadTypes answers the recorded-shape
// loadTypes reply of testdata/loadtypes-history-ext.json and counts its calls.
type typed struct {
	loads
	types any
	calls [][]string
}

func (c *typed) LoadTypes(specs []string) (any, error) {
	c.calls = append(c.calls, specs)
	return c.types, nil
}

func newTyped(t *testing.T, nodes *pyjson.Object) *typed {
	t.Helper()
	data, err := os.ReadFile("testdata/loadtypes-history-ext.json")
	if err != nil {
		t.Fatal(err)
	}
	types, err := pyjson.Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	return &typed{loads: loads{nodes: obj(ext, nodes)}, types: types}
}

// extNodes is what the live station returned for a NumericIntervalHistoryExt at its defaults
// (2026-11-10): no enabled, historyName or interval node, only the slots off their default.
func extNodes(extra ...any) *pyjson.Object {
	n := obj("", obj("h", "42", "t", "history:NumericIntervalHistoryExt"),
		"status", obj("t", "baja:Status", "v", "0"), "active", obj("t", "baja:Boolean", "v", "true"),
		"historyConfig", obj("t", "history:HistoryConfig"))
	for i := 0; i < len(extra); i += 2 {
		n.Set(extra[i].(string), extra[i+1])
	}
	return n
}

func extSet(slot string, value any, valueType string) *pyjson.Object {
	return obj("ord", ext, "slot", slot, "value", value, "value_type", valueType)
}

func js(t *testing.T, v any) string {
	t.Helper()
	b, err := pyjson.Marshal(v, pyjson.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// planAndReadback plans args on c, then reads back after the station holds after.
func planAndReadback(t *testing.T, c *typed, args *pyjson.Object, after *pyjson.Object) (*plan.Planned, []any) {
	t.Helper()
	p, err := writeops.SetSlot.Plan(c, args)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	c.nodes = obj(ext, after)
	requested, accepted, observed, verdict, err := writeops.SetSlot.Readback(c, args, p, []any{"reply"}, p.Inverse)
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	return p, []any{requested, accepted, observed, verdict}
}
