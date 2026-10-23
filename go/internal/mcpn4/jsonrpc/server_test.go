package jsonrpc

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/registry"
)

func handle(t *testing.T, s *Server, line string) string {
	t.Helper()
	out, ok := s.HandleLine([]byte(line))
	if !ok {
		return "<no reply>"
	}
	return string(out)
}

func echoServer(h Handler) *Server {
	s := New(registry.Default())
	s.tools = append([]registry.Tool{{Name: "echo", Schema: obj("type", "object", "properties", obj())}}, s.tools...)
	s.byName = map[string]*registry.Tool{"echo": &s.tools[0]}
	s.Handlers["echo"] = h
	return s
}

const callEcho = `{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"name": "echo", "arguments": {}}}`

func TestToolResultCarriesTextAndStructuredContent(t *testing.T) {
	t.Parallel()
	s := echoServer(func(*pyjson.Object) (*pyjson.Object, error) { return obj("a", 1, "é", "<&>"), nil })
	want := `{"jsonrpc": "2.0", "id": 1, "result": {"content": [{"type": "text", "text": ` +
		`"{\"a\": 1, \"\\u00e9\": \"<&>\"}"}], "structuredContent": {"a": 1, "\u00e9": "<&>"}}}`
	if got := handle(t, s, callEcho); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

type runtimeError struct{}

func (runtimeError) Error() string  { return "topsecret-value in /home/x/file.py" }
func (runtimeError) PyType() string { return "RuntimeError" }

func TestUnexpectedToolExceptionIsShortErrorWithoutMessageOrTrace(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		err  error
		want string
	}{
		{runtimeError{}, "internal error (RuntimeError)"},
		{&pyjson.OverflowError{}, "internal error (OverflowError)"},
		{errors.New("topsecret"), "internal error (Exception)"},
	} {
		s := echoServer(func(*pyjson.Object) (*pyjson.Object, error) { return nil, tc.err })
		want := `{"jsonrpc": "2.0", "id": 1, "result": {"content": [{"type": "text", "text": "` +
			tc.want + `"}], "isError": true}}`
		if got := handle(t, s, callEcho); got != want {
			t.Errorf("got  %s\nwant %s", got, want)
		}
	}
}

func TestToolErrorAndBoxErrorMessagesAreShown(t *testing.T) {
	t.Parallel()
	s := echoServer(func(*pyjson.Object) (*pyjson.Object, error) { return nil, &ToolError{"clear reason"} })
	if got := handle(t, s, callEcho); !strings.Contains(got, `"text": "clear reason"}], "isError": true`) {
		t.Errorf("got %s", got)
	}
}

func TestIdNullIsARequestAndGetsAReply(t *testing.T) {
	t.Parallel()
	got := handle(t, New(registry.Default()), `{"jsonrpc":"2.0","id":null,"method":"ping"}`)
	if want := `{"jsonrpc": "2.0", "id": null, "result": {}}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestMessageWithoutIdIsStillANotification(t *testing.T) {
	t.Parallel()
	if got := handle(t, New(registry.Default()), `{"jsonrpc":"2.0","method":"ping"}`); got != "<no reply>" {
		t.Errorf("got %s", got)
	}
}

func TestUnportedToolsAnswerNotImplementedAndSessionToolsNotConnected(t *testing.T) {
	t.Parallel()
	s := New(registry.Default())
	got := handle(t, s, `{"id": 1, "method": "tools/call", "params": {"name": "n4_connect", "arguments": {"station": "S"}}}`)
	if !strings.Contains(got, `"n4_connect `+NotImplemented+`"}], "isError": true`) {
		t.Errorf("n4_connect: %s", got)
	}
	got = handle(t, s, `{"id": 2, "method": "tools/call", "params": {"name": "n4_navigate"}}`)
	if !strings.Contains(got, `"text": "not connected: call n4_connect first"`) {
		t.Errorf("n4_navigate: %s", got)
	}
}

type brokenReader struct{ data []byte }

func (b *brokenReader) Read(p []byte) (int, error) {
	if len(b.data) == 0 {
		return 0, errors.New("stream broke")
	}
	n := copy(p, b.data)
	b.data = b.data[n:]
	return n, nil
}

func TestServeClosesTheSessionWhenTheLoopRaises(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	err := New(registry.Default()).Serve(&brokenReader{[]byte("{\"id\":1,\"method\":\"ping\"}\n")}, &out)
	if err == nil || err.Error() != "stream broke" {
		t.Errorf("Serve error = %v, want the read error", err)
	}
	if out.String() != "{\"jsonrpc\": \"2.0\", \"id\": 1, \"result\": {}}\n" {
		t.Errorf("out = %q", out.String())
	}
}

func TestServeSplitsOnNewlineOnlyAndHandlesTheLastLine(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	in := "\r\n{\"id\":1,\"method\":\"ping\"}\r\n\v\f{\"id\":2,\"method\":\"x\"}\r{\"id\":3,\"method\":\"ping\"}"
	if err := New(registry.Default()).Serve(strings.NewReader(in), &out); err != nil {
		t.Fatal(err)
	}
	want := `{"jsonrpc": "2.0", "id": 1, "result": {}}` + "\n" + `{"jsonrpc": "2.0", "id": null, "error": {"code": -32700, "message": "parse error"}}` + "\n"
	if out.String() != want {
		t.Errorf("out = %q\nwant  %q", out.String(), want)
	}
}

func TestStripRemovesPythonWhitespaceOnly(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"  \t{}\r\n":                     "{}",
		"\u00a0\u3000\x1c{}\u2028\u0085": "{}",
		"\ufeff{} ":                      "\ufeff{}", // a BOM is not whitespace
		"\xff \u00a0":                    "\xff",     // an undecodable byte stops the strip
		"\xa0":                           "\xa0",     // a lone continuation byte is not NBSP
		" \t ":                           "",
	} {
		if got := string(Strip([]byte(in))); got != want {
			t.Errorf("Strip(%q) = %q, want %q", in, got, want)
		}
	}
}
