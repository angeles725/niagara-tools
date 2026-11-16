package plan

// The channel op kind of the write engine: {channel, key, body} is one BOX call on that channel,
// through a client that can Call; a client that cannot refuses before anything is sent.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// wire is a Client that keeps every call it is given, by kind.
type wire struct{ calls []string }

func (w *wire) LoadTree(string, int, box.LoadOptions) (*pyjson.Object, error) { return nil, nil }
func (w *wire) SSC(key string, _ any) (any, error) {
	w.calls = append(w.calls, "ssc "+key)
	return nil, nil
}
func (w *wire) Sync(any) (any, error) { w.calls = append(w.calls, "sync"); return []any{}, nil }
func (w *wire) ClearHandles()         {}

// caller is a wire that can also Call a channel.
type caller struct{ wire }

func (c *caller) Call(channel, key string, body any) (any, error) {
	c.calls = append(c.calls, channel+" "+key+" "+pyjson.Repr(body))
	return []any{"Critical"}, nil
}

func TestAChannelOpIsOneCallOnItsChannel(t *testing.T) {
	c := &caller{}
	op := ChannelOp("alarm", "ackAlarms", obj("ids", []any{"aa11"}))
	if got := pyjson.Repr(op); got != "{'channel': 'alarm', 'key': 'ackAlarms', 'body': {'ids': ['aa11']}}" {
		t.Errorf("op %s", got)
	}
	reply, err := send(c, op)
	if err != nil || pyjson.Repr(reply) != "['Critical']" {
		t.Fatalf("reply %v, %v", reply, err)
	}
	if got := strings.Join(c.calls, "\n"); got != "alarm ackAlarms {'ids': ['aa11']}" {
		t.Errorf("calls %q", got)
	}
	if !CanCall(c) || CanCall(&wire{}) {
		t.Errorf("CanCall")
	}
}

func TestAClientWithoutCallRefusesAChannelOpBeforeSending(t *testing.T) {
	w := &wire{}
	_, err := send(w, ChannelOp("alarm", "ackAlarms", obj()))
	if err == nil || !strings.Contains(err.Error(), `cannot call the BOX channel "alarm"`) ||
		!strings.Contains(err.Error(), "nothing was sent") || len(w.calls) != 0 {
		t.Fatalf("err %v, calls %q", err, w.calls)
	}
}

func TestSscAndSyncOpsKeepTheirPaths(t *testing.T) {
	c := &caller{}
	send(c, obj("ssc", "invokeAction", "arg", obj()))
	send(c, obj("nm", "s", "h", "1"))
	if got := strings.Join(c.calls, ","); got != "ssc invokeAction,sync" {
		t.Errorf("calls %q", got)
	}
}

func TestAChannelOpOutsideTheAllowListRefusesBeforeSending(t *testing.T) {
	for _, op := range [][2]string{{"alarm", "clearAllRecords"}, {"file", "ackAlarms"}} {
		c := &caller{}
		_, err := send(c, ChannelOp(op[0], op[1], obj()))
		if err == nil || !strings.Contains(err.Error(), "is not an allowed channel op") ||
			!strings.Contains(err.Error(), "nothing was sent") || len(c.calls) != 0 {
			t.Errorf("%v: err %v, calls %q", op, err, c.calls)
		}
	}
}
