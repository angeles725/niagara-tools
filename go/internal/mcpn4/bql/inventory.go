package bql

import (
	"sort"
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

// SummarizeInventory is summarize_inventory: devices and points per network below base, local
// devices flagged and counted apart. A row's network is its first segment below base; a point
// belongs to the deepest device whose path prefixes its own, else it is unassigned.
func SummarizeInventory(base any, networkRows, deviceRows, pointRows []*pyjson.Object) (*pyjson.Object, error) {
	slot, err := BaseSlot(base)
	if err != nil {
		return nil, err
	}
	networks := pyjson.NewObject()
	net := func(name string) *pyjson.Object {
		if n, ok := networks.Get(name); ok {
			return n.(*pyjson.Object)
		}
		n := object("network", UnescapeSlot(name), "type", nil, "field_devices", 0, "local_devices", 0, "points", 0)
		networks.Set(name, n)
		return n
	}
	for _, row := range networkRows {
		if parts := relParts(row, slot); parts != nil {
			net(parts[0]).Set("type", cell(row, "Type"))
		}
	}
	devices := []*pyjson.Object{}
	for _, row := range deviceRows {
		parts := relParts(row, slot)
		if parts == nil {
			continue
		}
		local := IsLocalDevice(row)
		key := "field_devices"
		if local {
			key = "local_devices"
		}
		counter(net(parts[0]), key, 1)
		devices = append(devices, object("path", cell(row, "Slot Path"), "name", cell(row, "Name"),
			"type", cell(row, "Type"), "network", UnescapeSlot(parts[0]), "local", local, "points", 0))
	}
	byPath := append([]*pyjson.Object(nil), devices...)
	sort.SliceStable(byPath, func(i, j int) bool {
		return pyjson.Len(cell(byPath[i], "path")) > pyjson.Len(cell(byPath[j], "path"))
	})
	unassigned := 0
	for _, row := range pointRows {
		if parts := relParts(row, slot); parts != nil {
			counter(net(parts[0]), "points", 1)
		}
		path := cell(row, "Slot Path")
		var owner *pyjson.Object
		for _, d := range byPath {
			if p := cell(d, "path"); p != "" && strings.HasPrefix(path, p+"/") {
				owner = d
				break
			}
		}
		if owner == nil {
			unassigned++
		} else {
			counter(owner, "points", 1)
		}
	}
	nets := []any{}
	totals := object("networks", len(networks.Keys()), "field_devices", 0, "local_devices", 0,
		"points", len(pointRows), "unassigned_points", unassigned)
	for _, k := range networks.Keys() {
		n, _ := networks.Get(k)
		nets = append(nets, n)
		for _, key := range []string{"field_devices", "local_devices"} {
			v, _ := n.(*pyjson.Object).Get(key)
			counter(totals, key, v.(int))
		}
	}
	list := make([]any, len(devices))
	for i, d := range devices {
		list[i] = d
	}
	return object("networks", nets, "devices", list, "totals", totals), nil
}

func object(kv ...any) *pyjson.Object {
	o := pyjson.NewObject()
	for i := 0; i < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
	return o
}
