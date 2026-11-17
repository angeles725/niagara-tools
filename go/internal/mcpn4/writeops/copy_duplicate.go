package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// duplicateRegisters notes, per register address, the copies of a writable Modbus point that
// would use an address its source or an earlier copy already holds on the same device (the
// station resets each to Hex:-1), and returns that address by copy name. A source that is not a
// writable point with a Modbus client proxy is not subject to the rule: no notes.
func duplicateRegisters(body *pyjson.Object, names []string, overrides map[string]addressOverride) ([]any, map[string]string) {
	shared := map[string]string{}
	if !writableModbus(body) {
		return nil, shared
	}
	srcFormat, srcAddress := sourceAddress(body)
	used := map[string]bool{flexText(srcFormat, srcAddress): true}
	order, byAddress := []string{}, map[string][]string{}
	for _, n := range names {
		format, address := srcFormat, srcAddress
		if ov, ok := overrides[n]; ok {
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
		byAddress[text] = append(byAddress[text], pyjson.StrRepr(n))
		shared[n] = text
	}
	notes := []any{}
	for _, text := range order {
		notes = append(notes, strings.Join(byAddress[text], ", ")+": a writable Modbus point copied onto its "+
			"source's device uses the register address "+text+", and the device allows one writable point per "+
			"register: the station resets the copy's proxyExt/dataAddress to Hex:-1 (station log \"Duplicate "+
			"Writable point for register address\"). Give each copy its own address with copies[].data_address.")
	}
	return notes, shared
}

// duplicateCause is the read-back's explanation for a copy noted as sharing register text whose
// address differs from the plan.
func duplicateCause(text string) string {
	return "the station reset proxyExt/dataAddress to Hex:-1: another writable point on this device already " +
		"uses register " + text + " (\"Duplicate Writable point for register address\"); give the copy its own " +
		"address with copies[].data_address or n4_set_slot"
}

// writableModbus is a writable control point (control:NumericWritable, BooleanWritable, ...)
// whose proxyExt is a Modbus client proxy.
func writableModbus(body *pyjson.Object) bool {
	t, _ := get(body, "t").(string)
	ext, _ := get(kidNamed(body, "proxyExt"), "t").(string)
	return strings.HasSuffix(t, "Writable") && strings.HasPrefix(ext, "modbusCore:ModbusClient")
}

// sourceAddress is the format and address the body's proxyExt/dataAddress holds ("" when at its
// default).
func sourceAddress(body *pyjson.Object) (string, string) {
	addr := kidNamed(kidNamed(body, "proxyExt"), "dataAddress")
	format, _ := get(kidNamed(addr, "addressFormat"), "v").(string)
	address, _ := get(kidNamed(addr, "address"), "v").(string)
	return format, address
}

// flexText is how the station prints a FlexAddress: Modbus:40002, Hex:-1 (hex is the default
// format); the default address reads as such.
func flexText(format, address string) string {
	if format == "" {
		format = "hex"
	}
	if address == "" {
		address = "(default)"
	}
	return strings.ToUpper(format[:1]) + format[1:] + ":" + address
}

// touchesAddress is a difference list naming the proxy's dataAddress or a slot inside it.
func touchesAddress(diffs []any) bool {
	for _, d := range diffs {
		if p, _ := d.(string); p == "proxyExt/dataAddress" || strings.HasPrefix(p, "proxyExt/dataAddress/") {
			return true
		}
	}
	return false
}
