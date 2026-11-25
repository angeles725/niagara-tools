// Package jsonrpc is the newline-delimited JSON-RPC 2.0 layer of the mcp-n4 server
// (mcp-n4-kit/mcp_n4/server.py:22-190): line framing, dispatch of initialize, ping,
// tools/list and tools/call (plus prompts/* and resources/* for runbooks, runbooks.go), argument validation and the reply shapes. Every reply is encoded
// with pyjson exactly as json.dumps writes it.
package jsonrpc

import (
	"errors"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/registry"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// JSON-RPC error codes (server.py:24).
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParamsC = -32602
)

// NotImplemented ends the tool error of a tool this version of the Go server does not run
// yet; the black-box harness (tests/go_server.py) skips a test that meets it.
const NotImplemented = "is not implemented in this version of the Go server"

// ReasonNotConnected is safety.REASON_NOT_CONNECTED.
const ReasonNotConnected = safety.ReasonNotConnected

// Handler runs one tool on validated arguments. Its result becomes structuredContent and the
// text content. A *ToolError, *box.Error or *pyjson.ValueError is shown to the model (an auth
// *box.Error latches the Context first); *pyjson.OverflowError and any other error become
// "internal error (<Python type>)".
type Handler func(args *pyjson.Object) (*pyjson.Object, error)

// ToolError is tools_read.ToolError (and box.BoxError): a failure whose text is safe to show.
type ToolError struct{ Msg string }

func (e *ToolError) Error() string { return e.Msg }

// Context is the per-process state a tool call consults (session.Context): the station session,
// the authentication-failure latch and the secret scrub of every tool error text.
type Context interface {
	Connected() bool
	AuthBlock() string
	NoteAuthFailure(err error)
	Scrub(text string) string
	Close()
}

// PyTyped is an error that names the Python exception type it stands for.
type PyTyped interface{ PyType() string }

// Server holds what one server process answers with.
type Server struct {
	reg        *registry.Registry
	tools      []registry.Tool // listed by tools/list and callable, in table order
	byName     map[string]*registry.Tool
	writeNames map[string]bool
	// Handlers maps a tool name to its implementation; a tool without one answers the
	// NotImplemented tool error.
	Handlers map[string]Handler
	// Background is the work a handler left running after its reply (a write past its reply
	// budget); the serve loop waits for it before the next request line.
	Background *Background
	// Ctx is the session state; nil means no session can exist (every session tool answers
	// not connected) and nothing to scrub.
	Ctx Context
	// Runbooks are served as MCP prompts and resources (runbooks.go); none: those methods and
	// capabilities are absent.
	Runbooks []Runbook
	// lastConnect is the arguments of the last n4_connect that succeeded: a lost BOX session is
	// reopened with them (session_loss.go).
	lastConnect *pyjson.Object
}

// New returns the read-only server over reg: the read tools are listed and callable, and a
// write tool name is refused as disabled (server.py without --allow-writes).
func New(reg *registry.Registry) *Server {
	s := &Server{reg: reg, tools: reg.Read, byName: map[string]*registry.Tool{},
		writeNames: map[string]bool{}, Handlers: map[string]Handler{}, Background: &Background{}}
	for i := range s.tools {
		s.byName[s.tools[i].Name] = &s.tools[i]
	}
	for _, t := range reg.Write {
		s.writeNames[t.Name] = true
	}
	return s
}

// EnableWrites lists and serves the write tools after the read tools (--allow-writes), as
// tools_read.TOOLS + tools_write.TOOLS; a write tool without a handler is not ported yet.
func (s *Server) EnableWrites() {
	if len(s.tools) > len(s.reg.Read) {
		return
	}
	s.tools = append(append([]registry.Tool{}, s.reg.Read...), s.reg.Write...)
	for i := range s.tools {
		s.byName[s.tools[i].Name] = &s.tools[i]
	}
}

func encode(v any) []byte {
	out, err := pyjson.Marshal(v, pyjson.Options{})
	if err != nil { // every value here was decoded or built by this package
		panic("jsonrpc: unencodable reply: " + err.Error())
	}
	return out
}

func obj(kv ...any) *pyjson.Object {
	o := pyjson.NewObject()
	for i := 0; i < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
	return o
}

func errorReply(id any, code int, message string) *pyjson.Object {
	return obj("jsonrpc", "2.0", "id", id, "error", obj("code", code, "message", message))
}

// Dispatch is Server.dispatch: the reply object, or nil for a notification (no "id" key).
func (s *Server) Dispatch(msg any) *pyjson.Object {
	m, ok := msg.(*pyjson.Object)
	if !ok {
		return errorReply(nil, InvalidRequest, "request must be a JSON object")
	}
	method, _ := m.Get("method")
	params, _ := m.Get("params")
	id, hasID := m.Get("id")
	if !hasID {
		return nil
	}
	name, isStr := method.(string)
	var result any
	var err error
	switch {
	case !isStr: // "%s" % method: None, a number, a list...
		return errorReply(id, MethodNotFound, "method not found: "+pyjson.Str(method))
	case name == "initialize":
		result = s.initialize(params)
	case name == "ping":
		result = pyjson.NewObject()
	case name == "tools/list":
		list := make([]any, len(s.tools))
		for i, t := range s.tools {
			list[i] = t.Descriptor
		}
		result = obj("tools", list)
	case name == "tools/call":
		result, err = s.call(params)
	case s.isRunbookMethod(name):
		result, err = s.runbookMethod(name, params)
	default:
		return errorReply(id, MethodNotFound, "method not found: "+name)
	}
	var ip *InvalidParams
	var mnf *methodNotFound
	var rnf *resourceNotFound
	switch {
	case errors.As(err, &rnf):
		return errorReply(id, ResourceNotFound, rnf.msg)
	case errors.As(err, &ip):
		return errorReply(id, InvalidParamsC, ip.Msg)
	case errors.As(err, &mnf):
		return errorReply(id, MethodNotFound, mnf.msg)
	case err != nil: // a SchemaError: Python lets it escape and the process dies
		panic("jsonrpc: " + err.Error())
	}
	return obj("jsonrpc", "2.0", "id", id, "result", result)
}

type methodNotFound struct{ msg string }

func (e *methodNotFound) Error() string { return e.msg }

func (s *Server) initialize(params any) *pyjson.Object {
	version := s.reg.ProtocolVersions[0]
	if p, ok := params.(*pyjson.Object); ok {
		if asked, ok := lookup(p, "protocolVersion").(string); ok {
			for _, v := range s.reg.ProtocolVersions {
				if v == asked {
					version = asked
				}
			}
		}
	}
	caps := obj("tools", obj("listChanged", false))
	if len(s.Runbooks) > 0 {
		caps.Set("prompts", obj("listChanged", false))
		caps.Set("resources", obj("listChanged", false))
	}
	return obj("protocolVersion", version,
		"capabilities", caps,
		"serverInfo", obj("name", s.reg.Name, "version", s.reg.Version))
}
