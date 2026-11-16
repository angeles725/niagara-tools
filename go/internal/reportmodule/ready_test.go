package reportmodule

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var readyFull = []string{"--ready", "--niagara-home", "", "--expect-niagara", "4.15", "--expect-signer", "acmesigner"}

func fullArgs(fx readyFx) []string {
	a := append([]string(nil), readyFull...)
	a[2] = fx.home
	return a
}

// TestReadyEveryCheckPasses (P4): a signed jar of the expected signer, a baja floor and a compile home on the
// target line, a SAFE schema-risk and a mirror holding the deployed jar give one PASS row per check, the READY
// TO LOAD line and the post-load read-back, exit 0; verify-module is run with the expected signer.
func TestReadyEveryCheckPasses(t *testing.T) {
	t.Parallel()
	fx := newReadyFx(t)
	code, out, errOut := fx.run(t, fullArgs(fx)...)
	sum := sumOf(t, fx.jar)
	want := map[string]string{
		"module":         "Mod-rt  PASS  module  Mod-rt 2.8.6",
		"niagara-target": "Mod-rt  PASS  niagara-target  baja >= 4.14 (target 4.15)",
		"niagara-home":   "Mod-rt  PASS  niagara-home  compiled against Niagara 4.15.3.28 (target 4.15)",
		"signer":         "Mod-rt  PASS  signer  ACMESIGN (expected acmesigner)",
		"verify-module":  "Mod-rt  PASS  verify-module  passed",
		"schema-risk":    "Mod-rt  PASS  schema-risk  verdict=SAFE",
		"sha256":         "Mod-rt  PASS  sha256  " + sum,
	}
	for check, line := range want {
		if got := readyRowOf(out, check); got != line {
			t.Errorf("%s row = %q, want %q", check, got, line)
		}
	}
	if got := readyRowOf(out, "mirror"); !strings.HasPrefix(got, "Mod-rt  PASS  mirror  ") {
		t.Errorf("mirror row = %q, want PASS", got)
	}
	if code != ExitClean || errOut != "" || !strings.Contains(out, "READY TO LOAD: Mod-rt 2.8.6 for Niagara 4.15 (sha256 "+sum[:12]+")") {
		t.Errorf("exit %d (want 0), stderr %q, want the READY line:\n%s", code, errOut, out)
	}
	if !strings.Contains(out, "n4_station_runtime") || !strings.Contains(out, `"Mod-rt"`) || !strings.Contains(out, `"2.8.6"`) {
		t.Errorf("no post-load read-back naming the module and version:\n%s", out)
	}
	args, err := os.ReadFile(filepath.Join(fx.tb, "verify.args"))
	must(t, err)
	if got, w := strings.TrimSpace(string(args)), "--expected-signer acmesigner --src "+fx.root+" "+fx.jar; got != w {
		t.Errorf("verify-module args = %q, want %q", got, w)
	}
}

// TestReadyNotReadyListsTheFailingRows: a FAIL row makes the verdict NOT READY with the failing rows under it,
// exit 1, and no READY line or read-back.
func TestReadyNotReadyListsTheFailingRows(t *testing.T) {
	t.Parallel()
	fx := newReadyFx(t)
	args := fullArgs(fx)
	args[6] = "othersigner"
	code, out, _ := fx.run(t, args...)
	if got := readyRowOf(out, "signer"); got != "Mod-rt  FAIL  signer  ACMESIGN, expected othersigner (OTHERSIG)" {
		t.Errorf("signer row = %q", got)
	}
	if code != ExitFail || strings.Contains(out, "READY TO LOAD") || strings.Contains(out, "n4_station_runtime") ||
		!strings.Contains(out, "NOT READY: Mod-rt 2.8.6 — 1 failing row:\n  Mod-rt  FAIL  signer  ") {
		t.Errorf("exit %d, want 1 with NOT READY and the failing row:\n%s", code, out)
	}
}

// TestReadySigner: the development signer and an unsigned jar FAIL with or without --expect-signer; with no
// --expect-signer another signer is a WARN (the operator's signer was not checked).
func TestReadySigner(t *testing.T) {
	t.Parallel()
	cases := []struct{ signer, want string }{
		{"NIAGARA4", "Mod-rt  FAIL  signer  NIAGARA4 is the development signer: a station rejects the jar as self-signed"},
		{"", "Mod-rt  FAIL  signer  unsigned (no META-INF/<SIGNER>.SF with its block file)"},
		{"ACMESIGN", "Mod-rt  WARN  signer  ACMESIGN (no --expect-signer: the signer was not checked)"},
	}
	for _, c := range cases {
		fx := newReadyFx(t)
		writeJar(t, fx.jar, moduleJar("2.8.6", "4.14", c.signer, "new"))
		_, out, _ := fx.run(t, "--ready", "--niagara-home", fx.home)
		if got := readyRowOf(out, "signer"); got != c.want {
			t.Errorf("signer %q: row = %q, want %q", c.signer, got, c.want)
		}
	}
}

// TestReadyNiagaraTarget: the baja floor and the compile home are checked against --expect-niagara: a floor above
// it or a home on a newer line FAIL, a home on an older line WARNs; without the flag both WARN, and without
// --niagara-home the home row is a SKIP.
func TestReadyNiagaraTarget(t *testing.T) {
	t.Parallel()
	fx := newReadyFx(t)
	_, out, _ := fx.run(t, "--ready", "--niagara-home", fx.home, "--expect-niagara", "4.14")
	if got := readyRowOf(out, "niagara-home"); got != "Mod-rt  FAIL  niagara-home  compiled against Niagara 4.15.3.28, newer than the target 4.14" {
		t.Errorf("newer home: %q", got)
	}
	writeJar(t, fx.jar, moduleJar("2.8.6", "4.15", "ACMESIGN", "new"))
	_, out, _ = fx.run(t, "--ready", "--niagara-home", fx.home, "--expect-niagara", "4.14.1")
	if got := readyRowOf(out, "niagara-target"); got != "Mod-rt  FAIL  niagara-target  baja >= 4.15, above the target 4.14.1" {
		t.Errorf("floor above target: %q", got)
	}
	_, out, _ = fx.run(t, "--ready", "--niagara-home", fx.home, "--expect-niagara", "4.16")
	if got := readyRowOf(out, "niagara-home"); got != "Mod-rt  WARN  niagara-home  compiled against Niagara 4.15.3.28, an older line than the target 4.16" {
		t.Errorf("older home: %q", got)
	}
	code, out, _ := fx.run(t, "--ready", "--niagara-home", fx.home)
	if readyRowOf(out, "niagara-target") != "Mod-rt  WARN  niagara-target  baja >= 4.15 (no --expect-niagara: the target was not checked)" ||
		readyRowOf(out, "niagara-home") != "Mod-rt  WARN  niagara-home  compiled against Niagara 4.15.3.28 (no --expect-niagara: the target was not checked)" ||
		code != ExitClean || !strings.Contains(out, "READY TO LOAD: Mod-rt 2.8.6 for Niagara 4.15.3.28 (sha256 ") {
		t.Errorf("no target: exit %d\n%s", code, out)
	}
	_, out, _ = fx.run(t, "--ready")
	if got := readyRowOf(out, "niagara-home"); got != "Mod-rt  SKIP  niagara-home  no --niagara-home: the compile home is unknown" {
		t.Errorf("no home: %q", got)
	}
}

// TestReadySchemaRisk: no .deploy-baseline is a FAIL naming the command that creates it (SKIP with
// --first-deploy), OUTAGE is a FAIL, a schema-risk env fault an ERROR (exit 3).
func TestReadySchemaRisk(t *testing.T) {
	t.Parallel()
	fx := newReadyFx(t)
	must(t, os.Remove(filepath.Join(fx.art, ".deploy-baseline")))
	code, out, _ := fx.run(t, fullArgs(fx)...)
	if got := readyRowOf(out, "schema-risk"); code != ExitFail || !strings.HasPrefix(got, "Mod-rt  FAIL  schema-risk  no .deploy-baseline") || !strings.Contains(got, "archive") {
		t.Errorf("no baseline: exit %d row %q", code, got)
	}
	code, out, _ = fx.run(t, append(fullArgs(fx), "--first-deploy")...)
	if got := readyRowOf(out, "schema-risk"); code != ExitClean || got != "Mod-rt  SKIP  schema-risk  N/A (--first-deploy: never deployed, no baseline to diff)" {
		t.Errorf("first deploy: exit %d row %q", code, got)
	}
	must(t, os.Mkdir(filepath.Join(fx.art, ".deploy-baseline"), 0o755))
	fx.stub(t, "schema-risk", "exit 2")
	if code, out, _ = fx.run(t, fullArgs(fx)...); code != ExitFail || readyRowOf(out, "schema-risk") != "Mod-rt  FAIL  schema-risk  verdict=OUTAGE" {
		t.Errorf("outage: exit %d\n%s", code, out)
	}
	fx.stub(t, "schema-risk", "exit 3")
	if code, out, _ = fx.run(t, fullArgs(fx)...); code != ExitEnv || readyRowOf(out, "schema-risk") != "Mod-rt  ERROR  schema-risk  env fault (exit 3)" ||
		!strings.Contains(out, "NOT READY") {
		t.Errorf("env fault: exit %d\n%s", code, out)
	}
}

// TestReadyVerifyModule: a gate FAIL names the failing gate rows; its env fault is an ERROR, exit 3.
func TestReadyVerifyModule(t *testing.T) {
	t.Parallel()
	fx := newReadyFx(t)
	fx.stub(t, "verify-module", "printf 'PASS  signed  x  ok\\nFAIL  bytecode  x  major 65\\n'; exit 1")
	code, out, _ := fx.run(t, fullArgs(fx)...)
	if got := readyRowOf(out, "verify-module"); code != ExitFail || got != "Mod-rt  FAIL  verify-module  FAIL  bytecode  x  major 65" {
		t.Errorf("gate fail: exit %d row %q", code, got)
	}
	fx.stub(t, "verify-module", "exit 3")
	if code, out, _ = fx.run(t, fullArgs(fx)...); code != ExitEnv || readyRowOf(out, "verify-module") != "Mod-rt  ERROR  verify-module  env fault (exit 3)" {
		t.Errorf("env fault: exit %d\n%s", code, out)
	}
}
