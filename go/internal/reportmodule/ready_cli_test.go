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
	code, out, _ := fx.run(t, append(fullArgs(fx), "--json")...)
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

// TestReadyUsage: the --ready options need --ready, and --ready takes no lint-run option.
func TestReadyUsage(t *testing.T) {
	t.Parallel()
	fx := newReadyFx(t)
	for _, args := range [][]string{{"--json"}, {"--expect-niagara", "4.15"}, {"--expect-signer", "a"},
		{"--ready", "--lint-only"}, {"--ready", "--write-baseline"}, {"--ready", "--expect-niagara"}} {
		if code, _, errOut := fx.run(t, args...); code != ExitUsage || !strings.Contains(errOut, "--ready") {
			t.Errorf("%q: exit %d, want 2 with the usage:\n%s", args, code, errOut)
		}
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
