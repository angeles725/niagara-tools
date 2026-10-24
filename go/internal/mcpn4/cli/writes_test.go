package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestAllowWritesServesThePortedWriteTools: the write tools are listed after the read tools;
// set_slot and invoke_action answer, the tools of later slices say they are not ported yet.
func TestAllowWritesServesThePortedWriteTools(t *testing.T) {
	stdin := `{"id":1,"method":"tools/list"}` + "\n" +
		`{"id":2,"method":"tools/call","params":{"name":"n4_set_slot","arguments":{"ord":"o","slot":"s","value":1,"value_type":"baja:Double"}}}` + "\n" +
		`{"id":3,"method":"tools/call","params":{"name":"n4_save_station","arguments":{}}}` + "\n"
	code, out, errOut := run([]string{"--allow-writes", "--state-dir", filepath.Join(t.TempDir(), "s")}, stdin)
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if code != 0 || errOut != "" || len(lines) != 3 {
		t.Fatalf("got %d %q %q", code, out, errOut)
	}
	if !strings.Contains(lines[0], `"n4_set_slot"`) || !strings.Contains(lines[0], `"n4_save_station"`) ||
		!strings.Contains(lines[1], "not connected: call n4_connect first") ||
		!strings.Contains(lines[2], "n4_save_station is not implemented in this version of the Go server") {
		t.Errorf("replies:\n%s", out)
	}
}
