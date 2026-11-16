package skillinstall

import (
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/installtools"
)

func sha(t *testing.T, path string) string {
	t.Helper()
	sum, err := installtools.SHA256(path)
	if err != nil {
		t.Fatal(err)
	}
	return sum
}

// noKitState is the warning of a build-n4-module install over a kit without toolbelt/kit-state.sh.
func (f *fixture) noKitState() string {
	return "install-skill: no toolbelt/kit-state.sh in " + f.kit + "; generated kit state not materialized\n"
}

// TestInstallCopiesTheTrackedLauncher: the default skill is installed byte-identical under --home.
func TestInstallCopiesTheTrackedLauncher(t *testing.T) {
	f := newFixture(t)
	code, out, errText := f.install()
	expect(t, "install", code, out, errText, 0, "install-skill: installed "+f.installed("build-n4-module")+"\n", f.noKitState())
	if got := read(t, f.installed("build-n4-module")); got != buildSkill {
		t.Errorf("installed copy = %q", got)
	}
	absent(t, filepath.Join(f.home, ".claude", "skills", "mcp-n4"))
	absent(t, f.env["HOME"])
}

// TestInstallSkillSelectsTheTableEntry: --skill mcp-n4 installs that launcher only, with no kit state.
func TestInstallSkillSelectsTheTableEntry(t *testing.T) {
	f := newFixture(t)
	f.kitState(f.kit, "0")
	code, out, errText := f.install("--skill", "mcp-n4")
	expect(t, "mcp-n4", code, out, errText, 0, "install-skill: installed "+f.installed("mcp-n4")+"\n", "")
	if got := read(t, f.installed("mcp-n4")); got != mcpSkill {
		t.Errorf("installed copy = %q", got)
	}
	absent(t, filepath.Join(f.home, ".claude", "skills", "build-n4-module"))
	absent(t, filepath.Join(f.kit, kitStateLog))
}

// TestInstallAlreadyCurrent: a second run writes nothing and says so.
func TestInstallAlreadyCurrent(t *testing.T) {
	f := newFixture(t)
	f.install()
	code, out, errText := f.install()
	expect(t, "second run", code, out, errText, 0, "install-skill: already current: "+f.installed("build-n4-module")+"\n", f.noKitState())
}

// TestInstallRefusesADivergedCopyWithoutForce: exit 1, both digests named, the installed copy untouched.
func TestInstallRefusesADivergedCopyWithoutForce(t *testing.T) {
	f := newFixture(t)
	target := write(t, f.installed("build-n4-module"), "edited by hand\n")
	code, out, errText := f.install()
	expect(t, "diverged", code, out, errText, 1, "",
		"install-skill: installed copy diverges from tracked source\n"+
			"  tracked:  "+f.tracked("build-n4-module")+"\n"+
			"  installed: "+target+"\n"+
			"  tracked sha256:   "+sha(t, f.tracked("build-n4-module"))+"\n"+
			"  installed sha256: "+sha(t, target)+"\n"+
			"Re-run with --force to overwrite.\n")
	if got := read(t, target); got != "edited by hand\n" {
		t.Errorf("the diverged copy was changed: %q", got)
	}
}

// TestInstallForceOverwritesADivergedCopy: --force restores parity and says it overwrote.
func TestInstallForceOverwritesADivergedCopy(t *testing.T) {
	f := newFixture(t)
	target := write(t, f.installed("build-n4-module"), "edited by hand\n")
	code, out, errText := f.install("--force")
	expect(t, "force", code, out, errText, 0,
		"install-skill: --force: overwriting diverged copy\ninstall-skill: installed "+target+"\n", f.noKitState())
	if got := read(t, target); got != buildSkill {
		t.Errorf("installed copy = %q", got)
	}
}

// TestInstallDryRunWritesNothing: the three reports (absent, current, diverged), each naming the kit
// whose state a real run would materialize; no file and no kit-state run.
func TestInstallDryRunWritesNothing(t *testing.T) {
	f := newFixture(t)
	f.kitState(f.kit, "0")
	target := f.installed("build-n4-module")
	kitLine := "install-skill: [dry-run] would run kit-state --write in: " + f.kit + "\n"
	code, out, errText := f.install("--dry-run")
	expect(t, "absent", code, out, errText, 0, "install-skill: [dry-run] would install to: "+target+"\n"+kitLine, "")
	absent(t, filepath.Join(f.home, ".claude"))

	write(t, target, buildSkill)
	code, out, errText = f.install("--dry-run")
	expect(t, "current", code, out, errText, 0, "install-skill: [dry-run] already current: "+target+"\n"+kitLine, "")

	write(t, target, "edited by hand\n")
	code, out, errText = f.install("--dry-run", "--skill", "build-n4-module")
	expect(t, "diverged", code, out, errText, 0,
		"install-skill: [dry-run] would update (diverged): "+target+"\n"+
			"  tracked sha256: "+sha(t, f.tracked("build-n4-module"))+"\n"+
			"  current sha256: "+sha(t, target)+"\n"+kitLine, "")
	if got := read(t, target); got != "edited by hand\n" {
		t.Errorf("--dry-run changed the installed copy: %q", got)
	}
	absent(t, filepath.Join(f.kit, kitStateLog))

	code, out, errText = f.install("--dry-run", "--skill", "mcp-n4")
	expect(t, "mcp-n4", code, out, errText, 0, "install-skill: [dry-run] would install to: "+f.installed("mcp-n4")+"\n", "")
}
