package jsonrpc_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/registry"
)

var testBook = jsonrpc.Runbook{
	Name: "post-deploy-check", Title: "Post-deploy check", Description: "After a module deploy",
	Args: []jsonrpc.RunbookArg{
		{Name: "station", Description: "Station name", Required: true},
		{Name: "scope", Description: "Slot ORD", Default: "station:|slot:/"},
	},
	Body: "Connect to {{station}} and read {{scope}}.",
}

func runbookServer() *jsonrpc.Server {
	s := jsonrpc.New(registry.Default())
	s.Runbooks = []jsonrpc.Runbook{testBook}
	return s
}

// call sends one request line and returns its decoded reply.
func call(t *testing.T, s *jsonrpc.Server, line string) *pyjson.Object {
	t.Helper()
	out, ok := s.HandleLine([]byte(line))
	if !ok {
		t.Fatalf("no reply to %s", line)
	}
	v, err := pyjson.Unmarshal(out)
	if err != nil {
		t.Fatalf("reply %s: %v", out, err)
	}
	return v.(*pyjson.Object)
}

func text(t *testing.T, s *jsonrpc.Server, line string) string {
	t.Helper()
	out, _ := s.HandleLine([]byte(line))
	return string(out)
}

func TestInitializeAdvertisesPromptsAndResourcesOnlyWithRunbooks(t *testing.T) {
	init := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`
	with := text(t, runbookServer(), init)
	if !strings.Contains(with, `"capabilities": {"tools": {"listChanged": false}, "prompts": {"listChanged": false}, "resources": {"listChanged": false}}`) {
		t.Fatalf("with runbooks: %s", with)
	}
	without := text(t, jsonrpc.New(registry.Default()), init)
	if !strings.Contains(without, `"capabilities": {"tools": {"listChanged": false}}`) {
		t.Fatalf("without runbooks: %s", without)
	}
}

func TestPromptsListNamesEachRunbookWithItsArguments(t *testing.T) {
	got := text(t, runbookServer(), `{"jsonrpc":"2.0","id":2,"method":"prompts/list"}`)
	want := `{"jsonrpc": "2.0", "id": 2, "result": {"prompts": [{"name": "post-deploy-check", "title": "Post-deploy check", ` +
		`"description": "After a module deploy", "arguments": [{"name": "station", "description": "Station name", "required": true}, ` +
		`{"name": "scope", "description": "Slot ORD (default: station:|slot:/)", "required": false}]}]}}`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestPromptsGetFillsArgumentsAndDefaults(t *testing.T) {
	got := text(t, runbookServer(), `{"jsonrpc":"2.0","id":3,"method":"prompts/get","params":{"name":"post-deploy-check","arguments":{"station":"Prueba_4_15"}}}`)
	want := `{"jsonrpc": "2.0", "id": 3, "result": {"description": "After a module deploy", "messages": [{"role": "user", ` +
		`"content": {"type": "text", "text": "Connect to Prueba_4_15 and read station:|slot:/."}}]}}`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestPromptsGetRefusesUnknownNameMissingOrUnknownArgument(t *testing.T) {
	s := runbookServer()
	for _, c := range []struct{ params, msg string }{
		{`{"name":"nope"}`, "unknown prompt: nope (known: post-deploy-check)"},
		{`{"name":"post-deploy-check"}`, "prompt post-deploy-check needs argument station"},
		{`{"name":"post-deploy-check","arguments":{"station":"S","colour":"x"}}`, "prompt post-deploy-check has no argument colour"},
		{`{"name":"post-deploy-check","arguments":{"station":7}}`, "prompt argument station must be a string"},
		{`{}`, "prompts/get needs a string 'name'"},
	} {
		r := call(t, s, `{"jsonrpc":"2.0","id":4,"method":"prompts/get","params":`+c.params+`}`)
		e, _ := r.Get("error")
		eo, ok := e.(*pyjson.Object)
		if !ok {
			t.Fatalf("%s: no error in %v", c.params, r)
		}
		code, _ := eo.Get("code")
		msg, _ := eo.Get("message")
		if pyjson.Str(code) != "-32602" || msg != c.msg {
			t.Fatalf("%s: code %v message %q, want -32602 %q", c.params, code, msg, c.msg)
		}
	}
}

func TestResourcesListAndReadServeTheRunbookText(t *testing.T) {
	s := runbookServer()
	list := text(t, s, `{"jsonrpc":"2.0","id":5,"method":"resources/list"}`)
	wantList := `{"jsonrpc": "2.0", "id": 5, "result": {"resources": [{"uri": "runbook://post-deploy-check", "name": "post-deploy-check", ` +
		`"title": "Post-deploy check", "description": "After a module deploy", "mimeType": "text/markdown"}]}}`
	if list != wantList {
		t.Fatalf("got  %s\nwant %s", list, wantList)
	}
	read := text(t, s, `{"jsonrpc":"2.0","id":6,"method":"resources/read","params":{"uri":"runbook://post-deploy-check"}}`)
	wantRead := `{"jsonrpc": "2.0", "id": 6, "result": {"contents": [{"uri": "runbook://post-deploy-check", "mimeType": "text/markdown", ` +
		`"text": "Connect to {{station}} and read {{scope}}."}]}}`
	if read != wantRead {
		t.Fatalf("got  %s\nwant %s", read, wantRead)
	}
	missing := text(t, s, `{"jsonrpc":"2.0","id":7,"method":"resources/read","params":{"uri":"runbook://nope"}}`)
	if missing != `{"jsonrpc": "2.0", "id": 7, "error": {"code": -32002, "message": "resource not found: runbook://nope"}}` {
		t.Fatalf("missing: %s", missing)
	}
}

func TestPromptAndResourceMethodsStayUnknownWithoutRunbooks(t *testing.T) {
	s := jsonrpc.New(registry.Default())
	for _, m := range []string{"prompts/list", "prompts/get", "resources/list", "resources/read"} {
		got := text(t, s, `{"jsonrpc":"2.0","id":8,"method":"`+m+`"}`)
		if got != `{"jsonrpc": "2.0", "id": 8, "error": {"code": -32601, "message": "method not found: `+m+`"}}` {
			t.Fatalf("%s: %s", m, got)
		}
	}
}
