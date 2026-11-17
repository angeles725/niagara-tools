package readtools

import (
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Inventory diff (station-audits S8): n4_inventory_snapshot writes the components of a subtree
// and their configured slot values to a local file; n4_inventory_diff compares two such files.

// Bounds of the inventory tools.
const (
	InventorySnapshotDepth      = 6
	InventorySnapshotComponents = 20000
	InventoryDiffLimit          = 200
)

// InventorySnapshot is n4_inventory_snapshot: one BOX load of base, written under
// <state-dir>/exports/ with a name the server makes (read-only on the station).
func (t *Tools) InventorySnapshot(args *pyjson.Object) (*pyjson.Object, error) {
	base := strArg(args, "base", RootOrd)
	if !strings.HasPrefix(base, "station:") {
		return nil, toolError("base must be a station ORD such as %s, got %s", pyjson.StrRepr(RootOrd), pyjson.StrRepr(base))
	}
	chosen, err := slotsArg(args)
	if err != nil {
		return nil, err
	}
	s, notes, err := t.readInventory(base, intArg(args, "depth", InventorySnapshotDepth),
		intArg(args, "max_components", InventorySnapshotComponents), chosen)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	s.station, s.takenAt = t.Ctx.Session.StationName, now.UTC().Format(time.RFC3339)
	s.name = inventoryName(s.station, now)
	path, size, err := t.saveInventory(s)
	if err != nil {
		return nil, err
	}
	return obj("name", s.name, "path", path, "components", len(s.components), "bytes", size,
		"truncated", s.truncated, "base", base, "depth", s.depth, "notes", strs(notes)), nil
}

// slotsArg is the "slots" argument: nil when absent (configured properties), else its names.
func slotsArg(args *pyjson.Object) ([]string, error) {
	v, ok := args.Get("slots")
	if !ok || v == nil {
		return nil, nil
	}
	out := []string{}
	for _, item := range v.([]any) { // validated against the schema
		s, ok := item.(string)
		if !ok || s == "" || strings.Contains(s, "/") {
			return nil, toolError("slots must be a list of slot names such as 'facets' (no '/'), got %s", pyjson.Repr(v))
		}
		out = append(out, s)
	}
	return out, nil
}

// InventoryDiff is n4_inventory_diff: snapshot a against snapshot b, both local files (no station
// request, no session).
func (t *Tools) InventoryDiff(args *pyjson.Object) (*pyjson.Object, error) {
	a, err := t.loadInventory("a", strArg(args, "a", ""))
	if err != nil {
		return nil, err
	}
	b, err := t.loadInventory("b", strArg(args, "b", ""))
	if err != nil {
		return nil, err
	}
	return diffInventory(a, b).reply(a, b, intArg(args, "limit", InventoryDiffLimit)), nil
}
