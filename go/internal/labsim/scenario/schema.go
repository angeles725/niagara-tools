// Package scenario is the scenario file of lab-sim: its JSON shape, the
// validation that names every fault, and the construction of the simulated
// devices and their register maps from a valid file.
package scenario

// The values a scenario gets when it leaves a field out.
const (
	DefaultListen         = "127.0.0.1:1502"
	DefaultControl        = "127.0.0.1:1503"
	DefaultMaxConnections = 8
	DefaultIdleTimeoutMS  = 60000
	DefaultOrder          = "ABCD"
	DefaultBitType        = "bit"
	DefaultRegisterType   = "uint16"
)

// How a request for a unit id no device has is answered.
const (
	UnknownSilent  = "silent"            // no response, as a serial line behind a gateway
	UnknownGateway = "gateway_exception" // exception 0x0B, gateway target device failed to respond
)

// The behaviour kinds.
const (
	KindConstant   = "constant"
	KindRamp       = "ramp"
	KindSine       = "sine"
	KindRandomWalk = "random_walk"
	KindReplay     = "replay"
)

// What a replay does when its series ends.
const (
	EndHoldLast = "hold_last"
	EndLoop     = "loop"
)

// Limits of the scenario's own fields.
const (
	MinUnit          = 1
	MaxUnit          = 255
	MaxAddress       = 65535
	LimitConnections = 1024
)

// File is a scenario file.
type File struct {
	Name           string   `json:"name,omitempty"`
	Listen         string   `json:"listen,omitempty"`
	Control        string   `json:"control,omitempty"`
	UnknownUnit    string   `json:"unknown_unit,omitempty"`
	MaxConnections int      `json:"max_connections,omitempty"`
	IdleTimeoutMS  int      `json:"idle_timeout_ms,omitempty"`
	Devices        []Device `json:"devices"`

	dir string // where the files a scenario names are read from: the scenario's own directory
}

// Device is one simulated device, addressed by its Modbus unit id.
type Device struct {
	Name   string  `json:"name"`
	Unit   int     `json:"unit"`
	Faults *Faults `json:"faults,omitempty"`
	Points []Point `json:"points"`
	// BACnet, when set, also serves the device as a BACnet/IP device.
	BACnet *BACnetDevice `json:"bacnet,omitempty"`
}

// Point is one named value of a device and where it lives in the tables.
type Point struct {
	Name      string       `json:"name"`
	Table     string       `json:"table"`
	Address   int          `json:"address"`
	Type      string       `json:"type,omitempty"`
	Order     string       `json:"order,omitempty"`
	Value     float64      `json:"value,omitempty"`
	Writable  bool         `json:"writable,omitempty"`
	Behaviour *Behaviour   `json:"behaviour,omitempty"`
	Faults    *PointFaults `json:"faults,omitempty"`
	// BACnet, when set, is the object the point is in its BACnet device.
	BACnet *BACnetPoint `json:"bacnet,omitempty"`
}

// Behaviour makes a point's value change with time.
type Behaviour struct {
	Kind       string   `json:"kind"`
	Min        *float64 `json:"min,omitempty"`
	Max        *float64 `json:"max,omitempty"`
	Amplitude  float64  `json:"amplitude,omitempty"`
	PeriodMS   int      `json:"period_ms,omitempty"`
	Step       float64  `json:"step,omitempty"`
	IntervalMS int      `json:"interval_ms,omitempty"`
	Seed       *uint64  `json:"seed,omitempty"`
	// The fields of a replay: the CSV file (relative to the scenario), how many
	// times faster than recorded it plays (1 when left out), what it does at the
	// end of the series and how far into the series it starts.
	File          string   `json:"file,omitempty"`
	Speed         *float64 `json:"speed,omitempty"`
	End           string   `json:"end,omitempty"`
	StartOffsetMS int      `json:"start_offset_ms,omitempty"`

	series *Series // the replay file, read by Validate
}

// Faults are the failures of a device.
type Faults struct {
	FailAfterReads *int     `json:"fail_after_reads,omitempty"`
	DelayMS        int      `json:"delay_ms,omitempty"`
	Offline        []Window `json:"offline,omitempty"`
}

// PointFaults are the failures of one point.
type PointFaults struct {
	FailAfterReads *int `json:"fail_after_reads,omitempty"`
	DelayMS        int  `json:"delay_ms,omitempty"`
}

// Window is a span of milliseconds since the simulator started.
type Window struct {
	FromMS int `json:"from_ms"`
	ToMS   int `json:"to_ms"`
}
