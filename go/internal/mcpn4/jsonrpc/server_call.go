package jsonrpc

import (
	"errors"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

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
	if err := Validate(name, tool.Schema, args); err != nil {
		return nil, err
	}
	if tool.NeedsSession && (s.Ctx == nil || !s.Ctx.Connected()) {
		return s.toolError(ReasonNotConnected + ": call n4_connect first"), nil
	}
	if tool.NeedsSession || s.writeNames[name] { // a station call: refused while the auth latch holds (F4)
		if block := s.authBlock(); block != "" {
			return s.toolError(block), nil
		}
	}
	h := s.Handlers[name]
	if h == nil { // never scrubbed: the harness skips on this exact text
		return toolError(name + " " + NotImplemented), nil
	}
	out, err := s.run(name, args, h)
	var be *box.Error
	if errors.As(err, &be) && be.IsAuth() && s.Ctx != nil {
		// Always (re)latch: a client without the hook still latches here, and a held latch
		// may name other credentials (H2 review R4-001).
		s.Ctx.NoteAuthFailure(be)
	}
	if err != nil {
		return s.toolError(ErrorText(err)), nil
	}
	return obj("content", []any{obj("type", "text", "text", string(encode(out)))},
		"structuredContent", out), nil
}

func (s *Server) authBlock() string {
	if s.Ctx == nil {
		return ""
	}
	return s.Ctx.AuthBlock()
}

// ErrorText is the text of a failed tool: the words of a ToolError, BoxError (AuthError and
// HttpStatusError included) or ValueError, else the Python type only, never the message (it
// could hold a secret). A tool that reports a failure inside its reply words it with this.
func ErrorText(err error) string {
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
