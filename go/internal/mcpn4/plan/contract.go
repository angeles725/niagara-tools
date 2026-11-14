package plan

import "github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"

// TypeLoader is a client that can read type definitions from the station's registry channel
// (box.Client.LoadTypes). A client without it keeps the old behaviour: an omitted slot is refused.
type TypeLoader interface {
	LoadTypes(specs []string) (any, error)
}

// maxSupertypes bounds the walk up a type's supertypes (and so the registry calls it makes).
const maxSupertypes = 32

// DeclaredDefault is the default value, in BSON ({nm, t, v[, s]}), of the frozen property name
// that typeSpec or one of its supertypes declares; nil when none declares it (a dynamic slot, an
// action, a misspelled name) or the station's answer is not of the documented shape.
func DeclaredDefault(tl TypeLoader, typeSpec, name string) (*pyjson.Object, error) {
	var types *pyjson.Object
	for i := 0; i < maxSupertypes && typeSpec != ""; i++ {
		entry, ok := lookup(types, typeSpec)
		if !ok {
			reply, err := tl.LoadTypes([]string{typeSpec})
			if err != nil {
				return nil, err
			}
			types, _ = reply.(*pyjson.Object)
			if entry, ok = lookup(types, typeSpec); !ok {
				return nil, nil
			}
		}
		if v := declared(entry, name); v != nil {
			return v, nil
		}
		typeSpec, _ = field(entry, "p").(string)
	}
	return nil, nil
}

func lookup(types *pyjson.Object, spec string) (*pyjson.Object, bool) {
	if types == nil {
		return nil, false
	}
	v, _ := types.Get(spec)
	entry, ok := v.(*pyjson.Object)
	return entry, ok
}

// declared is the BSON default of property name in one type's own contract "c".
func declared(entry *pyjson.Object, name string) *pyjson.Object {
	slots, _ := field(entry, "c").([]any)
	for _, s := range slots {
		slot, ok := s.(*pyjson.Object)
		if !ok || field(slot, "st") != "p" || field(slot, "n") != name {
			continue
		}
		v, _ := field(slot, "v").(*pyjson.Object)
		if _, typed := field(v, "t").(string); typed {
			return v
		}
	}
	return nil
}

func field(o *pyjson.Object, k string) any {
	if o == nil {
		return nil
	}
	v, _ := o.Get(k)
	return v
}
