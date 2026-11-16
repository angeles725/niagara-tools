package register

import (
	"strings"
	"testing"
)

const claudeJSON = `{
  "numStartups": 3,
  "projects": {
    "/work/p": {
      "allowedTools": [],
      "mcpServers": {
        "other": {"command": "other-server", "args": ["a", "b"]}
      },
      "hasTrustDialogAccepted": true
    }
  },
  "userID": "x<y>&z"
}`

func claudeTarget(t *testing.T, h *fakeHome, project string) target {
	t.Helper()
	tgt, err := harnessTarget("claude", h.dir, project, "mcp-n4-lab", "/l/lab-run.sh")
	if err != nil {
		t.Fatal(err)
	}
	return tgt
}

func TestClaudeEntryGoesToTheProjectScopeOfClaudeJSON(t *testing.T) {
	h := newHome(t)
	h.write(t, ".claude.json", claudeJSON)
	tgt := claudeTarget(t, h, "/work/p")
	if tgt.file != h.path(".claude.json") {
		t.Fatalf("file = %s, want ~/.claude.json", tgt.file)
	}
	_, next, err := tgt.plan(false)
	want := strings.Replace(claudeJSON, `["a", "b"]}
`, `["a", "b"]},
        "mcp-n4-lab": {
          "args": [],
          "command": "/l/lab-run.sh",
          "env": {},
          "type": "stdio"
        }
`, 1)
	if err != nil || string(next) != want {
		t.Fatalf(".claude.json =\n%s\n%v\nwant\n%s", next, err, want)
	}
}

func TestClaudeRefusesAProjectClaudeCodeHasNotOpened(t *testing.T) {
	h := newHome(t)
	h.write(t, ".claude.json", claudeJSON)
	if _, _, err := claudeTarget(t, h, "/work/unknown").plan(false); err == nil ||
		!strings.Contains(err.Error(), "/work/unknown") || !strings.Contains(err.Error(), h.path(".claude.json")) {
		t.Fatalf("err = %v, want a refusal naming the project and the file", err)
	}
	if _, _, err := claudeTarget(t, newHome(t), "/work/p").plan(false); err == nil {
		t.Fatal("a missing ~/.claude.json was not refused")
	}
}
