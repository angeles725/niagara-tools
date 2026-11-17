package readtools

import (
	"fmt"
	"slices"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// device_detail=true extends the drivers section: each listed bad device gets what a technician
// needs to fix it and each network a down pattern. Two requests at most, whatever the device
// count: one detail query over every device and one port query over every network, sent only
// when a network lists a bad device. A value the station did not give is null, never a guess.

// detailRow is the detail of one device as the station answered it.
type detailRow struct{ faultCause, lastOk, lastFail, lastFailCause, address any }

// addDeviceDetail adds the detail to drivers (driversSection's object). devices and points are the
// rows of the health sections; err is only an error that fails the whole call.
func (t *Tools) addDeviceDetail(drivers *pyjson.Object, devices, points []*pyjson.Object, base string, maxRows,
	timeout int) error {
	networks, _ := lookup(drivers, "networks").([]any)
	bad := map[string]*pyjson.Object{}
	for _, n := range networks {
		for _, d := range lookup(n.(*pyjson.Object), "devices_bad").([]any) {
			bad[cellText(d.(*pyjson.Object), "device")] = d.(*pyjson.Object)
		}
	}
	var notes []string
	rows, ports := map[string]detailRow{}, map[string]any{}
	if len(bad) > 0 {
		var err error
		if rows, notes, err = t.detailRows(base, maxRows, timeout); err != nil {
			return err
		}
		if ports, err = t.detailPorts(base, maxRows, timeout, &notes); err != nil {
			return err
		}
	}
	stale := stalePoints(bad, points)
	for path, entry := range bad {
		row := rows[path]
		entry.Set("fault_cause", row.faultCause)
		entry.Set("last_fail_cause", row.lastFailCause)
		entry.Set("last_ok", row.lastOk)
		entry.Set("last_fail", row.lastFail)
		entry.Set("address", row.address)
		entry.Set("stale_points", stale[path])
	}
	tallies := deviceTallies(networks, devices)
	for _, n := range networks {
		entry := n.(*pyjson.Object)
		path := cellText(entry, "network")
		c := tallies[path]
		entry.Set("port", ports[path])
		entry.Set("pattern", downPattern(c[0], c[1], c[2], ports[path] != nil))
	}
	if len(notes) > 0 {
		drivers.Set("device_detail_note", strings.Join(notes, "; "))
	}
	return nil
}

// detailRows is the detail query's rows by device path; a failed query is no rows and a note.
func (t *Tools) detailRows(base string, maxRows, timeout int) (map[string]detailRow, []string, error) {
	out := map[string]detailRow{}
	columns, raw, _, _, err := t.query(base, detailDevicesQuery, maxRows, timeout)
	names := detailDeviceColumns
	if err == nil && len(raw) > 0 && len(columns) < len(names) && len(columns) >= detailFixedColumns {
		names = names[:detailFixedColumns]
	}
	var rows []*pyjson.Object
	if err == nil {
		rows, err = bql.ByPosition(columns, raw, names)
	}
	if err != nil {
		section, err := t.sectionError(err)
		if err != nil {
			return nil, nil, err
		}
		return out, []string{fmt.Sprintf(detailFailedNote, cellText(section, "error"))}, nil
	}
	var notes []string
	if len(names) < len(detailDeviceColumns) {
		notes = append(notes, detailNoAddress)
	}
	for _, row := range rows {
		out[cellText(row, "Slot Path")] = detailRow{faultCause: textOrNull(cellText(row, "FaultCause")),
			lastOk: textOrNull(cellText(row, "LastOk")), lastFail: textOrNull(cellText(row, "LastFail")),
			lastFailCause: textOrNull(cellText(row, "LastFailCause")), address: deviceAddress(row)}
	}
	return out, notes, nil
}

// detailPorts is each network's serial port by network path; a failed query adds a note.
func (t *Tools) detailPorts(base string, maxRows, timeout int, notes *[]string) (map[string]any, error) {
	out := map[string]any{}
	columns, raw, _, _, err := t.query(base, detailNetworksQuery, maxRows, timeout)
	var rows []*pyjson.Object
	if err == nil {
		rows, err = bql.ByPosition(columns, raw, detailNetworkColumns)
	}
	if err != nil {
		section, err := t.sectionError(err)
		if err == nil {
			*notes = append(*notes, fmt.Sprintf(detailPortNote, cellText(section, "error")))
		}
		return out, err
	}
	for _, row := range rows {
		out[cellText(row, "Slot Path")] = textOrNull(cellText(row, "Port"))
	}
	return out, nil
}

// deviceAddress is the address of a detail row from its type's known address slot, else null.
func deviceAddress(row *pyjson.Object) any {
	module, _, _ := strings.Cut(cellText(row, "Type"), ":")
	if column, ok := addressColumns[module]; ok {
		return textOrNull(cellText(row, column))
	}
	return nil
}

// stalePoints counts the stale proxy points under each bad device.
func stalePoints(bad map[string]*pyjson.Object, points []*pyjson.Object) map[string]int {
	out := map[string]int{}
	for path := range bad {
		out[path] = 0
	}
	for _, row := range points {
		if !isProxyPoint(row) || !slices.Contains(statusFlags(cellText(row, bitsColumn)), "stale") {
			continue
		}
		if owner := ownerOf(bad, cellText(row, "Slot Path")); owner != "" {
			out[owner]++
		}
	}
	return out
}

// deviceTallies is, per network path, its field devices: total, down and bad (in any device
// category), counted by the rule of driversSection.
func deviceTallies(networks []any, devices []*pyjson.Object) map[string][3]int {
	out := map[string][3]int{}
	for _, n := range networks {
		out[cellText(n.(*pyjson.Object), "network")] = [3]int{}
	}
	for _, row := range devices {
		owner := ownerOf(out, cellText(row, "Slot Path"))
		if owner == "" || bql.IsLocalDevice(row) {
			continue
		}
		flags := statusFlags(cellText(row, bitsColumn))
		got := driverCategories(flags, enabledCell(cellText(row, enabledColumn)), deviceCategories)
		c := out[owner]
		c[0]++
		if slices.Contains(flags, "down") {
			c[1]++
		}
		if len(got) > 0 {
			c[2]++
		}
		out[owner] = c
	}
	return out
}

// textOrNull is a cell's text, or null when the station gave none ("", a null time, or NULL for a
// path the row's type does not have).
func textOrNull(cell string) any {
	if s := strings.TrimSpace(cell); s != "" && !strings.EqualFold(s, "null") {
		return s
	}
	return nil
}
