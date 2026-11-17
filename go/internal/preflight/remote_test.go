package preflight

import (
	"os/exec"
	"strings"
	"testing"
)

// git runs git in dir for a test fixture.
func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(cmd.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.invalid", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.invalid")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// Retro delivery-guards-and-ui-contrast Δ3: a module checkout with no remote, or a branch with no upstream,
// keeps its commits on one disk only; preflight WARNs and leaves the backup decision to the operator.
func TestRemoteBackupWarnsWithoutRemoteOrUpstream(t *testing.T) {
	d := t.TempDir()
	mk(t, d, "nh/", "repo/g/build.gradle.kts=x")
	git(t, d+"/repo", "init", "-q", "-b", "main")
	git(t, d+"/repo", "add", ".")
	git(t, d+"/repo", "commit", "-q", "-m", "c")
	_, out, _ := pf((&fakeHost{}).host(), nil, "--jvm-dir", d, d+"/nh", d+"/repo/g")
	if want := "WARN  remote-backup  " + d + "/repo has no git remote: its commits exist on this disk only (push or back them up: the operator's call)"; !strings.Contains(out, want) {
		t.Errorf("rows\n%s\nwant %q", out, want)
	}
	git(t, d+"/repo", "remote", "add", "origin", d+"/elsewhere.git")
	_, out, _ = pf((&fakeHost{}).host(), nil, "--jvm-dir", d, d+"/nh", d+"/repo/g")
	if want := "WARN  remote-backup  " + d + "/repo branch main has no upstream: its commits are not pushed (git push -u origin main, or the operator's backup)"; !strings.Contains(out, want) {
		t.Errorf("rows\n%s\nwant %q", out, want)
	}
}

func TestRemoteBackupIsSilentOutsideGitAndWithAnUpstream(t *testing.T) {
	d := t.TempDir()
	mk(t, d, "nh/", "repo/g/build.gradle.kts=x")
	if _, out, _ := pf((&fakeHost{}).host(), nil, "--jvm-dir", d, d+"/nh", d+"/repo/g"); strings.Contains(out, "remote-backup") {
		t.Errorf("not a git work tree, rows\n%s", out)
	}
	git(t, d, "init", "-q", "--bare", "origin.git")
	git(t, d+"/repo", "init", "-q", "-b", "main")
	git(t, d+"/repo", "add", ".")
	git(t, d+"/repo", "commit", "-q", "-m", "c")
	git(t, d+"/repo", "remote", "add", "origin", d+"/origin.git")
	git(t, d+"/repo", "push", "-q", "-u", "origin", "main")
	if _, out, _ := pf((&fakeHost{}).host(), nil, "--jvm-dir", d, d+"/nh", d+"/repo/g"); strings.Contains(out, "remote-backup") {
		t.Errorf("pushed with an upstream, rows\n%s", out)
	}
}
