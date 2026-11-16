package blackbox

// A whole session over stdio, the way an MCP client runs the server (port of
// test_server.TestStdioEndToEnd).

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

const e2ePassword = "e2e-pw-value-77"

// runServer feeds lines to a server started with args, closes stdin and waits: exit code, the
// decoded replies and stderr.
func runServer(t *testing.T, lines []string, args ...string) (int, []map[string]any, string, string) {
	t.Helper()
	p := Spawn(t, Options{Extra: args, Env: map[string]string{"MCP_N4_USER": User, "MCP_N4_PASSWORD": e2ePassword}})
	for _, line := range lines {
		if err := p.Send(line); err != nil {
			t.Fatal(err)
		}
	}
	p.CloseStdin()
	code, rest, stderr := p.Wait(t)
	var replies []map[string]any
	for _, line := range rest {
		var reply map[string]any
		if err := json.Unmarshal([]byte(line), &reply); err != nil {
			t.Fatalf("stdout line %q: %v", line, err)
		}
		replies = append(replies, reply)
	}
	return code, replies, strings.Join(rest, "\n"), stderr
}

func call(id int, tool string, args Args) string {
	return RPC("tools/call", Args{"name": tool, "arguments": args}, id)
}

func structured(reply map[string]any) map[string]any {
	result, _ := reply["result"].(map[string]any)
	out, _ := result["structuredContent"].(map[string]any)
	return out
}

func TestSessionOverStdioAgainstTheFakeStation(t *testing.T) {
	st := boxstation.New(User, e2ePassword, StationName).Start(t)
	code, replies, stdout, stderr := runServer(t, []string{
		RPC("initialize", Args{"protocolVersion": "2025-06-18"}, 1), RPC("notifications/initialized", nil, nil),
		RPC("tools/list", nil, 2), call(3, "n4_connect", Args{"station": StationName}),
		call(4, "n4_navigate", Args{}), call(5, "n4_describe_session", Args{}),
	}, "--station", StationName+"="+st.URL, "--allow-http-for-tests")
	if code != 0 || len(replies) != 5 {
		t.Fatalf("exit %d, %d replies; stderr %s", code, len(replies), stderr)
	}
	for i, reply := range replies {
		if reply["id"] != float64(i+1) {
			t.Errorf("reply %d has id %v", i, reply["id"])
		}
	}
	if name := Dig(replies[0], "result", "serverInfo", "name"); name != "mcp-n4" {
		t.Errorf("serverInfo name %v", name)
	}
	if tools, _ := Dig(replies[1], "result", "tools").([]any); len(tools) != len(readTools) {
		t.Errorf("%d tools", len(tools))
	}
	if got := structured(replies[2])["station_name"]; got != StationName {
		t.Errorf("station_name %v", got)
	}
	children, _ := structured(replies[3])["children"].([]any)
	if !strings.Contains(jsonText(children), `"name":"Folder"`) {
		t.Errorf("children %v", children)
	}
	if structured(replies[4])["connected"] != true {
		t.Errorf("describe %v", replies[4])
	}
	if len(st.Sessions()) != 0 || strings.Contains(stdout+stderr, e2ePassword) {
		t.Errorf("sessions %v; the password shows in the output: %v", st.Sessions(), strings.Contains(stdout+stderr, e2ePassword))
	}
}

func TestHTTPStationIsRefusedWithoutTheFlagOverStdio(t *testing.T) {
	st := boxstation.New(User, e2ePassword, StationName).Start(t)
	code, replies, _, stderr := runServer(t, nil, "--station", StationName+"="+st.URL)
	if code == 0 || replies != nil || !strings.Contains(stderr, "https") || st.Requests() != 0 {
		t.Errorf("exit %d, replies %v, stderr %q, requests %d", code, replies, stderr, st.Requests())
	}
}

func TestConnectWithoutAConfiguredStationIsRefusedOverStdio(t *testing.T) {
	_, replies, _, _ := runServer(t, []string{call(1, "n4_connect", Args{"station": StationName})})
	result, _ := replies[0]["result"].(map[string]any)
	text := jsonText(result["content"])
	if result["isError"] != true || !strings.Contains(text, "--station") {
		t.Errorf("reply %v", replies[0])
	}
}

func jsonText(v any) string {
	out, _ := json.Marshal(v)
	return string(out)
}
