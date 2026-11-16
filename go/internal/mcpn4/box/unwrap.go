package box

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// unwrap is BoxClient._unwrap: validate a reply frame, return its first message body.
func (c *Client) unwrap(reply any, channel, key string) (any, error) {
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
	msg, ok := msgs[0].(*pyjson.Object)
	if !ok {
		return nil, boxError("malformed reply: missing or empty message list", channel, key)
	}
	if t, _ := msg.Get("t"); t == "e" {
		b, _ := msg.Get("b")
		text, ok := get(b, "m")
		if s, isStr := text.(string); ok && isStr {
			return nil, boxError(s, channel, key)
		}
		return nil, boxError("station error", channel, key)
	}
	b, _ := msg.Get("b")
	return b, nil
}
