package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	kitcli "github.com/angeles725/niagara-tools/go/internal/cli"
	mcpcli "github.com/angeles725/niagara-tools/go/internal/mcpn4/cli"
)

// describeSession serves one n4_describe_session call and returns its server_version and
// server.build.
func describeSession(t *testing.T) (version, build string) {
	t.Helper()
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"n4_describe_session","arguments":{}}}` + "\n")
	var out, errb bytes.Buffer
	if code := mcpcli.Run([]string{"--station", "LAB=https://127.0.0.1"}, in, kitcli.Stdio{Out: &out, Err: &errb}); code != 0 {
		t.Fatalf("server exit %d, stderr %q", code, errb.String())
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	var resp struct {
		Result struct {
			Content []struct{ Text string } `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &resp); err != nil || len(resp.Result.Content) == 0 {
		t.Fatalf("describe_session reply %q: %v", lines[len(lines)-1], err)
	}
	var body struct {
		ServerVersion string `json:"server_version"`
		Server        struct {
			Build string `json:"build"`
		} `json:"server"`
	}
	if err := json.Unmarshal([]byte(resp.Result.Content[0].Text), &body); err != nil {
		t.Fatalf("describe_session text %q: %v", resp.Result.Content[0].Text, err)
	}
	return body.ServerVersion, body.Server.Build
}

func TestVersionAgreesWithDescribeSession(t *testing.T) {
	version, build := describeSession(t)
	if version == "" || build == "" {
		t.Fatalf("describe_session gave version %q build %q", version, build)
	}
	var out, errb bytes.Buffer
	if code := run([]string{"--version"}, kitcli.Stdio{Out: &out, Err: &errb}); code != 0 {
		t.Fatalf("--version exit %d, stderr %q", code, errb.String())
	}
	if want := "mcp-n4 " + version + " (git " + build + ")\n"; out.String() != want {
		t.Fatalf("--version = %q, want %q", out.String(), want)
	}
}
