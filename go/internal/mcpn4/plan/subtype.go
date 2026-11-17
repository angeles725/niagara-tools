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
