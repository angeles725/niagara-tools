package jsonrpc

import (
	"errors"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// A station drops an idle or tunnelled BOX session and then answers every call with "Could not
// find Server Session" (retro 2026-11-16 U4: four drops in about 40 minutes; again four in one
// afternoon on 2026-11-18). The server reconnects once with the arguments of the last n4_connect
// that succeeded, so the identity, credential and lock-out checks of n4_connect all run again.
// A read or a dry run is then retried once; an execution is not, because its confirmation token
// was bound to the lost session: the operator redoes the dry run.

const sessionLostText = "Could not find Server Session"

const (
	msgSessionLost     = "the station dropped this server's BOX session (" + sessionLostText + ")"
	msgSessionLostRedo = msgSessionLost + "; mcp-n4 reconnected with the last n4_connect, which invalidates every confirmation token: " +
		"ops sent before the loss may have applied, so check n4_list_batches and read the targets back, then redo the dry run and execute with its new token"
	msgSessionLostRetry = msgSessionLost + "; reconnecting with the last n4_connect failed: "
)

// run is the tool handler call, with one reconnect after a lost session.
func (s *Server) run(name string, args *pyjson.Object, h Handler) (*pyjson.Object, error) {
	out, err := Recover(func() (*pyjson.Object, error) { return h(args) })
	if name == "n4_connect" {
		if err == nil {
			s.lastConnect = args
		}
		return out, err
	}
	if !sessionLost(err) || s.lastConnect == nil || s.Handlers["n4_connect"] == nil {
		return out, err
	}
	connect := s.Handlers["n4_connect"]
	if _, cerr := Recover(func() (*pyjson.Object, error) { return connect(s.lastConnect) }); cerr != nil {
		return nil, &ToolError{Msg: msgSessionLostRetry + ErrorText(cerr)}
	}
	if s.writeNames[name] && executes(args) {
		return nil, &ToolError{Msg: msgSessionLostRedo}
	}
	return Recover(func() (*pyjson.Object, error) { return h(args) })
}

// sessionLost is true for the station's lost-session reply, also when a tool passed it on as its
// own error text.
func sessionLost(err error) bool {
	var be *box.Error
	var te *ToolError
	switch {
	case errors.As(err, &be):
		return strings.Contains(be.Msg, sessionLostText)
	case errors.As(err, &te):
		return strings.Contains(te.Msg, sessionLostText)
	}
	return false
}

// executes is true when a write call is the execution (dry_run false), not the dry run.
func executes(args *pyjson.Object) bool {
	v, ok := args.Get("dry_run")
	return ok && v == false
}
