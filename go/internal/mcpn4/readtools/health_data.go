package readtools

// What n4_station_health reads and says (data only).

// systemServiceOrd is the station's SystemService; systemSlots are the slots of it the system
// section carries, by their station names, in reply order.
const systemServiceOrd = "station:|slot:/Services/PlatformServices/SystemService"

var systemSlots = []string{"stationName", "hostId", "model", "niagaraVersion", "osName", "javaVmVersion",
	"systemTime", "numCpus", "currentCpuUsage", "overallCpuUsage", "totalPhysicalMemory", "freePhysicalMemory"}

// The memory slots the derived percentage reads, and the BOX type whose values are whole numbers
// (a station omits one at its default, 0).
const (
	totalMemorySlot = "totalPhysicalMemory"
	freeMemorySlot  = "freePhysicalMemory"
	integerType     = "baja:Integer"
)

// healthQuery is one status query of the health tool: the section it fills and its select.
// status.bits is the status as a number, which no locale translates.
type healthQuery struct {
	section, query string
	enabled        bool // the query also selects enabled
}

var healthQueries = []healthQuery{
	{"networks", "select slotPath, name, type, status.bits, enabled from driver:DeviceNetwork", true},
	{"devices", "select slotPath, name, type, status.bits, enabled from driver:Device", true},
	{"points", "select slotPath, name, type, status.bits from control:ControlPoint", false},
}

// The columns of a health query by position: the inventory's, then the status bits and enabled.
const (
	bitsColumn    = "Bits"
	enabledColumn = "Enabled"
)

// unknownFlag is the flag of a row whose status bits the station did not answer as a number.
const unknownFlag = "unknown"

// The spy page capacity is read from and the notes of a null capacity.
const (
	capacityPage      = "metrics"
	capacitySource    = "spy:/" + capacityPage
	capacityNoTable   = "the spy metrics page has no Global Capacity table with Type, Limit and Used columns: the station has no global capacity licensing, or this Niagara version lays the page out differently (read it with n4_spy_page)"
	capacityNoPage    = "the spy metrics page is unreadable (status %s, HTTP %v): %s"
	capacityNoPageWhy = "capacity is read from " + capacitySource + ", which needs a station user allowed to read spy pages"
	capacityFailed    = "the spy metrics page could not be read: %s"
	totalsNote        = "the inventory totals need the networks, devices and points sections; failed: %s"
	systemNoSlots     = "the SystemService at " + systemServiceOrd + " has none of the expected slots"
)
