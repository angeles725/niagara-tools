package readtools

// downPattern is the pattern hint of one network from its field devices: total, down (the down
// flag) and bad (in any device category: down, fault, disabled or unknown), or nil when no
// pattern applies:
//   - bus: 2 or more devices and all but at most one are down (2 down at least: one down device
//     says nothing about the bus), so the shared link is the likely cause: the wiring on a serial
//     network (one with a port), else the network path (IP networks have no wiring to check);
//   - single: 2 or more devices and exactly one is bad, and it is down, so that device's own
//     address or power is the likely cause.
//
// It returns any so that "no pattern" is an untyped nil: a typed nil *pyjson.Object in a reply
// crashes the encoder.
func downPattern(total, down, bad int, serial bool) any {
	switch {
	case total >= 2 && down >= 2 && down >= total-1:
		hint := hintNetwork
		if serial {
			hint = hintBus
		}
		return obj("kind", patternBus, "down", down, "total", total, "hint", hint)
	case total >= 2 && down == 1 && bad == 1:
		return obj("kind", patternSingle, "down", down, "total", total, "hint", hintSingle)
	}
	return nil
}
