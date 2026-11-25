package readtools

import (
	"errors"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The bounds of n4_poll (#891): the whole poll must answer inside the 60 s an MCP client
// commonly waits for one request, with room for the last read.
const (
	pollDefaultSamples  = 6
	pollDefaultInterval = 5.0
	pollMaxSamples      = 40
	pollMaxSpan         = 40.0 // seconds from the first sample to the last
	pollMinInterval     = 1.0
	pollMaxRows         = 200
	pollTimeout         = 10
	sessionLost         = "Could not find Server Session"
)

const pollSpanRefused = "samples x interval_s spans %.0f s; at most %.0f s from the first sample to the last, " +
	"so the reply comes back inside the client's 60 s request limit: poll again for a longer series"

// Poll is n4_poll: one BQL select run samples times, interval_s apart, inside one call, with the
// station clock offset read once before the first sample and a per row and column summary.
// A failed sample is kept with its error and the poll goes on; a lost station session ends it
// early (ended session_lost) with the samples so far.
func (t *Tools) Poll(args *pyjson.Object) (*pyjson.Object, error) {
	base, q := strArg(args, "base", RootOrd), strArg(args, "query", "")
	n, interval := intArg(args, "samples", pollDefaultSamples), floatArg(args, "interval_s", pollDefaultInterval)
	maxRows, timeout := intArg(args, "max_rows", pollMaxRows), intArg(args, "timeout_s", pollTimeout)
	switch {
	case n < 1 || n > pollMaxSamples:
		return nil, toolError("samples must be 1 to %d, got %d", pollMaxSamples, n)
	case interval < pollMinInterval:
		return nil, toolError("interval_s must be at least %.0f, got %g", pollMinInterval, interval)
	case float64(n-1)*interval > pollMaxSpan:
		return nil, toolError(pollSpanRefused, float64(n-1)*interval, pollMaxSpan)
	}
	clock := t.stationClock()
	start, series := t.Ctx.Clock(), newPollSeries()
	samples, ended := []any{}, "complete"
	for i := 0; i < n; i++ {
		if i > 0 { // on a fixed schedule from the start, so a slow read does not drift the next
			if wait := start + float64(i)*interval - t.Ctx.Clock(); wait > 0 {
				t.sleep(wait)
			}
		}
		at := t.Ctx.Clock() - start
		columns, rows, cut, ms, err := t.query(base, q, maxRows, timeout)
		var te *jsonrpc.ToolError
		if i == 0 && errors.As(err, &te) { // a query the tool refuses fails the same way every time
			return nil, err
		}
		s := obj("t", round3(at), "local_time", t.now().Format(time.RFC3339Nano))
		if err != nil {
			s.Set("error", err.Error())
			samples = append(samples, s)
			if ended = pollStop(err); ended != "complete" {
				break
			}
			continue
		}
		s.Set("row_count", len(rows))
		s.Set("elapsed_ms", ms)
		if cut {
			s.Set("truncated", true)
		}
		s.Set("rows", list(rows))
		series.add(columns, rows)
		samples = append(samples, s)
	}
	t.recordRead("n4_poll", "base", args, len(samples))
	out := obj("base", base, "query", bql.Strip(q), "interval_s", interval, "samples_requested", n,
		"ended", ended, "clock", clock, "samples", samples, "series", series.summary())
	if note := pollNotes[ended]; note != "" {
		out.Set("note", note)
	}
	return out, nil
}

// pollNotes say what the caller does after a poll that ended early.
var pollNotes = map[string]string{
	"session_lost": "the station session was lost: reconnect with n4_connect, then poll again",
	"auth_failed":  "the station refused the credentials: the poll stopped so it does not add failed logins",
}

// pollStop is why a failed sample ends the poll: a lost session, or an authentication failure
// (another read would be one more failed login toward the station's lockout); "complete" goes on.
func pollStop(err error) string {
	var be *box.Error
	switch {
	case errors.As(err, &be) && be.IsAuth():
		return "auth_failed"
	case strings.Contains(err.Error(), sessionLost):
		return "session_lost"
	}
	return "complete"
}

// sleep and now are the Tools hooks, time.Sleep and time.Now unless a test set them.
func (t *Tools) sleep(seconds float64) {
	if t.Sleep != nil {
		t.Sleep(seconds)
		return
	}
	time.Sleep(time.Duration(seconds * float64(time.Second)))
}

func (t *Tools) now() time.Time {
	if t.Now != nil {
		return t.Now()
	}
	return time.Now()
}

func round3(v float64) float64 { return float64(int64(v*1000+0.5)) / 1000 }
