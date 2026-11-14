package blackbox

import (
	"os"
	"path/filepath"
	"testing"
)

// kitDir is mcp-n4-kit, seen from this package's directory (go test runs in it).
const kitDir = "../../../../mcp-n4-kit"

// mcpn4Dir is go/internal/mcpn4.
const mcpn4Dir = ".."

// kitDoc is the text of a file under mcp-n4-kit.
func kitDoc(t *testing.T, parts ...string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(append([]string{kitDir}, parts...)...))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
