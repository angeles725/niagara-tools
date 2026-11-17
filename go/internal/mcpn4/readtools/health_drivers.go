package readtools

import (
	"slices"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The drivers section of n4_station_health (drivers=true): per driver network, its field devices
// and proxy points counted per driver category, the bad ones listed up to maxBad. It reads
// nothing: it groups the rows of the networks, devices and points sections, so it needs all three.

// driverNet is one network entry and what was counted below it.
type driverNet struct {
	entry, devices, points *pyjson.Object
	devicesBad, pointsBad  []any
}

// driverCategories is the categories of cats a status falls in, each once whatever bits it holds:
// unknown alone for a status the station did not answer as a number, disabled also when enabled
// is false.
func driverCategories(flags []string, enabled any, cats []string) []string {
	if len(flags) == 1 && flags[0] == unknownFlag {
		return []string{unknownFlag}
	}
	var out []string
	for _, c := range cats {
		if slices.Contains(flags, c) || (c == disabledCategory && enabled == false) {
			out = append(out, c)
		}
	}
	return out
}

// isProxyPoint is a point row whose proxy extension the station named and is not NullProxyExt.
func isProxyPoint(row *pyjson.Object) bool {
	ext := strings.TrimSpace(cellText(row, proxyColumn))
	return ext != "" && ext != nullProxyExt
}

// driverCounts is a counts object: total and every category of cats, plus unknown, at 0.
func driverCounts(cats []string) *pyjson.Object {
	o := obj("total", 0)
	for _, c := range append(append([]string{}, cats...), unknownFlag) {
		o.Set(c, 0)
	}
	return o
}

// tally adds one row of categories got to each counts object.
func tally(got []string, counts ...*pyjson.Object) {
	for _, o := range counts {
		bump(o, "total")
		for _, c := range got {
			bump(o, c)
		}
	}
}

func bump(o *pyjson.Object, key string) { o.Set(key, lookup(o, key).(int)+1) }

// driversSection groups the networks, devices and points rows per network; cut says a query
// feeding them was cut, so the counts are lower bounds.
func driversSection(networks, devices, points []*pyjson.Object, maxBad int, cut bool) *pyjson.Object {
	nets, order := map[string]*driverNet{}, []string{}
	for _, row := range networks {
		path := cellText(row, "Slot Path")
		if _, seen := nets[path]; seen || path == "" {
			continue
		}
		nets[path], order = &driverNet{entry: obj("network", path, "name", cellText(row, "Name"),
			"type", cellText(row, "Type"), "status", strs(statusFlags(cellText(row, bitsColumn))),
			"enabled", enabledCell(cellText(row, enabledColumn))),
			devices: driverCounts(deviceCategories), points: driverCounts(pointCategories)}, append(order, path)
	}
	totalDevices, totalPoints := driverCounts(deviceCategories), driverCounts(pointCategories)
	unassigned, proxyUnknown := 0, 0
	for _, row := range devices {
		path := cellText(row, "Slot Path")
		n := nets[ownerOf(nets, path)]
		if n == nil || bql.IsLocalDevice(row) {
			continue
		}
		enabled := enabledCell(cellText(row, enabledColumn))
		got := driverCategories(statusFlags(cellText(row, bitsColumn)), enabled, deviceCategories)
		tally(got, n.devices, totalDevices)
		if len(got) > 0 {
			entry := obj("device", path, "status", strs(statusFlags(cellText(row, bitsColumn))))
			if enabled == false {
				entry.Set("enabled", false)
			}
			n.devicesBad = append(n.devicesBad, entry)
		}
	}
	for _, row := range points {
		path := cellText(row, "Slot Path")
		if strings.TrimSpace(cellText(row, proxyColumn)) == "" {
			proxyUnknown++
			continue
		}
		if !isProxyPoint(row) {
			continue
		}
		n := nets[ownerOf(nets, path)]
		if n == nil {
			unassigned++
			continue
		}
		flags := statusFlags(cellText(row, bitsColumn))
		got := driverCategories(flags, nil, pointCategories)
		tally(got, n.points, totalPoints)
		if len(got) > 0 {
			n.pointsBad = append(n.pointsBad, obj("point", path, "status", strs(flags)))
		}
	}
	list := make([]any, 0, len(order))
	for _, path := range order {
		n := nets[path]
		n.entry.Set("devices", n.devices)
		n.entry.Set("devices_bad", firstN(n.devicesBad, maxBad))
		n.entry.Set("devices_bad_truncated", len(n.devicesBad) > maxBad)
		n.entry.Set("points", n.points)
		n.entry.Set("points_bad", firstN(n.pointsBad, maxBad))
		n.entry.Set("points_bad_truncated", len(n.pointsBad) > maxBad)
		list = append(list, n.entry)
	}
	totals := obj("networks", len(order), "devices", totalDevices, "points", totalPoints,
		"unassigned_points", unassigned, "proxy_unknown", proxyUnknown, "rows_truncated", cut)
	return obj("networks", list, "totals", totals)
}

// firstN is the first max entries of list (never nil).
func firstN(list []any, max int) []any {
	if len(list) > max {
		list = list[:max]
	}
	return append([]any{}, list...)
}
