package bql

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// cell is row.get(key) of a by_position row: always a str there ("" when absent).
func cell(row *pyjson.Object, key string) string {
	v, _ := row.Get(key)
	s, _ := v.(string)
	return s
}

// relParts is _rel_parts: the path segments of a row's Slot Path below base, escaped as in the
// ORD; none when the row is not below base.
func relParts(row *pyjson.Object, base string) []string {
	path := strings.TrimPrefix(cell(row, "Slot Path"), "slot:")
	prefix := base
	if base == "/" {
		prefix = ""
	}
	if !strings.HasPrefix(path, prefix+"/") {
		return nil
	}
	var parts []string
	for _, p := range strings.Split(path[len(prefix)+1:], "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// IsLocalDevice is is_local_device: a network's built-in local device slot.
func IsLocalDevice(row *pyjson.Object) bool {
	path := cell(row, "Slot Path")
	return contains(LocalDeviceSlots, path[strings.LastIndexByte(path, '/')+1:])
}

// counter adds n to the int value of key.
func counter(o *pyjson.Object, key string, n int) {
	v, _ := o.Get(key)
	o.Set(key, v.(int)+n)
}

// networkName is how a network row is named below base: its path below base, each segment
// unescaped (its own name when it sits right under base), or its last segment when it is base.
func networkName(row *pyjson.Object, base string) (string, bool) {
	parts := relParts(row, base)
	if parts == nil {
		path := strings.TrimPrefix(cell(row, "Slot Path"), "slot:")
		if path != base || base == "/" {
			return "", false
		}
		parts = []string{path[strings.LastIndexByte(path, '/')+1:]}
	}
	for i, p := range parts {
		parts[i] = UnescapeSlot(p)
	}
	return strings.Join(parts, "/"), true
}

// owner is the entry of byPath that holds path: the one at its deepest ancestor path, nil when
// no ancestor has one.
func owner(byPath map[string]*pyjson.Object, path string) *pyjson.Object {
	for i := strings.LastIndexByte(path, '/'); i > 0; i = strings.LastIndexByte(path, '/') {
		path = path[:i]
		if o, ok := byPath[path]; ok {
			return o
		}
	}
	return nil
}

// SummarizeInventory is summarize_inventory: devices and points per network below base, local
// devices flagged and counted apart. A network is a row of the networks query (a
// driver:DeviceNetwork, whatever its depth below base), never a path segment: a device or a
// point belongs to the deepest network above it, and to none (network null) when no network row
// is above it. A point belongs to the deepest device above it, else it is unassigned.
func SummarizeInventory(base any, networkRows, deviceRows, pointRows []*pyjson.Object) (*pyjson.Object, error) {
	slot, err := BaseSlot(base)
	if err != nil {
		return nil, err
	}
	nets, netAt := []any{}, map[string]*pyjson.Object{}
	for _, row := range networkRows {
		name, ok := networkName(row, slot)
		if !ok {
			continue
		}
		path := cell(row, "Slot Path")
		if n, seen := netAt[path]; seen {
			n.Set("type", cell(row, "Type"))
			continue
		}
		netAt[path] = object("network", name, "type", cell(row, "Type"), "field_devices", 0, "local_devices", 0, "points", 0)
		nets = append(nets, netAt[path])
	}
	totals := object("networks", len(nets), "field_devices", 0, "local_devices", 0,
		"points", len(pointRows), "unassigned_points", 0)
	devices, deviceAt := []any{}, map[string]*pyjson.Object{}
	for _, row := range deviceRows {
		if relParts(row, slot) == nil {
			continue
		}
		path, local, key := cell(row, "Slot Path"), IsLocalDevice(row), "field_devices"
		if local {
			key = "local_devices"
		}
		counter(totals, key, 1)
		var network any
		if n := owner(netAt, path); n != nil {
			counter(n, key, 1)
			network, _ = n.Get("network")
		}
		d := object("path", path, "name", cell(row, "Name"), "type", cell(row, "Type"), "network", network,
			"local", local, "points", 0)
		devices = append(devices, d)
		if _, seen := deviceAt[path]; !seen && path != "" {
			deviceAt[path] = d
		}
	}
	for _, row := range pointRows {
		path := cell(row, "Slot Path")
		if n := owner(netAt, path); n != nil && relParts(row, slot) != nil {
			counter(n, "points", 1)
		}
		if d := owner(deviceAt, path); d != nil {
			counter(d, "points", 1)
		} else {
			counter(totals, "unassigned_points", 1)
		}
	}
	return object("networks", nets, "devices", devices, "totals", totals), nil
}

func object(kv ...any) *pyjson.Object {
	o := pyjson.NewObject()
	for i := 0; i < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
	return o
}
