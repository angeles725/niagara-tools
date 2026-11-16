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
	proxy          bool // the query also selects the type of the proxy extension
}

var healthQueries = []healthQuery{
	{"networks", "select slotPath, name, type, status.bits, enabled from driver:DeviceNetwork", true, false},
	{"devices", "select slotPath, name, type, status.bits, enabled from driver:Device", true, false},
	{"points", "select slotPath, name, type, status.bits, proxyExt.type from control:ControlPoint", false, true},
}

// The columns of a health query by position: the inventory's, then the status bits and enabled.
const (
	bitsColumn    = "Bits"
	enabledColumn = "Enabled"
	proxyColumn   = "Proxy"
)

// nullProxyExt is the proxy extension of a control point that is no proxy point: the licence
// counts proxy points only.
const nullProxyExt = "control:NullProxyExt"

// capacityLive maps a Global Capacity resource (lower case) to the section whose count is what
// the station really holds of it; the points count is the proxy points.
var capacityLive = map[string]string{"networks": "networks", "devices": "devices", "points": "points"}

// capacityMaxRows is the row cap of the metrics page read: more rows than the byte cap of a
// spy read can hold, so the row cap never hides a resource.
const capacityMaxRows = 300000

// unknownFlag is the flag of a row whose status bits the station did not answer as a number;
// nullFlag is the station's flag of a status with no value, which alone is not a fault.
const (
	unknownFlag = "unknown"
	nullFlag    = "null"
)

// The spy page capacity is read from and the notes of a null capacity.
const (
	capacityPage      = "metrics"
	capacitySource    = "spy:/" + capacityPage
	capacityNoTable   = "the spy metrics page has no Global Capacity table with Type, Limit and Used columns: the station has no global capacity licensing, or this Niagara version lays the page out differently (read it with n4_spy_page)"
	capacityCut       = "the spy metrics page was cut (%s) before its Global Capacity table was read whole: resources may be missing, read it with n4_spy_page"
	capacityCutNone   = "the spy metrics page was cut (%s) before a Global Capacity table was read: the capacity is unknown, not absent"
	capacitySkipped   = "%d row(s) of the Global Capacity table did not match its header and are not listed"
	inflationNote     = "The licence tally only grows: removing a component does not decrement it until a capacity recount or a station restart, so used can exceed what the station holds (live)."
	capacityNoPage    = "the spy metrics page is unreadable (status %s, HTTP %v): %s"
	capacityNoPageWhy = "capacity is read from " + capacitySource + ", which needs a station user allowed to read spy pages"
	capacityFailed    = "the spy metrics page could not be read: %s"
	totalsNote        = "the inventory totals need the networks, devices and points sections; failed: %s"
	systemNoSlots     = "the SystemService at " + systemServiceOrd + " has none of the expected slots"
)
