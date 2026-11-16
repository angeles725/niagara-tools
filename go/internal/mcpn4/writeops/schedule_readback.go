package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// handleInvalidator is a client whose cached handles can be dropped below an ORD
// (box.Client.InvalidateHandles).
type handleInvalidator interface{ InvalidateHandles(prefix string) }

// scheduleReadback reloads the schedule (its replaced subtrees have new handles, so the cached
// ones below its ORD are dropped first), decodes it and compares every part the batch wrote with
// what was requested, in n4_read_schedule's shape.
func scheduleReadback(c plan.Client, args *pyjson.Object, p *plan.Planned, replies, _ []any) (any, any, any, string, error) {
	ordText := str(args, "ord")
	if hi, ok := c.(handleInvalidator); ok {
		hi.InvalidateHandles(ordText)
	}
	s, err := loadSchedule(c, ordText)
	if err != nil {
		return nil, nil, nil, "", err
	}
	want := p.Data.(*pyjson.Object)
	observed := pyjson.NewObject()
	for _, k := range want.Keys() {
		w, _ := want.Get(k)
		got, _ := s.model.Get(k)
		if wo, ok := w.(*pyjson.Object); ok && k == "weekly" { // only the days written
			sub := pyjson.NewObject()
			for _, day := range wo.Keys() {
				v, _ := got.(*pyjson.Object).Get(day)
				sub.Set(day, v)
			}
			got = sub
		}
		if k == "special_events" {
			got = eventsView(got)
		}
		observed.Set(k, got)
	}
	verdict := "mismatch"
	a, errA := pyjson.Marshal(want, pyjson.Canonical)
	b, errB := pyjson.Marshal(observed, pyjson.Canonical)
	if errA == nil && errB == nil && string(a) == string(b) {
		verdict = "verified"
	}
	return want, replies, observed, verdict, nil
}

// eventsView is the special events read back as the plan compares them: name, rule and ranges.
func eventsView(v any) []any {
	out := []any{}
	list, _ := v.([]any)
	for _, e := range list {
		o := e.(*pyjson.Object)
		out = append(out, obj("name", get(o, "name"), "kind", get(o, "kind"), "rule", get(o, "rule"), "ranges", get(o, "ranges")))
	}
	return out
}

// putWant sets want[key] = v, merging a "weekly.<day>" key into a weekly object.
func putWant(want *pyjson.Object, key string, v any) {
	group, sub, nested := strings.Cut(key, ".")
	if !nested {
		want.Set(key, v)
		return
	}
	g, _ := want.Get(group)
	o, ok := g.(*pyjson.Object)
	if !ok {
		o = pyjson.NewObject()
		want.Set(group, o)
	}
	o.Set(sub, v)
}

// opsOf is v when it is a list of ops, else the one op v.
func opsOf(v any) []any {
	if l, ok := v.([]any); ok {
		return l
	}
	return []any{v}
}
