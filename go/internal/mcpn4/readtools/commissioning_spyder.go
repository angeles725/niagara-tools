package readtools

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The spyder section: the download state of each Honeywell BACnet Spyder below the scope
// (disabled, device_status, compile_status, error_status) and, with spyder_spec, io_mismatch,
// its IOComp configuration against the caller's table. Read-only: one BQL query, two with a
// spec. A Full Download can fail without a word in the station; these are the values that say
// it did not.

// spyderSection is the spyder section.
func (r *commRun) spyderSection() (*commSection, error) {
	devices, err := r.table("spyders")
	if err != nil {
		return nil, err
	}
	s := r.section("spyder", devices)
	for _, name := range spyderChecks {
		s.check(name)
	}
	if devices.failed != "" {
		s.notChecked(devices.failed, spyderChecks...)
		return s, nil
	}
	s.extra = []any{"spyders", len(devices.rows)}
	for _, row := range devices.rows {
		path, name := cellText(row, "Slot Path"), bql.UnescapeSlot(cellText(row, "Name"))
		if enabledCell(cellText(row, enabledColumn)) == false {
			s.check("disabled").add(path, name, factSpyderDisabled)
		}
		if got := strings.TrimSpace(cellText(row, "deviceStatus")); got != spyderGoodDeviceStatus {
			s.check("device_status").add(path, name, fmt.Sprintf(factSpyderStatus, "deviceStatus", cellOrEmpty(got), spyderGoodDeviceStatus))
		}
		if got := strings.TrimSpace(cellText(row, "compileStatusFlag")); got != spyderGoodCompileStatus {
			s.check("compile_status").add(path, name, fmt.Sprintf(factSpyderStatus, "compileStatusFlag", cellOrEmpty(got), spyderGoodCompileStatus))
		}
		if got := strings.TrimSpace(cellText(row, "sErrorStatus")); got != "" && got != nullCell {
			s.check("error_status").add(path, name, fmt.Sprintf(factSpyderError, bql.StripControl(got)))
		}
	}
	return s, r.spyderIO(s.check("io_mismatch"), devices)
}

// cellOrEmpty is a cell as a fact writes it: an empty cell is named, never left blank.
func cellOrEmpty(cell string) string {
	if cell == "" {
		return spyderEmptyCell
	}
	return cell
}

// spyderIO compares the IOComps of the Spyders in devices with spyder_spec into c. Without a
// spec the check is skipped and the IOComp table is not read.
func (r *commRun) spyderIO(c *commCheck, devices *commTable) error {
	if r.spyderSpec == nil {
		c.skipped = skipNoSpyderSpec
		return nil
	}
	io, err := r.table("spyder_io")
	switch {
	case err != nil:
		return err
	case io.failed != "":
		c.why = io.failed
		return nil
	case devices.cut:
		c.why = devices.cutWhy()
		return nil
	case io.cut:
		c.why = io.cutWhy()
		return nil
	}
	byName, paths := map[string]string{}, map[string]bool{}
	for _, row := range devices.rows {
		path := cellText(row, "Slot Path")
		byName[bql.UnescapeSlot(cellText(row, "Name"))], paths[path] = path, true
	}
	comps := spyderIOIndex(io, paths)
	for _, d := range r.spyderSpec {
		devicePath, ok := byName[d.name]
		if !ok {
			c.add("", d.name, fmt.Sprintf(factNoSpyder, d.name))
			continue
		}
		for _, p := range d.points {
			who := d.name + "/" + p.name
			row, ok := comps[devicePath+"\x00"+p.name]
			if !ok {
				c.add(devicePath, who, fmt.Sprintf(factNoIOComp, p.name))
			} else if diffs := p.diff(row); len(diffs) > 0 {
				c.add(cellText(row, "Slot Path"), who, strings.Join(diffs, "; "))
			}
		}
	}
	return nil
}

// spyderIOIndex keys each IOComp row by the Spyder holding it and its point's unescaped name;
// an IOComp outside every Spyder in paths is left out.
func spyderIOIndex(io *commTable, paths map[string]bool) map[string]*pyjson.Object {
	out := map[string]*pyjson.Object{}
	for _, row := range io.rows {
		path := cellText(row, "Slot Path")
		if lastSegment(path) != spyderIOSlot {
			continue
		}
		point := parentPath(path)
		if device := ownerOf(paths, point); device != "" {
			out[device+"\x00"+bql.UnescapeSlot(lastSegment(point))] = row
		}
	}
	return out
}
