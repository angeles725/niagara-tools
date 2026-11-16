package scenario

import "github.com/angeles725/niagara-tools/go/internal/labsim/regmap"

// span is the addresses a validated point occupies in its table.
type span struct {
	table       regmap.Table
	first, last int
	name        string
}

// validatePoints checks each point of a device and that no two share an address.
func validatePoints(d *Device, dir, deviceWhere string, out *faults) {
	names := map[string]bool{}
	var placed []span
	for i := range d.Points {
		p := &d.Points[i]
		where := deviceWhere + ", " + label("point", i, p.Name)
		switch {
		case p.Name == "":
			out.add(where, msgNoName)
		case names[p.Name]:
			out.add(where, msgNameUsed, p.Name)
		}
		names[p.Name] = true
		if p.Faults != nil {
			validateCounters(p.Faults.FailAfterReads, p.Faults.DelayMS, where, out)
		}
		s, ok := validatePlace(p, where, out)
		if !ok {
			continue
		}
		for _, other := range placed {
			if other.table == s.table && s.first <= other.last && other.first <= s.last {
				out.add(where, msgOverlap, s.first, s.last, p.Table, other.name)
				break
			}
		}
		placed = append(placed, s)
		validateValue(p, dir, where, out)
	}
}

// validatePlace checks table, type, order and address; it returns the point's
// span when they are sound enough to compare with the other points.
func validatePlace(p *Point, where string, out *faults) (span, bool) {
	table, ok := regmap.ParseTable(p.Table)
	if !ok {
		out.add(where, msgTable, p.Table)
		return span{}, false
	}
	typ := regmap.Type(p.Type)
	if !typ.Known() {
		out.add(where, msgType, p.Type)
		return span{}, false
	}
	if table.Bits() != (typ == regmap.Bit) {
		out.add(where, msgTypeTable, p.Type, p.Table)
		return span{}, false
	}
	if !regmap.Order(p.Order).Known() {
		out.add(where, msgOrder, p.Order)
	}
	if p.Writable && !table.Writable() {
		out.add(where, msgReadOnlyTable, p.Table)
	}
	last := p.Address + typ.Words() - 1
	if p.Address < 0 || last > MaxAddress {
		out.add(where, msgAddress, p.Address, typ.Words())
		return span{}, false
	}
	return span{table: table, first: p.Address, last: last, name: p.Name}, true
}

// validateValue checks the initial value and the behaviour against the type.
func validateValue(p *Point, dir, where string, out *faults) {
	typ := regmap.Type(p.Type)
	b := p.Behaviour
	if b == nil || b.Kind == KindConstant {
		if !typ.Fits(p.Value) {
			out.add(where, msgValue, p.Value, p.Type)
		}
		return
	}
	switch {
	case b.Kind != KindRamp && b.Kind != KindSine && b.Kind != KindRandomWalk && b.Kind != KindReplay:
		out.add(where, msgBehaviourKind, b.Kind)
	case typ == regmap.Bit:
		out.add(where, msgBehaviourBit, b.Kind)
	case p.Writable:
		out.add(where, msgBehaviourWritable, b.Kind)
	default:
		if low, high, ok := behaviourRange(p, dir, where, out); ok && (!typ.Fits(low) || !typ.Fits(high)) {
			out.add(where, msgBehaviourRange, b.Kind, low, high, p.Type)
		}
	}
}

// behaviourRange checks a behaviour's own parameters and returns the lowest and
// highest value it can reach.
func behaviourRange(p *Point, dir, where string, out *faults) (low, high float64, ok bool) {
	b := p.Behaviour
	if b.Kind == KindReplay {
		return replayRange(b, dir, where, out)
	}
	before := len(out.list)
	bounded := b.Min != nil && b.Max != nil && *b.Min < *b.Max
	switch b.Kind {
	case KindSine:
		if b.PeriodMS <= 0 {
			out.add(where, msgSinePeriod)
		}
		if b.Amplitude <= 0 {
			out.add(where, msgSineAmplitude)
		}
		return p.Value - b.Amplitude, p.Value + b.Amplitude, len(out.list) == before
	case KindRamp:
		if !bounded {
			out.add(where, msgRampBounds)
		}
		if b.PeriodMS <= 0 {
			out.add(where, msgRampPeriod)
		}
	case KindRandomWalk:
		if b.Seed == nil {
			out.add(where, msgWalkSeed)
		}
		if b.Step <= 0 {
			out.add(where, msgWalkStep)
		}
		if b.IntervalMS <= 0 {
			out.add(where, msgWalkInterval)
		}
		if !bounded {
			out.add(where, msgWalkBounds)
		} else if p.Value < *b.Min || p.Value > *b.Max {
			out.add(where, msgWalkStart)
		}
	}
	if len(out.list) != before {
		return 0, 0, false
	}
	return *b.Min, *b.Max, true
}
