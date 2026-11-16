package schedule

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Range is one requested time range: minutes of the day, end 1440 for midnight, and its value
// as given (bool, number, ordinal or string).
type Range struct {
	Start, End int
	Value      any
}

// Faults collects every fault of a request, one message each, before anything is sent.
type Faults []string

func (f *Faults) add(format string, a ...any) { *f = append(*f, fmt.Sprintf(format, a...)) }

// Err is nil without faults, else one error listing them all.
func (f Faults) Err() error {
	if len(f) == 0 {
		return nil
	}
	return fmt.Errorf("%d fault(s) in the request, nothing was sent:\n- %s", len(f), strings.Join(f, "\n- "))
}

var clockText = regexp.MustCompile(`^([01][0-9]|2[0-3]):([0-5][0-9])$`)

// minutes parses "HH:MM" (00:00..23:59; "24:00" only when end is true), -1 when malformed.
func minutes(v any, end bool) int {
	s, _ := v.(string)
	if end && s == "24:00" {
		return 1440
	}
	m := clockText.FindStringSubmatch(s)
	if m == nil {
		return -1
	}
	h, _ := strconv.Atoi(m[1])
	mm, _ := strconv.Atoi(m[2])
	return h*60 + mm
}

// Weekly parses the weekly argument: {day: [{start, end, value}]} for the days given only. Each
// fault (an unknown day, a malformed or reversed time, an overlap, a value of the wrong type) is
// added to f; the ranges come back sorted by start.
func Weekly(arg *pyjson.Object, k *Kind, facets string, f *Faults) map[string][]Range {
	out := map[string][]Range{}
	for _, day := range arg.Keys() {
		if !known(Days, day) {
			f.add("weekly: unknown day %s (use %s)", pyjson.StrRepr(day), strings.Join(Days, ", "))
			continue
		}
		raw, _ := arg.Get(day)
		list, ok := raw.([]any)
		if !ok {
			f.add("weekly.%s: must be a list of {start, end, value}", day)
			continue
		}
		out[day] = ranges(fmt.Sprintf("weekly.%s", day), list, k, facets, f)
	}
	return out
}

// ranges parses one day's list of ranges and checks them against each other.
func ranges(where string, list []any, k *Kind, facets string, f *Faults) []Range {
	var rs []Range
	for i, item := range list {
		at := fmt.Sprintf("%s[%d]", where, i)
		o, ok := item.(*pyjson.Object)
		if !ok {
			f.add("%s: must be an object {start, end, value}", at)
			continue
		}
		start, end := minutes(field(o, "start"), false), minutes(field(o, "end"), true)
		if start < 0 {
			f.add("%s: start %s is not a time HH:MM from 00:00 to 23:59", at, pyjson.Repr(field(o, "start")))
		}
		if end < 0 {
			f.add("%s: end %s is not a time HH:MM from 00:01 to 24:00", at, pyjson.Repr(field(o, "end")))
		}
		if start >= 0 && end >= 0 && end <= start {
			f.add("%s: end %s is not after start %s", at, pyjson.Repr(field(o, "end")), pyjson.Repr(field(o, "start")))
		}
		value, given := o.Get("value")
		if !given {
			f.add("%s: value is missing", at)
		} else if msg := CheckValue(k, facets, value, false); msg != "" {
			f.add("%s: %s", at, msg)
		}
		if start >= 0 && end > start && given {
			rs = append(rs, Range{start, end, value})
		}
	}
	sort.SliceStable(rs, func(i, j int) bool { return rs[i].Start < rs[j].Start })
	for i := 1; i < len(rs); i++ {
		if rs[i].Start < rs[i-1].End {
			f.add("%s: ranges %s-%s and %s-%s overlap", where, hhmm(rs[i-1].Start), hhmm(rs[i-1].End),
				hhmm(rs[i].Start), hhmm(rs[i].End))
		}
	}
	return rs
}

var rangeOrdinal = regexp.MustCompile(`=(-?\d+)`)

// CheckValue is "" when value fits the kind's value type, else why not. null is accepted only
// where allowNull is true (a default output).
func CheckValue(k *Kind, facets string, value any, allowNull bool) string {
	if value == nil {
		if allowNull {
			return ""
		}
		return "value null is not allowed for a range (only a default output may be null)"
	}
	switch k.Name {
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Sprintf("value %s is not a boolean (a boolean schedule takes true or false)", pyjson.Repr(value))
		}
	case "numeric":
		if _, ok := floatOf(value); !ok {
			return fmt.Sprintf("value %s is not a number (a numeric schedule takes a number)", pyjson.Repr(value))
		}
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Sprintf("value %s is not a string (a string schedule takes a string)", pyjson.Repr(value))
		}
	case "enum":
		n, ok := intOf(value)
		if !ok || n < 0 {
			return fmt.Sprintf("value %s is not an enum ordinal (an enum schedule takes a whole number >= 0)",
				pyjson.Repr(value))
		}
		if _, r, _ := strings.Cut(facets, "range="); r != "" {
			if allowed := rangeOrdinal.FindAllStringSubmatch(r, -1); len(allowed) > 0 && !hasOrdinal(allowed, n) {
				return fmt.Sprintf("value %d is not an ordinal of the schedule's range facet (%s)", n, r)
			}
		}
	}
	return ""
}

func hasOrdinal(allowed [][]string, n int) bool {
	for _, a := range allowed {
		if a[1] == strconv.Itoa(n) {
			return true
		}
	}
	return false
}

func hhmm(m int) string { return fmt.Sprintf("%02d:%02d", m/60, m%60) }

func known(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// floatOf is a finite JSON number as a float.
func floatOf(v any) (float64, bool) {
	n, ok := v.(pyjson.Number)
	if !ok {
		return 0, false
	}
	f, err := n.Float64()
	return f, err == nil && !math.IsInf(f, 0) && !math.IsNaN(f)
}

// intOf is a JSON integer that fits an int.
func intOf(v any) (int, bool) {
	n, ok := v.(pyjson.Number)
	if !ok || n.IsFloat() {
		return 0, false
	}
	b, ok := n.BigInt()
	if !ok || !b.IsInt64() || b.Int64() > math.MaxInt32 {
		return 0, false
	}
	return int(b.Int64()), true
}
