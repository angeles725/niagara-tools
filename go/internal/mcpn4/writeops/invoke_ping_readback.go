package writeops

import (
	"fmt"
	"slices"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// pingRowsMax is how many device rows the observed lists (the changed ones first); the counts
// cover them all.
const pingRowsMax = 50

// pingReadback reads the health again and compares it with the plan's read: verified when a
// device's status, lastOkTime, lastFailTime or lastFailCause changed (the driver ran the check),
// mismatch when none did (observed names why), unverified only when a read failed.
func pingReadback(c plan.Client, args *pyjson.Object, p *plan.Planned, reply any) (any, any, any, string, error) {
	requested := obj("action", "ping")
	ordText := str(args, "ord")
	before := p.Data.(*pingRead)
	observed := obj("target", ordText, "kind", before.kind)
	_, nodes, err := plan.Handle(c, ordText, 2)
	if err != nil {
		return nil, nil, nil, "", err
	}
	after := readPing(c, ordText, nodes)
	if before.err != "" || after.err != "" {
		observed.Set("hint", fmt.Sprintf("the health could not be read, so the effect of the ping is unknown (before: %s; after: %s)",
			orNone(before.err), orNone(after.err)))
		return requested, reply, observed, "unverified", nil
	}
	rows, changed := comparePing(before.rows, after.rows)
	observed.Set("devices_total", len(rows))
	observed.Set("devices_changed", changed)
	shown := rows
	if len(shown) > pingRowsMax {
		shown = shown[:pingRowsMax]
		observed.Set("devices_omitted", len(rows)-pingRowsMax)
	}
	list := make([]any, len(shown))
	for i, r := range shown {
		list[i] = r.row
	}
	observed.Set("devices", list)
	if changed > 0 {
		return requested, reply, observed, "verified", nil
	}
	observed.Set("note", "no health slot moved: "+pingWhy(after))
	return requested, reply, observed, "mismatch", nil
}

func orNone(s string) string {
	if s == "" {
		return "read"
	}
	return s
}

type pingCompared struct {
	row     *pyjson.Object
	changed bool
}

// comparePing is each device after the ping with its before and after, the changed ones first;
// changed counts them.
func comparePing(before, after []pingRow) (rows []pingCompared, changed int) {
	byPath := map[string]pingRow{}
	for _, b := range before {
		byPath[b.path] = b
	}
	for _, a := range after {
		b, known := byPath[a.path]
		diff := []any{}
		if !known {
			diff = append(diff, "added")
		} else {
			for _, f := range []struct {
				name string
				same bool
			}{{"status", slices.Equal(a.status, b.status)}, {"last_ok", a.lastOk == b.lastOk},
				{"last_fail", a.lastFail == b.lastFail}, {"last_fail_cause", a.cause == b.cause}} {
				if !f.same {
					diff = append(diff, f.name)
				}
			}
		}
		row := obj("device", a.path, "before", nil, "after", a.view(), "changed", diff)
		if known {
			row.Set("before", b.view())
		}
		rows = append(rows, pingCompared{row, len(diff) > 0})
	}
	slices.SortStableFunc(rows, func(x, y pingCompared) int {
		switch {
		case x.changed && !y.changed:
			return -1
		case !x.changed && y.changed:
			return 1
		}
		return 0
	})
	for _, r := range rows {
		if r.changed {
			changed++
		}
	}
	return rows, changed
}

func (r pingRow) view() *pyjson.Object {
	status := make([]any, len(r.status))
	for i, f := range r.status {
		status[i] = f
	}
	return obj("status", status, "last_ok", r.lastOk, "last_fail", r.lastFail, "last_fail_cause", r.cause)
}

// pingWhy names why a ping moved nothing: a disabled network, a disabled or down device, or a
// check that may still be running.
func pingWhy(r *pingRead) string {
	disabled, down := 0, 0
	for _, row := range r.rows {
		if hasFlag(row.status, "disabled") {
			disabled++
		}
		if hasFlag(row.status, "down") {
			down++
		}
	}
	switch {
	case r.networkDisabled:
		return "the network is disabled, so the driver does not check its devices; enable the network, then ping again"
	case disabled > 0:
		return countOf(disabled, len(r.rows), "disabled") + " (status {disabled}), and the driver does not check a disabled device; enable it, then ping again"
	case down > 0:
		return countOf(down, len(r.rows), "down") + " (status {down}); a failed check moves lastFailTime when it ends, so the check may still be running: read the health again"
	}
	return "the check may still be running (ping does not wait for it); read the health again"
}

// countOf is "the device is disabled" for one device, "2 of 5 devices are disabled" for more.
func countOf(n, total int, state string) string {
	if total == 1 {
		return "the device is " + state
	}
	return fmt.Sprintf("%d of %d devices are %s", n, total, state)
}
