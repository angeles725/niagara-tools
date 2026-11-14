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
	entry := func(spec string) (*pyjson.Object, bool, error) {
		if e, ok := lookup(types, spec); ok {
			return e, true, nil
		}
		reply, err := tl.LoadTypes([]string{spec})
		if err != nil {
			return nil, false, err
		}
		types, _ = reply.(*pyjson.Object)
		e, ok := lookup(types, spec)
		return e, ok, nil
	}
	slot, err := walkDeclared(entry, typeSpec, func(slot *pyjson.Object) bool {
		_, typed := field(asObject(field(slot, "v")), "t").(string)
		return field(slot, "n") == name && typed
	})
	if slot == nil {
		return nil, err
	}
	return asObject(field(slot, "v")), nil
}

// walkDeclared is the one walk up a type's contracts: typeSpec, then each supertype ("p"), at
// most maxSupertypes, and the first property slot ({st: "p", n, f?, v}) of a contract "c" that
// match accepts; nil when none does or entry stops the walk (ok false: a type it cannot answer).
func walkDeclared(entry func(spec string) (*pyjson.Object, bool, error), typeSpec string,
	match func(slot *pyjson.Object) bool) (*pyjson.Object, error) {
	for i := 0; i < maxSupertypes && typeSpec != ""; i++ {
		e, ok, err := entry(typeSpec)
		if err != nil || !ok {
			return nil, err
		}
		slots, _ := field(e, "c").([]any)
		for _, s := range slots {
			if slot, ok := s.(*pyjson.Object); ok && field(slot, "st") == "p" && match(slot) {
				return slot, nil
			}
		}
		typeSpec, _ = field(e, "p").(string)
	}
	return nil, nil
}

func asObject(v any) *pyjson.Object { o, _ := v.(*pyjson.Object); return o }

func lookup(types *pyjson.Object, spec string) (*pyjson.Object, bool) {
	if types == nil {
		return nil, false
	}
	v, _ := types.Get(spec)
	entry, ok := v.(*pyjson.Object)
	return entry, ok
}

func field(o *pyjson.Object, k string) any {
	if o == nil {
		return nil
	}
	v, _ := o.Get(k)
	return v
}
