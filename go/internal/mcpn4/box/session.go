package box

import (
	"errors"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// SSC is BoxClient.ssc: a callssc on the session component.
func (c *Client) SSC(key string, arg any) (any, error) {
	return c.Call(Channel, "callssc", obj("id", c.sid, "scid", SessionComponentID, "sck", key, "scarg", arg))
}

// Poll is BoxClient.poll: the pollchgs events, [] for null; any other non-list is malformed.
func (c *Client) Poll() ([]any, error) {
	r, err := c.Call(Channel, "pollchgs", obj("id", c.sid))
	if err != nil {
		return nil, err
	}
	switch events := r.(type) {
	case nil:
		return []any{}, nil
	case []any:
		return events, nil
	}
	return nil, malformed("pollchgs answered "+jsonKind(r)+", not a list", Channel, "pollchgs")
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
	if h, _ := get(root, "h"); pyjson.Truthy(h) {
		c.handles.set("station:", h)
	}
	return root, nil
}

// Close is BoxClient.close: delete the server session, swallowing any failure.
func (c *Client) Close() {
	if !pyjson.Truthy(c.sid) {
		return
	}
	c.Call(Channel, "del", obj("id", c.sid))
	c.sid = nil
}

// Sync is BoxClient.sync: exactly one op (a dict) per syncTo; anything else is a ValueError.
func (c *Client) Sync(op any) (any, error) {
	if _, ok := op.(*pyjson.Object); !ok {
		return nil, &pyjson.ValueError{Msg: "sync() takes exactly one op dict"}
	}
	return c.SSC("syncTo", obj("nm", "sync", "ver", 1.0, "ops", []any{op}))
}
