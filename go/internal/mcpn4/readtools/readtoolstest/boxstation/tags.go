package boxstation

import (
	"fmt"
	"strings"
)

// This file is the tag side of the fake, as measured on a 4.15 station (2026-11-15): a plain
// (non-component) add carries its flags ("f") and is refused when the name exists, a name ending
// in "?" is made unique (relations add "r?"), and the session handler answers impliedTags and
// impliedRelations for a handle with what SetImplied arranged.

// implied is what impliedTags / impliedRelations answer, by handle.
type implied struct {
	tags      map[string]any
	relations []any
}

// SetImplied arranges the implied tags (keys as the station sends them, id + type letter) and
// implied relations ({rId, inbound, entityOrd}) of the component with handle h.
func (st *Station) SetImplied(h string, tags map[string]any, relations []map[string]any) {
	st.Do(func(*Station) {
		if st.implied == nil {
			st.implied = map[string]*implied{}
		}
		rels := []any{}
		for _, r := range relations {
			rels = append(rels, r)
		}
		st.implied[h] = &implied{tags: tags, relations: rels}
	})
}

// impliedCall answers impliedTags / impliedRelations for the handle arg; an unknown handle is
// the station's "Could not find Component".
func (st *Station) impliedCall(key string, arg any) (any, error) {
	h, _ := arg.(string)
	if st.byHandle[h] == nil {
		return nil, fmt.Errorf("Exception: Could not find Component: %s", h)
	}
	im := st.implied[h]
	if im == nil {
		im = &implied{tags: map[string]any{}, relations: []any{}}
	}
	if key == "impliedTags" {
		return im.tags, nil
	}
	return im.relations, nil
}

// addPlain applies an add of a simple or struct value under parent: a tag, a relation.
func (st *Station) addPlain(parent *Node, name string, body map[string]any, flags string) (any, error) {
	if base, ok := strings.CutSuffix(name, "?"); ok {
		name = st.unique(parent, base)
	} else if parent.Child(name) != nil {
		return nil, fmt.Errorf("ValueError: Slot \"%s\" already exists.", name)
	}
	typ, _ := body["t"].(string)
	node := &Node{Name: name, Type: typ, Flags: flags}
	parent.Children = append(parent.Children, node)
	st.fill(node, body)
	return []any{map[string]any{"id": "a", "nn": name}}, nil
}

// AddSlot adds a simple or struct value (a tag, a relation) named name with flags to the
// component with handle h, as a syncTo add does; it returns the name given.
func (st *Station) AddSlot(h, name string, body map[string]any, flags string) (nn string, err error) {
	st.Do(func(*Station) {
		var out any
		op := map[string]any{"nm": "a", "h": h, "n": name, "b": body, "f": flags}
		if out, err = st.sync(op); err == nil {
			nn = out.([]any)[0].(map[string]any)["nn"].(string)
		}
	})
	return nn, err
}
