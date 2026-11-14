package blackbox

// Guard: every write tool the server lists is refused by the version-tier gate (port of
// test_tiers.TestTierB.test_every_mutating_tool_is_gated). The tool list and each tool's
// arguments come from the server's own tools/list, so a new write tool is covered with no edit
// here.
//
// Out of reach by design: a tool listed with readOnlyHint true (and named in readTools) whose
// schema is write-shaped is taken at its word and not called here. The guard trusts the
// annotation; whether a read tool really writes is the business of that tool's own tests.

import (
	"slices"
	"strings"
	"testing"
)

func TestEveryWriteToolIsGatedByTheTier(t *testing.T) {
	r := tierRig(t, "4.15.3.28", nil)
	tools := listTools(r.Srv)
	writes := WriteTools(tools)
	var unmarked []string
	for _, tool := range tools {
		name := tool["name"].(string)
		if !slices.Contains(readTools, name) && !slices.Contains(writes, name) {
			unmarked = append(unmarked, name) // neither a known read tool nor marked as writing
		}
	}
	t.Logf("%d write tools: %v", len(writes), writes)
	if len(writes) == 0 || len(unmarked) != 0 {
		t.Fatalf("write tools %v; tools neither read nor marked readOnlyHint false: %v", writes, unmarked)
	}
	for _, tool := range tools {
		name := tool["name"].(string)
		if !slices.Contains(writes, name) {
			continue
		}
		args := SchemaArgs(tool["inputSchema"].(map[string]any))
		if res := r.Srv.Call(name, Execute(args, "1.a.b")); !res.IsError || !strings.Contains(res.Text, reasonTier) {
			t.Errorf("%s is not refused by the tier gate: %.300s", name, res.Text)
		}
	}
}
