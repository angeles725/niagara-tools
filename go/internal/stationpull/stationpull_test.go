package stationpull

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const secret = "s3cr3t!pw"

// env writes a mode-0600 credential file; home is a Niagara home whose bin/plat is script (when not "").
func fixture(t *testing.T, script string) (home, env string) {
	t.Helper()
	home = t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if script != "" {
		if err := os.WriteFile(filepath.Join(home, "bin", "plat"), []byte("#!/bin/sh\n"+script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	env = filepath.Join(t.TempDir(), "plat.env")
	body := "NIAGARA_PLAT_HOST=localhost\nNIAGARA_PLAT_USER=admin\nNIAGARA_PLAT_PASSWORD='" + secret + "'\n"
	if err := os.WriteFile(env, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return home, env
}

func run(t *testing.T, home string, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	vars := map[string]string{"NIAGARA_HOME": home}
	code := Main(args, cli.Stdio{Out: &out, Err: &errb, Getenv: func(k string) string { return vars[k] }})
	return code, out.String(), errb.String()
}

func TestANonEmptyOrNonDirectoryDestIsRefusedBeforePlatRuns(t *testing.T) {
	home, env := fixture(t, "echo ran >> \"$0.log\"\n")
	full := t.TempDir()
	if err := os.WriteFile(filepath.Join(full, "keep.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "afile")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, dest := range []string{full, file} {
		code, out, errs := run(t, home, "--env", env, "--station", "LAB", "--dest", dest)
		if code != 3 || out != "" || !strings.Contains(errs, "--dest") {
			t.Errorf("%s: exit %d out %q err %q", dest, code, out, errs)
		}
	}
	if _, err := os.Stat(filepath.Join(home, "bin", "plat.log")); err == nil {
		t.Error("plat ran although the destination was refused")
	}
	if b, _ := os.ReadFile(filepath.Join(full, "keep.txt")); string(b) != "x" {
		t.Error("the non-empty destination was changed")
	}
}

func TestAStationNameOutsideTheAllowedSetIsRefused(t *testing.T) {
	home, env := fixture(t, "exit 0\n")
	for _, name := range []string{"a b", "a/b", "../x", "a;b", "", "$(x)"} {
		code, _, errs := run(t, home, "--env", env, "--station", name, "--dest", t.TempDir())
		if code != 3 || !strings.Contains(errs, "station") {
			t.Errorf("%q: exit %d %q", name, code, errs)
		}
	}
}

func TestUsageErrors(t *testing.T) {
	home, env := fixture(t, "exit 0\n")
	for _, args := range [][]string{
		{"--env", env, "--dest", t.TempDir()},
		{"--env", env, "--station", "LAB"},
		{"--env", env, "--station", "LAB", "--dest", t.TempDir(), "--bogus"},
		{"--env", env, "--station", "LAB", "--dest", t.TempDir(), "--exclude-files", "[a"},
	} {
		if code, _, errs := run(t, home, args...); code != 3 || !strings.Contains(errs, "usage: station-pull") {
			t.Errorf("%v: exit %d %q", args, code, errs)
		}
	}
}

func TestDryRunPrintsTheTwoPlatCallsMaskedAndRunsNothing(t *testing.T) {
	home, env := fixture(t, "echo ran >> \"$0.log\"\n")
	dest := t.TempDir()
	code, out, errs := run(t, home, "--env", env, "--dry-run", "--station", "LAB", "--dest", dest)
	if code != 0 || errs != "" || strings.Contains(out, secret) {
		t.Fatalf("exit %d err %q out %q", code, errs, out)
	}
	conn := "-h:ip:localhost -p:5011 -secure -noinput -usr:admin -pwd:<masked>"
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 ||
		lines[0] != "plat fget -recurse -overwrite:none "+conn+" ~stations/LAB "+filepath.Join(dest, "station") ||
		lines[1] != "plat flist -recurse "+conn+" ~stations/LAB" {
		t.Fatalf("lines:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(home, "bin", "plat.log")); err == nil {
		t.Error("plat ran under --dry-run")
	}
}
