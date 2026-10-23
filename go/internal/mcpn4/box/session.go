package box

import "errors"

// SSC is BoxClient.ssc: a callssc on the session component.
func (c *Client) SSC(key string, arg any) (any, error) {
	return c.Call(Channel, "callssc", obj("id", c.sid, "scid", SessionComponentID, "sck", key, "scarg", arg))
}

// Poll is BoxClient.poll as a for loop walks it: the pollchgs body, [] when falsy.
func (c *Client) Poll() ([]any, error) {
	r, err := c.Call(Channel, "pollchgs", obj("id", c.sid))
	if err != nil {
		return nil, err
	}
	if !truthy(r) {
		return []any{}, nil
	}
	return pyIter(r)
}

// Open is BoxClient.open: make, makessc, loadRoot; the loadRoot reply. A failure after make
// closes the server session (best effort) unless it is an AuthError, which only drops the
// local sid: one more rejected login counts toward the 5-in-30-s lockout.
func (c *Client) Open() (any, error) {
	sid, err := c.Call(Channel, "make", obj())
	if err != nil {
		return nil, err
	}
	c.sid = sid
	_, err = c.Call(Channel, "makessc", obj("id", c.sid, "scid", SessionComponentID,
		"scts", "box:ComponentSpaceSessionHandler", "scarg", "station:"))
	var root any
	if err == nil {
		root, err = c.SSC("loadRoot", nil)
	}
	if err != nil {
		var e *Error
		if errors.As(err, &e) && e.IsAuth() {
			c.sid = nil
		} else {
			c.Close()
		}
		return nil, err
	}
	if h, _ := get(root, "h"); truthy(h) {
		c.handles.set("station:", h)
	}
	return root, nil
}

// Close is BoxClient.close: delete the server session, swallowing any failure.
func (c *Client) Close() {
	if !truthy(c.sid) {
		return
	}
	c.Call(Channel, "del", obj("id", c.sid))
	c.sid = nil
}
