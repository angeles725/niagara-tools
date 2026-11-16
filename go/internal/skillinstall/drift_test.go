package skillinstall

import (
	"os"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/installtools"
)

const driftUsage = "usage: check-skill-drift.sh [--skill <name>] [--harness <claude|pi|gentle-shell|all> | --all] [--home <dir>]\n"

// TestDriftPassesOnAnIdenticalCopy: one PASS row naming the tracked path relative to the repository.
func TestDriftPassesOnAnIdenticalCopy(t *testing.T) {
	f := newFixture(t)
	write(t, f.installed("build-n4-module"), buildSkill)
	code, out, errText := f.drift()
	expect(t, "pass", code, out, errText, 0,
		"PASS  skill-drift  build-n4-module  installed copy matches build-n4-module-kit/skill/SKILL.md\n", "")
}

// TestDriftFailsOnADivergedCopy: a FAIL row with the --force remedy, exit 1, the copy never written.
func TestDriftFailsOnADivergedCopy(t *testing.T) {
	f := newFixture(t)
	target := write(t, f.installed("mcp-n4"), "stale\n")
	code, out, errText := f.drift("--skill", "mcp-n4")
	expect(t, "diverged", code, out, errText, 1,
		"FAIL  skill-drift  mcp-n4  installed copy "+target+" differs from mcp-n4-kit/skill/SKILL.md"+
			" — run scripts/install-skill.sh --force --skill mcp-n4\n", "")
	if got := read(t, target); got != "stale\n" {
		t.Errorf("the check wrote the installed copy: %q", got)
	}
}

// TestDriftFailsWhenNotInstalled: a FAIL row naming the path and the install remedy, exit 1.
func TestDriftFailsWhenNotInstalled(t *testing.T) {
	f := newFixture(t)
	code, out, errText := f.drift()
	expect(t, "not installed", code, out, errText, 1,
		"FAIL  skill-drift  build-n4-module  not installed at "+f.installed("build-n4-module")+
			" — run scripts/install-skill.sh --skill build-n4-module\n", "")
	absent(t, f.home)
}

// TestDriftUsageErrors: an unknown skill, an option without its value and an unknown argument are exit 2.
func TestDriftUsageErrors(t *testing.T) {
	f := newFixture(t)
	code, out, errText := f.drift("--skill", "nope")
	expect(t, "unknown skill", code, out, errText, 2, "", "check-skill-drift: unknown skill: nope\n"+driftUsage)
	for _, args := range [][]string{{"--skill"}, {"--home"}, {"--bogus"}, {"extra"}} {
		code, out, errText = f.drift(args...)
		expect(t, args[0], code, out, errText, 2, "", driftUsage)
	}
}

// TestDriftHelpPrintsTheLauncherHeader: the usage is the invoked script's header, read when asked.
func TestDriftHelpPrintsTheLauncherHeader(t *testing.T) {
	f := newFixture(t)
	for _, flag := range []string{"-h", "--help"} {
		code, out, errText := f.drift(flag)
		expect(t, flag, code, out, errText, 0, "check-skill-drift.sh — fixture header.\n\n  second line\n", "")
	}
}

// TestDriftEnvironmentErrors: no installer, no table in it, no tracked launcher, no home, no launcher
// variable and an installed copy that cannot be read are exit 3, never a PASS row.
func TestDriftEnvironmentErrors(t *testing.T) {
	f := newFixture(t)
	write(t, f.installed("build-n4-module"), buildSkill)

	f.env["HOME"] = ""
	code, out, errText := f.driftNoHome()
	expect(t, "no home", code, out, errText, 3, "", "check-skill-drift: no home directory (pass --home)\n")

	if os.Getuid() != 0 {
		if err := os.Chmod(f.installed("build-n4-module"), 0); err != nil {
			t.Fatal(err)
		}
		code, out, errText = f.drift()
		expect(t, "unreadable copy", code, out, errText, 3, "",
			"check-skill-drift: cannot read "+f.installed("build-n4-module")+": permission denied\n"+
				"check-skill-drift: could not compute a digest\n")
	}

	if err := os.Remove(f.tracked("build-n4-module")); err != nil {
		t.Fatal(err)
	}
	code, out, errText = f.drift()
	expect(t, "tracked missing", code, out, errText, 3, "",
		"check-skill-drift: tracked launcher not found: "+f.tracked("build-n4-module")+"\n")

	write(t, f.installer, "#!/usr/bin/env bash\nset -u\n")
	code, out, errText = f.drift()
	expect(t, "no table", code, out, errText, 3, "", "check-skill-drift: no SKILLS table in "+f.installer+"\n")

	if err := os.Remove(f.installer); err != nil {
		t.Fatal(err)
	}
	code, out, errText = f.drift()
	expect(t, "no installer", code, out, errText, 3, "", "check-skill-drift: installer not found: "+f.installer+"\n")

	f.driftCheck = "relative/check-skill-drift.sh"
	code, out, errText = f.drift()
	expect(t, "no launcher", code, out, errText, 3, "", "check-skill-drift: "+installtools.LauncherEnv+
		" is not the absolute path of the invoked script (\"relative/check-skill-drift.sh\"); run the scripts/ launcher\n")
}
