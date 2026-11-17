package readtools

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/refrig"
)

func checkSetpoint(r *refrigRun, spec *pyjson.Object) (*pyjson.Object, error) {
	ss, nc, err := r.need(spec, "temp_history")
	if nc != nil || err != nil {
		return nc, err
	}
	var exclude []refrig.Interval
	if pyStr(lookup(spec, "exclude_history")) != "" {
		ex, nc, err := r.need(spec, "exclude_history")
		if nc != nil || err != nil {
			return nc, err
		}
		exclude = refrig.OnIntervals(ex[0].samples, r.end)
	}
	found := refrig.SetpointNotReached(ss[0].samples, r.end, num(spec, "setpoint"), num(spec, "band"),
		minutes(spec, "min_minutes"), exclude)
	var items []any
	for _, e := range found {
		items = append(items, obj("fact", "above_setpoint", "period", span(e.Period), "peak", e.Peak.V, "peak_time", iso(e.Peak.T)))
	}
	return r.findings(items), nil
}

func checkOverAmps(r *refrigRun, spec *pyjson.Object) (*pyjson.Object, error) {
	ss, nc, err := r.need(spec, "amps_history")
	if nc != nil || err != nil {
		return nc, err
	}
	var items []any
	for _, h := range refrig.OverAmps(ss[0].samples, num(spec, "limit"), ss[0].loc) {
		items = append(items, obj("fact", "over_amps", "hour", iso(h.Hour), "samples_over", h.Over, "samples", h.Count,
			"peak", h.Peak.V, "peak_time", iso(h.Peak.T)))
	}
	return r.findings(items), nil
}

func checkDoorDrift(r *refrigRun, spec *pyjson.Object) (*pyjson.Object, error) {
	ss, nc, err := r.need(spec, "door_history", "temp_history")
	if nc != nil || err != nil {
		return nc, err
	}
	var items []any
	for _, d := range refrig.DoorDrift(ss[0].samples, ss[1].samples, r.end, num(spec, "max_rise"), minutes(spec, "after_minutes")) {
		items = append(items, obj("fact", "temperature_rose", "open", span(d.Open), "start_temp", d.StartTemp,
			"peak", d.Peak.V, "peak_time", iso(d.Peak.T), "rise", roundTo(d.Rise, 3)))
	}
	return r.findings(items), nil
}
