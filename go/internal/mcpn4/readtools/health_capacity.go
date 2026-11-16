package readtools

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/spy"
)

// capacityResult is the capacity section and what to say beside it: the section (nil when the
// capacity is unknown), the note, and whether the page it was read from was cut.
type capacityResult struct {
	section   *pyjson.Object
	note      string
	truncated bool
}

// healthCapacity is the capacity section: the Global Capacity table of the spy metrics page,
// each resource beside the live count the tool knows (live, by section name; absent is null).
// An unreadable page or one without the table is never a count, and a page that was cut is
// never read as whole: err is only an error that fails the whole call.
func (t *Tools) healthCapacity(timeout int, live map[string]int) (capacityResult, error) {
	reply, page, err := t.spyFetch(capacityPage, capacityMaxRows, box.MaxSpyBytes, timeout)
	if err != nil {
		section, fatal := t.sectionError(err)
		if fatal != nil {
			return capacityResult{}, fatal
		}
		return capacityResult{note: fmt.Sprintf(capacityFailed, lookup(section, "error"))}, nil
	}
	if page == nil {
		return capacityResult{note: fmt.Sprintf(capacityNoPage, lookup(reply, "status"), lookup(reply, "http_status"),
			capacityNoPageWhy)}, nil
	}
	capacity, skipped := spy.ReadCapacity(page)
	res := capacityResult{truncated: len(page.Cut) > 0}
	var notes []string
	switch cut := strings.Join(page.Cut, ", "); {
	case capacity == nil && res.truncated:
		notes = append(notes, fmt.Sprintf(capacityCutNone, cut))
	case capacity == nil:
		notes = append(notes, capacityNoTable)
	case res.truncated:
		notes = append(notes, fmt.Sprintf(capacityCut, cut))
	}
	if skipped > 0 {
		notes = append(notes, fmt.Sprintf(capacitySkipped, skipped))
	}
	res.note = strings.Join(notes, "; ")
	if capacity != nil {
		res.section = capacitySection(capacity, live)
	}
	return res, nil
}

// capacitySection is a read capacity as the reply's section: per resource the licence tally
// (used, limit) beside the live count, and may_be_inflated when the tally can be above what the
// station holds: a used above its live, or a recount that never ran.
func capacitySection(capacity *spy.Capacity, live map[string]int) *pyjson.Object {
	resources, inflated := make([]any, len(capacity.Resources)), capacity.NeverRan()
	for i, r := range capacity.Resources {
		unlimited := spy.NoLimit(r.LimitText)
		entry := obj("type", r.Type, "used", count(r.Used), "limit", count(r.Limit), "unlimited", unlimited, "live", nil)
		if n, ok := live[capacityLive[strings.ToLower(r.Type)]]; ok {
			entry.Set("live", n)
			inflated = inflated || r.Used != nil && *r.Used > n
		}
		if r.Used == nil {
			entry.Set("used_text", r.UsedText)
		}
		if r.Limit == nil && !unlimited {
			entry.Set("limit_text", r.LimitText)
		}
		resources[i] = entry
	}
	recount := pyjson.NewObject()
	for _, row := range capacity.Recount {
		recount.Set(row[0], row[1])
	}
	out := obj("resources", resources, "recount", recount, "may_be_inflated", inflated)
	if inflated {
		out.Set("inflation_note", inflationNote)
	}
	out.Set("source", capacitySource)
	return out
}

// count is a count the page gave, nil when the cell was not one.
func count(n *int) any {
	if n == nil {
		return nil
	}
	return *n
}
