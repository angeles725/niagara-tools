package reportmodule

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadyJSON: --json prints one object with the rows, the verdict, the line and the read-back.
func TestReadyJSON(t *testing.T) {
	t.Parallel()
	fx := newReadyFx(t)
	code, out, _ := fx.run(t, append(fullArgs(fx, "acmesigner"), "--json")...)
	var got readyJSON
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	sum := sumOf(t, fx.jar)
	if code != ExitClean || got.Verdict != verdictReady || len(got.Jars) != 1 {
		t.Fatalf("exit %d, %+v", code, got)
	}
	j := got.Jars[0]
	if j.Artifact != "Mod-rt" || j.Module != "Mod-rt" || j.Version != "2.8.6" || j.Niagara != "4.15" || j.SHA256 != sum ||
		len(j.Rows) != 8 || j.Rows[0].Check != "module" || j.Line != "READY TO LOAD: Mod-rt 2.8.6 for Niagara 4.15 (sha256 "+sum[:12]+")" ||
		!strings.Contains(j.Readback, "n4_station_runtime") {
		t.Errorf("jar = %+v", j)
	}
}

// TestReadyUsage: the --ready options need --ready, and --ready refuses every option of the lint run
// (R3-ready-ignores-lint-options): exit 2 with a first line naming the option, then the usage; a named
// --wiring-map or --console-dir that does not exist is refused the same way, never silently ignored.
func TestReadyUsage(t *testing.T) {
	t.Parallel()
	fx := newReadyFx(t)
	cases := []struct {
		args []string
		line string
	}{
		{[]string{"--json"}, "report-module: --json needs --ready"},
		{[]string{"--expect-niagara", "4.15"}, "report-module: --expect-niagara needs --ready"},
		{[]string{"--expect-signer", "a"}, "report-module: --expect-signer needs --ready"},
		{[]string{"--ready", "--lint-only"}, "report-module: --lint-only is not a --ready option"},
		{[]string{"--ready", "--write-baseline"}, "report-module: --write-baseline is not a --ready option"},
		{[]string{"--ready", "--no-baseline"}, "report-module: --no-baseline is not a --ready option"},
		{[]string{"--ready", "--console-dir", "/nonexistent"}, "report-module: --console-dir is not a --ready option"},
		{[]string{"--ready", "--profile", "lan"}, "report-module: --profile is not a --ready option"},
		{[]string{"--ready", "--legacy"}, "report-module: --legacy is not a --ready option"},
		{[]string{"--ready", "--wiring-map", "/nonexistent"}, "report-module: --wiring-map is not a --ready option"},
	}
	for _, c := range cases {
		code, out, errOut := fx.run(t, c.args...)
		first, rest, _ := strings.Cut(errOut, "\n")
		if code != ExitUsage || out != "" || first != c.line || !strings.HasPrefix(rest, "usage: report-module.sh") {
			t.Errorf("%q: exit %d, want 2 with %q then the usage:\n%s", c.args, code, c.line, errOut)
		}
	}
	if code, _, errOut := fx.run(t, "--ready", "--expect-niagara"); code != ExitUsage || !strings.HasPrefix(errOut, "usage: ") {
		t.Errorf("missing value: exit %d\n%s", code, errOut)
	}
}

// TestReadyNoJar: a module root with no built jar is an env fault naming where it looked.
func TestReadyNoJar(t *testing.T) {
	t.Parallel()
	fx := newReadyFx(t)
	must(t, os.RemoveAll(filepath.Join(fx.art, "build")))
	code, out, errOut := fx.run(t, "--ready")
	if code != ExitEnv || out != "" || errOut != "report-module: --ready: no built jar under "+fx.root+"/*/build/libs\n" {
		t.Errorf("exit %d stdout %q stderr %q", code, out, errOut)
	}
}
