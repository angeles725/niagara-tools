// Package model is the value model of lab-sim, free of any field protocol: a
// simulated device is a set of named points, each with a value that is constant,
// written, or generated from the time since the simulator started, plus the
// faults a scenario asks for. A protocol front end (Modbus today) maps its own
// addresses onto these points.
package model

import "time"

// Faults are the failures a scenario injects into a device or into one point.
type Faults struct {
	// FailAfterReads, when set, makes the subject fail once that many reads
	// were answered: every later request that touches it is refused.
	FailAfterReads *int
	// Delay is how long a response that touches the subject is held back.
	Delay time.Duration
}

// Window is a span of time since the simulator started, From included and To
// excluded.
type Window struct {
	From, To time.Duration
}

// PointSpec describes one point of a device.
type PointSpec struct {
	Name      string
	Initial   float64
	Writable  bool      // a protocol write is accepted
	Generator Generator // nil: the value stays at Initial until it is written
	Faults    Faults
}

// DeviceSpec describes one simulated device.
type DeviceSpec struct {
	Name    string
	Points  []PointSpec
	Faults  Faults
	Offline []Window // while inside one, the device does not answer at all
}
