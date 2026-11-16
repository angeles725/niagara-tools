package register

import (
	"os"
	"strings"
	"testing"
)

func TestAKeptLauncherExecutingTheDefaultBinSymlinkedIntoACheckoutIsRefusedLikeTheServerFlagsCase(t *testing.T) {
	h := newHome(t)
	before := keptLauncher(t, h)
	h.write(t, "wt/.git", "gitdir: /x\n")
	h.write(t, "wt/go/bin/mcp-n4", "#!/bin/sh\n")
	if err := os.MkdirAll(h.path(".local/bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(h.path("wt/go/bin/mcp-n4"), h.path(".local/bin/mcp-n4")); err != nil {
		t.Fatal(err)
	}
	if code := h.run(registerArgs("pi")...); code != 1 {
		t.Fatalf("server flags: exit %d, stderr %q, want the checkout refused", code, h.err.String())
	}
	withFlags := h.err.String()
	code := h.run("--station", "lab", "--harness", "pi")
	if code != 1 || h.err.String() != withFlags || !strings.Contains(withFlags, h.path("wt/.git")) {
		t.Fatalf("kept: exit %d, stderr %q, want exit 1 and the server-flags refusal %q", code, h.err.String(), withFlags)
	}
	if h.read(t, ".config/mcp-n4/lab-run.sh") != before {
		t.Fatal("the launcher changed")
	}
	if code := h.run("--station", "lab", "--harness", "pi", "--force"); code != 0 {
		t.Fatalf("--force exit %d, stderr %q, want the launcher kept", code, h.err.String())
	}
}

func TestADuplicateEntryNameIsRefusedWithNothingWritten(t *testing.T) {
	h := newHome(t)
	h.write(t, ".config/mcp-n4/lab-run.sh", "#!/bin/bash\nexec '/old/mcp-n4'\n")
	h.write(t, ".pi/agent/mcp.json", "{\n  \"mcpServers\": {\n    \"mcp-n4-lab\": {\"command\": \"/a\"},\n    \"mcp-n4-lab\": {\"command\": \"/b\"}\n  }\n}\n")
	before := h.snapshot(t)
	code := h.run(registerArgs("pi", "--force")...)
	msg := strings.TrimSpace(h.err.String())
	if code != 1 || !strings.Contains(msg, `duplicate key "mcp-n4-lab"`) || !strings.HasSuffix(msg, "; nothing written") {
		t.Fatalf("exit %d, stderr %q, want the duplicate name refused ending with \"; nothing written\"", code, msg)
	}
	if after := h.snapshot(t); !sameFiles(before, after) {
		t.Fatalf("files changed:\nbefore %v\nafter  %v", before, after)
	}
}

func TestAFailedEntryWriteRestoresTheLauncher(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	for _, prior := range []string{"", "#!/bin/bash\nexec '/old/mcp-n4'\n"} {
		h := newHome(t)
		if prior != "" {
			h.write(t, ".config/mcp-n4/lab-run.sh", prior)
		}
		if err := os.MkdirAll(h.path(".pi"), 0o555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chmod(h.path(".pi"), 0o755) })
		before := h.snapshot(t)
		if code := h.run(registerArgs("pi", "--force")...); code != 1 {
			t.Fatalf("prior %q: exit %d, stderr %q, want the entry write to fail", prior, code, h.err.String())
		}
		if after := h.snapshot(t); !sameFiles(before, after) {
			t.Fatalf("prior %q: files changed:\nbefore %v\nafter  %v", prior, before, after)
		}
	}
}
