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

func registerArgs(extra ...string) []string {
	args := append([]string{"--station", "lab"}, extra...)
	return append(args, serverFlags...)
}

func TestRegisterWritesTheLauncherExecutingTheInstalledBinary(t *testing.T) {
	h := newHome(t)
	if code := h.run(registerArgs()...); code != 0 {
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
}

func TestADifferingLauncherIsRefusedNamingItsPathUnlessForced(t *testing.T) {
	h := newHome(t)
	h.write(t, ".config/mcp-n4/lab-run.sh", "#!/bin/bash\nexec /old/worktree/mcp-n4\n")
	if code := h.run(registerArgs()...); code != 1 || !strings.Contains(h.err.String(), h.path(".config/mcp-n4/lab-run.sh")) ||
		!strings.Contains(h.err.String(), "line 2") {
		t.Fatalf("exit %d, stderr %q, want a refusal naming the launcher and line 2", code, h.err.String())
	}
	if h.read(t, ".config/mcp-n4/lab-run.sh") != "#!/bin/bash\nexec /old/worktree/mcp-n4\n" {
		t.Fatal("a refused launcher was changed")
	}
	if code := h.run(registerArgs("--force")...); code != 0 {
		t.Fatalf("--force exit %d, stderr %q", code, h.err.String())
	}
}

func TestABinInsideAGitCheckoutIsRefusedUnlessForced(t *testing.T) {
	h := newHome(t)
	h.write(t, "wt/.git", "gitdir: /elsewhere\n")
	bin := h.path("wt/go/bin/mcp-n4")
	if code := h.run(registerArgs("--bin", bin)...); code != 1 || !strings.Contains(h.err.String(), h.path("wt/.git")) {
		t.Fatalf("exit %d, stderr %q, want a refusal naming the checkout", code, h.err.String())
	}
	if _, err := os.Stat(h.path(".config/mcp-n4/lab-run.sh")); err == nil {
		t.Fatal("a refused --bin wrote the launcher")
	}
	if code := h.run(registerArgs("--bin", bin, "--force")...); code != 0 ||
		!strings.Contains(h.read(t, ".config/mcp-n4/lab-run.sh"), "exec '"+bin+"'") {
		t.Fatalf("--force exit %d, stderr %q", code, h.err.String())
	}
}

func TestBadInputIsRefusedBeforeAnyWrite(t *testing.T) {
	for _, args := range [][]string{
		{"--station", "lab", "--harness", "pi", "--", "--station", "LAB=https://h"}, // no harness entry in this commit
		{"--station", "Lab/x", "--", "--station", "LAB=https://h"},
		{"--station", "lab", "--", "--bogus"},
		{"--station", "lab", "--", "--station", "LAB=http://h"},
		{"--station", "lab", "--env", "/nonexistent/lab.env", "--", "--station", "LAB=https://h"},
		{"--station", "lab", "--", "--station", "LAB=https://h", "--write-scope", "station:|slot:/" + secretPass},
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

func TestWithoutServerFlagsNoLauncherIsWritten(t *testing.T) {
	h := newHome(t)
	if code := h.run("--station", "lab"); code != 2 || !strings.Contains(h.err.String(), "server flags") {
		t.Fatalf("exit %d, stderr %q, want a usage error naming the server flags", code, h.err.String())
	}
}
