package box

// RegistryChannel is the station's type registry channel (box-rt BRegistryChannel "reg").
const RegistryChannel = "reg"

// LoadTypes is the registry channel's loadTypes with contracts (ec: true): for each requested
// type spec and every type it depends on (its supertypes included), keyed by type spec, an
// object with "c", the frozen slots that type itself declares (each {st, n, v: the default
// value in BSON}), and "p", its supertype. Common baja types answer {} (box-rt
// RegistryEncoderVisitor.doVisit, ContractEncoderVisitor.encodeComplex). The reply is returned
// as it came; it reads type definitions only, never a component.
func (c *Client) LoadTypes(specs []string) (any, error) {
	t := make([]any, len(specs))
	for i, s := range specs {
		t[i] = s
	}
	return c.Call(RegistryChannel, "loadTypes", obj("t", t, "ec", true))
}
