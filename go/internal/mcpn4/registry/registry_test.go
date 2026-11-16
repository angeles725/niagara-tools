package registry

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultHoldsEveryToolInTableOrder(t *testing.T) {
	t.Parallel()
	r := Default()
	if r.Name != "mcp-n4" || r.Version == "" || r.ProtocolVersions[0] != "2025-06-18" {
		t.Fatalf("server = %q %q %v", r.Name, r.Version, r.ProtocolVersions)
	}
	// One descriptor file per read tool (../testdata/tools, the tools/list goldens), so a branch
	// that adds a read tool adds a file instead of editing a shared count.
	descriptors, err := filepath.Glob("../testdata/tools/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Read) != len(descriptors) || len(r.Write) != 12 {
		t.Fatalf("got %d read and %d write tools, want %d and 12", len(r.Read), len(r.Write), len(descriptors))
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
	if !strings.HasPrefix(r.Usage, "usage: mcp-n4 [-h]") || !strings.HasPrefix(r.Help, r.Usage) {
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
