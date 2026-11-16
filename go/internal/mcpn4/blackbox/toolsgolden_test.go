package blackbox

// The tools/list reply in testdata/protocol.json holds the placeholder toolsMarker instead of the
// descriptor array: each tool's descriptor bytes live in their own file,
// ../testdata/tools/<name>.json (shared with the jsonrpc differential), so branches that add
// different tools add different files instead of re-pinning the same golden line. The order is
// readTools, which TestToolsListEntriesHaveSchemaAndReadOnlyAnnotations pins against what the binary lists.

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const (
	toolsMarker = "@TOOLS@"
	toolsDir    = "../testdata/tools"
)

// assembleTools replaces every toolsMarker in s with the descriptors of names, in that order,
// joined as Python's json.dumps joins list items. A descriptor file for a tool outside names, a
// tool without a file, or an s without the marker fails t.
func assembleTools(t testing.TB, s string, names []string) string {
	t.Helper()
	if !strings.Contains(s, toolsMarker) {
		t.Fatalf("golden text holds no %s placeholder", toolsMarker)
	}
	files, err := filepath.Glob(filepath.Join(toolsDir, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if name := strings.TrimSuffix(filepath.Base(f), ".json"); !slices.Contains(names, name) {
			t.Errorf("descriptor file %s names tool %s, which readTools does not list", f, name)
		}
	}
	descs := make([]string, 0, len(names))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(toolsDir, name+".json"))
		if err != nil {
			t.Errorf("tool %s has no descriptor file %s/%s.json: %v", name, toolsDir, name, err)
			continue
		}
		descs = append(descs, strings.TrimSuffix(string(data), "\n"))
	}
	if t.Failed() {
		t.FailNow()
	}
	return strings.ReplaceAll(s, toolsMarker, strings.Join(descs, ", "))
}
