// Package control is the control surface of a running lab-sim: a loopback-only
// TCP port on which a test reads or sets a point, asks the status or stops the
// simulator, without going through the field protocol under test. One
// connection carries one request line and one response line, both JSON.
//
// The port has no authentication. That is acceptable only because it never
// leaves the loopback interface: Listen and Call refuse any other address.
package control

import "time"

// The operations.
const (
	OpGet    = "get"
	OpSet    = "set"
	OpStatus = "status"
	OpStop   = "stop"
)

// Limits of one exchange.
const (
	maxLine         = 4096 // bytes of a request line
	exchangeTimeout = 5 * time.Second
)

// The errors a request gets, as text.
const (
	errNotRequest = "not a control request (one JSON object on one line): %v"
	errUnknownOp  = "unknown op %q"
	errNoDevice   = "no device %q (devices: %s)"
	errNoPoint    = "device %q has no point %q (points: %s)"
	errNoValue    = "set needs a value"
)

// Request is one control request.
type Request struct {
	Op     string   `json:"op"`
	Device string   `json:"device,omitempty"`
	Point  string   `json:"point,omitempty"`
	Value  *float64 `json:"value,omitempty"`
}

// Response is the answer to a Request.
type Response struct {
	OK     bool     `json:"ok"`
	Error  string   `json:"error,omitempty"`
	Value  *float64 `json:"value,omitempty"`  // get and set: the point's value now
	Status *Status  `json:"status,omitempty"` // status
}

// Status describes a running simulator.
type Status struct {
	Scenario    string         `json:"scenario"`
	ElapsedMS   int64          `json:"elapsed_ms"`
	Listen      string         `json:"listen"`  // the field protocol's address
	Control     string         `json:"control"` // this port
	Requests    int64          `json:"requests"`
	Connections int            `json:"connections"`
	Refused     int64          `json:"refused"`
	Devices     []DeviceStatus `json:"devices"`
}

// DeviceStatus is one simulated device in a Status.
type DeviceStatus struct {
	Name    string `json:"name"`
	Unit    int    `json:"unit"`
	Points  int    `json:"points"`
	Reads   int    `json:"reads"` // reads it has answered
	Offline bool   `json:"offline"`
}
