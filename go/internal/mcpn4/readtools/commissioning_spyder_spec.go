package readtools

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// spyder_spec: the IO the caller expects on each Spyder, {<Spyder name>: {<point name>:
// {<field>: <value>}}}, names unescaped as Workbench shows them (2005, RA_T) and fields among
// spyderSpecFields. A malformed spec is refused: a spec silently read as empty would report
// every Spyder as matching.

type spyderDevice struct {
	name   string
	points []spyderPoint
}

type spyderPoint struct {
	name string
	want [][2]string // per field: the IOComp column and the expected value as text
}

// spyderSpecArg reads spyder_spec (nil when absent).
func spyderSpecArg(args *pyjson.Object) ([]spyderDevice, error) {
	v, ok := args.Get("spyder_spec")
	if !ok {
		return nil, nil
	}
	fields := make([]string, len(spyderSpecFields))
	for i, f := range spyderSpecFields {
		fields[i] = f[0]
	}
	refuse := func(format string, a ...any) ([]spyderDevice, error) {
		return nil, toolError(commBadSpyderSpec, strings.Join(fields, ", "), fmt.Sprintf(format, a...))
	}
	spec, ok := v.(*pyjson.Object)
	if !ok {
		return refuse("not an object")
	}
	out := []spyderDevice{}
	for _, device := range spec.Keys() {
		pv, _ := spec.Get(device)
		points, ok := pv.(*pyjson.Object)
		if !ok {
			return refuse("Spyder %s is not an object of points", pyjson.StrRepr(device))
		}
		d := spyderDevice{name: device}
		for _, point := range points.Keys() {
			fv, _ := points.Get(point)
			want, ok := fv.(*pyjson.Object)
			if !ok {
				return refuse("point %s/%s is not an object of fields", device, point)
			}
			p := spyderPoint{name: point}
			for _, field := range want.Keys() {
				column := spyderColumn(field)
				if column == "" {
					return refuse("unknown field %s of %s/%s", pyjson.StrRepr(field), device, point)
				}
				value, _ := want.Get(field)
				text, ok := specText(value)
				if !ok {
					return refuse("%s of %s/%s is %s, not a number or a text", field, device, point, pyjson.Repr(value))
				}
				p.want = append(p.want, [2]string{column, text})
			}
			d.points = append(d.points, p)
		}
		out = append(out, d)
	}
	return out, nil
}

// spyderColumn is the IOComp column of spec field ("" when unknown).
func spyderColumn(field string) string {
	for _, f := range spyderSpecFields {
		if f[0] == field {
			return f[1]
		}
	}
	return ""
}

// specText is a spec value as text: a number as Python writes it, or a string.
func specText(v any) (string, bool) {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x), true
	case pyjson.Number:
		return pyjson.Repr(x), true
	}
	return "", false
}

// diff is a fact per field of p the IOComp row does not hold, in the spec's order.
func (p spyderPoint) diff(row *pyjson.Object) []string {
	var out []string
	for _, w := range p.want {
		got := strings.TrimSpace(cellText(row, w[0]))
		if !specMatches(got, w[1]) {
			out = append(out, fmt.Sprintf(factSpyderMismatch, w[0], cellOrEmpty(got), w[1]))
		}
	}
	return out
}

// specMatches compares a cell with an expected value: as numbers when both are numbers (0 is
// 0.0), else as text, ignoring case.
func specMatches(got, want string) bool {
	g, gErr := strconv.ParseFloat(got, 64)
	w, wErr := strconv.ParseFloat(want, 64)
	if gErr == nil && wErr == nil {
		return g == w
	}
	return strings.EqualFold(got, want)
}
