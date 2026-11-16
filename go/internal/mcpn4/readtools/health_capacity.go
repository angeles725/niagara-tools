package readtools

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/spy"
)

// healthCapacity is the capacity section: the Global Capacity table of the spy metrics page, or
// nil and why. An unreadable page or one without the table is never a count; err is only an
// error that fails the whole call.
func (t *Tools) healthCapacity(timeout int) (*pyjson.Object, string, error) {
	reply, page, err := t.spyFetch(capacityPage, spy.DefaultMaxRows, box.MaxSpyBytes, timeout)
	if err != nil {
		section, fatal := t.sectionError(err)
		if fatal != nil {
			return nil, "", fatal
		}
		return nil, fmt.Sprintf(capacityFailed, lookup(section, "error")), nil
	}
	if page == nil {
		return nil, fmt.Sprintf(capacityNoPage, lookup(reply, "status"), lookup(reply, "http_status"), capacityNoPageWhy), nil
	}
	capacity := spy.ReadCapacity(page)
	if capacity == nil {
		return nil, capacityNoTable, nil
	}
	resources := make([]any, len(capacity.Resources))
	for i, r := range capacity.Resources {
		unlimited := spy.NoLimit(r.LimitText)
		entry := obj("type", r.Type, "used", count(r.Used), "limit", count(r.Limit), "unlimited", unlimited)
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
	return obj("resources", resources, "recount", recount, "source", capacitySource), "", nil
}

// count is a count the page gave, nil when the cell was not one.
func count(n *int) any {
	if n == nil {
		return nil
	}
	return *n
}
