package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// targetRegisters notes the copies of writable Modbus points that would share a register address
// on target t (the station resets each later one to Hex:-1, copy_duplicate.go): a copy at the
// register a writable point already on t holds (held, retro 2026-11-17 Δ6), two points replicated
// onto the same register, or, when t is the source parent itself, a copy at a source's address. It
// returns that address by source point name, for the read-back's cause.
func targetRegisters(t replicaTarget, points []string, sources map[string]*copySource, srcOrd string, held map[string]string) ([]any, map[string]string) {
	used, shared := map[string]bool{}, map[string]string{}
	if t.ord == srcOrd {
		for _, p := range points {
			if body := sources[p].body; writableModbus(body) {
				used[flexText(sourceAddress(body))] = true
			}
		}
	}
	order, byAddress, notes := []string{}, map[string][]string{}, []any{}
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
		if address == unassigned {
			continue // no register yet: nothing to share
		}
		text := flexText(format, address)
		if holder, ok := held[text]; ok {
			notes = append(notes, t.ord+": "+pyjson.StrRepr(t.newName(p))+" would use the register address "+
				text+", which the writable point "+pyjson.StrRepr(holder)+" already on that device uses"+oneWritable)
			shared[p] = text
			continue
		}
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
	for _, text := range order {
		notes = append(notes, t.ord+": "+strings.Join(byAddress[text], ", ")+" would use the register address "+
			text+", which a source point or an earlier copy already uses on that device"+oneWritable)
	}
	return notes, shared
}

// oneWritable is the rule and remedy every shared-register note ends with.
const oneWritable = ", and the device allows one writable point per register: the station resets the copy's " +
	"proxyExt/dataAddress to Hex:-1 (station log \"Duplicate Writable point for register address\"). Give it its " +
	"own address with copies[].data_address of that target."

// unassigned is the address the station resets a duplicate writable point to (Hex:-1): no register, so
// it is never held or shared (verifier probe G2 of #798).
const unassigned = "-1"

// registerDepth reaches a point's proxyExt/dataAddress/address from its parent.
const registerDepth = 4

// heldRegisters is the register each writable Modbus point under a target already holds, by its
// FlexAddress text (the first holder named), from the target's flat load of registerDepth. On the
// source parent itself the replicated points are left out: the source rule covers them.
func heldRegisters(nodes *pyjson.Object, isSource bool, points []string) map[string]string {
	skip := map[string]bool{}
	for _, p := range points {
		skip[p] = isSource
	}
	out := map[string]string{}
	kids := box.Children(nodes, "")
	for _, name := range kids.Keys() {
		t, _ := get(asObj(get(kids, name)), "t").(string)
		ext, _ := get(box.Node(nodes, name+"/proxyExt"), "t").(string)
		if skip[name] || !strings.HasSuffix(t, "Writable") || !strings.HasPrefix(ext, "modbusCore:ModbusClient") {
			continue
		}
		format, _ := get(box.Node(nodes, name+"/proxyExt/dataAddress/addressFormat"), "v").(string)
		address, _ := get(box.Node(nodes, name+"/proxyExt/dataAddress/address"), "v").(string)
		if address == unassigned {
			continue
		}
		if text := flexText(format, address); out[text] == "" {
			out[text] = name
		}
	}
	return out
}
