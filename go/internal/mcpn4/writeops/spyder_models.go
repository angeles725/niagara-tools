package writeops

import (
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// spyderModel is one Honeywell BACnet Spyder hardware model.
type spyderModel struct{ name, io string }

// spyderModels maps the hardware id a Spyder publishes as analogValue:0 "AV_HardwareID" to its
// model, as honeywellBacnetSpyder XL10NextGenXML/BacnetSpyder.xml declares them (hw_id,
// displayName, ModelDescription). The station's download refuses a database model whose hw_id
// differs (BBacnetSpyder.readControllerModelFromDevice, BSpyderDownloadUtility.checkModel).
var spyderModels = map[int]spyderModel{
	31: {"PVB6436AS", "6 Universal Inputs, 4 Binary Inputs, 3 Modulating Outputs, 6 Binary Outputs"},
	32: {"PVB6438NS", "6 Universal Inputs, 4 Binary Inputs, 3 Modulating Outputs, 8 Binary Outputs"},
	33: {"PUB6438S", "6 Universal Inputs, 4 Binary Inputs, 3 Modulating Outputs, 8 Binary Outputs"},
	34: {"PUB6438SR", "6 Universal Inputs, 4 Binary Inputs, 3 Modulating Outputs, 8 Binary Outputs"},
	51: {"PVB4024NS", "4 Universal Inputs, 2 Modulating Outputs, 4 Binary Outputs"},
	52: {"PVB4022AS", "4 Universal Inputs, 2 Modulating Outputs, 2 Binary Outputs"},
	53: {"PUB4024S", "4 Universal Inputs, 2 Modulating Outputs, 4 Binary Outputs"},
	54: {"PUB1012S", "1 Universal Inputs, 1 Modulating Outputs, 2 Binary Outputs"},
	55: {"PVB0000AS", "no wired I/O (VAV actuator and Sylk bus only)"},
}

const (
	hardwareIDObject = "analogValue:0"
	hardwareIDName   = "hardware" // AV_HardwareID (PUB6438S), AV0MObjHardwareI (PUB1012S)
	unknownDescPfx   = "Property:Unknown Property:"
)

// spyderHardware is the model a discovered analogValue:0 hardware-id row names (its name holds
// "hardware": AV_HardwareID on a PUB6438S, AV0MObjHardwareI on a PUB1012S) (nil for other rows); an
// unknown id keeps model null.
func spyderHardware(name, id string, value any) *pyjson.Object {
	if id != hardwareIDObject || !strings.Contains(strings.ToLower(name), hardwareIDName) {
		return nil
	}
	f, err := strconv.ParseFloat(pyjson.Str(value), 64)
	if err != nil || f != float64(int(f)) {
		return nil
	}
	hw := int(f)
	m, ok := spyderModels[hw]
	if !ok {
		return obj("hw_id", hw, "model", nil, "io", nil)
	}
	return obj("hw_id", hw, "model", m.name, "io", m.io)
}

// cleanDescription drops the text the station writes when an object has no Description property.
func cleanDescription(d string) string {
	if strings.HasPrefix(d, unknownDescPfx) {
		return ""
	}
	return d
}
