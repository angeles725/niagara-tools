package box

// TypeHeld is true when the registry reply this session already kept holds spec, so a LoadTypes
// of it is answered without a call (and without telling whether the station still describes it).
func (c *Client) TypeHeld(spec string) bool {
	if c.types == nil {
		return false
	}
	_, held := c.types.Get(spec)
	return held
}

// ForgetTypes drops every type the session kept, so the next LoadTypes asks the station again: the
// station may have restarted with a new or upgraded module while the session carried on (#1166).
func (c *Client) ForgetTypes() { c.types = nil }
