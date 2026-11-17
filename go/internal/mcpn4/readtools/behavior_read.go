package readtools

import (
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
)

// behaviorQuery reads every control point below the scope: path, out value and status bits.
const behaviorQuery = "select slotPath, out.value, status.bits from control:ControlPoint"

var behaviorColumns = []string{"path", "value", "status"}

// readBehavior takes a snapshot of scope now: at most maxPoints rows (truncated beyond).
func (t *Tools) readBehavior(name, scope string, maxPoints, timeout int) (*snapshot, error) {
	columns, raw, cut, _, err := t.query(scope, behaviorQuery, maxPoints, timeout)
	if err != nil {
		return nil, err
	}
	rows, err := bql.ByPosition(columns, raw, behaviorColumns)
	if err != nil {
		return nil, err
	}
	s := &snapshot{name: name, station: t.Ctx.Session.StationName, scope: scope, truncated: cut,
		takenAt: time.Now().UTC().Format(time.RFC3339)}
	for _, r := range rows {
		s.points = append(s.points, pointState{cellText(r, "path"), cellText(r, "value"), cellText(r, "status")})
	}
	return s, nil
}
