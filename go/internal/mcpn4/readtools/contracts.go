package readtools

import (
	"errors"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// loadContracts is the one read of the type contracts a listing needs: the registry entries of
// specs and of every type they depend on, in one call (more only when the reply names a
// supertype it does not describe), each entry {c: declared slots, p: supertype}. The client keeps
// every entry for the session (box.Client.LoadTypes), so a type is asked for once per session.
// A type the station left out of a reply is never kept by the client, so gone remembers it for
// the client's session and it is not asked for again. No specs is no call and nil.
func loadContracts(tl typeLoader, gone *undescribed, specs []string) (*pyjson.Object, error) {
	seen := map[string]bool{}
	for _, s := range specs {
		seen[s] = true
	}
	if len(specs) == 0 {
		return nil, nil
	}
	specs = gone.drop(tl, specs)
	var types *pyjson.Object // the first round asks even with every spec dropped: the kept entries
	for round := 0; round < maxSupertypes && (round == 0 || len(specs) > 0); round++ {
		reply, err := tl.LoadTypes(specs)
		got, isObj := reply.(*pyjson.Object)
		if err == nil && !isObj {
			err = errors.New("its answer is not an object")
		}
		if err != nil {
			return nil, err
		}
		gone.remember(got, specs)
		types, specs = got, nil
		for _, spec := range types.Keys() { // a supertype the reply names but does not describe
			if super := box.Supertype(types, spec); super != "" && !seen[super] && box.Node(types, super) == nil {
				seen[super] = true
				specs = append(specs, super)
			}
		}
		specs = gone.drop(tl, specs)
	}
	return types, nil
}

// undescribed is the type specs the station left out of a registry reply, for one client (a new
// session's client starts empty): asking again would get the same answer.
type undescribed struct {
	client typeLoader
	specs  map[string]bool
}

// drop is specs without the ones remembered for tl.
func (u *undescribed) drop(tl typeLoader, specs []string) []string {
	if u.client != tl {
		u.client, u.specs = tl, map[string]bool{}
	}
	var out []string
	for _, s := range specs {
		if !u.specs[s] {
			out = append(out, s)
		}
	}
	return out
}

// remember keeps each asked spec the reply leaves out.
func (u *undescribed) remember(reply *pyjson.Object, asked []string) {
	for _, s := range asked {
		if box.Node(reply, s) == nil {
			u.specs[s] = true
		}
	}
}
