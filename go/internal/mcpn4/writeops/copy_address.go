package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// A Modbus client device allows one writable point per register address: a second writable
// proxy at the same address is reset to Hex:-1 by the station (BModbusClientProxyExt.started and
// changed: "Duplicate Writable point for register address"). A copy of a writable point keeps its
// source's address, so copies[] gives a copy its own proxyExt/dataAddress inside the same add.

const (
	addressFormatType = "modbusCore:AddressFormatEnum"
	flexAddressType   = "modbusCore:FlexAddress"
)

// addressOverride is one copies[] entry: the address and, when given, its format.
type addressOverride struct{ address, format string }

// copyAddresses is the add body of each new name (body, or a copy of it carrying the name's
// copies[] address), the notes for copies that would share a writable register, and the address
// each such copy would share, by name (for the read-back's cause).
func copyAddresses(ct *plan.Contracts, root, body *pyjson.Object, source string, names []string,
	args *pyjson.Object) ([]*pyjson.Object, []any, map[string]string, error) {
	overrides, err := copyOverrides(args, names)
	if err != nil {
		return nil, nil, nil, err
	}
	addrType := ""
	if len(overrides) > 0 {
		if addrType, err = declaredAddressType(ct, root); err != nil || addrType == "" {
			if err == nil {
				err = plan.ToolError("copies[].data_address needs a source with a proxyExt that declares dataAddress; " +
					pyjson.StrRepr(source) + " has none")
			}
			return nil, nil, nil, err
		}
	}
	bodies := make([]*pyjson.Object, len(names))
	for i, n := range names {
		bodies[i] = body
		if ov, ok := overrides[n]; ok {
			bodies[i] = withAddress(body, ov, addrType)
		}
	}
	notes, shared := duplicateRegisters(body, names, overrides)
	return bodies, notes, shared, nil
}

// copyOverrides reads copies[]: each name one of new_names, at most once, with a non-empty address.
func copyOverrides(args *pyjson.Object, names []string) (map[string]addressOverride, error) {
	raw, _ := args.Get("copies")
	list, _ := raw.([]any)
	out := map[string]addressOverride{}
	for _, e := range list {
		entry, _ := e.(*pyjson.Object)
		n, _ := get(entry, "name").(string)
		if !containsName(names, n) {
			return nil, plan.ToolError("copies[].name " + pyjson.StrRepr(n) + " is not one of new_names")
		}
		if _, seen := out[n]; seen {
			return nil, plan.ToolError("copies[].name " + pyjson.StrRepr(n) + " is given twice")
		}
		address, _ := get(entry, "data_address").(string)
		format, _ := get(entry, "address_format").(string)
		if strings.TrimSpace(address) == "" {
			return nil, plan.ToolError("copies[].data_address of " + pyjson.StrRepr(n) + " must be a non-empty address")
		}
		out[n] = addressOverride{address, format}
	}
	return out, nil
}

// declaredAddressType is the type of the dataAddress slot root's proxyExt declares; "" when root
// has no proxyExt or its type declares no dataAddress.
func declaredAddressType(ct *plan.Contracts, root *pyjson.Object) (string, error) {
	ext := kidNamed(root, "proxyExt")
	extType, _ := get(ext, "t").(string)
	if extType == "" {
		return "", nil
	}
	slot, err := ct.Declared(extType, obj("n", "dataAddress"))
	if err != nil || slot == nil {
		return "", typeRefusal(err)
	}
	if t, ok := get(asObj(get(slot, "v")), "t").(string); ok {
		return t, nil
	}
	return flexAddressType, nil
}

// withAddress is a copy of body whose proxyExt/dataAddress holds ov (format kept unless given).
func withAddress(body *pyjson.Object, ov addressOverride, addrType string) *pyjson.Object {
	out := cloneObject(body)
	ext := kidNamed(out, "proxyExt")
	addr := kidNamed(ext, "dataAddress")
	if addr == nil {
		addr = obj("nm", "p", "t", addrType, "n", "dataAddress")
		kids, _ := get(ext, "s").([]any)
		ext.Set("s", append(kids, addr))
	}
	if ov.format != "" {
		setChild(addr, "addressFormat", addressFormatType, ov.format)
	}
	setChild(addr, "address", "baja:String", ov.address)
	return out
}

// setChild sets the value of node's child n, adding it (of type t) when node lacks it.
func setChild(node *pyjson.Object, n, t, v string) {
	if kid := kidNamed(node, n); kid != nil {
		kid.Set("v", v)
		return
	}
	kids, _ := get(node, "s").([]any)
	node.Set("s", append(kids, obj("nm", "p", "t", t, "n", n, "v", v)))
}

// cloneObject is a deep copy of o (objects and lists; scalars are immutable).
func cloneObject(o *pyjson.Object) *pyjson.Object {
	out := pyjson.NewObject()
	for _, k := range o.Keys() {
		v, _ := o.Get(k)
		out.Set(k, cloneValue(v))
	}
	return out
}

func cloneValue(v any) any {
	switch x := v.(type) {
	case *pyjson.Object:
		return cloneObject(x)
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = cloneValue(e)
		}
		return out
	}
	return v
}

func containsName(names []string, n string) bool {
	for _, x := range names {
		if x == n {
			return true
		}
	}
	return false
}
