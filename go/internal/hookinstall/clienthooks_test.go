package hookinstall

import (
	"os"
	"path/filepath"
	"testing"
)

const clientUsage = "usage: install-client-hooks.sh [--force] <client-repo>\n"

func (f *fixture) installedLines() string {
	return "install-client-hooks: " + f.repo + "/.githooks/pre-push active (split-package-check --strict on every push).\n" +
		"                      Commit .githooks/pre-push; each new clone re-runs this installer once.\n"
}

// TestClientInstall: the hook is copied executable, core.hooksPath and n4kit.toolbelt are set in the
// clone's local config; a second run and a run on a sub-directory change nothing.
func TestClientInstall(t *testing.T) {
	f := newFixture(t)
	sub := filepath.Join(f.repo, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, arg := range []string{f.repo, f.repo, sub} {
		code, out, errText := f.client(arg)
		expect(t, arg, code, out, errText, 0, f.installedLines(), "")
		data, _ := os.ReadFile(f.hook())
		if st, err := os.Stat(f.hook()); err != nil || string(data) != templateText || st.Mode().Perm()&0o100 == 0 {
			t.Errorf("hook: %v, content %q; want the template, executable", err, data)
		}
		if hp, tb := f.local("core.hooksPath"), f.local("n4kit.toolbelt"); hp != ".githooks" || tb != f.toolbelt() {
			t.Errorf("core.hooksPath = %q, n4kit.toolbelt = %q", hp, tb)
		}
	}
	absent(t, filepath.Join(sub, ".githooks"))
}

// TestClientMakesAnIdenticalHookExecutable: a committed hook that lost its bit is fixed, not rewritten.
func TestClientMakesAnIdenticalHookExecutable(t *testing.T) {
	f := newFixture(t)
	write(t, f.hook(), templateText, 0o644)
	code, out, errText := f.client(f.repo)
	expect(t, "identical", code, out, errText, 0, f.installedLines(), "")
	if st, _ := os.Stat(f.hook()); st.Mode().Perm()&0o100 == 0 {
		t.Errorf("hook mode %v; want it executable", st.Mode().Perm())
	}
}

// TestClientRefusesWhatIsNotAWorkTree: exit 2 and nothing written.
func TestClientRefusesWhatIsNotAWorkTree(t *testing.T) {
	f := newFixture(t)
	plain := filepath.Join(f.base, "plain")
	if err := os.MkdirAll(plain, 0o755); err != nil {
		t.Fatal(err)
	}
	code, out, errText := f.client(plain)
	expect(t, "plain directory", code, out, errText, 2, "", "install-client-hooks: not a git work tree: "+plain+"\n")
	absent(t, filepath.Join(plain, ".githooks"))
	gone := filepath.Join(f.base, "gone")
	code, out, errText = f.client(gone)
	expect(t, "missing", code, out, errText, 2, "", "install-client-hooks: not a directory: "+gone+"\n")
}

// TestClientNeverOverwritesADifferentHook: exit 3 and the client's hook untouched, unless --force.
func TestClientNeverOverwritesADifferentHook(t *testing.T) {
	f := newFixture(t)
	write(t, f.hook(), "#!/bin/sh\necho mine\n", 0o755)
	code, out, errText := f.client(f.repo)
	expect(t, "different", code, out, errText, 3, "", "install-client-hooks: REFUSING — "+f.hook()+
		" exists and differs from the kit template; nothing changed. Re-run with --force.\n")
	if data, _ := os.ReadFile(f.hook()); string(data) != "#!/bin/sh\necho mine\n" || f.local("core.hooksPath") != "" {
		t.Errorf("refused, yet hook = %q, core.hooksPath = %q", data, f.local("core.hooksPath"))
	}
	code, out, errText = f.client("--force", f.repo)
	expect(t, "--force", code, out, errText, 0, f.installedLines(), "")
	if data, _ := os.ReadFile(f.hook()); string(data) != templateText {
		t.Errorf("--force left hook = %q", data)
	}
}

// TestClientNeverReplacesACustomHooksPath: a local or a global core.hooksPath is exit 3, nothing
// changed; --force sets the local one.
func TestClientNeverReplacesACustomHooksPath(t *testing.T) {
	f := newFixture(t)
	git(t, f.repo, "config", "--local", "core.hooksPath", ".husky")
	code, out, errText := f.client(f.repo)
	expect(t, "local", code, out, errText, 3, "",
		"install-client-hooks: REFUSING — core.hooksPath is already \".husky\"; nothing changed. Re-run with --force.\n")
	git(t, f.repo, "config", "--local", "--unset", "core.hooksPath")
	git(t, f.repo, "config", "--global", "core.hooksPath", "/global/hooks")
	code, out, errText = f.client(f.repo)
	expect(t, "global", code, out, errText, 3, "",
		"install-client-hooks: REFUSING — core.hooksPath is already \"/global/hooks\"; nothing changed. Re-run with --force.\n")
	if f.local("core.hooksPath") != "" {
		t.Error("refused, yet the local core.hooksPath is set")
	}
	absent(t, f.hook())
	code, out, errText = f.client(f.repo, "--force")
	expect(t, "--force", code, out, errText, 0, f.installedLines(), "")
	if got := f.local("core.hooksPath"); got != ".githooks" {
		t.Errorf("--force left the local core.hooksPath = %q", got)
	}
}

// TestClientNeverDisablesActiveDefaultHooks: executable non-sample hooks of the default hooks directory
// are named and the install refused; --force installs.
func TestClientNeverDisablesActiveDefaultHooks(t *testing.T) {
	f := newFixture(t)
	hooksDir := filepath.Join(f.repo, ".git", "hooks")
	write(t, filepath.Join(hooksDir, "pre-commit"), "#!/bin/sh\n", 0o755)
	write(t, filepath.Join(hooksDir, "commit-msg"), "#!/bin/sh\n", 0o755)
	write(t, filepath.Join(hooksDir, "pre-rebase"), "#!/bin/sh\n", 0o644)    // not executable: inactive
	write(t, filepath.Join(hooksDir, "active.sample"), "#!/bin/sh\n", 0o755) // a sample: inactive
	code, out, errText := f.client(f.repo)
	expect(t, "active hooks", code, out, errText, 3, "",
		"install-client-hooks: REFUSING — active hooks in "+hooksDir+" would stop running: commit-msg pre-commit; nothing changed.\n"+
			"                      Move them into .githooks/ (or chain them) and re-run with --force.\n")
	absent(t, f.hook())
	if f.local("core.hooksPath") != "" {
		t.Error("refused, yet core.hooksPath is set")
	}
	code, out, errText = f.client("--force", f.repo)
	expect(t, "--force", code, out, errText, 0, f.installedLines(), "")
}

// TestClientUsage: no repository, two of them and an unknown option are exit 2; --help is the usage
// line on stdout.
func TestClientUsage(t *testing.T) {
	f := newFixture(t)
	for _, args := range [][]string{nil, {"--force"}, {f.repo, f.repo}} {
		code, out, errText := f.client(args...)
		expect(t, "usage", code, out, errText, 2, "", clientUsage)
	}
	code, out, errText := f.client("--bogus", f.repo)
	expect(t, "unknown option", code, out, errText, 2, "", "install-client-hooks: unknown option: --bogus\n"+clientUsage)
	for _, flag := range []string{"-h", "--help"} {
		code, out, errText = f.client(f.repo, flag, "--bogus")
		expect(t, flag, code, out, errText, 0, clientUsage, "")
	}
	absent(t, f.hook())
}

// TestClientKitLayoutErrors: a missing template is exit 2 naming it; a missing toolbelt directory is
// exit 1 (the shell's failed `cd`); an unusable launcher variable is exit 2.
func TestClientKitLayoutErrors(t *testing.T) {
	f := newFixture(t)
	if err := os.Remove(f.template()); err != nil {
		t.Fatal(err)
	}
	code, out, errText := f.client(f.repo)
	expect(t, "template", code, out, errText, 2, "",
		"install-client-hooks: template missing: "+f.toolbelt()+"/../templates/client-pre-push\n")
	if err := os.Remove(f.toolbelt()); err != nil {
		t.Fatal(err)
	}
	code, out, errText = f.client(f.repo)
	expect(t, "toolbelt", code, out, errText, 1, "", "install-client-hooks: kit toolbelt not found: "+f.toolbelt()+"\n")
	f.launcher = ""
	code, out, errText = f.client(f.repo)
	expect(t, "launcher", code, out, errText, 2, "", "install-client-hooks: INSTALL_TOOLS_LAUNCHER is not the absolute path of the invoked script (\"\"); run the scripts/ launcher\n")
	absent(t, f.hook())
}
