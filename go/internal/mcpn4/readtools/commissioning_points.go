package readtools

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The points section: every control point in scope by status flag, per network and device. A
// device that is down, disabled or in fault is listed itself, first; its points are counted in
// their flags and not listed.

// commDevice is a device or network row and what was counted below it.
type commDevice struct {
	row                      *pyjson.Object
	listed                   bool // it is a finding itself
	fact                     string
	devices, devicesListed   int // of a network
	points, pointsWithStatus int
}

// commDevices reads a device or network table into its entries by path (in row order) and
// lists the not-ok ones in check. A failed table marks check not checked and answers no entry.
func (r *commRun) commDevices(s *commSection, table, check string) (map[string]*commDevice, []string, error) {
	tb, err := r.table(table)
	if err != nil {
		return nil, nil, err
	}
	if tb.failed != "" {
		s.notChecked(tb.failed, check)
		return nil, nil, nil
	}
	if tb.cut && s.truncated == 0 {
		s.truncated = max(len(tb.rows), 1)
	}
	s.check(check)
	at, order := map[string]*commDevice{}, []string{}
	for _, row := range tb.rows {
		path := cellText(row, "Slot Path")
		if _, seen := at[path]; seen {
			continue
		}
		d := &commDevice{row: row}
		var facts []string
		flags := statusFlags(cellText(row, bitsColumn))
		for _, f := range flags {
			d.listed = d.listed || deviceListedFlags[f] || f == unknownFlag
		}
		if len(flags) > 0 {
			facts = append(facts, fmt.Sprintf(factStatus, strings.Join(flags, ", ")))
		}
		if enabledCell(cellText(row, enabledColumn)) == false {
			d.listed, facts = true, append(facts, "enabled false")
		}
		d.fact = strings.Join(facts, ", ")
		at[path], order = d, append(order, path)
	}
	return at, order, nil
}

// pointsSection is the points section.
func (r *commRun) pointsSection() (*commSection, error) {
	points, err := r.table("points")
	if err != nil {
		return nil, err
	}
	s := r.section("points", points)
	names := []string{"devices_down_or_disabled", "networks_not_ok"}
	for _, fc := range pointFlagChecks {
		names = append(names, fc[1])
	}
	if points.failed != "" { // nothing below is judged without the points
		s.notChecked(points.failed, names...)
		return s, nil
	}
	devices, deviceOrder, err := r.commDevices(s, "devices", names[0])
	if err != nil {
		return nil, err
	}
	networks, networkOrder, err := r.commDevices(s, "networks", names[1])
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		s.check(name)
	}
	for _, path := range deviceOrder {
		if n := networks[ownerOf(networks, path)]; n != nil {
			if n.devices++; devices[path].listed {
				n.devicesListed++
			}
		}
	}
	for _, row := range points.rows {
		r.countPoint(s, row, devices[ownerOf(devices, cellText(row, "Slot Path"))], networks[ownerOf(networks, cellText(row, "Slot Path"))])
	}
	for _, name := range names[2:] {
		if n := s.hidden[name]; n > 0 {
			s.check(name).extra = []any{"under_listed_devices", n}
		}
	}
	for _, path := range deviceOrder {
		if d := devices[path]; d.listed {
			s.check(names[0]).add(path, cellText(d.row, "Name"), fmt.Sprintf(factDevice, d.fact, d.points, d.pointsWithStatus))
		}
	}
	byNetwork := []any{}
	for _, path := range networkOrder {
		n := networks[path]
		if n.fact != "" {
			s.check(names[1]).add(path, cellText(n.row, "Name"), n.fact)
		}
		if len(byNetwork) < r.limit {
			byNetwork = append(byNetwork, obj("path", path, "name", cellText(n.row, "Name"), "devices", n.devices,
				"devices_listed", n.devicesListed, "points", n.points, "points_not_ok", n.pointsWithStatus))
		}
	}
	s.extra = []any{"total_points", len(points.rows)}
	if networks != nil {
		s.extra = append(s.extra, "networks", len(networkOrder), "by_network", byNetwork)
	}
	return s, nil
}

// countPoint counts one point row in the check of each of its status flags and below its
// device and network (either may be nil). Under a listed device it is counted, not listed.
func (r *commRun) countPoint(s *commSection, row *pyjson.Object, device, network *commDevice) {
	flags, finding := statusFlags(cellText(row, bitsColumn)), false
	for _, fc := range pointFlagChecks {
		if !contains(flags, fc[0]) {
			continue
		}
		finding = true
		if c := s.check(fc[1]); device != nil && device.listed {
			c.count++
			s.hidden[fc[1]]++
		} else {
			c.add(cellText(row, "Slot Path"), pointName(row), fmt.Sprintf(factStatus, strings.Join(flags, ", ")))
		}
	}
	for _, owner := range []*commDevice{device, network} {
		if owner != nil {
			if owner.points++; finding {
				owner.pointsWithStatus++
			}
		}
	}
}

// contains reports whether list holds s.
func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
