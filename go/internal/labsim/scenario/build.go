package scenario

import (
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/model"
	"github.com/angeles725/niagara-tools/go/internal/labsim/regmap"
)

// Built is a scenario turned into running state.
type Built struct {
	Devices []*model.Device       // in scenario order
	Units   map[byte]*regmap.Unit // the Modbus register map of each unit id
	BACnet  []BACnetEnd           // the devices that are BACnet devices too, in scenario order
}

// Build makes the devices and register maps of a scenario Validate found no
// fault in.
func (f *File) Build() Built {
	b := Built{Units: map[byte]*regmap.Unit{}}
	for i := range f.Devices {
		d := &f.Devices[i]
		spec := model.DeviceSpec{Name: d.Name}
		if d.Faults != nil {
			spec.Faults = model.Faults{FailAfterReads: d.Faults.FailAfterReads, Delay: millis(d.Faults.DelayMS)}
			for _, w := range d.Faults.Offline {
				spec.Offline = append(spec.Offline, model.Window{From: millis(w.FromMS), To: millis(w.ToMS)})
			}
		}
		bindings := make([]regmap.Binding, 0, len(d.Points))
		for j := range d.Points {
			p := &d.Points[j]
			spec.Points = append(spec.Points, pointSpec(p))
			table, _ := regmap.ParseTable(p.Table)
			bindings = append(bindings, regmap.Binding{
				Point: p.Name, Table: table, Address: uint16(p.Address),
				Type: regmap.Type(p.Type), Order: regmap.Order(p.Order),
			})
		}
		dev := model.NewDevice(spec)
		b.Devices = append(b.Devices, dev)
		b.Units[byte(d.Unit)] = regmap.NewUnit(dev, bindings)
		if d.BACnet != nil {
			b.BACnet = append(b.BACnet, bacnetEnd(d, dev))
		}
	}
	return b
}

func pointSpec(p *Point) model.PointSpec {
	spec := model.PointSpec{Name: p.Name, Initial: p.Initial(), Writable: p.Writable, Generator: generator(p)}
	if p.Faults != nil {
		spec.Faults = model.Faults{FailAfterReads: p.Faults.FailAfterReads, Delay: millis(p.Faults.DelayMS)}
	}
	return spec
}

// Initial is the value the point has when the simulator starts: its value, the
// low end of a ramp, or where a replay starts.
func (p *Point) Initial() float64 {
	if p.Behaviour != nil && p.Behaviour.Kind == KindRamp && p.Behaviour.Min != nil {
		return *p.Behaviour.Min
	}
	if p.Behaviour != nil && p.Behaviour.Kind == KindReplay && p.Behaviour.series != nil {
		return replayGenerator(p.Behaviour).At(0)
	}
	return p.Value
}

// generator is the point's behaviour; nil for a constant.
func generator(p *Point) model.Generator {
	b := p.Behaviour
	if b == nil {
		return nil
	}
	switch b.Kind {
	case KindRamp:
		return model.Ramp(*b.Min, *b.Max, millis(b.PeriodMS))
	case KindSine:
		return model.Sine(p.Value, b.Amplitude, millis(b.PeriodMS))
	case KindRandomWalk:
		return model.RandomWalk(*b.Seed, p.Value, b.Step, millis(b.IntervalMS), *b.Min, *b.Max)
	case KindReplay:
		return replayGenerator(b)
	}
	return nil
}

func millis(n int) time.Duration { return time.Duration(n) * time.Millisecond }
