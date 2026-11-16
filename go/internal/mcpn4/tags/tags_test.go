package tags_test

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/tags"
)

func TestParseIDAndSlot(t *testing.T) {
	id, ok := tags.ParseID("hs:point")
	if !ok || id.Slot() != "hs$3apoint" || id.String() != "hs:point" {
		t.Fatalf("hs:point -> %+v %v", id, ok)
	}
	for _, bad := range []string{"point", "hs:", ":point", "hs:po int", "hs:a:b", "hs:a|b", "9s:x", "hs:a-b"} {
		if _, ok := tags.ParseID(bad); ok {
			t.Errorf("%q accepted", bad)
		}
	}
	if got := tags.FromSlot("ntpl$3atemplate"); got != "ntpl:template" {
		t.Errorf("FromSlot %q", got)
	}
}

func TestIsTagSlot(t *testing.T) {
	cases := []struct {
		nm, name, flags string
		handle, want    bool
	}{
		{"p", "hs$3apoint", "m", false, true},
		{"p", "ntpl$3atemplate", "rhm", false, true},
		{"p", "plain", "m", false, true},
		{"p", "hs$3apoint", "", false, false},
		{"p", "hs$3apoint", "mt", false, false},
		{"p", "hs$3apoint", "m", true, false},
		{"a", "hs$3apoint", "m", false, false},
		{"p", "a$3ab$3ac", "m", false, false},
	}
	for _, c := range cases {
		if got := tags.IsTagSlot(c.nm, c.name, c.flags, c.handle); got != c.want {
			t.Errorf("%+v: got %v", c, got)
		}
	}
}

func TestDecodeImplied(t *testing.T) {
	for key, want := range map[string]tags.Implied{
		"n:pointM":        {ID: "n:point", Type: "baja:Marker"},
		"n:nameS":         {ID: "n:name", Type: "baja:String", Value: "Pt1"},
		"n:ordInSessionO": {ID: "n:ordInSession", Type: "baja:Ord", Value: "station:|h:2"},
		"x:capN":          {ID: "x:cap", Type: "baja:Float", Value: "fNaN"},
		"x:dN":            {ID: "x:d", Type: "baja:Double", Value: "dNaN"},
	} {
		got, ok := tags.DecodeImplied(key, want.Value)
		if !ok || got != want {
			t.Errorf("%s: got %+v %v", key, got, ok)
		}
	}
	if _, ok := tags.DecodeImplied("n:pointQ", nil); ok {
		t.Error("unknown letter accepted")
	}
}

func TestRelationFields(t *testing.T) {
	r := tags.RelationFields("r", map[string]any{"relationId": "n:tagGroup", "sourceOrd": "slot:/G"})
	if !r.IsTagGroup() || r.Endpoint != "slot:/G" || r.Inbound {
		t.Fatalf("%+v", r)
	}
	if tags.RelationFields("r1", map[string]any{"relationId": "n:tagGroup", "inbound": "true"}).IsTagGroup() {
		t.Fatal("inbound tag group")
	}
}
