package registry

import (
	"strings"
	"testing"
)

func TestDefaultHoldsEveryToolInTableOrder(t *testing.T) {
	t.Parallel()
	r := Default()
	if r.Name != "mcp-n4" || r.Version == "" || r.ProtocolVersions[0] != "2025-06-18" {
		t.Fatalf("server = %q %q %v", r.Name, r.Version, r.ProtocolVersions)
	}
	if len(r.Read) != 13 || len(r.Write) != 7 {
		t.Fatalf("got %d read and %d write tools, want 13 and 7", len(r.Read), len(r.Write))
	}
	if r.Read[0].Name != "n4_connect" || r.Read[0].NeedsSession || !r.Read[2].NeedsSession {
		t.Errorf("read tools start %q (session %v), third needs session %v",
			r.Read[0].Name, r.Read[0].NeedsSession, r.Read[2].NeedsSession)
	}
	seen := map[string]bool{}
	for _, tool := range append(append([]Tool{}, r.Read...), r.Write...) {
		if seen[tool.Name] {
			t.Errorf("tool %s is registered twice", tool.Name)
		}
		seen[tool.Name] = true
		if got := strings.Join(tool.Descriptor.Keys(), ","); got != "name,description,inputSchema,annotations" {
			t.Errorf("%s descriptor keys = %s", tool.Name, got)
		}
	}
	if !strings.HasPrefix(r.Usage, "usage: mcp_n4.server [-h]") || !strings.HasPrefix(r.Help, r.Usage) {
		t.Errorf("usage/help do not hold the argparse text")
	}
}

func TestParseRefusesAMalformedDocument(t *testing.T) {
	t.Parallel()
	base := `{"server": {"name": "x", "version": "1", "protocol_versions": ["v"]},
	          "cli": {"usage": "u", "help": "h"}, "read_tools": %s, "write_tools": []}`
	for name, tools := range map[string]string{
		"not JSON":              `[`,
		"tool not an object":    `[1]`,
		"needs_session missing": `[{"name": "a", "description": "d", "inputSchema": {}, "annotations": {}}]`,
		"needs_session string":  `[{"name": "a", "description": "d", "inputSchema": {}, "annotations": {}, "needs_session": "y"}]`,
		"schema not an object":  `[{"name": "a", "description": "d", "inputSchema": [], "annotations": {}, "needs_session": true}]`,
	} {
		if _, err := Parse([]byte(strings.Replace(base, "%s", tools, 1))); err == nil {
			t.Errorf("%s: Parse accepted it", name)
		}
	}
	if _, err := Parse([]byte(strings.Replace(base, "%s", `[]`, 1))); err != nil {
		t.Errorf("a well-formed document: %v", err)
	}
	if _, err := Parse([]byte(`{"server": {}}`)); err == nil {
		t.Errorf("a document without keys was accepted")
	}
}
