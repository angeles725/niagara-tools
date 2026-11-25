package box

// Subscribe is the component-space sub of ords (#875, box-rt BComponentSpaceSessionHandler
// .subscribe): the station subscribes each component for this session, so a driver polls a
// subscribed proxy point. A BOX subscription has no lease: pair every Subscribe with Unsubscribe.
func (c *Client) Subscribe(ords []string) error {
	_, err := c.SSC("sub", ordList(ords))
	return err
}

// Unsubscribe is the component-space unsub of ords; the station answers null.
func (c *Client) Unsubscribe(ords []string) error {
	_, err := c.SSC("unsub", ordList(ords))
	return err
}

func ordList(ords []string) []any {
	out := make([]any, len(ords))
	for i, o := range ords {
		out[i] = o
	}
	return out
}
