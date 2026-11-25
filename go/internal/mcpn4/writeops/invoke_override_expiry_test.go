package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// #1064 item 2: the dry run of active/inactive on a BooleanWritable says when the override will
// hold with no expiry, i.e. the point's maxOverrideDuration facet is 0 or not set.
func TestTheOverrideDryRunNotesWhenItHasNoExpiry(t *testing.T) {
	types := obj("control:BooleanWritable", obj("c", []any{}, "p", "baja:Component"), "acme:Pulse",
		obj("c", []any{}, "p", "baja:Component"), "baja:Component", obj("c", []any{}))
	for name, tc := range map[string]struct {
		typ, facets string
		want        bool
	}{
		"no facets":         {"control:BooleanWritable", "", true},
		"duration 0":        {"control:BooleanWritable", "trueText=s:On|maxOverrideDuration=r:0", true},
		"one hour":          {"control:BooleanWritable", "maxOverrideDuration=r:3600000|trueText=s:On", false},
		"no override sent":  {"acme:Pulse", "", false},
		"unreadable facets": {"control:BooleanWritable", "maxOverrideDuration=r:soon", false},
	} {
		root := obj("", obj("h", "5", "t", tc.typ))
		if tc.facets != "" {
			root.Set("facets", obj("t", "baja:Facets", "v", tc.facets))
		}
		c := &typed{loads: loads{nodes: obj(boolWritable, root)}, types: types}
		p, err := writeops.InvokeAction.Plan(c, obj("ord", boolWritable, "action", "inactive"))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got := strings.Contains(js(t, p.Notes), "no expiry"); got != tc.want {
			t.Errorf("%s: no-expiry note = %v, want %v (%s)", name, got, tc.want, js(t, p.Notes))
		}
	}
}
