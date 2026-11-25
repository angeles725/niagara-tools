package skillinstall

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/installtools"
)

// The command line and the environment of install-skill: usage errors, the home, the SKILLS table and
// what makes a run an environment error. The file placement rules are in install_test.go.

// TestInstallUsageErrors: every malformed command line is exit 2 with its message and nothing written.
func TestInstallUsageErrors(t *testing.T) {
	f := newFixture(t)
	for _, c := range []struct {
		name string
		args []string
		want string
	}{
		{"--skill without a value", []string{"--skill"}, "install-skill: --skill requires an argument\n"},
		{"--home without a value", []string{"--home"}, "install-skill: --home requires an argument\n"},
		{"unknown option", []string{"--bogus"}, "install-skill: unknown option: --bogus\n" + wantUsage},
		{"positional", []string{"extra"}, "install-skill: unexpected argument: extra\n" + wantUsage},
		{"unknown skill", []string{"--skill", "nope"},
			"install-skill: unknown skill: nope (expected build-n4-module or mcp-n4)\n" + wantUsage},
	} {
		code, out, errText := f.install(c.args...)
		expect(t, c.name, code, out, errText, 2, "", c.want)
	}
	absent(t, f.home)
}

// TestInstallDoubleDashEndsTheOptions: what follows `--` is not read (the bash loop stopped there).
func TestInstallDoubleDashEndsTheOptions(t *testing.T) {
	f := newFixture(t)
	code, out, errText := f.install("--dry-run", "--", "--bogus", "extra")
	expect(t, "--", code, out, errText, 0,
		"install-skill: [dry-run] would install to: "+f.installed("build-n4-module")+"\n"+
			"install-skill: [dry-run] would run kit-state --write in: "+f.kit+"\n", "")
}

// TestInstallHomeDefaultsToTheEnvironment: without --home the base is $HOME; an empty one is refused
// instead of installing under the filesystem root.
func TestInstallHomeDefaultsToTheEnvironment(t *testing.T) {
	f := newFixture(t)
	f.env["HOME"] = f.home
	out, errb := new(bytes.Buffer), new(bytes.Buffer)
	code := Main([]string{"--dry-run", "--skill", "mcp-n4"}, f.stdio(f.installer, out, errb))
	expect(t, "$HOME", code, out.String(), errb.String(), 0, "install-skill: [dry-run] would install to: "+f.installed("mcp-n4")+"\n", "")

	f.env["HOME"] = ""
	out.Reset()
	code = Main([]string{"--skill", "mcp-n4"}, f.stdio(f.installer, out, errb))
	expect(t, "no home", code, out.String(), errb.String(), 3, "", "install-skill: no home directory (pass --home)\n")
}

// TestInstallTableIsTheSingleSource: a skill added to the launcher's SKILLS line installs, and the
// messages list the names from that line.
func TestInstallTableIsTheSingleSource(t *testing.T) {
	f := newFixture(t)
	write(t, f.installer, "#!/usr/bin/env bash\nSKILLS=\"extra:extra-kit/skill/SKILL.md build-n4-module:build-n4-module-kit/skill/SKILL.md\"\n")
	write(t, filepath.Join(f.root, "extra-kit", "skill", "SKILL.md"), "stub extra skill\n")
	code, out, errText := f.install("--skill", "extra")
	expect(t, "extra", code, out, errText, 0, "install-skill: installed "+f.installed("extra")+"\n", "")
	code, out, errText = f.install("--skill", "mcp-n4")
	expect(t, "removed from the table", code, out, errText, 2, "",
		"install-skill: unknown skill: mcp-n4 (expected extra or build-n4-module)\n"+
			"usage: install-skill.sh [--skill <extra|build-n4-module>] [--harness <claude|pi|gentle-shell|all>] [--home <dir>] [--dry-run] [--force]\n"+
			"       install-skill.sh --check [--home <dir>]\n")

	write(t, f.installer, "#!/usr/bin/env bash\nset -u\n")
	code, out, errText = f.install()
	expect(t, "no table", code, out, errText, 3, "", "install-skill: no SKILLS table in "+f.installer+"\n")
}

// TestInstallEnvironmentErrors: a missing tracked source, an unusable launcher variable and a target
// directory that cannot be created are exit 3, nothing installed.
func TestInstallEnvironmentErrors(t *testing.T) {
	f := newFixture(t)
	if err := os.Remove(f.tracked("mcp-n4")); err != nil {
		t.Fatal(err)
	}
	code, out, errText := f.install("--skill", "mcp-n4")
	expect(t, "tracked missing", code, out, errText, 3, "", "install-skill: tracked source not found: "+f.tracked("mcp-n4")+"\n")

	saved := f.installer
	f.installer = ""
	code, out, errText = f.install()
	expect(t, "no launcher", code, out, errText, 3, "",
		"install-skill: "+installtools.LauncherEnv+" is not the absolute path of the invoked script (\"\"); run the scripts/ launcher\n")
	f.installer = saved

	write(t, filepath.Join(f.home, ".claude"), "a file where the directory goes\n")
	code, out, errText = f.install()
	expect(t, "target dir", code, out, errText, 3, "",
		"install-skill: cannot create target directory: "+filepath.Dir(f.installed("build-n4-module"))+"\n")
}
