package writeops

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Twin of test_snapshot_omits_t_for_typeless_nodes_and_recognizes_status_by_name: frozen
// children carry no type (no "t": null), and a status child of a Status* parent loses its
// runtime facets.
func TestSnapshotOmitsTForTypelessNodesAndRecognizesStatusByName(t *testing.T) {
	n, err := pyjson.Unmarshal([]byte(`{"t": "baja:StatusNumeric", "h": "9", "s": [
		{"n": "value", "v": "1.0"}, {"n": "status", "v": "40;activeLevel=e:17"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	s := &snapshot{handles: map[string]string{}}
	snap, err := s.take(n.(*pyjson.Object), "x", SnapshotDepth)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := pyjson.Marshal(get(snap, "s"), pyjson.Options{})
	if want := `[{"nm": "p", "n": "value", "v": "1.0"}, {"nm": "p", "n": "status", "v": "40"}]`; string(got) != want {
		t.Fatalf("got %s want %s", got, want)
	}
	if s.handles["s9"] != "x" {
		t.Fatalf("handles %v", s.handles)
	}
}
