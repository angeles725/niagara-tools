package register

import (
	"os"
	"strings"
	"testing"

	kitcli "github.com/angeles725/niagara-tools/go/internal/cli"
)

// run is `mcp-n4 register <args>` with HOME pointing at the fake home.
func (h *fakeHome) run(args ...string) int {
	h.out.Reset()
	h.err.Reset()
	return Main(args, kitcli.Stdio{Out: &h.out, Err: &h.err, Getenv: func(n string) string {
		if n == "HOME" {
			return h.dir
		}
		return ""
	}})
}

// serverFlags are the flags a lab station launcher passes to the server.
var serverFlags = []string{"--", "--station", "LAB=https://127.0.0.1:18443", "--insecure-tls", "LAB",
	"--credential-env", "LAB_N4", "--load-wait", "10", "--http-timeout", "60"}

func registerArgs(harness string, extra ...string) []string {
	args := append([]string{"--station", "lab", "--harness", harness}, extra...)
	return append(args, serverFlags...)
}

func TestRegisterWritesTheLauncherExecutingTheInstalledBinary(t *testing.T) {
	h := newHome(t)
	if code := h.run(registerArgs("pi")...); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, h.err.String())
	}
	got := h.read(t, ".config/mcp-n4/lab-run.sh")
	if !strings.Contains(got, "set -a; . '"+h.path(".config/mcp-n4/lab.env")+"'; set +a\n") ||
		!strings.Contains(got, "exec '"+h.path(".local/bin/mcp-n4")+"' \\\n  '--station' 'LAB=https://127.0.0.1:18443'") {
		t.Fatalf("launcher =\n%s\nwant it to source lab.env and exec ~/.local/bin/mcp-n4", got)
	}
	if strings.Contains(got, secretUser) || strings.Contains(got, secretPass) {
		t.Fatal("the launcher holds an env file value")
	}
	if st, err := os.Stat(h.path(".config/mcp-n4/lab-run.sh")); err != nil || st.Mode().Perm() != 0o755 {
		t.Fatalf("launcher mode %v %v, want 0755", st, err)
	}
	if !strings.Contains(h.read(t, ".pi/agent/mcp.json"), `"command": "`+h.path(".config/mcp-n4/lab-run.sh")+`"`) {
		t.Fatal("the pi entry does not run the launcher")
	}
}

func TestASecondRunChangesNoByteForEveryHarness(t *testing.T) {
	for _, hc := range []struct{ harness, file string }{
		{"claude", ".claude.json"}, {"pi", ".pi/agent/mcp.json"}, {"gentle-shell", ".gentle-shell/agent/mcp.json"},
	} {
		h := newHome(t)
		h.write(t, ".claude.json", claudeJSON)
		args := registerArgs(hc.harness, "--project", "/work/p")
		if code := h.run(args...); code != 0 {
			t.Fatalf("%s: first exit %d, stderr %q", hc.harness, code, h.err.String())
		}
		entry, launcher := h.read(t, hc.file), h.read(t, ".config/mcp-n4/lab-run.sh")
		if code := h.run(args...); code != 0 || strings.Count(h.out.String(), "unchanged") != 2 {
			t.Fatalf("%s: second exit %d, stdout %q, want both unchanged", hc.harness, code, h.out.String())
		}
		if h.read(t, hc.file) != entry || h.read(t, ".config/mcp-n4/lab-run.sh") != launcher {
			t.Fatalf("%s: a second run changed bytes", hc.harness)
		}
	}
}

func TestADifferingEntryIsRefusedNamingFileAndEntryAndWritesNothing(t *testing.T) {
	h := newHome(t)
	before := "{\n  \"mcpServers\": {\n    \"mcp-n4-lab\": {\n      \"command\": \"/old/run.sh\",\n      \"args\": []\n    }\n  }\n}\n"
	h.write(t, ".pi/agent/mcp.json", before)
	if code := h.run(registerArgs("pi")...); code != 1 || !strings.Contains(h.err.String(), h.path(".pi/agent/mcp.json")) ||
		!strings.Contains(h.err.String(), "mcp-n4-lab") || !strings.Contains(h.err.String(), "--force") {
		t.Fatalf("exit %d, stderr %q, want a refusal naming file, entry and --force", code, h.err.String())
	}
	if h.read(t, ".pi/agent/mcp.json") != before {
		t.Fatal("a refused registration changed mcp.json")
	}
	if _, err := os.Stat(h.path(".config/mcp-n4/lab-run.sh")); err == nil {
		t.Fatal("a refused entry still wrote the launcher")
	}
	if code := h.run(registerArgs("pi", "--force")...); code != 0 {
		t.Fatalf("--force exit %d, stderr %q", code, h.err.String())
	}
}

func TestADifferingLauncherIsRefusedNamingItsPathUnlessForced(t *testing.T) {
	h := newHome(t)
	h.write(t, ".config/mcp-n4/lab-run.sh", "#!/bin/bash\nexec /old/worktree/mcp-n4\n")
	if code := h.run(registerArgs("pi")...); code != 1 || !strings.Contains(h.err.String(), h.path(".config/mcp-n4/lab-run.sh")) ||
		!strings.Contains(h.err.String(), "line 2") {
		t.Fatalf("exit %d, stderr %q, want a refusal naming the launcher and line 2", code, h.err.String())
	}
	if _, err := os.Stat(h.path(".pi/agent/mcp.json")); err == nil {
		t.Fatal("a refused launcher still wrote the harness entry")
	}
	if code := h.run(registerArgs("pi", "--force")...); code != 0 {
		t.Fatalf("--force exit %d, stderr %q", code, h.err.String())
	}
}

func TestABinInsideAGitCheckoutIsRefusedUnlessForced(t *testing.T) {
	h := newHome(t)
	h.write(t, "wt/.git", "gitdir: /elsewhere\n")
	bin := h.path("wt/go/bin/mcp-n4")
	if code := h.run(registerArgs("pi", "--bin", bin)...); code != 1 || !strings.Contains(h.err.String(), h.path("wt/.git")) {
		t.Fatalf("exit %d, stderr %q, want a refusal naming the checkout", code, h.err.String())
	}
	if _, err := os.Stat(h.path(".config/mcp-n4/lab-run.sh")); err == nil {
		t.Fatal("a refused --bin wrote the launcher")
	}
	if code := h.run(registerArgs("pi", "--bin", bin, "--force")...); code != 0 ||
		!strings.Contains(h.read(t, ".config/mcp-n4/lab-run.sh"), "exec '"+bin+"'") {
		t.Fatalf("--force exit %d, stderr %q", code, h.err.String())
	}
}

func TestWithoutServerFlagsALauncherExecutingTheInstalledBinaryIsKept(t *testing.T) {
	h := newHome(t)
	if code := h.run("--station", "lab", "--harness", "pi"); code != 2 || !strings.Contains(h.err.String(), "server flags") {
		t.Fatalf("no launcher, no flags: exit %d, stderr %q", code, h.err.String())
	}
	kept := "#!/bin/bash\nexec " + h.path(".local/bin/mcp-n4") + " --station LAB=https://h\n"
	h.write(t, ".config/mcp-n4/lab-run.sh", kept)
	if code := h.run("--station", "lab", "--harness", "pi"); code != 0 || h.read(t, ".config/mcp-n4/lab-run.sh") != kept {
		t.Fatalf("exit %d, stderr %q, want the launcher kept and the entry written", code, h.err.String())
	}
}

func TestBadInputIsRefusedBeforeAnyWrite(t *testing.T) {
	for _, args := range [][]string{
		{"--station", "lab", "--harness", "vim", "--", "--station", "LAB=https://h"},
		{"--station", "Lab/x", "--harness", "pi", "--", "--station", "LAB=https://h"},
		{"--station", "lab", "--harness", "pi", "--", "--bogus"},
		{"--station", "lab", "--harness", "pi", "--", "--station", "LAB=http://h"},
		{"--station", "lab", "--harness", "pi", "--env", "/nonexistent/lab.env", "--", "--station", "LAB=https://h"},
		{"--station", "lab", "--harness", "pi", "--", "--station", "LAB=https://h", "--write-scope", "station:|slot:/" + secretPass},
	} {
		h := newHome(t)
		if code := h.run(args...); code != 2 || strings.Contains(h.err.String(), secretPass) {
			t.Errorf("%v: exit %d, stderr %q, want 2 and no secret value", args, code, h.err.String())
		}
		if _, err := os.Stat(h.path(".config/mcp-n4/lab-run.sh")); err == nil {
			t.Errorf("%v: refused input wrote the launcher", args)
		}
	}
}

func TestAKeptLauncherNotExecutingTheInstalledBinaryIsRefusedNamingBothUnlessForced(t *testing.T) {
	h := newHome(t)
	h.write(t, "wt/.git", "gitdir: /x\n")
	h.write(t, ".config/mcp-n4/lab-run.sh", "#!/bin/bash\nexec '"+h.path("wt/go/bin/mcp-n4")+"' --station LAB=https://h\n")
	if code := h.run("--station", "lab", "--harness", "pi"); code != 1 ||
		!strings.Contains(h.err.String(), h.path(".config/mcp-n4/lab-run.sh")) || !strings.Contains(h.err.String(), h.path("wt/go/bin/mcp-n4")) {
		t.Fatalf("exit %d, stderr %q, want a refusal naming the launcher and the binary", code, h.err.String())
	}
	if !strings.Contains(h.err.String(), h.path("wt/.git")) {
		t.Fatalf("stderr %q, want the checkout named", h.err.String())
	}
	h.write(t, ".config/mcp-n4/lab-run.sh", "#!/bin/bash\nset -a\n")
	if code := h.run("--station", "lab", "--harness", "pi"); code != 1 || !strings.Contains(h.err.String(), "no exec line") {
		t.Fatalf("no exec line: exit %d, stderr %q", code, h.err.String())
	}
	if code := h.run("--station", "lab", "--harness", "pi", "--force"); code != 0 {
		t.Fatalf("--force exit %d, stderr %q", code, h.err.String())
	}
	if code := h.run(registerArgs("pi", "--force")...); code != 0 || !strings.Contains(h.read(t, ".config/mcp-n4/lab-run.sh"), h.path(".local/bin/mcp-n4")) {
		t.Fatalf("rewrite exit %d, stderr %q, want the launcher migrated to the installed binary", code, h.err.String())
	}
}
