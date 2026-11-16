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
// slots, three status queries, the inventory summary of their rows, the spy metrics page). Each
// section answers or carries its own error; only an authentication failure fails the call, so
// the server latches on it like on any other read.

// sectionError is err as a section's {error}, or err itself when it must fail the whole call
// (an AuthError). The text is what a failed tool would show, scrubbed of the secret.
func (t *Tools) sectionError(err error) (*pyjson.Object, error) {
	var be *box.Error
	if errors.As(err, &be) && be.IsAuth() {
		return nil, err
	}
	return obj("error", t.Ctx.Scrub(errorWords(err))), nil
}

// errorWords is the text of err that is safe to show (the dispatcher's rule): the words of a
// tool, station or value error, else the error's type only.
func errorWords(err error) string {
	var te *jsonrpc.ToolError
	var be *box.Error
	var ve *pyjson.ValueError
	var typed jsonrpc.PyTyped
	switch {
	case errors.As(err, &te):
		return te.Msg
	case errors.As(err, &be):
		return be.Msg
	case errors.As(err, &ve):
		return ve.Msg
	case errors.As(err, &typed):
		return "internal error (" + typed.PyType() + ")"
	}
	return "internal error (Exception)"
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
	keyed, rowCount, failed := map[string][]*pyjson.Object{}, 0, []string{}
	for _, q := range healthQueries {
		section, rows, err := t.healthStatus(q, base, maxRows, timeout, maxNotOk)
		if err != nil {
			return nil, err
		}
		if rows == nil {
			failed = append(failed, q.section)
		}
		keyed[q.section], rowCount = rows, rowCount+len(rows)
		out.Set(q.section, section)
	}
	out.Set("totals", nil)
	if len(failed) > 0 {
		out.Set("totals_note", fmt.Sprintf(totalsNote, strings.Join(failed, ", ")))
	} else if sum, err := bql.SummarizeInventory(base, keyed["networks"], keyed["devices"], keyed["points"]); err != nil {
		out.Set("totals_note", errorWords(err))
	} else {
		out.Set("totals", lookup(sum, "totals"))
	}
	capacity, note, err := t.healthCapacity(timeout)
	if err != nil {
		return nil, err
	}
	if capacity == nil {
		out.Set("capacity", nil)
		out.Set("capacity_note", note)
	} else {
		out.Set("capacity", capacity)
	}
	out.Set("elapsed_ms", int(math.RoundToEven((t.Ctx.Clock()-start)*1000)))
	t.recordRead("n4_station_health", "base", args, rowCount)
	return out, nil
}
