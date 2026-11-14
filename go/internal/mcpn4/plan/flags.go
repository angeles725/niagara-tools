package plan

import "github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"

// UnansweredType is a type spec the registry channel's reply left out. The station answers every
// requested type, a common baja type with {} (box-rt RegistryEncoderVisitor.doVisit), so a
// missing spec is a type the station could not describe.
type UnansweredType struct{ Spec string }

func (e *UnansweredType) Error() string {
	return "the station did not describe type " + pyjson.StrRepr(e.Spec)
}

// Contracts are the type contracts read from the station's registry channel, keyed by type spec
// ({c: [frozen slots {st, n, f?, v}], p: supertype}); an entry {} is a type with no declared slot.
type Contracts struct {
	tl    TypeLoader
	types *pyjson.Object
}

// NewContracts holds no type yet; Load and Slot read them through tl.
func NewContracts(tl TypeLoader) *Contracts { return &Contracts{tl: tl, types: pyjson.NewObject()} }

// Load reads every spec not held yet in one registry call and keeps each entry of the reply (the
// requested types and the types they depend on). A spec the reply leaves out is *UnansweredType.
func (c *Contracts) Load(specs []string) error {
	missing, seen := []string{}, map[string]bool{}
	for _, s := range specs {
		if _, held := lookup(c.types, s); !held && !seen[s] {
			seen[s] = true
			missing = append(missing, s)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	reply, err := c.tl.LoadTypes(missing)
	if err != nil {
		return err
	}
	if got, ok := reply.(*pyjson.Object); ok {
		for _, k := range got.Keys() {
			if e, ok := lookup(got, k); ok {
				c.types.Set(k, e)
			}
		}
	}
	for _, s := range missing {
		if _, ok := lookup(c.types, s); !ok {
			return &UnansweredType{Spec: s}
		}
	}
	return nil
}

// Slot is the contract slot of property name that typeSpec or one of its supertypes declares;
// nil when none does (a dynamic slot). A type not held yet is loaded first.
func (c *Contracts) Slot(typeSpec, name string) (*pyjson.Object, error) {
	entry := func(spec string) (*pyjson.Object, bool, error) {
		if err := c.Load([]string{spec}); err != nil {
			return nil, false, err
		}
		e, ok := lookup(c.types, spec)
		return e, ok, nil
	}
	return walkDeclared(entry, typeSpec, func(slot *pyjson.Object) bool { return field(slot, "n") == name })
}

// Flags are the effective flags (Flags.encodeToString letters) of child, a loaded child node of a
// parent typed parentType, and the contract slot that declares it (nil for a dynamic slot). The
// station writes "f" only when the instance flags differ from the declared ones, or on a dynamic
// slot with flags (box-rt ValueDocEncoder), so: child's own "f" when present, else the declared
// slot's "f" (missing: no flags), else none.
func (c *Contracts) Flags(parentType string, child *pyjson.Object) (string, *pyjson.Object, error) {
	var slot *pyjson.Object
	if name, ok := field(child, "n").(string); ok && parentType != "" {
		var err error
		if slot, err = c.Slot(parentType, name); err != nil {
			return "", nil, err
		}
	}
	if f, ok := field(child, "f").(string); ok {
		return f, slot, nil
	}
	f, _ := field(slot, "f").(string)
	return f, slot, nil
}
