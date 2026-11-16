package register

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// T17 F1: chain and relative symlinks into a checkout are refused when kept; a link outside any checkout is kept.
func TestT17KeptLauncherResolvesSymlinkVariants(t *testing.T) {
	for _, tc := range []struct {
		name    string
		link    func(h *fakeHome) string
		refused bool
	}{
		{"relative", func(h *fakeHome) string { return "../../wt/go/bin/mcp-n4" }, true},
		{"chain", func(h *fakeHome) string {
			os.Symlink(h.path("wt/go/bin/mcp-n4"), h.path("out/link2"))
			return h.path("out/link2")
		}, true},
		{"outside", func(h *fakeHome) string { return "../../out/mcp-n4" }, false},
	} {
		h := newHome(t)
		before := keptLauncher(t, h)
		h.write(t, "wt/.git", "gitdir: /x\n")
		h.write(t, "wt/go/bin/mcp-n4", "#!/bin/sh\n")
		h.write(t, "out/mcp-n4", "#!/bin/sh\n")
		os.MkdirAll(h.path(".local/bin"), 0o755)
		if err := os.Symlink(tc.link(h), h.path(".local/bin/mcp-n4")); err != nil {
			t.Fatal(err)
		}
		code := h.run("--station", "lab", "--harness", "pi")
		if tc.refused != (code == 1 && strings.Contains(h.err.String(), "inside a git checkout")) {
			t.Fatalf("%s: exit %d, stderr %q", tc.name, code, h.err.String())
		}
		if h.read(t, ".config/mcp-n4/lab-run.sh") != before {
			t.Fatalf("%s: launcher changed", tc.name)
		}
	}
}

// T17 F2: a failed entry write restores the existing launcher's mode, not only its bytes.
func TestT17FailedEntryWriteRestoresLauncherMode(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	h := newHome(t)
	h.write(t, ".config/mcp-n4/lab-run.sh", "#!/bin/bash\nexec '/old/mcp-n4'\n")
	os.Chmod(h.path(".config/mcp-n4/lab-run.sh"), 0o700)
	os.MkdirAll(h.path(".pi"), 0o555)
	t.Cleanup(func() { os.Chmod(h.path(".pi"), 0o755) })
	if code := h.run(registerArgs("pi", "--force")...); code != 1 || !strings.HasSuffix(strings.TrimSpace(h.err.String()), "; nothing written") {
		t.Fatalf("exit %d, stderr %q", code, h.err.String())
	}
	st, _ := os.Stat(h.path(".config/mcp-n4/lab-run.sh"))
	if st.Mode().Perm() != 0o700 {
		t.Fatalf("mode %v, want 0700", st.Mode().Perm())
	}
	if m, _ := filepath.Glob(h.path(".config/mcp-n4/.*.tmp")); len(m) > 0 {
		t.Fatalf("temp files left: %v", m)
	}
}

// T17 F2: when the rollback itself fails, undo reports an error (so register never claims "nothing written").
func TestT17UndoFailureIsReported(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	for _, prior := range []string{"", "old\n"} {
		d := t.TempDir()
		p := filepath.Join(d, "lab-run.sh")
		if prior != "" {
			os.WriteFile(p, []byte(prior), 0o700)
		}
		if _, undo, err := writeLauncher("launcher "+p, p, []byte("new\n")); err != nil {
			t.Fatal(err)
		} else {
			os.Chmod(d, 0o555)
			uerr := undo()
			os.Chmod(d, 0o755)
			if uerr == nil {
				t.Fatalf("prior %q: undo in a read-only dir returned nil", prior)
			}
		}
	}
}

// T17 follow-up: a dangling ~/.local/bin/mcp-n4 is judged by its link text: a target inside a
// checkout (still present, or gone with a checkout among its ancestors) is refused when kept; a
// dangling link outside any checkout is kept.
func TestT17KeptLauncherJudgesADanglingSymlinkByItsTarget(t *testing.T) {
	for _, tc := range []struct {
		name, dotgit, link string
		refused            bool
	}{
		{"binary gone", "wt/.git", "../../wt/go/bin/gone", true},
		{"worktree gone", "repo/.git", "../../repo/wt/go/bin/mcp-n4", true},
		{"outside", "", "../../out/gone", false},
	} {
		h := newHome(t)
		before := keptLauncher(t, h)
		if tc.dotgit != "" {
			h.write(t, tc.dotgit, "gitdir: /x\n")
		}
		if err := os.MkdirAll(h.path(".local/bin"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(tc.link, h.path(".local/bin/mcp-n4")); err != nil {
			t.Fatal(err)
		}
		code := h.run("--station", "lab", "--harness", "pi")
		if tc.refused != (code == 1 && strings.Contains(h.err.String(), "inside a git checkout")) {
			t.Errorf("%s: exit %d, stderr %q", tc.name, code, h.err.String())
		}
		if h.read(t, ".config/mcp-n4/lab-run.sh") != before {
			t.Errorf("%s: launcher changed", tc.name)
		}
	}
}
