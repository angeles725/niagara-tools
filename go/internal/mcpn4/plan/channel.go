package plan

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Channel ops: a write that is one BOX call on a channel other than the session's (the alarm
// channel's ackAlarms), planned as {channel, key, body}, journaled like any op and sent through a
// client that can Call. The ssc and sync paths are unchanged.

// ChannelCaller is a client that can make one BOX call on any channel (box.Client.Call).
type ChannelCaller interface {
	Call(channel, key string, body any) (any, error)
}

// msgNoChannel is the refusal of a channel op by a client that cannot Call.
const msgNoChannel = "the station client cannot call the BOX channel %q (key %q): nothing was sent"

// ChannelOp is the planned op of one call of key on channel with body.
func ChannelOp(channel, key string, body any) *pyjson.Object {
	return obj("channel", channel, "key", key, "body", body)
}

// CanCall is whether c can send channel ops; a tool planning one refuses before its token when not.
func CanCall(c Client) bool {
	_, ok := c.(ChannelCaller)
	return ok
}

// NoChannelError is the refusal of a channel op on a client that cannot Call.
func NoChannelError(channel, key string) error {
	return &jsonrpc.ToolError{Msg: fmt.Sprintf(msgNoChannel, channel, key)}
}

// sendChannel sends the channel op o, or refuses it before sending when c cannot Call.
func sendChannel(c Client, o *pyjson.Object) (any, error) {
	channel, key := pyjson.Str(field(o, "channel")), pyjson.Str(field(o, "key"))
	cc, ok := c.(ChannelCaller)
	if !ok {
		return nil, NoChannelError(channel, key)
	}
	return cc.Call(channel, key, field(o, "body"))
}
