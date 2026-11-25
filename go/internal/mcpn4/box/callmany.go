package box

import (
	"context"
	"fmt"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// MaxFrameMessages is how many messages CallMany puts in one frame; a longer list goes in several.
const MaxFrameMessages = 50

// TraceEnv, set to 1, makes every new Client print one stderr line per BOX frame it sends
// ("box frame seq=3 messages=50 key=callssc"): count the lines of a tool run to count its round trips.
const TraceEnv = "MCP_N4_BOX_TRACE"

// Msg is one BOX message of CallMany: the channel, key and body Call takes.
type Msg struct {
	Channel, Key string
	Body         any
}

// Reply is the station's answer to one Msg: its body, or Err when the station answered that
// message with an error ("t": "e"; a *Error with Station set).
type Reply struct {
	Body any
	Err  error
}

// Frames is how many BOX frames this client has POSTed (Call is one each, CallMany one per
// FrameCap messages): the round trips of a tool run, and what tests count.
func (c *Client) Frames() int { return c.frames }

func frameTrace() func(seq, messages int, key string) {
	if os.Getenv(TraceEnv) != "1" {
		return nil
	}
	return func(seq, messages int, key string) {
		fmt.Fprintf(os.Stderr, "box frame seq=%d messages=%d key=%s\n", seq, messages, key)
	}
}

// exchange POSTs one frame of messages and returns the decoded reply frame. channel and key name
// the request in its errors; the reply is refused past MaxReplyBytes or MaxReplyNesting.
func (c *Client) exchange(ctx context.Context, msgs []any, channel, key string) (any, error) {
	c.seq++
	frame := obj("p", "box", "v", ProtocolVersion, "m", msgs, "n", c.seq)
	if pyjson.Truthy(c.sid) {
		frame.Set("id", c.sid)
	}
	data, err := pyjson.Marshal(frame, pyjson.Options{})
	if err != nil {
		return nil, err
	}
	c.frames++
	if c.OnFrame != nil {
		c.OnFrame(c.seq, len(msgs), key)
	}
	resp, err := c.do(ctx, "POST", "/box/", data, map[string]string{"Content-Type": "application/json"},
		0, c.MaxReplyBytes+1)
	if err != nil {
		return nil, c.mapError(err, channel, key, false)
	}
	if len(resp.body) > c.MaxReplyBytes {
		return nil, boxError(fmt.Sprintf("BOX reply larger than %d bytes: refused, not parsed in part",
			c.MaxReplyBytes), channel, key)
	}
	reply, err := pyjson.Loads(resp.body)
	if err != nil {
		return nil, boxError("transport failure: "+err.Error(), channel, key)
	}
	if nestsDeeper(reply, MaxReplyNesting) {
		return nil, malformed(fmt.Sprintf("nested deeper than %d levels", MaxReplyNesting), channel, key)
	}
	return reply, nil
}

// CallMany sends msgs in as few frames as FrameCap (MaxFrameMessages) allows and returns one Reply
// per message, in msgs order. BBoxService answers each message of a frame on its own (an error is
// that entry's "t": "e" reply, never the frame's), so a failing message is its Reply.Err and its
// siblings are unaffected. A frame whose reply is malformed (an r missing, repeated, out of range
// or not an integer), too large, too nested, refused (AuthError) or never answered fails the call:
// the replies of the frames before it are dropped, the messages having reached the station.
func (c *Client) CallMany(msgs []Msg) ([]Reply, error) {
	limit := c.FrameCap
	if limit <= 0 {
		limit = MaxFrameMessages
	}
	out := make([]Reply, 0, len(msgs))
	for start := 0; start < len(msgs); start += limit {
		part, err := c.frame(c.interrupt, msgs[start:min(start+limit, len(msgs))])
		if err != nil {
			return nil, err
		}
		out = append(out, part...)
	}
	return out, nil
}

// frame is one CallMany frame: the messages with r = 0..n-1, the replies matched back by r.
func (c *Client) frame(ctx context.Context, msgs []Msg) ([]Reply, error) {
	wire := make([]any, len(msgs))
	for i, m := range msgs {
		wire[i] = obj("r", i, "t", "rt", "c", m.Channel, "k", m.Key, "b", m.Body)
	}
	first := msgs[0]
	reply, err := c.exchange(ctx, wire, first.Channel, first.Key)
	if err != nil {
		return nil, err
	}
	entries, err := c.messages(reply, first.Channel, first.Key)
	if err != nil {
		return nil, err
	}
	if len(entries) != len(msgs) {
		return nil, malformed(fmt.Sprintf("%d replies to %d messages", len(entries), len(msgs)),
			first.Channel, first.Key)
	}
	out := make([]Reply, len(msgs))
	got := make([]bool, len(msgs))
	for _, raw := range entries {
		entry, _ := raw.(*pyjson.Object)
		i, ok := replyIndex(entry, len(msgs))
		if !ok || got[i] {
			return nil, malformed("reply entry with a missing, repeated or out-of-range r",
				first.Channel, first.Key)
		}
		got[i] = true
		body, err := entryBody(entry, msgs[i].Channel, msgs[i].Key)
		out[i] = Reply{Body: body, Err: err}
	}
	return out, nil
}

// replyIndex is the r of a reply entry when it is an integer in 0..n-1.
func replyIndex(entry *pyjson.Object, n int) (int, bool) {
	if entry == nil {
		return 0, false
	}
	r, _ := entry.Get("r")
	num, ok := r.(pyjson.Number)
	if !ok {
		return 0, false
	}
	v, ok := num.BigInt()
	if !ok || !v.IsInt64() || v.Int64() < 0 || v.Int64() >= int64(n) {
		return 0, false
	}
	return int(v.Int64()), true
}
