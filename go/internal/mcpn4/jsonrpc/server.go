// Package jsonrpc is the newline-delimited JSON-RPC 2.0 layer of the mcp-n4 server
// (mcp-n4-kit/mcp_n4/server.py:22-190): line framing, dispatch of initialize, ping,
// tools/list and tools/call, argument validation and the reply shapes. Every reply is encoded
// with pyjson exactly as json.dumps writes it.
package jsonrpc

import (
	"bufio"
	"errors"
	"io"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
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
	// Ctx is the session state; nil means no session can exist (every session tool answers
	// not connected) and nothing to scrub.
	Ctx Context
}

// New returns the read-only server over reg: the read tools are listed and callable, and a
// write tool name is refused as disabled (server.py without --allow-writes).
func New(reg *registry.Registry) *Server {
	s := &Server{reg: reg, tools: reg.Read, byName: map[string]*registry.Tool{},
		writeNames: map[string]bool{}, Handlers: map[string]Handler{}}
	for i := range s.tools {
		s.byName[s.tools[i].Name] = &s.tools[i]
	}
	for _, t := range reg.Write {
		s.writeNames[t.Name] = true
	}
	return s
}

// Serve is Server.serve: one reply line per answered input line, flushed at once, until EOF.
// Lines end at "\n" only, as Python's stdin splits them on POSIX; a last line without one is
// still handled. A read or write error ends the loop and is returned. However the loop ends, the
// station session is closed.
func (s *Server) Serve(in io.Reader, out io.Writer) error {
	if s.Ctx != nil {
		defer s.Ctx.Close()
	}
	r := bufio.NewReader(in)
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 {
			if reply, ok := s.HandleLine(line); ok {
				if _, werr := out.Write(append(reply, '\n')); werr != nil {
					return werr
				}
			}
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// HandleLine is Server.handle_line: the reply line (without "\n") and true, or false for a
// blank line or a notification.
func (s *Server) HandleLine(line []byte) ([]byte, bool) {
	line = Strip(line)
	if len(line) == 0 {
		return nil, false
	}
	var msg any
	err := errTooDeep
	if !NestsDeeper(line, MaxNesting) { // checked first: the decoder recurses per level
		msg, err = pyjson.Unmarshal(line)
	}
	var reply *pyjson.Object
	if err != nil {
		reply = errorReply(nil, ParseError, "parse error")
	} else if reply = s.Dispatch(msg); reply == nil {
		return nil, false
	}
	return encode(reply), true
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
	default:
		return errorReply(id, MethodNotFound, "method not found: "+name)
	}
	var ip *InvalidParams
	var mnf *methodNotFound
	switch {
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
	return obj("protocolVersion", version,
		"capabilities", obj("tools", obj("listChanged", false)),
		"serverInfo", obj("name", s.reg.Name, "version", s.reg.Version))
}

func (s *Server) call(params any) (any, error) {
	p, ok := params.(*pyjson.Object)
	name, isStr := lookup(p, "name").(string)
	if !ok || !isStr {
		return nil, &InvalidParams{"tools/call needs a string 'name'"}
	}
	tool := s.byName[name]
	if tool == nil && s.writeNames[name] {
		return nil, &methodNotFound{"write tools are disabled: start the server with --allow-writes"}
	}
	if tool == nil {
		return nil, &InvalidParams{"unknown tool: " + name}
	}
	raw, present := p.Get("arguments")
	args := pyjson.NewObject()
	if present && raw != nil {
		if args, ok = raw.(*pyjson.Object); !ok {
			return nil, &InvalidParams{"'arguments' must be an object"}
		}
	}
	if err := Validate(tool.Schema, args); err != nil {
		return nil, err
	}
	if tool.NeedsSession { // a station call: refused while the auth latch holds (F4)
		if s.Ctx == nil || !s.Ctx.Connected() {
			return s.toolError(ReasonNotConnected + ": call n4_connect first"), nil
		}
		if block := s.Ctx.AuthBlock(); block != "" {
			return s.toolError(block), nil
		}
	}
	h := s.Handlers[name]
	if h == nil { // never scrubbed: the harness skips on this exact text
		return toolError(name + " " + NotImplemented), nil
	}
	out, err := h(args)
	var be *box.Error
	if errors.As(err, &be) && be.IsAuth() && s.Ctx != nil {
		// Always (re)latch: a client without the hook still latches here, and a held latch
		// may name other credentials (H2 review R4-001).
		s.Ctx.NoteAuthFailure(be)
	}
	if err != nil {
		return s.toolError(errorText(err)), nil
	}
	return obj("content", []any{obj("type", "text", "text", string(encode(out)))},
		"structuredContent", out), nil
}

// errorText is the text of a failed tool: the words of a ToolError, BoxError (AuthError and
// HttpStatusError included) or ValueError, else the Python type only, never the message (it
// could hold a secret).
func errorText(err error) string {
	var te *ToolError
	var be *box.Error
	var ve *pyjson.ValueError
	var oe *pyjson.OverflowError
	var typed PyTyped
	switch {
	case errors.As(err, &te):
		return te.Msg
	case errors.As(err, &be):
		return be.Msg
	case errors.As(err, &ve):
		return ve.Msg
	case errors.As(err, &oe):
		return "internal error (OverflowError)"
	case errors.As(err, &typed):
		return "internal error (" + typed.PyType() + ")"
	}
	return "internal error (Exception)"
}

// toolError is Server._tool_error: the text, scrubbed of the secret.
func (s *Server) toolError(text string) *pyjson.Object {
	if s.Ctx != nil {
		text = s.Ctx.Scrub(text)
	}
	return toolError(text)
}

func toolError(text string) *pyjson.Object {
	return obj("content", []any{obj("type", "text", "text", text)}, "isError", true)
}
