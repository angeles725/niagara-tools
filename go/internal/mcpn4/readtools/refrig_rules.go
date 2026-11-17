package readtools

import (
	"math"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/refrig"
)

// refrigRuleNames lists the rules n4_refrigeration_faults knows, as its refusals name them.
const refrigRuleNames = "short_cycling, defrost_termination, setpoint_not_reached, over_amps, door_drift"

// refrigCheck runs one rule from its spec (the registry schema checked the fields and types).
type refrigCheck func(r *refrigRun, spec *pyjson.Object) (*pyjson.Object, error)

var refrigChecks = map[string]refrigCheck{
	"short_cycling":        checkShortCycling,
	"defrost_termination":  checkDefrost,
	"setpoint_not_reached": checkSetpoint,
	"over_amps":            checkOverAmps,
	"door_drift":           checkDoorDrift,
}

// need reads the named histories of spec; a failed one makes the rule not_checked (section).
func (r *refrigRun) need(spec *pyjson.Object, keys ...string) ([]*refrigSeries, *pyjson.Object, error) {
	var out []*refrigSeries
	for _, k := range keys {
		s, err := r.series(pyStr(lookup(spec, k)))
		if err != nil {
			return nil, nil, err
		}
		if s.failed != "" {
			return nil, obj("not_checked", k+" "+s.id+" could not be read: "+s.failed), nil
		}
		out = append(out, s)
	}
	return out, nil, nil
}

func num(spec *pyjson.Object, k string) float64 {
	n, _ := lookup(spec, k).(pyjson.Number)
	f, _ := n.Float64()
	return f
}

func minutes(spec *pyjson.Object, k string) time.Duration {
	return time.Duration(num(spec, k) * float64(time.Minute))
}

// findings is {count, items, more} of items cut to the run's limit.
func (r *refrigRun) findings(items []any) *pyjson.Object {
	shown := items[:min(r.limit, len(items))]
	if shown == nil {
		shown = []any{}
	}
	return obj("count", len(items), "items", shown, "more", len(items)-len(shown))
}

func iso(t time.Time) string { return t.Format(time.RFC3339) }

func span(i refrig.Interval) *pyjson.Object {
	return obj("start", iso(i.Start), "end", iso(i.End), "minutes", roundTo(i.Duration().Minutes(), 2), "open", i.Open)
}

func checkShortCycling(r *refrigRun, spec *pyjson.Object) (*pyjson.Object, error) {
	ss, nc, err := r.need(spec, "run_history")
	if nc != nil || err != nil {
		return nc, err
	}
	run := ss[0]
	short, busy := refrig.ShortCycling(run.samples, r.end, time.Duration(num(spec, "min_run_s")*float64(time.Second)),
		int(num(spec, "max_starts_per_hour")), run.loc)
	var items []any
	for _, s := range short {
		items = append(items, obj("fact", "short_run", "run", span(s.Run)))
	}
	for _, b := range busy {
		items = append(items, obj("fact", "too_many_starts", "hour", iso(b.Hour), "starts", b.Starts))
	}
	return r.findings(items), nil
}

func checkDefrost(r *refrigRun, spec *pyjson.Object) (*pyjson.Object, error) {
	ss, nc, err := r.need(spec, "defrost_history", "coil_history")
	if nc != nil || err != nil {
		return nc, err
	}
	cycles := refrig.DefrostTermination(ss[0].samples, ss[1].samples, r.end, num(spec, "terminate_temp"),
		minutes(spec, "max_minutes"), minutes(spec, "slack_minutes"))
	var items []any
	counts := obj("temperature", 0, "time", 0, "early", 0, "open", 0)
	for _, c := range cycles {
		n, _ := lookup(counts, c.EndedBy).(int)
		counts.Set(c.EndedBy, n+1)
		if c.EndedBy != "time" {
			continue // only a cycle ended by time is a finding: the coil never reached the termination temperature
		}
		item := obj("fact", "ended_by_time", "cycle", span(c.Cycle), "max_coil_temp", nil)
		if c.HasMax {
			item.Set("max_coil_temp", c.MaxTemp)
		}
		items = append(items, item)
	}
	out := r.findings(items)
	out.Set("cycles", counts)
	return out, nil
}

// roundTo rounds f to places decimals, so replies do not show float noise (14.999999).
func roundTo(f float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(f*p) / p
}
