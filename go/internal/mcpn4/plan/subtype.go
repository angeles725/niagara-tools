package plan

// Is reports whether typeSpec is super or one of its supertypes ("p") is, walking at most
// maxSupertypes contracts; false when the station does not describe the chain that far (an
// entry {} names no supertype), so a caller must not refuse on false alone.
func (c *Contracts) Is(typeSpec, super string) (bool, error) {
	for i := 0; i < maxSupertypes && typeSpec != ""; i++ {
		if typeSpec == super {
			return true, nil
		}
		if err := c.Load([]string{typeSpec}); err != nil {
			return false, err
		}
		e, _ := lookup(c.types, typeSpec)
		typeSpec, _ = field(e, "p").(string)
	}
	return false, nil
}

// IsKnown is Is plus whether the answer is complete: false is complete only when the walk reached a
// type with no supertype (the chain is described to its root) within maxSupertypes contracts.
func (c *Contracts) IsKnown(typeSpec, super string) (is, complete bool, err error) {
	for i := 0; i < maxSupertypes; i++ {
		if typeSpec == super {
			return true, true, nil
		}
		if err := c.Load([]string{typeSpec}); err != nil {
			return false, false, err
		}
		e, _ := lookup(c.types, typeSpec)
		next, _ := field(e, "p").(string)
		if next == "" {
			return false, true, nil
		}
		typeSpec = next
	}
	return false, false, nil
}
