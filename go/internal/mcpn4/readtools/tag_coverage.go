package readtools

import (
	"math"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// n4_tag_coverage (station-audits S9): how far the Haystack tagging below a base reaches, in
// one read-only call. It composes the existing reads: select slotPath from control:ControlPoint
// (the points), n4_neql_query hs:equip (the equips), n4_read_tags' tagEntry per component and
// the dictionary catalog n4_set_tags validates against.

// Bounds of n4_tag_coverage.
const (
	TcDefaultMaxPoints = 2000
	tcMaxItems         = 100 // per list; counts hold the totals
	tcTimeout          = 60
)

// tcRun is one coverage read: the entities read and what was left out.
type tcRun struct {
	t               *Tools
	notes           []any
	unreadable      []any
	unreadableCount int
	truncated       bool
	impliedMissing  bool
}

// TagCoverage is n4_tag_coverage.
func (t *Tools) TagCoverage(args *pyjson.Object) (*pyjson.Object, error) {
	base := strArg(args, "base", RootOrd)
	slot, err := bql.BaseSlot(base)
	if err != nil {
		return nil, toolError("base must be a station slot ORD such as %s, got %s", pyjson.StrRepr(RootOrd),
			pyjson.StrRepr(base))
	}
	start := t.Ctx.Clock()
	maxPoints := intArg(args, "max_points", TcDefaultMaxPoints) // 1..NeqlMaxRows: the registry schema
	r := &tcRun{t: t, notes: []any{}, unreadable: []any{}}
	dicts, checked, err := r.dictionaries(strArg(args, "dictionary", ""))
	if err != nil {
		return nil, err
	}
	paths, err := r.pointPaths(bql.StationSlot+slot, maxPoints)
	if err != nil {
		return nil, err
	}
	excluded := 0
	if boolArg(args, "exclude_services", true) {
		paths, excluded = withoutServices(paths)
	}
	points, err := r.read(paths, "station:|", true)
	if err != nil {
		return nil, err
	}
	equipOrds, equipsRead, err := r.equipOrds(base, maxPoints)
	if err != nil {
		return nil, err
	}
	equips, err := r.read(equipOrds, "", false)
	if err != nil {
		return nil, err
	}
	gaps, noPoint, noEquip := tcPointGaps(points)
	unknown, err := tcUnknownTags(append(append([]tcEntity{}, points...), equips...), dicts)
	if err != nil {
		return nil, err
	}
	var orphans any = nil
	orphanCount := any(nil)
	if equipsRead && !r.truncated {
		list := tcOrphanEquips(points, equips)
		orphans, orphanCount = tcCapped(list), len(list)
	} else {
		r.notes = append(r.notes, "equips_without_points not checked: the points or equips were not all read")
	}
	out := obj("base", base, "dictionaries", checked, "points_without_tags", tcCapped(gaps),
		"equips_without_points", orphans, "unknown_tags", tcCapped(unknown),
		"counts", obj("points", len(points), "equips", len(equips), "points_without_tags", len(gaps),
			"missing_point_tag", noPoint, "missing_equip_ref", noEquip, "equips_without_points", orphanCount,
			"unknown_tags", len(unknown), "services_excluded", excluded, "unreadable", r.unreadableCount),
		"truncated", r.truncated, "unreadable", r.unreadable, "notes", r.notes)
	if r.impliedMissing {
		out.Set("implied_unavailable", msgImpliedUnavailable)
	}
	out.Set("elapsed_ms", int(math.RoundToEven((t.Ctx.Clock()-start)*1000)))
	t.recordRead("n4_tag_coverage", "base", args, len(points)+len(equips))
	return out, nil
}

// tcCapped is a list kept to tcMaxItems entries.
func tcCapped(list []*pyjson.Object) []any {
	out := []any{}
	for i, x := range list {
		if i == tcMaxItems {
			break
		}
		out = append(out, x)
	}
	return out
}
