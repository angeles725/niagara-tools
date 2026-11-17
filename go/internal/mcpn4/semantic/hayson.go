// Package semantic turns Niagara tags into a Project Haystack grid in the Haystack 4 JSON
// encoding (Hayson) for n4_semantic_export (opportunity O10). Pure functions: the read tool
// reads the tags; nothing here talks to a station.
package semantic

import (
	"fmt"
	"strconv"
	"strings"
)

// Tag is one tag of a component: its id (hs:temp), Niagara value type and the value as the
// station encoded it.
type Tag struct {
	ID, Type string
	Value    any
}

// Relation is one relation of a component: its id (hs:equipRef) and the ORD of the other end.
type Relation struct {
	ID, Endpoint string
}

// Entity is one exported component.
type Entity struct {
	Ord       string
	Dis       string
	Tags      []Tag
	Relations []Relation
}

// Skipped is a tag that could not be converted, with why (never silently dropped).
type Skipped struct {
	Ord, Tag, Why string
}

// marker is the Hayson marker.
func marker() map[string]any { return map[string]any{"_kind": "marker"} }

// Ref is a Hayson ref to the entity with id (and its display name when known).
func Ref(id, dis string) map[string]any {
	r := map[string]any{"_kind": "ref", "val": id}
	if dis != "" {
		r["dis"] = dis
	}
	return r
}

// value converts one tag value to Hayson; ok false with the reason when it cannot.
func value(t Tag) (any, string) {
	s, isStr := t.Value.(string)
	switch t.Type {
	case "baja:Marker":
		return marker(), ""
	case "baja:Boolean":
		if v, ok := t.Value.(bool); ok {
			return v, ""
		}
		if b, err := strconv.ParseBool(s); isStr && err == nil {
			return b, ""
		}
	case "baja:Double", "baja:Float", "baja:Integer", "baja:Long":
		if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); isStr && err == nil {
			return map[string]any{"_kind": "number", "val": f}, ""
		}
		if n, ok := t.Value.(float64); ok {
			return map[string]any{"_kind": "number", "val": n}, ""
		}
	case "baja:String":
		if isStr {
			return s, ""
		}
	default:
		return nil, "type " + t.Type + " has no Haystack conversion here"
	}
	return nil, fmt.Sprintf("value %v is not a %s", t.Value, t.Type)
}
