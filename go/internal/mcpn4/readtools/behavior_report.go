package readtools

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func behaviorCutNote(n int) string {
	return fmt.Sprintf("truncated at %d points: raise max_points or narrow scope; a diff does not assert missing or new points from a truncated set", n)
}

// behaviorReport is the n4_behavior_diff reply: both sides, the summary counts and each check as
// {count, items, more}. Missing and new points are not judged when either side was truncated.
func behaviorReport(before, after *snapshot, tolerance float64, limit int) *pyjson.Object {
	d := compareStates(before.points, after.points, tolerance)
	out := obj("before", side(before), "after", side(after), "tolerance", tolerance)
	var notes []any
	if before.station != after.station {
		notes = append(notes, fmt.Sprintf("the snapshots come from different stations (%s, %s)", before.station, after.station))
	}
	if before.scope != after.scope {
		notes = append(notes, fmt.Sprintf("the snapshots cover different scopes (%s, %s)", before.scope, after.scope))
	}
	changes := func(l []pointChange, flags bool) *pyjson.Object {
		items := []any{}
		for _, c := range l[:min(limit, len(l))] {
			b, a := any(c.before), any(c.after)
			if flags {
				b, a = strs(statusFlags(c.before)), strs(statusFlags(c.after))
			}
			items = append(items, obj("path", c.path, "before", b, "after", a))
		}
		return obj("count", len(l), "items", items, "more", len(l)-len(items))
	}
	paths := func(l []string) *pyjson.Object {
		shown := l[:min(limit, len(l))]
		return obj("count", len(l), "items", strs(shown), "more", len(l)-len(shown))
	}
	out.Set("status_changed", changes(d.statusChanged, true))
	out.Set("value_changed", changes(d.valueChanged, false))
	summary := obj("status_changed", len(d.statusChanged), "value_changed", len(d.valueChanged), "unchanged", d.unchanged)
	if before.truncated || after.truncated {
		why := "not_checked: a side was truncated (" + behaviorCutNote(max(len(before.points), len(after.points))) + ")"
		out.Set("missing", obj("not_checked", why))
		out.Set("new", obj("not_checked", why))
	} else {
		out.Set("missing", paths(d.missing))
		out.Set("new", paths(d.added))
		summary.Set("missing", len(d.missing))
		summary.Set("new", len(d.added))
	}
	out.Set("summary", summary)
	if notes != nil {
		out.Set("notes", notes)
	}
	return out
}

func side(s *snapshot) *pyjson.Object {
	return obj("name", s.name, "station", s.station, "scope", s.scope, "taken_at", s.takenAt,
		"points", len(s.points), "truncated", s.truncated)
}
