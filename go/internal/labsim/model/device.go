package model

import (
	"sync"
	"time"
)

type point struct {
	writable bool
	gen      Generator
	faults   Faults
	reads    int
	override *float64 // a commanded value that hides the generator while set
}

// Device is one simulated device. Every method is safe for concurrent use.
type Device struct {
	name    string
	order   []string
	faults  Faults
	offline []Window

	mu     sync.Mutex
	points map[string]*point
	reads  int
}

// NewDevice builds a device from its specification.
func NewDevice(spec DeviceSpec) *Device {
	d := &Device{name: spec.Name, faults: spec.Faults, offline: spec.Offline, points: map[string]*point{}}
	for _, p := range spec.Points {
		gen := p.Generator
		if gen == nil {
			gen = constant(p.Initial)
		}
		d.order = append(d.order, p.Name)
		d.points[p.Name] = &point{writable: p.Writable, gen: gen, faults: p.Faults}
	}
	return d
}

// Name is the device's name in its scenario.
func (d *Device) Name() string { return d.name }

// Points lists the point names in scenario order.
func (d *Device) Points() []string { return append([]string(nil), d.order...) }

// Writable reports whether a protocol write to the point is accepted.
func (d *Device) Writable(name string) bool {
	p := d.points[name]
	return p != nil && p.writable
}

// Offline reports whether the device is inside one of its offline windows.
func (d *Device) Offline(elapsed time.Duration) bool {
	for _, w := range d.offline {
		if elapsed >= w.From && elapsed < w.To {
			return true
		}
	}
	return false
}

// Delay is how long a response that touches the named points is held back: the
// longest of the device's delay and theirs.
func (d *Device) Delay(points []string) time.Duration {
	delay := d.faults.Delay
	for _, name := range points {
		if p := d.points[name]; p != nil {
			delay = max(delay, p.faults.Delay)
		}
	}
	return delay
}

// Admit decides whether a request that touches the named points is served. It
// is refused once the device, or one of those points, has answered the reads
// its fail_after_reads allows; an admitted read is counted against them.
func (d *Device) Admit(points []string, read bool) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if spent(d.faults, d.reads) {
		return false
	}
	for _, name := range points {
		if p := d.points[name]; p != nil && spent(p.faults, p.reads) {
			return false
		}
	}
	if read {
		d.reads++
		for _, name := range points {
			if p := d.points[name]; p != nil {
				p.reads++
			}
		}
	}
	return true
}

func spent(f Faults, reads int) bool {
	return f.FailAfterReads != nil && reads >= *f.FailAfterReads
}

// Reads is the number of reads the device has admitted.
func (d *Device) Reads() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.reads
}

// Tx reads and writes points inside one Do call.
type Tx struct {
	d       *Device
	elapsed time.Duration
}

// Get is the point's value now: the override while one is set, else the field
// value; 0 for a point the device does not have.
func (tx Tx) Get(name string) float64 {
	if p := tx.d.points[name]; p != nil && p.override != nil {
		return *p.override
	}
	return tx.Field(name)
}

// Field is the value the point has by itself (constant, written, pinned or
// generated), whatever overrides it.
func (tx Tx) Field(name string) float64 {
	if p := tx.d.points[name]; p != nil {
		return p.gen.At(tx.elapsed)
	}
	return 0
}

// Override holds the point at *v over its field value until it is released
// with nil. It is how a protocol with command priorities (BACnet) shows the
// commanded value while the field value stays underneath.
func (tx Tx) Override(name string, v *float64) {
	if p := tx.d.points[name]; p != nil {
		p.override = nil
		if v != nil {
			held := *v
			p.override = &held
		}
	}
}

// Set fixes the point's field value at v from now on, replacing any generated
// behaviour; an override, if one is set, keeps hiding it.
func (tx Tx) Set(name string, v float64) {
	if p := tx.d.points[name]; p != nil {
		p.gen = constant(v)
	}
}

// Do runs fn with the device locked, so a request that spans several points
// reads or writes them as one.
func (d *Device) Do(elapsed time.Duration, fn func(Tx)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	fn(Tx{d: d, elapsed: elapsed})
}

// Get is one point's value at elapsed, and whether the device has the point.
func (d *Device) Get(name string, elapsed time.Duration) (float64, bool) {
	if d.points[name] == nil {
		return 0, false
	}
	var v float64
	d.Do(elapsed, func(tx Tx) { v = tx.Get(name) })
	return v, true
}

// Pin holds the point at v, whatever its behaviour was; it reports whether the
// device has the point. It is the control surface's write: it ignores Writable.
func (d *Device) Pin(name string, v float64) bool {
	if d.points[name] == nil {
		return false
	}
	d.Do(0, func(tx Tx) { tx.Set(name, v) })
	return true
}
