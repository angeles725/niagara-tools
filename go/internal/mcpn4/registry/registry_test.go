package registry_test

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/registry"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

func TestDefaultHoldsEveryToolInTableOrder(t *testing.T) {
	t.Parallel()
	r := registry.Default()
	if r.Name != "mcp-n4" || r.Version == "" || r.ProtocolVersions[0] != "2025-06-18" {
		t.Fatalf("server = %q %q %v", r.Name, r.Version, r.ProtocolVersions)
	}
	// No tool count is a literal, so branches that add different tools never edit the same line:
	// one descriptor file per read tool (../testdata/tools, the tools/list goldens), one served
	// handler per write tool (writeops.Register).
	descriptors, err := filepath.Glob("../testdata/tools/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Read) != len(descriptors) {
		t.Fatalf("got %d read tools, want %d (one per descriptor file)", len(r.Read), len(descriptors))
	}
	if got, want := writeNames(r), servedWriteTools(); !slices.Equal(got, want) {
		t.Fatalf("registered write tools %v, served write handlers %v", got, want)
	}
	if r.Read[0].Name != "n4_connect" || r.Read[0].NeedsSession || !r.Read[2].NeedsSession {
		t.Errorf("read tools start %q (session %v), third needs session %v",
			r.Read[0].Name, r.Read[0].NeedsSession, r.Read[2].NeedsSession)
	}
	seen := map[string]bool{}
	for _, tool := range append(append([]registry.Tool{}, r.Read...), r.Write...) {
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
		if _, err := registry.Parse([]byte(strings.Replace(base, "%s", tools, 1))); err == nil {
			t.Errorf("%s: Parse accepted it", name)
		}
	}
	if _, err := registry.Parse([]byte(strings.Replace(base, "%s", `[]`, 1))); err != nil {
		t.Errorf("a well-formed document: %v", err)
	}
	if _, err := registry.Parse([]byte(`{"server": {}}`)); err == nil {
		t.Errorf("a document without keys was accepted")
	}
}

// writeNames are the registry's write tools, sorted.
func writeNames(r *registry.Registry) []string {
	var names []string
	for _, tool := range r.Write {
		names = append(names, tool.Name)
	}
	slices.Sort(names)
	return names
}

// servedWriteTools are the tools writeops.Register gives a handler, sorted. The handlers are only
// counted, never called, so the engine can be nil.
func servedWriteTools() []string {
	s := jsonrpc.New(registry.Default())
	writeops.Register(s, nil)
	var names []string
	for name := range s.Handlers {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
