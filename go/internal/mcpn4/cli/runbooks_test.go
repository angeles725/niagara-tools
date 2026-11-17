package cli

import (
	"strings"
	"testing"
)

// The server main() builds serves the shipped runbooks as prompts and resources, read-only mode
// included (they are text; no station call).
func TestServerServesTheShippedRunbooks(t *testing.T) {
	srv, err := NewServer(Config{StateDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	out, _ := srv.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	if !strings.Contains(string(out), `"prompts": {"listChanged": false}, "resources": {"listChanged": false}`) {
		t.Fatalf("initialize: %s", out)
	}
	out, _ = srv.HandleLine([]byte(`{"jsonrpc":"2.0","id":2,"method":"prompts/get","params":{"name":"commissioning",` +
		`"arguments":{"station":"Prueba_4_15","scope":"station:|slot:/Drivers"}}}`))
	if !strings.Contains(string(out), "# Commissioning close-out: station:|slot:/Drivers on Prueba_4_15") {
		t.Fatalf("prompts/get: %s", out)
	}
	for _, name := range []string{"alarm-triage", "commissioning", "post-deploy-check"} {
		out, _ = srv.HandleLine([]byte(`{"jsonrpc":"2.0","id":3,"method":"resources/read","params":{"uri":"runbook://` + name + `"}}`))
		if !strings.Contains(string(out), `"mimeType": "text/markdown"`) {
			t.Fatalf("resources/read %s: %s", name, out)
		}
	}
}
