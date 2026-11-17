package mstpplan

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Main(args, cli.Stdio{Out: &out, Err: &errb})
	return code, out.String(), errb.String()
}

// The field case that was computed by hand three times: Spyders 8-23 at MAC 11-26 on a 7-switch DIP.
func TestPlanPrintsMacSwitchesAndDeviceId(t *testing.T) {
	code, out, errOut := run(t, "--first-mac", "11", "--count", "16", "--label-start", "8")
	if code != ExitOK || errOut != "" {
		t.Fatalf("exit %d stderr %q", code, errOut)
	}
	for _, row := range []string{
		"| Device 8 | 11 | 4, 2, 1 | 2011 |",
		"| Device 13 | 16 | 5 | 2016 |",
		"| Device 23 | 26 | 5, 4, 2 | 2026 |",
	} {
		if !strings.Contains(out, row) {
			t.Errorf("missing row %q in:\n%s", row, out)
		}
	}
	if !strings.HasPrefix(out, "| Label | MAC | Switches ON | Device ID |\n|---|---|---|---|\n") {
		t.Errorf("header:\n%s", out)
	}
}

func TestCsvFormatAndPrefix(t *testing.T) {
	code, out, _ := run(t, "--first-mac", "4", "--count", "2", "--format", "csv", "--label", "Spyder", "--id-base", "3000")
	want := "label,mac,switches_on,device_id\nSpyder 1,4,3,3004\nSpyder 2,5,3 1,3005\n"
	if code != ExitOK || out != want {
		t.Errorf("exit %d out %q, want %q", code, out, want)
	}
}

func TestReservedMacIsRefused(t *testing.T) {
	code, out, errOut := run(t, "--first-mac", "2", "--count", "3")
	if code != ExitConflict || out != "" || !strings.Contains(errOut, "MAC 2 is reserved (0-3: the JACE and spare masters)") {
		t.Errorf("exit %d out %q err %q", code, out, errOut)
	}
}

func TestMacAboveMaxMasterIsRefused(t *testing.T) {
	code, _, errOut := run(t, "--first-mac", "120", "--count", "5", "--max-master", "122")
	if code != ExitConflict || !strings.Contains(errOut, "MAC 123 is above maxMaster 122") {
		t.Errorf("exit %d err %q", code, errOut)
	}
}

func TestMacBeyondTheDipSwitchesIsRefused(t *testing.T) {
	code, _, errOut := run(t, "--first-mac", "14", "--count", "3", "--dip", "4")
	if code != ExitConflict || !strings.Contains(errOut, "MAC 16 needs more than 4 DIP switches (max 15)") {
		t.Errorf("exit %d err %q", code, errOut)
	}
}

func TestClashWithExistingDevicesIsRefused(t *testing.T) {
	f := filepath.Join(t.TempDir(), "existing.csv")
	if err := os.WriteFile(f, []byte("# existing devices\nmac,device_id\n4,2004\n12,2099\n50,2013\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errOut := run(t, "--first-mac", "11", "--count", "3", "--existing", f)
	if code != ExitConflict || !strings.Contains(errOut, "MAC 12 is already used by an existing device") ||
		!strings.Contains(errOut, "device ID 2013 is already used by an existing device") {
		t.Errorf("exit %d err %q", code, errOut)
	}
}

func TestUsageErrors(t *testing.T) {
	for _, args := range [][]string{{}, {"--count", "2"}, {"--first-mac", "x", "--count", "1"}, {"--first-mac", "5", "--count", "0"}, {"--first-mac", "5", "--count", "1", "--format", "xml"}, {"--bogus"}} {
		if code, _, errOut := run(t, args...); code != ExitUsage || !strings.Contains(errOut, "usage: mstp-plan") {
			t.Errorf("%v: exit %d err %q", args, code, errOut)
		}
	}
}
