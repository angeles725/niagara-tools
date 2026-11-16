package register

import (
	"strings"
	"testing"
)

// keptLauncher writes the launcher register would write for lab, so a later run without server
// flags has one to keep.
func keptLauncher(t *testing.T, h *fakeHome) string {
	t.Helper()
	if code := h.run(registerArgs("pi")...); code != 0 {
		t.Fatalf("setup exit %d, stderr %q", code, h.err.String())
	}
	return h.read(t, ".config/mcp-n4/lab-run.sh")
}

func TestWithoutServerFlagsAnExplicitBinOrEnvEqualToTheLauncherIsUnchanged(t *testing.T) {
	h := newHome(t)
	before := keptLauncher(t, h)
	code := h.run("--station", "lab", "--harness", "pi", "--bin", h.path(".local/bin/mcp-n4"),
		"--env", h.path(".config/mcp-n4/lab.env"))
	if code != 0 || !strings.Contains(h.out.String(), "unchanged launcher") || h.read(t, ".config/mcp-n4/lab-run.sh") != before {
		t.Fatalf("exit %d, stdout %q, stderr %q, want the launcher unchanged", code, h.out.String(), h.err.String())
	}
}

func TestWithoutServerFlagsADifferingExplicitBinOrEnvIsRefusedNotIgnored(t *testing.T) {
	for _, c := range []struct{ flag, rel string }{{"--bin", "opt/mcp-n4"}, {"--env", "other.env"}} {
		for _, force := range []bool{false, true} {
			h := newHome(t)
			before := keptLauncher(t, h)
			h.write(t, "other.env", "LAB_N4_USER=u\n")
			args := []string{"--station", "lab", "--harness", "pi", c.flag, h.path(c.rel)}
			if force {
				args = append(args, "--force")
			}
			code := h.run(args...)
			msg := h.err.String()
			if code != 1 || !strings.Contains(msg, h.path(".config/mcp-n4/lab-run.sh")) || !strings.Contains(msg, h.path(c.rel)) ||
				!strings.Contains(msg, "server flags after --") {
				t.Errorf("%s force=%v: exit %d, stderr %q, want a refusal naming the launcher, the requested value and how to rewrite",
					c.flag, force, code, msg)
			}
			if h.read(t, ".config/mcp-n4/lab-run.sh") != before {
				t.Errorf("%s force=%v: the launcher changed", c.flag, force)
			}
		}
	}
}

func TestWithoutServerFlagsAnExplicitBinInsideACheckoutIsRefusedUnlessForced(t *testing.T) {
	h := newHome(t)
	h.write(t, "wt/.git", "gitdir: /x\n")
	bin := h.path("wt/go/bin/mcp-n4")
	h.write(t, ".config/mcp-n4/lab-run.sh", "#!/bin/bash\nexec '"+bin+"' \\\n  '--station' 'LAB=https://h'\n")
	if code := h.run("--station", "lab", "--harness", "pi", "--bin", bin); code != 1 || !strings.Contains(h.err.String(), h.path("wt/.git")) {
		t.Fatalf("exit %d, stderr %q, want the checkout refused", code, h.err.String())
	}
	if code := h.run("--station", "lab", "--harness", "pi", "--bin", bin, "--force"); code != 0 {
		t.Fatalf("--force exit %d, stderr %q", code, h.err.String())
	}
}

func TestLauncherWordsReadTheExecTargetAndTheSourcedEnvFile(t *testing.T) {
	text := "#!/bin/bash\nset -a; . '/h/a b.env'; set +a\nexec '/a b/mcp-n4' \\\n  '--station' 'x'\n"
	if got, ok := launcherWord(text, "exec "); !ok || got != "/a b/mcp-n4" {
		t.Errorf("exec target = %q %v", got, ok)
	}
	if got, ok := launcherWord(text, "set -a; . "); !ok || got != "/h/a b.env" {
		t.Errorf("env file = %q %v", got, ok)
	}
	for text, want := range map[string]string{
		"  exec /w/go/bin/mcp-n4 --station lab=https://x\n": "/w/go/bin/mcp-n4",
		"exec \"/q/mcp-n4\"\n":                              "/q/mcp-n4",
		"exec /e\\ x/mcp-n4\n":                              "/e x/mcp-n4",
	} {
		if got, ok := launcherWord(text, "exec "); !ok || got != want {
			t.Errorf("launcherWord(%q) = %q %v, want %q", text, got, ok, want)
		}
	}
	if _, ok := launcherWord("#!/bin/bash\n/usr/bin/mcp-n4\n", "exec "); ok {
		t.Error("a launcher without exec was read as having one")
	}
}
