package readtools

import (
	"math"
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// healthSystem is the system section: the SystemService slots of systemSlots as n4_read_slots
// decodes them (whole numbers as numbers), plus memory_used_percent when both memory slots are
// numbers. The memory values are the station's, in the unit it reports them.
//
// The CPU usage slots are refreshed by the service's poll, which runs when the service is
// subscribed (BPlatformService.subscribed); until then they hold -1, their declared default,
// and a load leaves them out. A load subscribes the service, so when they are missing or -1
// the service is loaded once more; still unread, the section carries cpu_note.
func (t *Tools) healthSystem() (*pyjson.Object, error) {
	nodes, _, err := t.timedLoad(systemServiceOrd, 2)
	if err != nil {
		return t.sectionError(err)
	}
	out := pyjson.NewObject()
	if err := systemSlotsFrom(nodes, out, systemSlots); err != nil {
		return t.sectionError(err)
	}
	if out.Len() == 0 {
		return obj("error", systemNoSlots), nil
	}
	if !cpuRead(out) {
		if again, _, err := t.timedLoad(systemServiceOrd, 2); err == nil {
			if err := systemSlotsFrom(again, out, cpuSlots); err != nil {
				return t.sectionError(err)
			}
		} else if _, fatal := t.sectionError(err); fatal != nil {
			return nil, fatal
		}
		if !cpuRead(out) {
			out.Set("cpu_note", cpuNotMeasured)
		}
	}
	total, okT := lookup(out, totalMemorySlot).(int)
	free, okF := lookup(out, freeMemorySlot).(int)
	if okT && okF && total > 0 && free >= 0 && free <= total {
		out.Set("memory_used_percent", math.Round(float64(total-free)/float64(total)*1000)/10)
	}
	return out, nil
}

// systemSlotsFrom sets in out each of names that the load nodes holds.
func systemSlotsFrom(nodes, out *pyjson.Object, names []string) error {
	kids := box.Children(nodes, "")
	for _, name := range names {
		if box.Node(kids, name) == nil {
			continue
		}
		entry, err := slotEntry(nodes, name)
		if err != nil {
			return err
		}
		out.Set(name, wholeNumber(lookup(entry, "type"), lookup(entry, "value")))
	}
	return nil
}

// cpuRead reports whether every CPU usage slot is in out with a measured value (not -1).
func cpuRead(out *pyjson.Object) bool {
	for _, name := range cpuSlots {
		if v, ok := out.Get(name); !ok || v == -1 {
			return false
		}
	}
	return true
}

// wholeNumber is the value of a baja:Integer slot as a number (0 when the station omitted it:
// the type default); any other value is returned as it is.
func wholeNumber(kind, value any) any {
	if kind != integerType {
		return value
	}
	if value == nil {
		return 0
	}
	if s, ok := value.(string); ok {
		if n, err := strconv.Atoi(s); err == nil {
			return n
		}
	}
	return value
}
