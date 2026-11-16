package hostbudget

import (
	"strings"
	"testing"
)

// ugb is one decimal GB as the uint64 a Disk counts in.
const ugb = uint64(gb)

// The free-space floor (retro coordination-stretch T2). User rule, 2026-11-16: "el limite maximo es
// 40 GB libres". A 1 TB drive at 90% used still has 90 GB free; the percent rule refused it.

func TestDiskFloorDecidesInsteadOfThePercent(t *testing.T) {
	cases := []struct {
		name  string
		avail uint64 // GB free on a 1000 GB mount
		want  Status
	}{
		{"90 GB free on a 1 TB mount passes with the floor", 90, Pass},
		{"exactly 40 GB free passes", 40, Pass},
		{"39 GB free fails", 39, Fail},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := healthy(t)
			c.Mounts = []string{"/mnt/c"}
			c.DiskMinFree = 40 * ugb
			p := Probe{Statfs: func(path string) (Disk, error) {
				if path == "/mnt/c" {
					return Disk{Used: (1000 - tc.avail) * ugb, Avail: tc.avail * ugb}, nil
				}
				return Disk{Used: 1, Avail: 9}, nil
			}}
			rows, err := Check(c, p)
			if err != nil {
				t.Fatal(err)
			}
			r, ok := rowFor(rows, "disk:/mnt/c")
			if !ok || r.Status != tc.want {
				t.Errorf("disk row %q (present %v), want %v", r, ok, tc.want)
			}
		})
	}
}

func TestDiskFloorFailRowNamesTheFloorAndTheFreeBytes(t *testing.T) {
	code, out, _ := run(t, Disk{Used: 961 * ugb, Avail: 39 * ugb},
		append(fakeHost(t, "700", "48129"), "--disk-min-free", "40G")...)
	want := "FAIL disk:/mnt/c 961.0GB/1000.0GB (96% used, 39.0GB free = 39000000000 bytes, fail below 40.0GB free)"
	if code != ExitFail || !strings.Contains(out, want+"\n") {
		t.Errorf("exit %d, want 1 and the row\n%s\nout:\n%s", code, want, out)
	}
}

// The user's final rule of 2026-11-16 is a percent again: /mnt/c up to 95% used (scripts/ci-local.sh passes
// --disk-pct 95). The row names the percent, the free bytes and the limit.
func TestDiskPercentRuleUnchangedWithoutTheFloor(t *testing.T) {
	code, out, _ := run(t, Disk{Used: 910 * ugb, Avail: 90 * ugb}, fakeHost(t, "700", "48129")...)
	if code != ExitFail || !strings.Contains(out, "FAIL disk:/mnt/c 910.0GB/1000.0GB (91% used, 90.0GB free = 90000000000 bytes, fail above 90%)\n") {
		t.Errorf("exit %d; the percent rule must still fail 91%% used without a floor; out:\n%s", code, out)
	}
	code, out, _ = run(t, Disk{Used: 910 * ugb, Avail: 90 * ugb}, append(fakeHost(t, "700", "48129"), "--disk-pct", "95")...)
	if !strings.Contains(out, "WARN disk:/mnt/c 910.0GB/1000.0GB (91% used, 90.0GB free = 90000000000 bytes, fail above 95%)\n") {
		t.Errorf("--disk-pct 95: exit %d; 91%% used must not fail; out:\n%s", code, out)
	}
	code, out, _ = run(t, Disk{Used: 960 * ugb, Avail: 40 * ugb}, append(fakeHost(t, "700", "48129"), "--disk-pct", "95")...)
	if code != ExitFail || !strings.Contains(out, "FAIL disk:/mnt/c 960.0GB/1000.0GB (96% used, 40.0GB free = 40000000000 bytes, fail above 95%)\n") {
		t.Errorf("--disk-pct 95: exit %d; 96%% used must fail; out:\n%s", code, out)
	}
}

func TestParseSize(t *testing.T) {
	for in, want := range map[string]uint64{
		"40G": 40 * ugb, "40GB": 40 * ugb, "40g": 40 * ugb, "1T": 1000 * ugb, "500M": 500_000_000, "123": 123, "1.5G": 1_500_000_000,
	} {
		if got, err := parseSize(in); err != nil || got != want {
			t.Errorf("parseSize(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "G", "-1G", "40X", "4 0G"} {
		if _, err := parseSize(bad); err == nil {
			t.Errorf("parseSize(%q) accepted", bad)
		}
	}
	if code, _, errb := run(t, calm, append(fakeHost(t, "700", "48129"), "--disk-min-free", "lots")...); code != ExitEnv ||
		!strings.Contains(errb, "--disk-min-free") {
		t.Errorf("a bad size: exit %d, stderr %q; want 3 naming the flag", code, errb)
	}
}
