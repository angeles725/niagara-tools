package skillinstall

import (
	"os"
	"path/filepath"
	"testing"
)

// TestInstallMaterializesTheKitState: a build-n4-module install runs `kit-state.sh --write <kit>` in the
// kit $BUILD_N4_KIT names, after the copy and again when the launcher is already current.
func TestInstallMaterializesTheKitState(t *testing.T) {
	f := newFixture(t)
	f.kitState(f.kit, "0")
	target := f.installed("build-n4-module")
	code, out, errText := f.install()
	expect(t, "install", code, out, errText, 0, "install-skill: installed "+target+"\nkit-state: wrote "+f.kit+"\n", "")
	code, out, errText = f.install()
	expect(t, "current", code, out, errText, 0, "install-skill: already current: "+target+"\nkit-state: wrote "+f.kit+"\n", "")
	if got := read(t, filepath.Join(f.kit, kitStateLog)); got != "--write "+f.kit+"\n--write "+f.kit+"\n" {
		t.Errorf("kit-state runs = %q; want two `--write <kit>`", got)
	}
}

// TestInstallKitStateFailureIsExit3: the launcher stays installed and the message names the kit.
func TestInstallKitStateFailureIsExit3(t *testing.T) {
	f := newFixture(t)
	f.kitState(f.kit, "7")
	code, out, errText := f.install()
	expect(t, "kit-state fails", code, out, errText, 3,
		"install-skill: installed "+f.installed("build-n4-module")+"\nkit-state: wrote "+f.kit+"\n",
		"install-skill: kit-state --write failed in "+f.kit+"; the launcher is installed, the kit state is not\n")
	if got := read(t, f.installed("build-n4-module")); got != buildSkill {
		t.Errorf("installed copy = %q", got)
	}
}

// TestInstallKitFallsBackToTheRepository: without $BUILD_N4_KIT, or when it holds no METHODOLOGY.md, the
// kit is <repo>/build-n4-module-kit.
func TestInstallKitFallsBackToTheRepository(t *testing.T) {
	f := newFixture(t)
	repoKit := filepath.Join(f.root, "build-n4-module-kit")
	f.kitState(repoKit, "0")
	f.kitState(f.kit, "0")
	if err := os.Remove(filepath.Join(f.kit, "METHODOLOGY.md")); err != nil {
		t.Fatal(err)
	}
	for _, kit := range []string{f.kit, ""} {
		f.env["BUILD_N4_KIT"] = kit
		code, out, errText := f.install("--force")
		if code != 0 || errText != "" {
			t.Fatalf("BUILD_N4_KIT=%q: exit %d, stdout %q, stderr %q", kit, code, out, errText)
		}
	}
	if got := read(t, filepath.Join(repoKit, kitStateLog)); got != "--write "+repoKit+"\n--write "+repoKit+"\n" {
		t.Errorf("repository kit runs = %q", got)
	}
	absent(t, filepath.Join(f.kit, kitStateLog))
}

// TestInstallRefusesWhatItCannotCheckOrWrite: the bash tool printed "installed" and exited 0 over a cp
// that failed, and read an unreadable $BUILD_N4_KIT as "no kit there"; each is exit 3 here.
func TestInstallRefusesWhatItCannotCheckOrWrite(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root writes a read-only file and reads through a mode-000 directory")
	}
	f := newFixture(t)
	target := write(t, f.installed("mcp-n4"), "edited by hand\n")
	if err := os.Chmod(target, 0o444); err != nil {
		t.Fatal(err)
	}
	code, out, errText := f.install("--skill", "mcp-n4", "--force")
	expect(t, "read-only target", code, out, errText, 3, "install-skill: --force: overwriting diverged copy\n",
		"install-skill: cannot write "+target+": permission denied\n")

	dirTarget := f.installed("build-n4-module")
	if err := os.MkdirAll(dirTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	code, out, errText = f.install()
	expect(t, "target is a directory", code, out, errText, 3, "",
		"install-skill: cannot write "+dirTarget+": is a directory\n")
	if err := os.Remove(dirTarget); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(f.kit, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(f.kit, 0o755) })
	methodology := filepath.Join(f.kit, "METHODOLOGY.md")
	code, out, errText = f.install()
	expect(t, "unreadable kit", code, out, errText, 3, "",
		"install-skill: cannot check "+methodology+": permission denied\n")
	absent(t, dirTarget)
}
