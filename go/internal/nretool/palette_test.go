package nretool

import (
	"path/filepath"
	"strings"
	"testing"
)

const wantPaletteClass = "devkit:com.tridium.devkit.util.PaletteGenerator"

// partDir is a module part directory as the kit lays one out: module-include.xml and src/.
func partDir(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	write(t, filepath.Join(d, "module-include.xml"), "<module/>", 0o644)
	write(t, filepath.Join(d, "src", "a.gif"), "", 0o644)
	return d
}

func TestPaletteRunsPaletteGeneratorAndSaysWhatItWritesFirst(t *testing.T) {
	home, argsFile := stubHome(t, "nre.exe", "echo RAN >&2")
	dir := partDir(t)
	code, _, errs := run("palette", home, dir)
	want := filepath.Join(home, "bin") + "\n" + wantPaletteClass + "\nW:" + dir + "\n"
	got, _ := readFile(argsFile)
	note := "PaletteGenerator writes " + filepath.Join(dir, "module.palette")
	if code != 0 || got != want || !strings.Contains(errs, note) || strings.Index(errs, note) > strings.Index(errs, "RAN") {
		t.Errorf("exit %d, nre ran as %q (want %q), err %q", code, got, want, errs)
	}
}

func TestPaletteExitsWithNreExitCode(t *testing.T) {
	home, _ := stubHome(t, "nre.exe", "exit 5")
	if code, _, _ := run("palette", home, partDir(t)); code != 5 {
		t.Errorf("exit %d, want 5", code)
	}
}

func TestPaletteFailureThatNreExitsZeroOnIsStillAFailure(t *testing.T) {
	home, _ := stubHome(t, "nre.exe", "echo 'Failed to generate palette: java.io.IOException' >&2")
	code, _, errs := run("palette", home, partDir(t))
	if code != 1 || !strings.Contains(errs, "nre-tool: PaletteGenerator reported a failure") {
		t.Errorf("exit %d, err %q", code, errs)
	}
}

func TestPaletteRefusesWhatIsNotAModulePartDirectory(t *testing.T) {
	noSrc, noInclude := t.TempDir(), t.TempDir()
	write(t, filepath.Join(noSrc, "module-include.xml"), "<module/>", 0o644)
	write(t, filepath.Join(noInclude, "src", "a.gif"), "", 0o644)
	file := filepath.Join(t.TempDir(), "module-include.xml")
	write(t, file, "<module/>", 0o644)
	for _, d := range []string{filepath.Join(t.TempDir(), "missing"), noSrc, noInclude, file} {
		home, argsFile := stubHome(t, "nre.exe", "")
		code, out, errs := run("palette", home, d)
		if code != 3 || out != "" || exists(argsFile) ||
			!strings.Contains(errs, "nre-tool: palette: "+d+" is not a module part directory (needs module-include.xml and src/)") {
			t.Errorf("%s: exit %d, out %q, err %q, nre ran %v", d, code, out, errs, exists(argsFile))
		}
	}
}

func TestPaletteNeedsTheWindowsLauncher(t *testing.T) {
	home, argsFile := stubHome(t, "nre", "")
	code, _, errs := run("palette", home, partDir(t))
	if code != 3 || exists(argsFile) || !strings.Contains(errs, "writes <moduleDir>\\module.palette") {
		t.Errorf("exit %d, err %q, nre ran %v", code, errs, exists(argsFile))
	}
}

func TestPaletteUsage(t *testing.T) {
	code, _, errs := run("palette", t.TempDir())
	if code != 3 || !strings.Contains(errs, "usage: nre-tool palette <niagara_home> <moduleDir>") {
		t.Errorf("exit %d, err %q", code, errs)
	}
}
