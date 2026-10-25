package niagaratest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var keptRe = regexp.MustCompile(`(?m)^FAIL  run-niagara-test  diagnostics kept: (\S+)$`)

// logName is <UTC timestamp>-<module>.log.
var logName = regexp.MustCompile(`^[0-9]{8}T[0-9]{6}Z-ModuleA(-[0-9]+)?\.log$`)

// keptLog is the run log a FAIL row names: under the fixture's state dir, named and moded as the contract says.
func (fx *fixture) keptLog(t *testing.T, out string) string {
	t.Helper()
	m := keptRe.FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("no diagnostics kept row in\n%s", out)
	}
	dir := filepath.Join(fx.state, "run-niagara-test")
	if filepath.Dir(m[1]) != dir || !logName.MatchString(filepath.Base(m[1])) {
		t.Errorf("log %q is not <UTC timestamp>-ModuleA.log under %s", m[1], dir)
	}
	return readLogFile(t, m[1])
}

func readLogFile(t *testing.T, p string) string {
	t.Helper()
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("log mode %v, want 0600", info.Mode().Perm())
	}
	if d, err := os.Stat(filepath.Dir(p)); err != nil || d.Mode().Perm() != 0o700 {
		t.Errorf("log dir mode %v (%v), want 0700", d.Mode().Perm(), err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// numbered is n lines "<tag> <i>" (1-based), CRLF-terminated as test.exe writes them.
func numbered(tag string, n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "%s %d\r\n", tag, i)
	}
	return b.String()
}

// A rejected run (no testng-results.xml) keeps its evidence: the rows, the stderr and stdout tails, the
// -output: listing and the exit status, all with the scratch paths redacted; the scratch dir is still deleted.
func TestRunFailKeepsDiagnostics(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	fx.f.on["test"] = func(_ context.Context, c Cmd) (int, error) {
		outDir := f2wsl(fx, argValue(c.Args, "-output:"))
		write(t, filepath.Join(outDir, "junitreports", "TEST-a.xml"), "<x/>")
		fmt.Fprint(c.Stderr, numbered("err", 300))
		fmt.Fprintf(c.Stderr, "Cannot load %s\\modules\\moduleA-rt.jar\r\n", argValue(c.Env, "niagara_user_home="))
		fmt.Fprint(c.Stderr, "invalid test target(s): ModuleA\r\n")
		fmt.Fprint(c.Stdout, numbered("out", 120))
		fx.f.useNUH(c)
		return 0, nil
	}
	code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
	if code != exitFail {
		t.Fatalf("exit %d, want 1:\n%s", code, out)
	}
	log := fx.keptLog(t, out)
	for _, w := range []string{
		"FAIL  run-niagara-test  testng: ",          // a runner row
		"PASS  run-niagara-test  install unchanged", // a cleanup row
		"err 300\n", "err 103\n", "invalid test target(s): ModuleA\n",
		"Cannot load <scratch>\\nuh\\modules\\moduleA-rt.jar\n",
		"out 120\n", "out 73\n",
		"junitreports/TEST-a.xml  4 bytes",
		"exit: 1 (FAIL)",
	} {
		if !strings.Contains(log, w) {
			t.Errorf("log lacks %q:\n%s", w, log)
		}
	}
	for _, nw := range []string{"err 102\n", "out 70\n", "run-niagara-test-", "\r"} {
		if strings.Contains(log, nw) {
			t.Errorf("log holds %q (want only the last 200 stderr / 50 stdout lines, redacted, no CR):\n%s", nw, log)
		}
	}
	if l := fx.leftovers(t); len(l) != 0 {
		t.Errorf("left in %%TEMP%%: %q", l)
	}
}

func f2wsl(fx *fixture, win string) string { return fx.f.wsl(win) }

// Every failing exit keeps a log: 3 (build failed) and 4 (timeout) as well as 1.
func TestRunFailExitsKeepDiagnostics(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		setup  func(fx *fixture)
		extra  []string
		code   int
		status string
	}{
		{"build failed", func(fx *fixture) {
			fx.f.on["gradle"] = func(_ context.Context, c Cmd) (int, error) { fmt.Fprint(c.Stdout, "BUILD FAILED\n"); return 1, nil }
		}, nil, exitEnv, "exit: 3 (environment)"},
		{"timeout", func(fx *fixture) {
			fx.f.on["test"] = func(ctx context.Context, _ Cmd) (int, error) { <-ctx.Done(); return -1, ctx.Err() }
		}, []string{"--timeout", "200ms"}, exitTimeout, "exit: 4 (timeout)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			tc.setup(fx)
			code, out := fx.run(t, append([]string{"run"}, fx.args(tc.extra...)...)...)
			if code != tc.code {
				t.Fatalf("exit %d, want %d:\n%s", code, tc.code, out)
			}
			if log := fx.keptLog(t, out); !strings.Contains(log, tc.status) {
				t.Errorf("log lacks %q:\n%s", tc.status, log)
			}
		})
	}
}

// A green run writes no log by default; an explicit --log-dir writes one anyway; --keep-home writes one too.
func TestRunPassLog(t *testing.T) {
	t.Parallel()
	t.Run("default: no log", func(t *testing.T) {
		t.Parallel()
		fx := newFixture(t)
		code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
		if code != exitPass || strings.Contains(out, "diagnostics") {
			t.Errorf("exit %d:\n%s", code, out)
		}
		if _, err := os.Stat(filepath.Join(fx.state, "run-niagara-test")); !os.IsNotExist(err) {
			t.Errorf("a green run made the log dir: %v", err)
		}
	})
	t.Run("--log-dir", func(t *testing.T) {
		t.Parallel()
		fx := newFixture(t)
		dir := filepath.Join(t.TempDir(), "logs")
		code, out := fx.run(t, append([]string{"run"}, fx.args("--log-dir", dir)...)...)
		m := regexp.MustCompile(`(?m)^PASS  run-niagara-test  run log: (\S+)$`).FindStringSubmatch(out)
		if code != exitPass || m == nil || filepath.Dir(m[1]) != dir {
			t.Fatalf("exit %d, no run log row under %s:\n%s", code, dir, out)
		}
		if log := readLogFile(t, m[1]); !strings.Contains(log, "exit: 0 (PASS)") || !strings.Contains(log, "verdict: 1 test run") {
			t.Errorf("log:\n%s", log)
		}
	})
	t.Run("--keep-home", func(t *testing.T) {
		t.Parallel()
		fx := newFixture(t)
		code, out := fx.run(t, append([]string{"run"}, fx.args("--keep-home")...)...)
		if code != exitPass || len(fx.leftovers(t)) != 1 || !strings.Contains(out, "PASS  run-niagara-test  run log: "+fx.state) {
			t.Errorf("exit %d, left %q:\n%s", code, fx.leftovers(t), out)
		}
	})
}

// A huge single stderr line is cut at the per-line cap: the log stays bounded.
func TestRunDiagnosticsBoundLongLine(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	fx.f.on["test"] = func(_ context.Context, c Cmd) (int, error) {
		fmt.Fprint(c.Stderr, strings.Repeat("x", 4<<20)+"\nlast line\n")
		fx.f.useNUH(c)
		return 0, nil
	}
	code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
	if code != exitFail {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	log := fx.keptLog(t, out)
	if len(log) > 256<<10 || !strings.Contains(log, "last line") || !strings.Contains(log, "[line cut at") {
		t.Errorf("log of %d bytes, want the long line cut and the last line kept", len(log))
	}
}

// Redaction: the scratch root and the home copy in Windows, forward-slash and /mnt forms, any case; the user
// profile and the WSL home; never the install, never a longer name sharing a prefix.
func TestRedactMixedPathForms(t *testing.T) {
	t.Parallel()
	r := newRedactor(
		redaction{`C:\Users\you\AppData\Local\Temp\run-niagara-test-123\home`, "/mnt/c/Users/you/AppData/Local/Temp/run-niagara-test-123/home", "<copy>"},
		redaction{`C:\Users\you\AppData\Local\Temp\run-niagara-test-123`, "/mnt/c/Users/you/AppData/Local/Temp/run-niagara-test-123", "<scratch>"},
		redaction{`C:\Users\you`, "/mnt/c/Users/you", "<you>"},
		redaction{"", "/home/dev", "<you>"},
	)
	in := strings.Join([]string{
		`load C:\Users\you\AppData\Local\Temp\run-niagara-test-123\home\modules\a.jar`,
		`c:\users\YOU\appdata\local\temp\RUN-NIAGARA-TEST-123\out\testng`,
		`C:/Users/you/AppData/Local/Temp/run-niagara-test-123/nuh/etc`,
		`/mnt/c/Users/you/AppData/Local/Temp/run-niagara-test-123/home/bin/test.exe`,
		`/MNT/C/users/you/AppData/Local/Temp/run-niagara-test-123`,
		`profile C:\Users\you\Niagara4.15\brand; wsl /home/dev/.gradle`,
		`other C:\Users\young\x and /home/developer and install C:\Niagara\N4\bin`,
	}, "\n")
	want := strings.Join([]string{
		`load <copy>\modules\a.jar`,
		`<scratch>\out\testng`,
		`<scratch>/nuh/etc`,
		`<copy>/bin/test.exe`,
		`<scratch>`,
		`profile <you>\Niagara4.15\brand; wsl <you>/.gradle`,
		`other C:\Users\young\x and /home/developer and install C:\Niagara\N4\bin`,
	}, "\n")
	if got := r.apply(in); got != want {
		t.Errorf("redacted\n%s\nwant\n%s", got, want)
	}
}
