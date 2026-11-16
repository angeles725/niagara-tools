package schedule

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Event is one requested special event: its name (as shown and as a slot name), its day rule's
// body and its ranges.
type Event struct {
	Name, Slot string
	Days       *pyjson.Object
	Ranges     []Range
}

// SpecialEvents parses add_special_event: [{name, kind, rule, ranges}]. A fixed date or a date
// range that ends before today can never occur again: it is a fault unless allowExpired.
func SpecialEvents(list []any, k *Kind, facets string, today time.Time, allowExpired bool, f *Faults) []Event {
	var out []Event
	for i, item := range list {
		where := fmt.Sprintf("add_special_event[%d]", i)
		o, ok := item.(*pyjson.Object)
		if !ok {
			f.add("%s: must be an object {name, kind, rule, ranges}", where)
			continue
		}
		r := ruleReader{o, where, f}
		r.only("name", "kind", "rule", "ranges")
		name, _ := field(o, "name").(string)
		if strings.TrimSpace(name) == "" {
			f.add("%s.name: must be a non-empty string", where)
		}
		kind, _ := field(o, "kind").(string)
		rule, _ := field(o, "rule").(*pyjson.Object)
		if rule == nil {
			rule = pyjson.NewObject()
		}
		days := EncodeRule(kind, rule, where+".rule", f)
		if Expired(kind, rule, today) && !allowExpired {
			f.add("%s: %s can never occur again (it ends before %s); pass allow_expired: true to add it anyway",
				where, pyjson.StrRepr(name), today.Format("2006-01-02"))
		}
		given, ok := field(o, "ranges").([]any)
		if !ok {
			f.add("%s.ranges: must be a list of {start, end, value}", where)
		}
		ranges := ranges(where+".ranges", given, k, facets, f)
		if days != nil {
			days.Set("n", "days")
		}
		out = append(out, Event{name, EscapeName(name), days, ranges})
	}
	return out
}

// EventBody is the BOX body of an event: a DailySchedule with its day and its rule as days.
func EventBody(k *Kind, e Event) *pyjson.Object {
	day := DayBody(k, e.Ranges)
	day.Set("n", "day")
	return obj("nm", "p", "n", e.Slot, "t", "schedule:DailySchedule", "s", []any{day, e.Days})
}

// Expired reports whether a date or date-range rule can never match today or later: its (end)
// date names a year, month and day before today, or a whole year or month before it.
func Expired(kind string, rule *pyjson.Object, today time.Time) bool {
	switch kind {
	case "date":
		return dateBefore(rule, today)
	case "date_range":
		end, _ := field(rule, "end").(*pyjson.Object)
		return end != nil && dateBefore(end, today)
	}
	return false
}

func dateBefore(rule *pyjson.Object, today time.Time) bool {
	y, ok := intOf(field(rule, "year"))
	if !ok || y != today.Year() {
		return ok && y < today.Year()
	}
	month, _ := field(rule, "month").(string)
	m := enumOrdinal(Months[:12], month) + 1
	if m < 1 || m != int(today.Month()) {
		return m >= 1 && m < int(today.Month())
	}
	d, ok := intOf(field(rule, "day"))
	return ok && d < today.Day()
}

// EscapeName is a display name as a Niagara slot name (SlotPath.escape): letters, digits and _
// kept (a leading digit escaped), anything else $hh, or $uhhhh beyond ASCII.
func EscapeName(name string) string {
	var b strings.Builder
	for i, r := range name {
		switch {
		case r == '_' || r < utf8.RuneSelf && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'),
			r >= '0' && r <= '9' && i > 0:
			b.WriteRune(r)
		case r < 0x80:
			fmt.Fprintf(&b, "$%02x", r)
		default:
			fmt.Fprintf(&b, "$u%04x", r)
		}
	}
	return b.String()
}
