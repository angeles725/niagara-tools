package readtools

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// n4_station_health: one reply composed of the reads the server already has (the SystemService
// slots, three status queries, the inventory summary of their rows, the spy metrics page, the
// licence manager page). Each
// section answers or carries its own error; only an authentication failure fails the call, so
// the server latches on it like on any other read.

// sectionError is err as a section's {error}, or err itself when it must fail the whole call
// (an AuthError). The text is what a failed tool would show, scrubbed of the secret.
func (t *Tools) sectionError(err error) (*pyjson.Object, error) {
	var be *box.Error
	if errors.As(err, &be) && be.IsAuth() {
		return nil, err
	}
	return obj("error", t.Ctx.Scrub(jsonrpc.ErrorText(err))), nil
}

// StationHealth is n4_station_health.
func (t *Tools) StationHealth(args *pyjson.Object) (*pyjson.Object, error) {
	base := strArg(args, "base", RootOrd)
	maxRows, timeout := intArg(args, "max_rows", bql.DefaultMaxRows), intArg(args, "timeout_s", 60)
	maxNotOk := intArg(args, "max_not_ok", 50)
	start := t.Ctx.Clock()
	out := pyjson.NewObject()
	system, err := t.healthSystem()
	if err != nil {
		return nil, err
	}
	out.Set("system", system)
	keyed, rowCount, failed, cut := map[string][]*pyjson.Object{}, 0, []string{}, false
	live := map[string]int{} // what the station holds, by section: only for the whole station
	slot, _ := bql.BaseSlot(base)
	for _, q := range healthQueries {
		res, err := t.healthStatus(q, base, maxRows, timeout, maxNotOk)
		if err != nil {
			return nil, err
		}
		if res.rows == nil {
			failed = append(failed, q.section)
		} else if res.liveKnown && slot == "/" {
			live[q.section] = res.live
		}
		keyed[q.section], rowCount, cut = res.rows, rowCount+len(res.rows), cut || res.cut
		out.Set(q.section, res.section)
	}
	out.Set("totals", nil)
	if len(failed) > 0 {
		out.Set("totals_note", fmt.Sprintf(totalsNote, strings.Join(failed, ", ")))
	} else if sum, err := bql.SummarizeInventory(base, keyed["networks"], keyed["devices"], keyed["points"]); err != nil {
		out.Set("totals_note", jsonrpc.ErrorText(err))
	} else {
		totals := lookup(sum, "totals").(*pyjson.Object)
		totals.Set("rows_truncated", cut) // a cut query makes these lower bounds
		out.Set("totals", totals)
	}
	capacity, err := t.healthCapacity(timeout, live)
	if err != nil {
		return nil, err
	}
	if capacity.section == nil {
		out.Set("capacity", nil)
	} else {
		out.Set("capacity", capacity.section)
	}
	out.Set("capacity_truncated", capacity.truncated)
	if capacity.note != "" {
		out.Set("capacity_note", capacity.note)
	}
	license, err := t.healthLicense(timeout, system)
	if err != nil {
		return nil, err
	}
	out.Set("license", license)
	out.Set("elapsed_ms", int(math.RoundToEven((t.Ctx.Clock()-start)*1000)))
	t.recordRead("n4_station_health", "base", args, rowCount)
	return out, nil
}
