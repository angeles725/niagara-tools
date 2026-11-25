package box

import (
	"fmt"
	"html"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// messages validates a reply frame (an object, its seq the request's, a non-empty "m" list) and
// returns the list; unwrap and CallMany read their entries from it.
func (c *Client) messages(reply any, channel, key string) ([]any, error) {
	r, ok := reply.(*pyjson.Object)
	if !ok {
		return nil, boxError("malformed reply: not an object", channel, key)
	}
	if n, ok := r.Get("n"); ok && !PyEqual(n, c.seq) {
		return nil, boxError(fmt.Sprintf("reply seq %s does not match request seq %d", pyjson.Repr(n), c.seq),
			channel, key)
	}
	m, _ := r.Get("m")
	msgs, ok := m.([]any)
	if !ok || len(msgs) == 0 {
		return nil, boxError("malformed reply: missing or empty message list", channel, key)
	}
	return msgs, nil
}

// unwrap is BoxClient._unwrap: validate a reply frame, return its first message body.
func (c *Client) unwrap(reply any, channel, key string) (any, error) {
	msgs, err := c.messages(reply, channel, key)
	if err != nil {
		return nil, err
	}
	msg, ok := msgs[0].(*pyjson.Object)
	if !ok {
		return nil, boxError("malformed reply: missing or empty message list", channel, key)
	}
	return entryBody(msg, channel, key)
}

// entryBody is a reply message's body, or the station's error when its type is "e".
func entryBody(msg *pyjson.Object, channel, key string) (any, error) {
	if t, _ := msg.Get("t"); t == "e" {
		b, _ := msg.Get("b")
		text, ok := get(b, "m")
		s, isStr := text.(string)
		if !ok || !isStr {
			s = "station error"
		}
		return nil, stationError(s, channel, key)
	}
	b, _ := msg.Get("b")
	return b, nil
}

// stationError is the BoxError of the station's own error reply ("t": "e"): its text with the
// HTML entities the station escapes (&quot;, &lt;, &amp; ...) decoded, flagged as an explicit
// station answer rather than a transport or protocol failure.
func stationError(text, channel, key string) *Error {
	e := boxError(html.UnescapeString(text), channel, key)
	e.Station = true
	return e
}
