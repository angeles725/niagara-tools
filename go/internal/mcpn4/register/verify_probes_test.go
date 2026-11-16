package register

import (
	"os"
	"strings"
	"testing"
)

// Probes from the independent verification of PR #691 (H5): register never damages a harness config
// or keeps a launcher that executes a binary inside a git checkout.

const piConfig = ".pi/agent/mcp.json"

func TestRegisterWithoutServerFlagsRefusesAKeptLauncherExecutingAWorktreeBinary(t *testing.T) {
	h := newHome(t)
	h.write(t, ".config/mcp-n4/lab-run.sh",
		"#!/bin/bash\nexec /home/u/niagara-tools-worktrees/mcpn4-live/go/bin/mcp-n4 --station lab=https://x\n")
	if code := h.run("--station", "lab", "--harness", "pi"); code == 0 {
		t.Fatalf("exit 0 (stdout %q): a kept launcher that execs a worktree binary was registered", h.out.String())
	}
	if _, err := os.Stat(h.path(piConfig)); err == nil {
		t.Fatalf("%s was written although the launcher execs a worktree binary", piConfig)
	}
}

func TestRegisterKeepsASymlinkedHarnessConfigALink(t *testing.T) {
	h := newHome(t)
	h.write(t, "dotfiles/mcp.json", "{\"mcpServers\": {}}\n")
	if err := os.MkdirAll(h.path(".pi/agent"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(h.path("dotfiles/mcp.json"), h.path(piConfig)); err != nil {
		t.Fatal(err)
	}
	code := h.run(registerArgs("pi")...)
	st, err := os.Lstat(h.path(piConfig))
	if err != nil || st.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("exit %d: %s is no longer a symlink to the dotfiles copy", code, piConfig)
	}
	target := h.read(t, "dotfiles/mcp.json")
	if code == 0 && !strings.Contains(target, `"mcp-n4-lab"`) {
		t.Fatalf("exit 0 but the link target lacks the entry:\n%s", target)
	}
	if code != 0 && target != "{\"mcpServers\": {}}\n" {
		t.Fatalf("exit %d but the link target changed:\n%s", code, target)
	}
}

func TestRegisterRefusesTrailingDataAfterTheDocumentAndWritesNothing(t *testing.T) {
	for _, doc := range []string{"{\"mcpServers\": {}} garbage\n", "{\"mcpServers\": {}}\n{\"x\": 1}\n"} {
		h := newHome(t)
		h.write(t, piConfig, doc)
		if code := h.run(registerArgs("pi")...); code == 0 {
			t.Errorf("%q: exit 0, want a refusal of a malformed file", doc)
		}
		if got := h.read(t, piConfig); got != doc {
			t.Errorf("%q: file changed to %q", doc, got)
		}
	}
}

func TestRegisterTreatsAReorderedIdenticalEntryAsUnchanged(t *testing.T) {
	h := newHome(t)
	if code := h.run(registerArgs("pi")...); code != 0 {
		t.Fatalf("first run: exit %d, %s", code, h.err.String())
	}
	doc := "{\"mcpServers\": {\"mcp-n4-lab\": {\"exposure\": \"direct\", \"args\": [], \"command\": \"" +
		h.path(".config/mcp-n4/lab-run.sh") + "\"}}}\n"
	h.write(t, piConfig, doc)
	if code := h.run(registerArgs("pi")...); code != 0 || !strings.Contains(h.out.String(), "unchanged pi entry") {
		t.Fatalf("exit %d, stdout %q, stderr %q: an entry equal in value must be unchanged", code, h.out.String(), h.err.String())
	}
	if got := h.read(t, piConfig); got != doc {
		t.Fatalf("file rewritten:\n%s", got)
	}
}
