package reportmodule

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Edge cases of report-module --ready: the read error of the module root, an artifact with no built jar, a gradle
// directory beside the artifacts, a baja floor that cannot be compared with the target, the two distinct
// no-baseline results of schemaRiskVerdict (schema_verdict_test.go) and --target-version reaching verify-module.

// TestReadyUnreadableRootNamesTheReadError: a module root that cannot be listed is an env fault that
// names the read error, never "no built jar under ...".
func TestReadyUnreadableRootNamesTheReadError(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root reads a mode-000 directory")
	}
	fx := newReadyFx(t)
	must(t, os.Chmod(fx.root, 0))
	t.Cleanup(func() { _ = os.Chmod(fx.root, 0o755) })
	code, out, errOut := fx.run(t, "--ready")
	if want := "report-module: --ready: cannot list " + fx.root + ": Permission denied\n"; code != ExitEnv || out != "" || errOut != want {
		t.Errorf("exit %d stdout %q stderr %q, want exit 3 and %q", code, out, errOut, want)
	}
}

// TestReadyArtifactWithoutJarIsAFailRow: an artifact of the module (a directory with module-include.xml or
// its gradle build file) that has no built jar is a FAIL row naming it, and its verdict line says it is not built
// instead of a "?" version, so the module is NOT READY; a directory that is no artifact (docs/) gets no row.
func TestReadyArtifactWithoutJarIsAFailRow(t *testing.T) {
	t.Parallel()
	fx := newReadyFx(t)
	ux := filepath.Join(fx.root, "Mod-ux")
	must(t, os.MkdirAll(ux, 0o755))
	must(t, os.WriteFile(filepath.Join(ux, "Mod-ux.gradle.kts"), nil, 0o644))
	must(t, os.MkdirAll(filepath.Join(fx.root, "docs"), 0o755))
	must(t, os.WriteFile(filepath.Join(fx.root, "docs", "write-path-matrix.md"), nil, 0o644))
	code, out, _ := fx.run(t, fullArgs(fx, "acmesigner")...)
	want := "Mod-ux  FAIL  jar  no built jar under build/libs: this artifact of the module would not be loaded (build it first)"
	if got := rowOf(out, "Mod-ux", "jar"); got != want {
		t.Errorf("Mod-ux row = %q, want %q", got, want)
	}
	if code != ExitFail || strings.Contains(out, "docs  ") || !strings.Contains(out, "NOT READY: Mod-ux (not built) — 1 failing row:\n  Mod-ux  FAIL  jar  ") {
		t.Errorf("exit %d, want 1 with NOT READY for Mod-ux and no docs row:\n%s", code, out)
	}
}

// TestReadyIncomparableFloorBlocksReady: a baja floor that cannot be compared with --expect-niagara FAILs,
// so the jar never reads READY with its target unchecked.
func TestReadyIncomparableFloorBlocksReady(t *testing.T) {
	t.Parallel()
	fx := newReadyFx(t)
	writeJar(t, fx.jar, moduleJar("2.8.6", "4.14-rc", "ACMESIGN", "new"))
	code, out, _ := fx.run(t, fullArgs(fx, "acmesigner")...)
	want := "Mod-rt  FAIL  niagara-target  baja >= 4.14-rc cannot be compared with the target 4.15: the station may refuse the module"
	if got := readyRowOf(out, "niagara-target"); got != want || code != ExitFail || strings.Contains(out, "READY TO LOAD") {
		t.Errorf("exit %d row %q, want exit 1 and %q:\n%s", code, got, want, out)
	}
}

// TestReadyTargetVersionReachesVerifyModule: --target-version is passed to the verify-module gate before
// the expected signer.
func TestReadyTargetVersionReachesVerifyModule(t *testing.T) {
	t.Parallel()
	fx := newReadyFx(t)
	if code, out, _ := fx.run(t, append(fullArgs(fx, "acmesigner"), "--target-version", "4.15")...); code != ExitClean {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	args, err := os.ReadFile(filepath.Join(fx.tb, "verify.args"))
	must(t, err)
	if got, w := strings.TrimSpace(string(args)), "--target-version 4.15 --expected-signer acmesigner --src "+fx.root+" "+fx.jar; got != w {
		t.Errorf("verify-module args = %q, want %q", got, w)
	}
}

// TestReadyGradleDirBesideTheArtifactsGetsNoRow: a gradle directory that is not a part of the module (buildSrc/, a
// shared conventions build) holds a build.gradle.kts but is no <MOD>-rt/-ux/-wb/-se/-doc artifact and has no
// module-include.xml, so it is never reported as an unbuilt artifact and the module stays READY.
func TestReadyGradleDirBesideTheArtifactsGetsNoRow(t *testing.T) {
	t.Parallel()
	fx := newReadyFx(t)
	for _, d := range []string{"buildSrc", "conventions"} {
		must(t, os.MkdirAll(filepath.Join(fx.root, d), 0o755))
		must(t, os.WriteFile(filepath.Join(fx.root, d, "build.gradle.kts"), nil, 0o644))
	}
	code, out, _ := fx.run(t, fullArgs(fx, "acmesigner")...)
	if code != ExitClean || strings.Contains(out, "buildSrc") || strings.Contains(out, "conventions") || !strings.Contains(out, "READY TO LOAD") {
		t.Errorf("exit %d, want 0, READY TO LOAD and no buildSrc/conventions row:\n%s", code, out)
	}
}
