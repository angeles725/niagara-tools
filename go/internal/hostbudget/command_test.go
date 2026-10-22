package hostbudget

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// run executes Main against a fake host: cgroup and proc fixtures under dir, a fixed statfs.
// Not parallel: it swaps the package-level statfs seam.
func run(t *testing.T, disk Disk, args ...string) (int, string, string) {
	t.Helper()
	saved := statfs
	statfs = func(string) (Disk, error) { return disk, nil }
	t.Cleanup(func() { statfs = saved })
	var out, errb bytes.Buffer
	code := Main(args, cli.Stdio{Out: &out, Err: &errb, Getenv: func(string) string { return "" }})
	return code, out.String(), errb.String()
}

// fakeHost returns the common flags for a host whose pids cgroup is cur/max.
func fakeHost(t *testing.T, cur, max string) []string {
	t.Helper()
	dir := t.TempDir()
	cgroup(t, filepath.Join(dir, "cg"), "init.scope", cur, max)
	cgroup(t, filepath.Join(dir, "cg"), "system.slice", "16", "max")
	proc(t, filepath.Join(dir, "proc"), "320861", "900")
	return []string{"--cgroup-root", filepath.Join(dir, "cg"), "--proc-root", filepath.Join(dir, "proc"),
		"--gocache", filepath.Join(dir, "gocache"), "--mount", "/mnt/c"}
}

var calm = Disk{Used: 10, Avail: 90}

func TestMainCleanHostExits0(t *testing.T) {
	code, out, _ := run(t, calm, fakeHost(t, "700", "48129")...)
	if code != ExitOK {
		t.Fatalf("exit %d, want 0; out:\n%s", code, out)
	}
	for _, want := range []string{"PASS pids:init.scope 700/48129", "PASS threads ", "PASS gocache ", "PASS disk:/ ", "PASS disk:/mnt/c "} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "system.slice") {
		t.Errorf("an unlimited cgroup was reported:\n%s", out)
	}
}

func TestMainFailExits1(t *testing.T) {
	code, out, _ := run(t, calm, fakeHost(t, "30000", "48129")...)
	if code != ExitFail || !strings.Contains(out, "FAIL pids:init.scope 30000/48129") {
		t.Fatalf("exit %d, want 1 with the FAIL row; out:\n%s", code, out)
	}
}

func TestMainWarnOnlyExits0UnlessStrict(t *testing.T) {
	args := fakeHost(t, "24064", "48128")
	if code, out, _ := run(t, calm, args...); code != ExitOK || !strings.Contains(out, "WARN pids:init.scope 24064/48128") {
		t.Fatalf("exit %d, want 0 with a WARN row; out:\n%s", code, out)
	}
	// Mutation: HB-strict -- ignoring --strict lets a WARN-only host exit 0.
	if code, _, _ := run(t, calm, append(args, "--strict")...); code != ExitFail {
		t.Errorf("--strict exit %d, want 1", code)
	}
}

func TestMainDiskFailExits1(t *testing.T) {
	code, out, _ := run(t, Disk{Used: 95, Avail: 5}, fakeHost(t, "700", "48129")...)
	if code != ExitFail || !strings.Contains(out, "FAIL disk:/mnt/c ") {
		t.Fatalf("exit %d, want 1 with a disk FAIL row; out:\n%s", code, out)
	}
}

func TestMainThresholdFlags(t *testing.T) {
	args := append(fakeHost(t, "30000", "48129"), "--pids-pct", "70")
	if code, out, _ := run(t, calm, args...); code != ExitOK {
		t.Errorf("--pids-pct 70: exit %d, want 0 (62%% is under 70%%); out:\n%s", code, out)
	}
	args = append(fakeHost(t, "700", "48129"), "--disk-pct", "5")
	if code, _, _ := run(t, calm, args...); code != ExitFail {
		t.Errorf("--disk-pct 5: exit %d, want 1 (10%% is above 5%%)", code)
	}
}

func TestMainUnreadableRootExits3(t *testing.T) {
	args := fakeHost(t, "700", "48129")
	args[1] = filepath.Join(args[1], "missing")
	code, out, errs := run(t, calm, args...)
	if code != ExitEnv || !strings.Contains(errs, "host-budget:") {
		t.Fatalf("exit %d, want 3 with a message; out %q err %q", code, out, errs)
	}
}

func TestMainUsageExits3(t *testing.T) {
	for _, args := range [][]string{
		{"--bogus"},
		{"positional"},
		{"--pids-pct", "101"},
		{"--disk-pct", "-1"},
		{"--gocache-gb", "-2"},
		{"--mount", ""},
	} {
		if code, _, _ := run(t, calm, args...); code != ExitEnv {
			t.Errorf("Main(%q) exit %d, want 3", args, code)
		}
	}
}

func TestResolveGocacheOrder(t *testing.T) {
	saved := goEnvGOCACHE
	t.Cleanup(func() { goEnvGOCACHE = saved })
	env := cli.Stdio{Getenv: func(n string) string {
		if n == "GOCACHE" {
			return "/from/env"
		}
		return ""
	}}
	goEnvGOCACHE = func() (string, error) { return "/from/go-env", nil }
	if got := resolveGocache(env); got != "/from/go-env" {
		t.Errorf("with go env: %q, want /from/go-env", got)
	}
	goEnvGOCACHE = func() (string, error) { return "", errors.New("no go") }
	if got := resolveGocache(env); got != "/from/env" {
		t.Errorf("without go: %q, want /from/env", got)
	}
}
