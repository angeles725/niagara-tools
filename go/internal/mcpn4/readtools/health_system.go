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
func (t *Tools) healthSystem() (*pyjson.Object, error) {
	nodes, _, err := t.timedLoad(systemServiceOrd, 2)
	if err != nil {
		return t.sectionError(err)
	}
	out, kids := pyjson.NewObject(), box.Children(nodes, "")
	for _, name := range systemSlots {
		if box.Node(kids, name) == nil {
			continue
		}
		entry, err := slotEntry(nodes, name)
		if err != nil {
			return t.sectionError(err)
		}
		out.Set(name, wholeNumber(lookup(entry, "type"), lookup(entry, "value")))
	}
	if out.Len() == 0 {
		return obj("error", systemNoSlots), nil
	}
	total, okT := lookup(out, totalMemorySlot).(int)
	free, okF := lookup(out, freeMemorySlot).(int)
	if okT && okF && total > 0 && free >= 0 && free <= total {
		out.Set("memory_used_percent", math.Round(float64(total-free)/float64(total)*1000)/10)
	}
	return out, nil
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
