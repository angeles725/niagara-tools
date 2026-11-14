package box

import "github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"

// RegistryChannel is the station's type registry channel (box-rt BRegistryChannel "reg").
const RegistryChannel = "reg"

// LoadTypes is the registry channel's loadTypes with contracts (ec: true): for each requested
// type spec and every type it depends on (its supertypes included), keyed by type spec, an
// object with "c", the frozen slots that type itself declares (each {st, n, f?, v: the default
// value in BSON}), and "p", its supertype. Common baja types answer {} (box-rt
// RegistryEncoderVisitor.doVisit, ContractEncoderVisitor.encodeComplex). Types do not change
// within a station session, so every entry of each reply is kept per client: a request whose
// specs are all kept is answered without a call. The answer holds every kept type (at least the
// requested specs the station described and their dependencies); a reply that is not an object
// is returned as it came. It reads type definitions only, never a component.
func (c *Client) LoadTypes(specs []string) (any, error) {
	if c.types == nil {
		c.types = pyjson.NewObject()
	}
	missing := []any{}
	for _, s := range specs {
		if _, held := c.types.Get(s); !held {
			missing = append(missing, s)
		}
	}
	if len(missing) > 0 {
		reply, err := c.Call(RegistryChannel, "loadTypes", obj("t", missing, "ec", true))
		got, ok := reply.(*pyjson.Object)
		if err != nil || !ok {
			return reply, err
		}
		for _, k := range got.Keys() {
			v, _ := got.Get(k)
			c.types.Set(k, v)
		}
	}
	out := pyjson.NewObject()
	for _, k := range c.types.Keys() {
		v, _ := c.types.Get(k)
		out.Set(k, v)
	}
	return out, nil
}
