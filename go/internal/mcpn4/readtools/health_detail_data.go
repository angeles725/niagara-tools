package readtools

// What n4_station_health device_detail=true reads and says (data only). Slot facts, from the
// decompiled driver-rt, bacnet-rt, modbusCore-rt, modbusTcp-rt and serial-rt sources:
//   - BDevice.faultCause (BDevice.java:176) and BDevice.health, a BPingHealth (BDevice.java:202)
//     with lastOkTime (BPingHealth.java:159), lastFailTime (:188) and lastFailCause (:215). A ping
//     failure sets down and lastFailCause but never faultCause, which only a config fault fills.
//   - The address of a device is driver-specific: BModbusDevice.deviceAddress
//     (modbusCore BModbusDevice.java:30; BModbusTcpDevice.java:58) and BBacnetDevice.address, a
//     BBacnetAddress shown as network:mac (bacnet-rt BBacnetDevice.java:294).
//   - A serial network's port is serialPortConfig.portName, a BSerialHelper slot
//     (BSerialHelper.java:136) on BSerialNetwork (:38), BModbusAsyncNetwork (:80) and
//     BAbstractMbusNetwork (:238). A path a row does not have answers an empty cell (BNull).

// detailDevicesQuery reads, for every device in one request, what a technician needs to fix a
// bad one; detailNetworksQuery reads every network's serial port in one more.
const (
	detailDevicesQuery = "select slotPath, type, faultCause, health.lastOkTime, health.lastFailTime, " +
		"health.lastFailCause, deviceAddress, address from driver:Device"
	detailNetworksQuery = "select slotPath, serialPortConfig.portName from driver:DeviceNetwork"
)

// The columns of the detail queries by position; the address columns are last, so an answer
// without them still places the others.
var (
	detailDeviceColumns  = []string{"Slot Path", "Type", "FaultCause", "LastOk", "LastFail", "LastFailCause", "DeviceAddress", "Address"}
	detailNetworkColumns = []string{"Slot Path", "Port"}
)

// detailFixedColumns is how many detail columns every driver:Device row has (the BDevice slots).
const detailFixedColumns = 6

// addressColumns is the known per-type address slot: a device type whose module is listed reports
// its address from that column; any other type reports null, never a guess.
var addressColumns = map[string]string{
	"modbusAsync": "DeviceAddress",
	"modbusTcp":   "DeviceAddress",
	"modbusCore":  "DeviceAddress",
	"bacnet":      "Address",
}

// The pattern hints of a network with down devices.
const (
	patternBus      = "bus"
	patternSingle   = "single"
	hintBus         = "check bus wiring, termination, polarity and the port"
	hintNetwork     = "check the shared network path: the station's interface, switches or routers, and the devices' power"
	hintSingle      = "check that device's address and power"
	deviceDetailArg = "device_detail"
)

// The notes of device_detail.
const (
	detailNeedsDrivers = "device_detail=true needs drivers=true: the detail extends the drivers section"
	detailFailedNote   = "the device detail query failed (%s): its fields are null"
	detailNoAddress    = "the device detail answer has no address columns: address is null"
	detailPortNote     = "the network port query failed (%s): port is null"
)
