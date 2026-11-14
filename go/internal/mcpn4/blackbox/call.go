package blackbox

import (
	"encoding/json"
)

// Args are a tool's arguments, or any JSON object.
type Args = map[string]any

// Result is a tools/call result: Structured is structuredContent, Text the first content text.
type Result struct {
	Raw        map[string]any
	IsError    bool
	Text       string
	Structured map[string]any
}

// RPC is a JSON-RPC message; id nil makes it a notification, params nil leaves them out.
func RPC(method string, params any, id any) string {
	msg := map[string]any{"jsonrpc": "2.0", "method": method}
	if id != nil {
		msg["id"] = id
	}
	if params != nil {
		msg["params"] = params
	}
	out, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return string(out)
}

// Request sends a request with id 1 and returns the decoded reply, nil when there was none.
func (s *Server) Request(method string, params any) map[string]any {
	s.t.Helper()
	return s.decode(s.Line(RPC(method, params, 1)))
}

// Notify sends a notification and returns the decoded reply (nil, as the protocol wants).
func (s *Server) Notify(method string, params any) map[string]any {
	s.t.Helper()
	return s.decode(s.Line(RPC(method, params, nil)))
}

// Initialize is the MCP handshake: initialize with protocolVersion, then the initialized
// notification; it returns the initialize result.
func (s *Server) Initialize(protocolVersion string) map[string]any {
	s.t.Helper()
	res := s.Request("initialize", Args{"protocolVersion": protocolVersion})
	if s.Notify("notifications/initialized", nil) != nil {
		s.t.Fatal("the initialized notification got a reply")
	}
	result, _ := res["result"].(map[string]any)
	return result
}

// Call is tools/call of tool with args; a JSON-RPC error (not a tool error) fails the test.
func (s *Server) Call(tool string, args Args) Result {
	s.t.Helper()
	if args == nil {
		args = Args{}
	}
	res := s.Request("tools/call", Args{"name": tool, "arguments": args})
	if _, bad := res["error"]; bad || res["result"] == nil {
		s.t.Fatalf("%s: not a tool result: %v", tool, res)
	}
	r := Result{Raw: res["result"].(map[string]any)}
	r.IsError, _ = r.Raw["isError"].(bool)
	r.Structured, _ = r.Raw["structuredContent"].(map[string]any)
	if content, _ := r.Raw["content"].([]any); len(content) > 0 {
		first, _ := content[0].(map[string]any)
		r.Text, _ = first["text"].(string)
	}
	return r
}

// OK is Call that fails the test unless the tool succeeded; it returns structuredContent.
func (s *Server) OK(tool string, args Args) map[string]any {
	s.t.Helper()
	r := s.Call(tool, args)
	if r.IsError || r.Structured == nil {
		s.t.Fatalf("%s: want success, got %v", tool, r.Raw)
	}
	return r.Structured
}

// Err is Call that fails the test unless the tool answered an error; it returns its text.
func (s *Server) Err(tool string, args Args) string {
	s.t.Helper()
	r := s.Call(tool, args)
	if !r.IsError {
		s.t.Fatalf("%s: want a tool error, got %v", tool, r.Raw)
	}
	return r.Text
}

// CallError is tools/call that expects a JSON-RPC error; it returns the error code (0: none).
func (s *Server) CallError(tool any, args any) int {
	s.t.Helper()
	params := Args{"name": tool}
	if args != nil {
		params["arguments"] = args
	}
	return ErrorCode(s.Request("tools/call", params))
}

// ErrorCode is the JSON-RPC error code of a decoded reply, 0 when it has none.
func ErrorCode(reply map[string]any) int {
	e, _ := reply["error"].(map[string]any)
	code, _ := e["code"].(float64)
	return int(code)
}

func (s *Server) decode(line string) map[string]any {
	s.t.Helper()
	if line == "" {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(line), &out); err != nil {
		s.t.Fatalf("reply is not a JSON object: %v: %s", err, line)
	}
	return out
}

// Dig walks nested JSON objects by key; it is nil where the path breaks.
func Dig(v any, path ...string) any {
	for _, key := range path {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[key]
	}
	return v
}
