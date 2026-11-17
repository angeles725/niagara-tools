package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// targetRegisters notes the copies of writable Modbus points that would share a register address
// on target t (the station resets each later one to Hex:-1, copy_duplicate.go): two points
// replicated onto the same register, or, when t is the source parent itself, a copy at a source's
// address. It returns that address by source point name, for the read-back's cause. Points already
// on t before the replication are not read.
func targetRegisters(t replicaTarget, points []string, sources map[string]*copySource, srcOrd string) ([]any, map[string]string) {
	used, shared := map[string]bool{}, map[string]string{}
	if t.ord == srcOrd {
		for _, p := range points {
			if body := sources[p].body; writableModbus(body) {
				used[flexText(sourceAddress(body))] = true
			}
		}
	}
	order, byAddress := []string{}, map[string][]string{}
	for _, p := range points {
		body := sources[p].body
		if !writableModbus(body) {
			continue
		}
		format, address := sourceAddress(body)
		if ov, ok := t.addrOf[p]; ok {
			address = ov.address
			if ov.format != "" {
				format = ov.format
			}
		}
		text := flexText(format, address)
		if !used[text] {
			used[text] = true
			continue
		}
		if byAddress[text] == nil {
			order = append(order, text)
		}
		byAddress[text] = append(byAddress[text], pyjson.StrRepr(t.newName(p)))
		shared[p] = text
	}
	notes := []any{}
	for _, text := range order {
		notes = append(notes, t.ord+": "+strings.Join(byAddress[text], ", ")+" would use the register address "+
			text+", which a source point or an earlier copy already uses on that device, and the device allows "+
			"one writable point per register: the station resets the copy's proxyExt/dataAddress to Hex:-1 "+
			"(station log \"Duplicate Writable point for register address\"). Give it its own address with "+
			"copies[].data_address of that target.")
	}
	return notes, shared
}
