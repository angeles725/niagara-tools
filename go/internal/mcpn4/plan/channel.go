package plan

import (
	"fmt"
	"sort"
	"strings"

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

// ChannelAllowList is every channel op the engine sends, as channel: keys (data). Any other
// channel or key is refused before sending: a channel op can only do what a tool was built for.
var ChannelAllowList = map[string][]string{"alarm": {"ackAlarms"}}

// Refusals of a channel op, as data.
const (
	msgNoChannel  = "the station client cannot call the BOX channel %q (key %q): nothing was sent"
	msgNotAllowed = "channel %q key %q is not an allowed channel op (allowed: %s): nothing was sent"
	msgNotPlanned = "channel %q key %q is not an allowed channel op (allowed: %s): nothing was planned"
)

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
	if !allowedChannelOp(channel, key) {
		return nil, &jsonrpc.ToolError{Msg: fmt.Sprintf(msgNotAllowed, channel, key, allowedText())}
	}
	cc, ok := c.(ChannelCaller)
	if !ok {
		return nil, NoChannelError(channel, key)
	}
	return cc.Call(channel, key, field(o, "body"))
}

// checkChannelOps refuses a plan holding a channel op outside ChannelAllowList, before its
// token and journal intent exist; sendChannel checks again as defence in depth.
func checkChannelOps(ops []any) error {
	for _, op := range ops {
		o, _ := op.(*pyjson.Object)
		if _, ok := o.Get("channel"); !ok {
			continue
		}
		channel, key := pyjson.Str(field(o, "channel")), pyjson.Str(field(o, "key"))
		if !allowedChannelOp(channel, key) {
			return &jsonrpc.ToolError{Msg: fmt.Sprintf(msgNotPlanned, channel, key, allowedText())}
		}
	}
	return nil
}

// allowedChannelOp is whether key on channel is in ChannelAllowList.
func allowedChannelOp(channel, key string) bool {
	for _, k := range ChannelAllowList[channel] {
		if k == key {
			return true
		}
	}
	return false
}

// allowedText is ChannelAllowList as "channel key, ...", sorted.
func allowedText() string {
	var pairs []string
	for channel, keys := range ChannelAllowList {
		for _, k := range keys {
			pairs = append(pairs, channel+" "+k)
		}
	}
	sort.Strings(pairs)
	return strings.Join(pairs, ", ")
}
