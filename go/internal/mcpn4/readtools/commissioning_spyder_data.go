package readtools

// What the spyder section of n4_commissioning_report reads and says (data only). The slot names
// are those of a Honeywell BACnet Spyder after a Full Download, read live on a client B field job
// (deviceStatus=1, compileStatusFlag=0, sErrorStatus empty, enabled true on every Spyder whose
// download succeeded); the IOComp slots are those of honeywellSpyderTool's BIOComponent
// (SensorType and the ranges are dynamic slots, Terminal1 is frozen, -1 unassigned).

// The two queries: the Spyders below the scope, and (only with spyder_spec) their IOComps.
const (
	spyderDeviceQuery = "select slotPath, name, enabled, deviceStatus, compileStatusFlag, sErrorStatus from honeywellBacnetSpyder:BacnetSpyder"
	spyderIOQuery     = "select slotPath, SensorType, InputLow, InputHigh, OutputLow, OutputHigh, Terminal1 from honeywellSpyderTool:IOComponent"
)

var (
	spyderDeviceColumns = []string{"Slot Path", "Name", enabledColumn, "deviceStatus", "compileStatusFlag", "sErrorStatus"}
	spyderIOColumns     = []string{"Slot Path", "SensorType", "InputLow", "InputHigh", "OutputLow", "OutputHigh", "Terminal1"}
)

// spyderSpecFields maps a field of a spyder_spec point to the IOComp column it is compared with,
// in the order a mismatch fact lists them.
var spyderSpecFields = [][2]string{{"sensor_type", "SensorType"}, {"input_low", "InputLow"}, {"input_high", "InputHigh"},
	{"output_low", "OutputLow"}, {"output_high", "OutputHigh"}, {"terminal", "Terminal1"}}

// The download state a Spyder must read back after a successful Full Download.
const (
	spyderGoodDeviceStatus  = "1"
	spyderGoodCompileStatus = "0"
	spyderIOSlot            = "IOComp"
)

// The checks of the section, in reply order.
var spyderChecks = []string{"disabled", "device_status", "compile_status", "error_status", "io_mismatch"}

// Messages of the spyder section.
const (
	commBadSpyderSpec = "spyder_spec must be an object {<Spyder name>: {<point name>: {<field>: <number or text>}}} with fields among %s: %s"

	factSpyderDisabled = "Enabled=false: a Spyder left disabled after a download is neither polled nor pinged; set Enabled=true, Ping, then read this again"
	factSpyderStatus   = "%s %s, want %s"
	factSpyderError    = "sErrorStatus %q"
	factSpyderMismatch = "%s %s, want %s"
	factNoSpyder       = "no Spyder named %s in scope"
	factNoIOComp       = "no point %s with an IOComp below this Spyder"
	spyderEmptyCell    = "(empty)"

	skipNoSpyderSpec = "spyder_spec not given: the expected IO of each Spyder is the caller's to say"
)
