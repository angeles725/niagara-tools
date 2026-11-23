package labstation

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// fakeStation behaves as the 4.14 station console did on a lab unit (research: the UMBRELLA_LAB probe of
// 2026-11-23): a boot line, the Station Started line, the niagara> prompt, a few verbs, and quit's shutdown lines.
const fakeStation = `#!/usr/bin/env bash
echo "boot $1"
[ "$1" = BAD ] && { echo "SEVERE [sys] Cannot boot station"; exit 1; }
echo "INFO [sys] *** Station Started (5ms) [9ms total] ***"
printf 'niagara>'
while IFS= read -r l; do l=${l%$'\r'}
  case $l in
  version) printf '\nNiagara Runtime Environment\n  java.version: 1.8\n\nniagara>' ;;
  quit) echo "INFO [sys] Station stopped"; echo "INFO [sys] *** Station shutdown ***"; exit 0 ;;
  *) printf '\nniagara>' ;;
  esac
done
`

func fakeHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "bin", "station"), []byte(fakeStation), 0o755); err != nil {
		t.Fatal(err)
	}
	return home
}

func run(home, state string, args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	env := map[string]string{"NIAGARA_HOME": home, "LAB_STATION_STATE": state}
	code := Main(args, cli.Stdio{Out: &out, Err: &errb, Getenv: func(k string) string { return env[k] }})
	return code, out.String(), errb.String()
}

var pidRe = regexp.MustCompile(`\(pid \d+\)`)

func TestStartConsoleStopRoundTrip(t *testing.T) {
	home, state := fakeHome(t), t.TempDir()
	code, out, errs := run(home, state, "start", "LAB")
	if code != 0 || pidRe.ReplaceAllString(out, "(pid N)") !=
		"lab-station: LAB started (pid N): INFO [sys] *** Station Started (5ms) [9ms total] ***\n" {
		t.Fatalf("start: exit %d, out %q, err %q", code, out, errs)
	}
	if code, _, errs := run(home, state, "start", "LAB"); code != 1 || !strings.HasPrefix(errs, "lab-station: LAB is already running (pid ") {
		t.Errorf("a second start: exit %d, err %q", code, errs)
	}
	code, out, _ = run(home, state, "status", "LAB")
	if code != 0 || pidRe.ReplaceAllString(out, "(pid N)") != "lab-station: LAB running (pid N)\n" {
		t.Errorf("status: exit %d, out %q", code, out)
	}
	code, out, errs = run(home, state, "console", "LAB", "version")
	if code != 0 || out != "Niagara Runtime Environment\n  java.version: 1.8\n" {
		t.Errorf("console version: exit %d, out %q, err %q", code, out, errs)
	}
	code, out, errs = run(home, state, "stop", "LAB")
	if code != 0 || out != "lab-station: LAB stopped: INFO [sys] *** Station shutdown ***\n" {
		t.Fatalf("stop: exit %d, out %q, err %q", code, out, errs)
	}
	if code, out, _ := run(home, state, "status", "LAB"); code != 1 || out != "lab-station: LAB stopped\n" {
		t.Errorf("status after stop: exit %d, out %q", code, out)
	}
}

func TestStartThatCannotBootFails(t *testing.T) {
	home, state := fakeHome(t), t.TempDir()
	code, _, errs := run(home, state, "start", "BAD")
	if code != 1 || !strings.Contains(errs, "lab-station: BAD did not start: SEVERE [sys] Cannot boot station") {
		t.Errorf("exit %d, err %q", code, errs)
	}
}

func TestStopWithoutAConsolePipeIsRefused(t *testing.T) {
	code, _, errs := run(fakeHome(t), t.TempDir(), "stop", "LAB")
	want := "lab-station: LAB is not running under lab-station (no console pipe): stop it another way\n"
	if code != 1 || errs != want {
		t.Errorf("exit %d, err %q", code, errs)
	}
}

func TestConsoleRefusesStateChangingVerbs(t *testing.T) {
	home, state := fakeHome(t), t.TempDir()
	for _, v := range []string{"kill", "quit", "tell"} {
		code, _, errs := run(home, state, "console", "LAB", v)
		if code != 3 || !strings.Contains(errs, "lab-station: console verb "+v+" is refused (allowed: gc heap help threads version; stop with lab-station stop)") {
			t.Errorf("%s: exit %d, err %q", v, code, errs)
		}
	}
}

func TestDryRunTouchesNothing(t *testing.T) {
	home, state := fakeHome(t), t.TempDir()
	code, out, _ := run(home, state, "restart", "--dry-run", "LAB")
	exe := filepath.Join(home, "bin", "station")
	want := "lab-station: dry run, nothing done\n" +
		"  stop: write quit to " + filepath.Join(state, "LAB", "in") + ", wait for *** Station shutdown ***\n" +
		"  start: " + exe + " LAB, stdin " + filepath.Join(state, "LAB", "in") + ", log " +
		filepath.Join(state, "LAB", "out.log") + ", wait for *** Station Started\n"
	if code != 0 || out != want {
		t.Errorf("exit %d, out %q\nwant %q", code, out, want)
	}
	if es, _ := os.ReadDir(state); len(es) != 0 {
		t.Errorf("the dry run wrote %d entries in the state dir", len(es))
	}
}

func TestUsage(t *testing.T) {
	for _, a := range [][]string{{}, {"start"}, {"frob", "LAB"}, {"start", "../x"}} {
		if code, _, errs := run(fakeHome(t), t.TempDir(), a...); code != 3 || !strings.Contains(errs, usage) {
			t.Errorf("%v: exit %d, err %q", a, code, errs)
		}
	}
}
