package readtools

import (
	"errors"
	"sync"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// loadContracts is the one read of the type contracts a listing needs: the registry entries of
// specs and of every type they depend on, in one call (more only when the reply names a
// supertype it does not describe), each entry {c: declared slots, p: supertype}. The client keeps
// every entry for the session (box.Client.LoadTypes), so a type is asked for once per session.
// gone remembers, for the client's session, the types the station answered as unknown or left
// out of a reply that answered other types; they are not asked for again, and when nothing is
// left to ask there is no call (an empty answer). No specs is no call and nil.
func loadContracts(tl typeLoader, gone *undescribed, specs []string) (*pyjson.Object, error) {
	if len(specs) == 0 {
		return nil, nil
	}
	seen := map[string]bool{}
	for _, s := range specs {
		seen[s] = true
	}
	types := pyjson.NewObject()
	specs = gone.drop(tl, specs)
	for round := 0; round < maxSupertypes && len(specs) > 0; round++ {
		reply, err := tl.LoadTypes(specs)
		got, isObj := reply.(*pyjson.Object)
		if err == nil && !isObj {
			err = errors.New("its answer is not an object")
		}
		if err != nil {
			return nil, err
		}
		gone.remember(tl, got, specs)
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

// undescribed is the type specs the station does not describe, for one client (a new session's
// client starts empty): asking again would get the same answer. Listings may share it.
type undescribed struct {
	mu     sync.Mutex
	client typeLoader
	specs  map[string]bool
}

// drop is specs without the ones remembered for tl.
func (u *undescribed) drop(tl typeLoader, specs []string) []string {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.reset(tl)
	var out []string
	for _, s := range specs {
		if !u.specs[s] {
			out = append(out, s)
		}
	}
	return out
}

// remember keeps each asked spec the reply answers with an error entry, and each one it leaves
// out when it answered another asked spec: a reply that answered none of them proves nothing.
func (u *undescribed) remember(tl typeLoader, reply *pyjson.Object, asked []string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.reset(tl)
	answered := false
	for _, s := range asked {
		if e := box.Node(reply, s); e != nil && !isErrEntry(e) {
			answered = true
		}
	}
	for _, s := range asked {
		if e := box.Node(reply, s); e == nil && answered || isErrEntry(e) {
			u.specs[s] = true
		}
	}
}

// reset forgets what was remembered for another client; u is locked.
func (u *undescribed) reset(tl typeLoader) {
	if u.client != tl || u.specs == nil {
		u.client, u.specs = tl, map[string]bool{}
	}
}

// has reports whether spec is remembered.
func (u *undescribed) has(spec string) bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.specs[spec]
}

// isErrEntry reports whether a registry entry is the error entry of a type the station does not
// know ({"isErr": true, ...}).
func isErrEntry(e *pyjson.Object) bool {
	if e == nil {
		return false
	}
	v, _ := e.Get("isErr")
	return v == true
}
