package readtools

import (
	"math"
	"sort"
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// n4_override_audit (retro R8 delta N7): the writables that hold an override below a base, each
// with its active level (read from the out text), its expiry (none is permanent) and the link
// that feeds its in16, with the schedule behind it when that is one. Four fixed BQL reads (one per
// writable type), one for the schedule handles and one load per overridden writable.

// Bounds of n4_override_audit.
const (
	OaDefaultMaxWritables  = 5000
	oaTimeout              = 60
	oaMaxIn16Loads         = 200
	oaMinLevel, oaMaxLevel = 1, 15
)

// oaDefaultLevels are the levels the override actions write: in1 (emergencyOverride) and in8
// (override).
var oaDefaultLevels = []int{1, 8}

// oaWritable is one writable as the BQL read gives it.
type oaWritable struct {
	path, typ, out string
	level          int
	levelName      string
	expires        any // text, nil when it has no expiry
	in16           *in16
}

// oaRun is one audit.
type oaRun struct {
	t       *Tools
	levels  map[int]bool
	max     int
	notes   []any
	unread  []any
	cut     bool
	writes  []oaWritable
	byLevel map[string]int
}

// OverrideAudit is n4_override_audit.
func (t *Tools) OverrideAudit(args *pyjson.Object) (*pyjson.Object, error) {
	base := strArg(args, "base", RootOrd)
	slot, err := bql.BaseSlot(base)
	if err != nil {
		return nil, toolError("base must be a station slot ORD such as %s, got %s", pyjson.StrRepr(RootOrd), pyjson.StrRepr(base))
	}
	levels, err := oaLevels(args)
	if err != nil {
		return nil, err
	}
	start := t.Ctx.Clock()
	r := &oaRun{t: t, levels: levels, max: intArg(args, "max_writables", OaDefaultMaxWritables), notes: []any{},
		unread: []any{}, byLevel: map[string]int{}}
	if err := r.readWritables(bql.StationSlot + slot); err != nil {
		return nil, err
	}
	over := r.overrides()
	if err := r.feedIn16(over); err != nil {
		return nil, err
	}
	t.recordRead("n4_override_audit", "base", args, len(r.writes))
	out := obj("base", base, "levels", oaLevelList(levels), "overrides", oaEntries(over), "counts", r.counts(over),
		"unreadable", r.unread, "truncated", r.cut, "notes", r.notes)
	out.Set("elapsed_ms", int(math.RoundToEven((t.Ctx.Clock()-start)*1000)))
	return out, nil
}

// oaLevels is the levels argument as a set (the default is the override actions' levels).
func oaLevels(args *pyjson.Object) (map[int]bool, error) {
	set := map[int]bool{}
	v, ok := args.Get("levels")
	if !ok {
		for _, l := range oaDefaultLevels {
			set[l] = true
		}
		return set, nil
	}
	list, _ := v.([]any)
	for _, item := range list {
		if n, ok := item.(pyjson.Number); ok {
			if b, ok := n.BigInt(); ok && b.IsInt64() && b.Int64() >= oaMinLevel && b.Int64() <= oaMaxLevel {
				set[int(b.Int64())] = true
				continue
			}
		}
		return nil, toolError("levels takes whole numbers from %d to %d (level 16 is the base level, never an override), got %s",
			oaMinLevel, oaMaxLevel, pyjson.Repr(item))
	}
	if len(set) == 0 {
		return nil, toolError("levels needs at least one level from %d to %d", oaMinLevel, oaMaxLevel)
	}
	return set, nil
}

func oaLevelList(levels map[int]bool) []any {
	list := make([]int, 0, len(levels))
	for l := range levels {
		list = append(list, l)
	}
	sort.Ints(list)
	out := make([]any, len(list))
	for i, l := range list {
		out[i] = l
	}
	return out
}

// kind names a level: the two override actions' levels, any other an input.
func oaKind(level int) string {
	switch level {
	case 1:
		return "emergency"
	case 8:
		return "manual"
	}
	return "input"
}

// overrides is the writables at a listed level, the permanent ones first, then by level and path.
func (r *oaRun) overrides() []oaWritable {
	var over []oaWritable
	for _, w := range r.writes {
		if w.level > 0 && r.levels[w.level] {
			over = append(over, w)
		}
	}
	sort.SliceStable(over, func(i, j int) bool {
		a, b := over[i], over[j]
		if (a.expires == nil) != (b.expires == nil) {
			return a.expires == nil
		}
		if a.level != b.level {
			return a.level < b.level
		}
		return a.path < b.path
	})
	return over
}

// counts is the totals: writables read, overridden, permanent and timed, and every level the out
// texts show (so a level that is no override is still counted).
func (r *oaRun) counts(over []oaWritable) *pyjson.Object {
	permanent := 0
	for _, w := range over {
		if w.expires == nil {
			permanent++
		}
	}
	names := make([]string, 0, len(r.byLevel))
	for n := range r.byLevel {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		a, aerr := strconv.Atoi(names[i])
		b, berr := strconv.Atoi(names[j])
		if aerr == nil && berr == nil {
			return a < b
		}
		return aerr == nil && berr != nil || aerr != nil && berr != nil && names[i] < names[j]
	})
	by := pyjson.NewObject()
	for _, n := range names {
		by.Set(n, r.byLevel[n])
	}
	return obj("writables", len(r.writes), "overridden", len(over), "permanent", permanent, "timed", len(over)-permanent,
		"unreadable", len(r.unread), "by_level", by)
}
