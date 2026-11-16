package register

import (
	"os"
	"path/filepath"
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

func TestClaudeReportsAMalformedClaudeJSONNotAMissingProject(t *testing.T) {
	h := newHome(t)
	h.write(t, ".claude.json", "{\"projects\": {\"/work/p\": {}},}")
	_, _, err := claudeTarget(t, h, "/work/p").plan(false)
	if err == nil || strings.Contains(err.Error(), "no Claude Code project") || !strings.Contains(err.Error(), h.path(".claude.json")) {
		t.Fatalf("err = %v, want the parse error of the file", err)
	}
}

func TestClaudeJSONSymlinkStaysALinkAndItsTargetGetsTheEntry(t *testing.T) {
	h := newHome(t)
	h.write(t, "dotfiles/claude.json", claudeJSON)
	if err := os.Symlink(h.path("dotfiles/claude.json"), h.path(".claude.json")); err != nil {
		t.Fatal(err)
	}
	if code := h.run(registerArgs("claude", "--project", "/work/p")...); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, h.err.String())
	}
	if st, err := os.Lstat(h.path(".claude.json")); err != nil || st.Mode()&os.ModeSymlink == 0 {
		t.Fatal("~/.claude.json is no longer a symlink")
	}
	if !strings.Contains(h.read(t, "dotfiles/claude.json"), `"mcp-n4-lab"`) {
		t.Fatal("the link target lacks the entry")
	}
}

func TestARelativeProjectIsMadeAbsolute(t *testing.T) {
	h := newHome(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	h.write(t, ".claude.json", strings.Replace(claudeJSON, "/work/p", filepath.Join(cwd, "sub"), 1))
	if code := h.run(registerArgs("claude", "--project", "sub")...); code != 0 {
		t.Fatalf("exit %d, stderr %q, want the relative project resolved against the cwd", code, h.err.String())
	}
}
