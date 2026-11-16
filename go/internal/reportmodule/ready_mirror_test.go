package reportmodule

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadyMirror: the mirror row says whether <home>/modules/<jar> is the deployed jar again. A live
// first-replaced pointer is the deployed jar: modules/ equal to it PASSes, anything else WARNs with the exact
// --restore-mirror command; with no pointer, modules/ holding this very build WARNs; a home that is not a mirror
// is a SKIP. The mirror row never blocks READY.
func TestReadyMirror(t *testing.T) {
	t.Parallel()
	fx := newReadyFx(t)
	modJar := filepath.Join(fx.home, "modules", "Mod-rt.jar")
	kept := filepath.Join(fx.home, ReplacedDir)
	must(t, os.MkdirAll(kept, 0o755))
	deployed := sumOf(t, modJar)[:12]
	keptJar := filepath.Join(kept, "Mod-rt.jar."+deployed)
	data, err := os.ReadFile(modJar)
	must(t, err)
	must(t, os.WriteFile(keptJar, data, 0o644))
	must(t, os.WriteFile(filepath.Join(kept, "Mod-rt.jar"+FirstSuffix), []byte(deployed+"\n"), 0o644))

	_, out, _ := fx.run(t, fullArgs(fx)...)
	if got := readyRowOf(out, "mirror"); got != "Mod-rt  PASS  mirror  modules/Mod-rt.jar is the deployed jar again (sha256 "+deployed+")" {
		t.Errorf("restored: %q", got)
	}

	built, err := os.ReadFile(fx.jar)
	must(t, err)
	must(t, os.WriteFile(modJar, built, 0o644)) // the build replaced the mirror's copy
	code, out, _ := fx.run(t, fullArgs(fx)...)
	want := "Mod-rt  WARN  mirror  modules/Mod-rt.jar (sha256 " + sumOf(t, fx.jar)[:12] + ") is not the deployed jar (sha256 " +
		deployed + "); restore with: " + ShellWord(fx.tb+"/build.sh") + " --restore-mirror " + ShellWord(keptJar)
	if got := readyRowOf(out, "mirror"); got != want || code != ExitClean || !strings.Contains(out, "READY TO LOAD") {
		t.Errorf("replaced: exit %d row\n%q\nwant\n%q", code, got, want)
	}

	must(t, os.Remove(filepath.Join(kept, "Mod-rt.jar"+FirstSuffix)))
	_, out, _ = fx.run(t, fullArgs(fx)...)
	if got := readyRowOf(out, "mirror"); !strings.HasPrefix(got, "Mod-rt  WARN  mirror  modules/Mod-rt.jar is this build (sha256 ") {
		t.Errorf("this build, no pointer: %q", got)
	}

	must(t, os.Remove(filepath.Join(fx.home, ".niagara-mirror")))
	_, out, _ = fx.run(t, fullArgs(fx)...)
	if got := readyRowOf(out, "mirror"); got != "Mod-rt  SKIP  mirror  "+fx.home+" is not a mirror (no .niagara-mirror): nothing to restore" {
		t.Errorf("not a mirror: %q", got)
	}
}

// TestReadyLeavesTheFullReportAlone: --niagara-home, --first-deploy and --target-version, which --ready reads
// too, keep their old meaning without --ready: the full report runs (its summary line) and no ready row or line.
func TestReadyLeavesTheFullReportAlone(t *testing.T) {
	t.Parallel()
	fx := newReadyFx(t)
	_, out, _ := fx.run(t, "--lint-only", "--niagara-home", fx.home, "--first-deploy", "--target-version", "4.14")
	if !strings.Contains(out, "report-module: 1 artifact · ") || strings.Contains(out, "READY") || readyRowOf(out, "sha256") != "" {
		t.Errorf("the full report changed:\n%s", out)
	}
}
