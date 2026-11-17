package readtools

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The rules of n4_histories_audit, pure over the rows the station answered: a point with no
// history, colliding ids, a stale history, one close to its capacity.

// histExt is one history extension row, every cell as the station wrote it.
type histExt struct {
	path, typ, id, capacity, policy, interval, last string
	enabled                                         any // true, false or nil (not answered)
}

// histAuditCfg holds the thresholds of one audit and the time it measures ages from.
type histAuditCfg struct {
	staleFactor float64       // a regular history is stale past staleFactor x its interval
	staleMin    time.Duration // ... but never before this age (records show up in batches)
	covMaxAge   time.Duration // a COV history is stale past this age
	warn        float64       // a record-capped history is near capacity from this fraction
	now         time.Time
}

// pointsWithoutHistory are the points with no history extension right below them.
func pointsWithoutHistory(points []string, exts []histExt) []string {
	has := map[string]bool{}
	for _, e := range exts {
		has[parentPath(e.path)] = true
	}
	out := []string{}
	for _, p := range points {
		if !has[p] {
			out = append(out, p)
		}
	}
	return out
}

// servicesPrefix is the slot path of the station's services folder.
const servicesPrefix = "slot:/Services"

// withoutServices is points without those under the services folder, and how many it dropped.
func withoutServices(points []string) ([]string, int) {
	kept := []string{}
	for _, p := range points {
		if p != servicesPrefix && !strings.HasPrefix(p, servicesPrefix+"/") {
			kept = append(kept, p)
		}
	}
	return kept, len(points) - len(kept)
}

// collidingIds are the history ids two or more extensions record into, sorted by id.
func collidingIds(exts []histExt) []*pyjson.Object {
	byId := map[string][]any{}
	for _, e := range exts {
		if e.id != "" {
			byId[e.id] = append(byId[e.id], e.path)
		}
	}
	ids := []string{}
	for id, paths := range byId {
		if len(paths) > 1 {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	out := []*pyjson.Object{}
	for _, id := range ids {
		out = append(out, obj("history_id", id, "exts", byId[id]))
	}
	return out
}

// isCov is whether e records on change of value: its collection interval says irregular, or,
// when that cell is unreadable, its type is a COV extension.
func isCov(e histExt) (cov bool, every time.Duration, ok bool) {
	if irregular, d, ok := bql.ParseCollectionInterval(e.interval); ok {
		return irregular, d, true
	}
	return strings.Contains(e.typ, "CovHistoryExt"), 0, strings.Contains(e.typ, "CovHistoryExt")
}

// stale is the stale item of an enabled extension, nil when its last record is recent enough;
// why names the cell (last_record or interval) that left it undecided.
func (c histAuditCfg) stale(e histExt) (item *pyjson.Object, why string) {
	cov, every, ok := isCov(e)
	if !ok {
		return nil, "interval"
	}
	kind, limit := "cov", c.covMaxAge
	if !cov {
		kind, limit = spanText(every), max(time.Duration(c.staleFactor*float64(every)), c.staleMin)
	}
	out := obj("history_id", e.id, "ext", e.path)
	if strings.TrimSpace(e.last) == "null" {
		out.Set("last_record", nil)
		out.Set("interval", kind)
		out.Set("threshold", spanText(limit))
		out.Set("age", nil)
		out.Set("note", "never recorded")
		return out, ""
	}
	at, ok := bql.ParseAbsTime(strings.TrimSpace(e.last))
	if !ok {
		return nil, "last_record"
	}
	if age := c.now.Sub(at); age > limit {
		out.Set("last_record", strings.TrimSpace(e.last))
		out.Set("interval", kind)
		out.Set("threshold", spanText(limit))
		out.Set("age", spanText(age))
		return out, ""
	}
	return nil, ""
}

// needsCount is whether e's record count is worth a read: an enabled history with an id, capped
// by record count, that has recorded.
func needsCount(e histExt) bool {
	capacity, ok := bql.ParseCapacity(e.capacity)
	return e.enabled != false && e.id != "" && ok && capacity.RestrictBy == bql.CapacityRecords &&
		capacity.Max > 0 && strings.TrimSpace(e.last) != "null"
}

// nearCapacity is the near-capacity item of a record-capped history holding records, nil below
// the warning fraction. A stop policy stops recording when full; roll overwrites the oldest.
func (c histAuditCfg) nearCapacity(e histExt, records int) *pyjson.Object {
	capacity, _ := bql.ParseCapacity(e.capacity)
	fraction := float64(records) / float64(capacity.Max)
	if capacity.Max <= 0 || fraction < c.warn {
		return nil
	}
	policy := strings.TrimSpace(e.policy)
	severity, effect := "info", "rollover: the oldest records are overwritten"
	switch {
	case policy == "stop" && int64(records) >= capacity.Max:
		severity, effect = "warning", "full: the history has stopped recording"
	case policy == "stop":
		severity, effect = "warning", "recording stops when the history is full"
	case policy != "roll":
		effect = "full policy not answered"
	}
	return obj("history_id", e.id, "ext", e.path, "records", records, "capacity", int(capacity.Max),
		"fraction", math.Round(fraction*1000)/1000, "full_policy", policy, "severity", severity, "effect", effect)
}

// spanText is d as days, hours, minutes and seconds, the zero parts left out (5m, 1d2h, 0s).
func spanText(d time.Duration) string {
	d = d.Round(time.Second)
	var b strings.Builder
	for _, u := range []struct {
		unit time.Duration
		tag  string
	}{{24 * time.Hour, "d"}, {time.Hour, "h"}, {time.Minute, "m"}, {time.Second, "s"}} {
		if n := d / u.unit; n > 0 {
			fmt.Fprintf(&b, "%d%s", n, u.tag)
			d -= n * u.unit
		}
	}
	if b.Len() == 0 {
		return "0s"
	}
	return b.String()
}
