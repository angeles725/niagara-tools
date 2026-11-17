package blackbox

// The JSON-RPC protocol over stdio (port of test_server.TestProtocol).

import (
	"reflect"
	"strings"
	"testing"
)

func TestInitializeReportsServerInfoAndCapabilities(t *testing.T) {
	res := Start(t, Options{}).Request("initialize", Args{"protocolVersion": "2025-03-26"})
	result, _ := res["result"].(map[string]any)
	want := map[string]any{"serverInfo": map[string]any{"name": "mcp-n4", "version": serverVersion},
		"capabilities": map[string]any{"tools": map[string]any{"listChanged": false},
			"prompts": map[string]any{"listChanged": false}, "resources": map[string]any{"listChanged": false}},
		"protocolVersion": "2025-03-26"}
	if res["id"] != 1.0 || !reflect.DeepEqual(result, want) {
		t.Errorf("initialize = %v", res)
	}
}

func protocolOf(s *Server, params any) any {
	result, _ := s.Request("initialize", params)["result"].(map[string]any)
	return result["protocolVersion"]
}

func TestInitializeFallsBackToNewestSupportedVersion(t *testing.T) {
	s := Start(t, Options{})
	for _, params := range []any{Args{"protocolVersion": "1999-01-01"}, nil} {
		if got := protocolOf(s, params); got != supportedProtocols[0] {
			t.Errorf("params %v: protocolVersion %v", params, got)
		}
	}
}

func TestANewerProtocolVersionIsNotEchoedUntilItsSemanticsExist(t *testing.T) {
	s := Start(t, Options{})
	for _, v := range supportedProtocols {
		if got := protocolOf(s, Args{"protocolVersion": v}); got != v {
			t.Errorf("%s answered %v", v, got)
		}
	}
	if got := protocolOf(s, Args{"protocolVersion": "2025-11-25"}); got != "2025-06-18" {
		t.Errorf("2025-11-25 answered %v", got)
	}
}

func TestInitializedNotificationGetsNoReply(t *testing.T) {
	if got := Start(t, Options{}).Notify("notifications/initialized", nil); got != nil {
		t.Errorf("reply %v", got)
	}
}

func TestPingReturnsEmptyResult(t *testing.T) {
	if got := Start(t, Options{}).Request("ping", nil); !isEmptyObject(got["result"]) {
		t.Errorf("ping = %v", got)
	}
}

func TestUnknownMethodIs32601AndUnknownNotificationIsSilent(t *testing.T) {
	s := Start(t, Options{})
	if code := ErrorCode(s.Request("nope", nil)); code != -32601 {
		t.Errorf("code %d", code)
	}
	if got := s.Notify("nope", nil); got != nil {
		t.Errorf("notification reply %v", got)
	}
}

func TestParseErrorIs32700WithNullId(t *testing.T) {
	s := Start(t, Options{})
	out := s.decode(s.Line("{not json"))
	if id, has := out["id"]; ErrorCode(out) != -32700 || !has || id != nil {
		t.Errorf("reply %v", out)
	}
}

func TestNonObjectMessageIsInvalidRequest(t *testing.T) {
	s := Start(t, Options{})
	if code := ErrorCode(s.decode(s.Line("[1, 2]"))); code != -32600 {
		t.Errorf("code %d", code)
	}
}

func TestBlankLineIsIgnored(t *testing.T) {
	if got := Start(t, Options{}).Line("   "); got != "" {
		t.Errorf("reply %s", got)
	}
}

func TestToolsListEntriesHaveSchemaAndReadOnlyAnnotations(t *testing.T) {
	tools := listTools(Start(t, Options{}))
	var names []string
	for _, tool := range tools {
		names = append(names, tool["name"].(string))
		schema, _ := tool["inputSchema"].(map[string]any)
		want := map[string]any{"readOnlyHint": true, "openWorldHint": false}
		if schema["type"] != "object" || !reflect.DeepEqual(tool["annotations"], want) || tool["description"] == "" {
			t.Errorf("%v: schema %v annotations %v", tool["name"], schema["type"], tool["annotations"])
		}
	}
	if !reflect.DeepEqual(names, readTools) {
		t.Errorf("tools %v", names)
	}
}

func TestToolsCallRejectsBadParamsWith32602(t *testing.T) {
	s := Start(t, Options{})
	for _, params := range []any{nil, Args{}, Args{"name": 5}, Args{"name": "ghost"},
		Args{"name": "n4_navigate", "arguments": []any{}}} {
		if code := ErrorCode(s.Request("tools/call", params)); code != -32602 {
			t.Errorf("%v: code %d", params, code)
		}
	}
}

func TestToolsCallValidatesRequiredTypeAndRange(t *testing.T) {
	s := Start(t, Options{})
	for _, c := range []struct {
		tool string
		args Args
	}{{"n4_connect", Args{}}, {"n4_connect", Args{"station": 5}}, {"n4_navigate", Args{"depth": 4}},
		{"n4_navigate", Args{"depth": "1"}}, {"n4_connect", Args{"station": "x", "expected_station": 7}}} {
		if code := s.CallError(c.tool, c.args); code != -32602 {
			t.Errorf("%s %v: code %d", c.tool, c.args, code)
		}
	}
}

func TestToolNeedingASessionReportsNotConnected(t *testing.T) {
	r := Start(t, Options{}).Call("n4_navigate", nil)
	if !r.IsError || !strings.Contains(r.Text, "not connected") {
		t.Errorf("result %v", r.Raw)
	}
}

// listTools is the tools/list answer's tools.
func listTools(s *Server) []map[string]any {
	result, _ := s.Request("tools/list", nil)["result"].(map[string]any)
	list, _ := result["tools"].([]any)
	var out []map[string]any
	for _, tool := range list {
		out = append(out, tool.(map[string]any))
	}
	return out
}
