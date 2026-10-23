package configsanity

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-config-sanity.bats run (LCS-finderr aside: its locked directory is not a
// golden tree) and every edge tree, recorded from the bash lint-config-sanity.sh.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 46) }

// TestAwkNum pins gawk's string-to-number reading of the CS2 min facet argument.
func TestAwkNum(t *testing.T) {
	for in, want := range map[string]float64{"2.5": 2.5, "0": 0, "0.0": 0, "5,x": 5, "1e3": 1000, "1e": 1, "0x5": 0, "3.": 3, "": 0, "x": 0} {
		if got := awkNum(in); got != want {
			t.Errorf("awkNum(%q) = %v, want %v", in, got, want)
		}
	}
}

// TestUnreadableFile pins declared difference (a): a listed file the scan cannot read is "cannot scan" on
// stderr and exit 3, and the other files' rows still print. The bash lint printed four gawk fatals and
// exited by the other rows (fail-open).
func TestUnreadableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := t.TempDir()
	zero := "class Z {\n  @NiagaraProperty(name = \"coolSetpoint\", type = \"double\", defaultValue = \"BDouble.make(0)\",\n    flags = Flags.OPERATOR)\n  double c;\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "Z.java"), []byte(zero), 0o644); err != nil {
		t.Fatal(err)
	}
	u := filepath.Join(dir, "A.java")
	if err := os.WriteFile(u, []byte(zero), 0); err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	code := lint.Command(Spec())([]string{id, dir}, cli.Stdio{Out: &o, Err: &e})
	wantOut := "FAIL  lint-config-sanity  " + filepath.Join(dir, "Z.java") +
		":2  CS2: zero-default setpoint \"coolSetpoint\" with OPERATOR flag and no min>0 facet\n"
	wantErr := id + ": cannot scan " + u + ": awk: fatal: cannot open file `" + u + "' for reading: Permission denied\n"
	if code != 3 || o.String() != wantOut || e.String() != wantErr {
		t.Errorf("exit %d out %q err %q, want 3, %q and %q", code, o.String(), e.String(), wantOut, wantErr)
	}
}
