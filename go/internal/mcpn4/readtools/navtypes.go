package readtools

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The types filter of n4_navigate: a type spec keeps the entries of that type and of its
// subtypes. The ancestry is the station's (its registry channel), read once per listing for the
// types the listing holds and kept by the client for the session.

// typeLoader is a client that reads type definitions from the station's registry channel
// (*box.Client.LoadTypes). With a client that cannot, the filter matches exact type specs.
type typeLoader interface {
	LoadTypes(specs []string) (any, error)
}

// maxSupertypes bounds the walk up a type's supertypes and the registry calls made for it.
const maxSupertypes = 32

// exactTypesNote is the types_note of a listing whose filter could not read the registry.
const exactTypesNote = "the station's type registry could not be read (%s): types kept exact type specs only, not their subtypes"

// typeFilter is the types filter of one listing: the wanted specs and the registry entries of
// the listed types (nil when none was read). note is set when the registry could not be read.
type typeFilter struct {
	wanted map[string]bool
	types  *pyjson.Object
	note   string
}

// match is whether kind is a wanted type or has one among its supertypes. A list or dict kind
// is unhashable (Python's `kind in types`).
func (f *typeFilter) match(kind any) (bool, error) {
	switch k := kind.(type) {
	case string:
		for i := 0; i <= maxSupertypes && k != ""; i++ {
			if f.wanted[k] {
				return true, nil
			}
			k = box.Supertype(f.types, k)
		}
	case []any, *pyjson.Object:
		return false, typeError
	}
	return false, nil
}

// typeFilter builds the filter for the entries of nodes down to depth levels. It asks the
// registry about the listed types that are not wanted themselves (loadContracts); a registry it
// cannot read leaves exact matching and a note.
func (t *Tools) typeFilter(wanted map[string]bool, nodes *pyjson.Object, depth int) *typeFilter {
	f := &typeFilter{wanted: wanted}
	tl, ok := t.Ctx.Session.Client.(typeLoader)
	if !ok {
		return f
	}
	specs, seen := []string{}, map[string]bool{}
	for _, path := range nodes.Keys() {
		kind, _ := box.Node(nodes, path).Get("t")
		spec, isStr := kind.(string)
		if path == "" || strings.Count(path, "/") >= depth || !isStr || wanted[spec] || seen[spec] {
			continue
		}
		seen[spec] = true
		specs = append(specs, spec)
	}
	types, err := loadContracts(tl, &t.undescribed, specs)
	if err != nil {
		f.note = t.Ctx.Scrub(strings.Replace(exactTypesNote, "%s", err.Error(), 1))
		return f
	}
	f.types = types
	return f
}
