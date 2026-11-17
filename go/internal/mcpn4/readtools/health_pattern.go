package readtools

import "github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"

// downPattern is the pattern hint of one network from its field devices: total, down (the down
// flag) and bad (in any device category: down, fault, disabled or unknown), or nil when no
// pattern applies:
//   - bus: 2 or more devices and all but at most one are down (2 down at least: one down device
//     says nothing about the bus), so the shared wiring is the likely cause;
//   - single: 2 or more devices and exactly one is bad, and it is down, so that device's own
//     address or power is the likely cause.
func downPattern(total, down, bad int) *pyjson.Object {
	switch {
	case total >= 2 && down >= 2 && down >= total-1:
		return obj("kind", patternBus, "down", down, "total", total, "hint", hintBus)
	case total >= 2 && down == 1 && bad == 1:
		return obj("kind", patternSingle, "down", down, "total", total, "hint", hintSingle)
	}
	return nil
}
