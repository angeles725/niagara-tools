package readtools

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The point facts n4_histories_audit needs for flat_scope=sensors (R8 delta N4: settings and enums that hold one
// value are normal) and flat_down (R8 delta N3: a down device is not a placeholder): one read of every control
// point's status bits and facets, taken only when one of the two options is on.

const haFlatPointsFailed = "the flat point read failed, so flat_scope and flat_down were not applied: %s"

// flatPoint is what the point read says of one control point.
type flatPoint struct {
	flags []string // status flags set, as n4_read_slots status_flags
	units string   // the facets' unit name ("fahrenheit"), "" without units
}

// sensorUnits are the unit names (lowercase) that mark a sensor reading: temperature, humidity, pressure, flow,
// concentration and light.
var sensorUnits = map[string]bool{
	"fahrenheit": true, "celsius": true, "kelvin": true, "deltafahrenheit": true, "deltacelsius": true,
	"percentrelativehumidity": true,
	"pascal":                  true, "kilopascal": true, "bar": true, "millibar": true, "poundspersquareinch": true,
	"inchesofwater": true, "inchesofmercury": true, "millimetersofmercury": true, "millimetersofwater": true,
	"cubicfeetperminute": true, "cubicmetersperhour": true, "cubicmeterspersecond": true, "literspersecond": true,
	"gallonsperminute": true, "partspermillion": true, "lux": true, "footcandle": true,
}

// readFlatPoints reads the control points when flat_scope or flat_down needs them; a failed read leaves flatPoints
// nil (flat keeps every history) and says so in the notes.
func (r *haRun) readFlatPoints() {
	if !r.flatSensors && !r.flatDown {
		return
	}
	r.requests++
	r.flatReads++
	columns, raw, _, _, err := r.t.query(r.scope, bql.HistoriesAuditFlatPointQuery, haMaxPoints, r.timeout)
	var rows []*pyjson.Object
	if err == nil {
		rows, err = bql.ByPosition(columns, raw, haFlatPointFields)
	}
	if err != nil {
		why := err.Error()
		if section, serr := r.t.sectionError(err); serr == nil {
			why = fmt.Sprint(lookup(section, "error"))
		}
		r.notes = append(r.notes, fmt.Sprintf(haFlatPointsFailed, why))
		return
	}
	r.flatPoints = map[string]flatPoint{}
	for _, row := range rows {
		p := flatPoint{}
		if flags, err := box.StatusFlags(strings.TrimSpace(cellText(row, "status"))); err == nil {
			p.flags = flags
		}
		if units, _, _ := strings.Cut(parseFacetsText(cellText(row, "facets"), "units"), ";"); units != "" {
			p.units = units
		}
		r.flatPoints[cellText(row, "path")] = p
	}
}

// parseFacetsText is the value of one facet key in a facets string, "" when absent.
func parseFacetsText(facets, key string) string {
	v, _ := parseFacets(facets).Get(key)
	s, _ := v.(string)
	return s
}

// flatInScope is false for a history flat_scope=sensors leaves out: a COV history of a point whose units are no
// sensor unit. Without the option, or when the point read failed, every history is in scope.
func (r *haRun) flatInScope(e histExt) bool {
	if !r.flatSensors || r.flatPoints == nil {
		return true
	}
	if !strings.HasPrefix(strings.TrimSpace(e.interval), "true") { // collected at an interval: a reading
		return true
	}
	return sensorUnits[strings.ToLower(r.flatPoints[parentPath(e.path)].units)]
}

// flatPointDown is e's point's status flags and true when flat_down is on and the point is down, stale or in fault.
func (r *haRun) flatPointDown(e histExt) ([]any, bool) {
	if !r.flatDown || r.flatPoints == nil {
		return nil, false
	}
	flags, down := []any{}, false
	for _, f := range r.flatPoints[parentPath(e.path)].flags {
		flags = append(flags, f)
		down = down || f == "down" || f == "stale" || f == "fault"
	}
	return flags, down
}
