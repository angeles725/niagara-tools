package hookinstall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	activated = "install-hooks: activated — core.hooksPath = .githooks; live: the retro-enforcement pre-push gate, the\n" +
		"              client-identifier guard at commit time (pre-commit, commit-msg) and the\n" +
		"              kit-state regenerate hooks (post-merge, post-checkout, post-rewrite, post-commit).\n"
	undoLine = "              Undo with: scripts/install-hooks.sh --uninstall\n"
)

// TestHooksActivate: the local core.hooksPath becomes .githooks, and a second run changes nothing.
func TestHooksActivate(t *testing.T) {
	f := newFixture(t)
	for _, run := range []string{"first", "second"} {
		code, out, errText := f.hooks(f.repo)
		expect(t, run, code, out, errText, 0, activated+undoLine, "")
		if got := f.local("core.hooksPath"); got != ".githooks" {
			t.Errorf("%s run: core.hooksPath = %q", run, got)
		}
	}
}

// TestHooksUninstall: only the value this installer set is unset; any other is left and named.
func TestHooksUninstall(t *testing.T) {
	f := newFixture(t)
	code, out, errText := f.hooks(f.repo, "--uninstall")
	expect(t, "unset", code, out, errText, 0, "install-hooks: core.hooksPath is not .githooks (it is \"\"); left untouched.\n", "")
	git(t, f.repo, "config", "--local", "core.hooksPath", ".custom")
	code, out, errText = f.hooks(f.repo, "--uninstall")
	expect(t, "custom", code, out, errText, 0, "install-hooks: core.hooksPath is not .githooks (it is \".custom\"); left untouched.\n", "")
	if got := f.local("core.hooksPath"); got != ".custom" {
		t.Errorf("a custom core.hooksPath was changed to %q", got)
	}
	git(t, f.repo, "config", "--local", "core.hooksPath", ".githooks")
	code, out, errText = f.hooks(f.repo, "--uninstall")
	expect(t, "ours", code, out, errText, 0, "install-hooks: deactivated — core.hooksPath unset (default hooks restored).\n", "")
	if got := f.local("core.hooksPath"); got != "" {
		t.Errorf("core.hooksPath still %q after --uninstall", got)
	}
}

// TestHooksRefuseACustomHooksPath: a user's own hooks directory survives unless --force is given.
func TestHooksRefuseACustomHooksPath(t *testing.T) {
	f := newFixture(t)
	git(t, f.repo, "config", "--local", "core.hooksPath", ".custom")
	code, out, errText := f.hooks(f.repo)
	expect(t, "refuse", code, out, errText, 3, "",
		"install-hooks: REFUSING — core.hooksPath is already set to a custom value \".custom\".\n"+
			"              Your hooks are left untouched. Re-run with --force to override.\n")
	if got := f.local("core.hooksPath"); got != ".custom" {
		t.Errorf("a custom core.hooksPath was changed to %q", got)
	}
	code, out, errText = f.hooks(f.repo, "--force")
	expect(t, "--force", code, out, errText, 0, activated+undoLine, "")
	if got := f.local("core.hooksPath"); got != ".githooks" {
		t.Errorf("--force left core.hooksPath = %q", got)
	}
}

// TestHooksRunPostMergeOnceAtTheTopLevel: an executable .githooks/post-merge runs from the work tree
// root (also when the installer starts in a sub-directory), its output between the two messages; one
// without the execute bit is not run.
func TestHooksRunPostMergeOnceAtTheTopLevel(t *testing.T) {
	f := newFixture(t)
	hook := write(t, filepath.Join(f.repo, ".githooks", "post-merge"), "#!/usr/bin/env bash\necho \"ran in $PWD\"\necho x >> ran\n", 0o644)
	code, out, errText := f.hooks(f.repo)
	expect(t, "not executable", code, out, errText, 0, activated+undoLine, "")
	if err := os.Chmod(hook, 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(f.repo, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	code, out, errText = f.hooks(sub)
	expect(t, "executable", code, out, errText, 0, activated+"ran in "+f.repo+"\n"+undoLine, "")
	if data, _ := os.ReadFile(filepath.Join(f.repo, "ran")); string(data) != "x\n" {
		t.Errorf("post-merge runs = %q; want one", data)
	}
}

// TestHooksPostMergeFailureIsItsExitCode: the run stops there, as the `set -e` script did.
func TestHooksPostMergeFailureIsItsExitCode(t *testing.T) {
	f := newFixture(t)
	write(t, filepath.Join(f.repo, ".githooks", "post-merge"), "#!/usr/bin/env bash\necho broken >&2\nexit 7\n", 0o755)
	code, out, errText := f.hooks(f.repo)
	expect(t, "hook fails", code, out, errText, 7, activated, "broken\n")
}

// TestHooksPostMergeWithoutAShebangRunsAsShell: bash ran such a file itself; so does the port.
func TestHooksPostMergeWithoutAShebangRunsAsShell(t *testing.T) {
	f := newFixture(t)
	write(t, filepath.Join(f.repo, ".githooks", "post-merge"), "echo plain\n", 0o755)
	code, out, errText := f.hooks(f.repo)
	expect(t, "no shebang", code, out, errText, 0, activated+"plain\n"+undoLine, "")
}

// TestHooksUsage: --help prints the one usage line; an unknown argument is exit 2; both are decided
// in argument order before anything is read or written.
func TestHooksUsage(t *testing.T) {
	f := newFixture(t)
	outside := t.TempDir()
	for _, flag := range []string{"-h", "--help"} {
		code, out, errText := f.hooks(outside, "--force", flag, "--bogus")
		expect(t, flag, code, out, errText, 0, "Usage: install-hooks.sh [--force] [--uninstall]\n", "")
	}
	code, out, errText := f.hooks(f.repo, "--bogus", "--help")
	expect(t, "unknown", code, out, errText, 2, "", "install-hooks: unknown arg: --bogus\n")
	if got := f.local("core.hooksPath"); got != "" {
		t.Errorf("a usage error set core.hooksPath = %q", got)
	}
}

// TestHooksOutsideARepository: exit 2, nothing written.
func TestHooksOutsideARepository(t *testing.T) {
	f := newFixture(t)
	outside := filepath.Join(f.base, "plain")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{nil, {"--uninstall"}} {
		code, out, errText := f.hooks(outside, args...)
		expect(t, "outside", code, out, errText, 2, "", "install-hooks: not a git repository\n")
	}
}

// TestHooksAFailingGitCommandEndsWithItsStatus: the `set -e` script stopped with git's own exit status
// and message; a bare repository has no work tree root (128, after core.hooksPath is set) and a key
// set twice cannot be unset (5).
func TestHooksAFailingGitCommandEndsWithItsStatus(t *testing.T) {
	f := newFixture(t)
	bare := filepath.Join(f.base, "bare.git")
	git(t, f.base, "init", "-q", "--bare", "-b", "main", bare)
	code, out, errText := f.hooks(bare)
	if code != 128 || out != activated || !strings.HasPrefix(errText, "fatal:") {
		t.Errorf("bare: exit %d, stdout %q, stderr %q; want 128, the activation lines and git's fatal line", code, out, errText)
	}
	git(t, f.repo, "config", "--local", "--add", "core.hooksPath", ".githooks")
	git(t, f.repo, "config", "--local", "--add", "core.hooksPath", ".githooks")
	code, out, errText = f.hooks(f.repo, "--uninstall")
	if code != 5 || out != "" || errText == "" {
		t.Errorf("two values: exit %d, stdout %q, stderr %q; want git's 5 and its message", code, out, errText)
	}
	if got := f.local("core.hooksPath"); got != ".githooks\n.githooks" {
		t.Errorf("core.hooksPath = %q; want both values left", got)
	}
}
