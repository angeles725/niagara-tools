package plan_test

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// A Preview is volatile (a link source's current value): it never joins the hashed plan, so a
// token issued on one Preview executes under another.
func TestPreviewNeverChangesThePlanHash(t *testing.T) {
	sess := &session.Session{StationName: "S", BaseURL: "http://127.0.0.1", SessionID: "x"}
	planned := func(preview *pyjson.Object) *plan.Planned {
		return &plan.Planned{Ops: []any{node("ssc", "checkLinks")}, Inverse: []any{}, Notes: []any{},
			Preview: preview}
	}
	_, none, err := plan.Hashed("n4_create_link", planned(nil), sess)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []*pyjson.Object{node("link_sources", []any{node("value", 0.0, "status", "{ok}")}),
		node("link_sources", []any{node("value", 21.5, "status", "{null}")})} {
		hashed, h, err := plan.Hashed("n4_create_link", planned(p), sess)
		if err != nil || h != none {
			t.Errorf("hash = %s (%v), want %s", h, err, none)
		}
		if _, in := hashed.Get("link_sources"); in {
			t.Errorf("hashed plan carries the preview: %v", hashed)
		}
	}
}
