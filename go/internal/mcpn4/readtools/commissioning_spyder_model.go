package readtools

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
)

// The model_mismatch check of the spyder section (issue #820): each Honeywell BACnet Spyder's
// configured model against the hardware id its analogValue:0 point reads. Read-only, two BQL
// queries, only with spyder_hardware. The hardware id exists only on the controller (BBacnetSpyder.readControllerModelFromDevice
// reads the presentValue of analogValue:0, BBacnetSpyder.java:2871-2880), so its only station-side
// copy is a point proxying that object (BBacnetProxyExt.objectId, BBacnetProxyExt.java:131); a
// Spyder with no such point is counted, never judged. [INFER] The two selects were not run on a
// live station: a station that refuses one answers it as not_checked, never as a clean section.

const (
	spyderModelQuery = "select slotPath, name, controllerModel from honeywellBacnetSpyder:BacnetSpyder"
	spyderHWQuery    = "select slotPath, proxyExt.objectId, out.value from control:NumericPoint"
	spyderHWObject   = "analogValue:0"
)

var (
	spyderModelColumns = []string{"Slot Path", "Name", "Model"}
	spyderHWColumns    = []string{"Slot Path", "Object Id", "Value"}
)

// spyderHardware maps the hardware id to the model's display name and the name the station
// stores in controllerModel (honeywellBacnetSpyder BacnetSpyder.xml hw_id, displayName, model).
var spyderHardware = map[int][2]string{
	31: {"PVB6436AS", "ModelBACnet1"}, 32: {"PVB6438NS", "ModelBACnet2"}, 33: {"PUB6438S", "ModelBACnet3"},
	34: {"PUB6438SR", "ModelBACnet4"}, 51: {"PVB4024NS", "ModelMicroBACnet1"}, 52: {"PVB4022AS", "ModelMicroBACnet2"},
	53: {"PUB4024S", "ModelMicroBACnet3"}, 54: {"PUB1012S", "ModelMicroBACnet4"}, 55: {"PVB0000AS", "ModelMicroBACnet5"},
}

const (
	factModelMismatch = "configured model %s, but analogValue:0 reads hardware id %d: %s"
	factModelUnknown  = "configured model %s, but analogValue:0 reads hardware id %d, which is in no known model"
)

// spyderModel adds the model_mismatch check to the spyder section s (only with spyder_hardware):
// the Spyders of the model query against the hardware ids of their analogValue:0 points.
func (r *commRun) spyderModel(s *commSection) error {
	c := s.check("model_mismatch")
	devices, err := r.table("spyder_model")
	if err != nil {
		return err
	}
	if devices.failed != "" {
		c.why = devices.failed
		return nil
	}
	hw, err := r.table("spyder_hw")
	if err != nil {
		return err
	}
	if hw.failed != "" {
		c.why = hw.failed
		return nil
	}
	for _, tb := range []*commTable{devices, hw} {
		if tb.cut && s.truncated == 0 {
			s.truncated = max(len(tb.rows), 1)
		}
	}
	ids, unreadable := spyderHardwareIDs(hw, devices)
	for _, row := range devices.rows {
		path, name := cellText(row, "Slot Path"), bql.UnescapeSlot(cellText(row, "Name"))
		if id, ok := ids[path]; ok {
			if fact := modelFact(strings.TrimSpace(cellText(row, "Model")), id); fact != "" {
				c.add(path, name, fact)
			}
		}
	}
	s.extra = append(s.extra, "with_hardware_point", len(ids)+unreadable, "hardware_unreadable", unreadable)
	return nil
}

// spyderHardwareIDs is the hardware id each Spyder's analogValue:0 point reads, by Spyder path,
// and how many Spyders have that point with a value that is no whole number (a point in fault).
func spyderHardwareIDs(hw, devices *commTable) (map[string]int, int) {
	paths := map[string]bool{}
	for _, row := range devices.rows {
		paths[cellText(row, "Slot Path")] = true
	}
	ids, seen, unreadable := map[string]int{}, map[string]bool{}, 0
	for _, row := range hw.rows {
		if strings.TrimSpace(cellText(row, "Object Id")) != spyderHWObject {
			continue
		}
		device := ownerOf(paths, cellText(row, "Slot Path"))
		if device == "" || seen[device] {
			continue
		}
		seen[device] = true
		if id, ok := wholeCell(cellText(row, "Value")); ok {
			ids[device] = id
		} else {
			unreadable++
		}
	}
	return ids, unreadable
}

// wholeCell is the whole number a cell starts with ("54.0 {ok}" is 54).
func wholeCell(cell string) (int, bool) {
	words := strings.Fields(cell)
	if len(words) == 0 {
		return 0, false
	}
	f, err := strconv.ParseFloat(words[0], 64)
	if err != nil || f != float64(int(f)) {
		return 0, false
	}
	return int(f), true
}

// modelFact is what is wrong with configured (the stored controllerModel) given the hardware id
// the controller reads ("" when they agree).
func modelFact(configured string, id int) string {
	want, known := spyderHardware[id]
	if known && want[1] == configured {
		return ""
	}
	shown := configured
	for _, m := range spyderHardware {
		if m[1] == configured {
			shown = fmt.Sprintf("%s (%s)", m[0], configured)
		}
	}
	if !known {
		return fmt.Sprintf(factModelUnknown, shown, id)
	}
	return fmt.Sprintf(factModelMismatch, shown, id, fmt.Sprintf("%s (%s)", want[0], want[1]))
}
