package niagaratest

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// homeKeptRe is the row a --keep-home run prints for the dir it kept.
var homeKeptRe = regexp.MustCompile(`(?m)cleanup: kept (\S+)$`)

// markerFile is the marker a run writes in the scratch dir it creates or takes.
const markerFile = "run-niagara-test.marker"

// resetCalls forgets the calls of an earlier run of the same fixture.
func (fx *fixture) resetCalls() {
	fx.f.mu.Lock()
	defer fx.f.mu.Unlock()
	fx.f.calls = nil
}

// keepRun is boot 1: a green --keep-home run; it returns the kept dir in its Windows and WSL forms.
func (fx *fixture) keepRun(t *testing.T) (win, wsl string) {
	t.Helper()
	code, out := fx.run(t, append([]string{"run"}, fx.args("--keep-home")...)...)
	m := homeKeptRe.FindStringSubmatch(out)
	if code != exitPass || m == nil {
		t.Fatalf("boot 1: exit %d, no kept row:\n%s", code, out)
	}
	fx.resetCalls()
	return m[1], fx.f.wsl(m[1])
}

// reuse is boot 2 on dir, with extra flags.
func (fx *fixture) reuse(t *testing.T, dir string, extra ...string) (int, string) {
	t.Helper()
	return fx.run(t, append([]string{"run"}, fx.args(append([]string{"--reuse-home", dir}, extra...)...)...)...)
}

// editMarker rewrites the marker line that starts with key.
func editMarker(t *testing.T, dir, key, value string) {
	t.Helper()
	p := filepath.Join(dir, markerFile)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("no marker in the kept dir: %v", err)
	}
	re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(key) + `: .*$`)
	if !re.Match(b) {
		t.Fatalf("marker has no %q line:\n%s", key, b)
	}
	write(t, p, re.ReplaceAllLiteralString(string(b), key+": "+value))
}

// copyMarker puts the kept run's marker into dst, as a hand copy would.
func copyMarker(t *testing.T, kept, dst string) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(kept, markerFile))
	if err != nil {
		t.Fatalf("no marker in the kept dir: %v", err)
	}
	write(t, filepath.Join(dst, markerFile), string(b))
}

// (a) A second boot on a kept home sees the state the first boot left there: in the scratch niagara_user_home
// and in the home copy. It copies nothing again, rebuilds the module jars, clears the old -output: dir, and
// still runs every gate (the preflight, the real user home snapshot).
func TestRunReuseHomeSeesPreviousBoot(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	var seen []string
	stale := false
	fx.f.on["test"] = func(ctx context.Context, c Cmd) (int, error) {
		nuh := fx.f.wsl(argValue(c.Env, "niagara_user_home="))
		data := filepath.Join(nuh, "stations", "S", "boots.txt")
		reg := filepath.Join(c.Dir, "registry", "boots.txt")
		a, _ := os.ReadFile(data)
		b, _ := os.ReadFile(reg)
		seen = append(seen, string(a)+"|"+string(b))
		write(t, data, string(a)+"boot\n")
		write(t, reg, string(b)+"boot\n")
		if _, err := os.Stat(filepath.Join(fx.f.wsl(argValue(c.Args, "-output:")), "testng-results.xml")); err == nil {
			stale = true
		}
		return fx.f.std(ctx, c)
	}
	win, wsl := fx.keepRun(t)
	code, out := fx.reuse(t, win)
	if code != exitPass {
		t.Fatalf("boot 2: exit %d\n%s", code, out)
	}
	if len(seen) != 2 || seen[1] != "boot\n|boot\n" {
		t.Errorf("boot 2 saw %q, want the state boot 1 left (station data and registry)", seen)
	}
	if stale {
		t.Error("boot 2 found boot 1's testng-results.xml: the old -output: dir was not cleared")
	}
	want := "temp profile preflight git-rev-parse git-status gradle test procs git-status"
	if got := strings.Join(fx.f.kinds(), " "); got != want {
		t.Errorf("boot 2 steps %q, want %q (no robocopy: the kept home is not copied again)", got, want)
	}
	if g := fx.f.of("gradle"); len(g) != 1 || argValue(g[0].Args, "-Pniagara_home=") != wsl+"/home" {
		t.Errorf("boot 2 did not rebuild against the kept home %s/home: %v", wsl, g)
	}
	for _, w := range []string{"reuse: " + win, "real user homes unchanged (5 entries checked)", "install unchanged ("} {
		if !strings.Contains(out, w) {
			t.Errorf("no %q in\n%s", w, out)
		}
	}
}

// (b)-(f) Every refusal is exit 2, before anything is built or run, and names its reason.
func TestRunReuseHomeRefusals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		setup func(t *testing.T, fx *fixture) string // returns the --reuse-home value
		want  string
	}{
		{"(b) outside the scratch parent", func(t *testing.T, fx *fixture) string {
			_, kept := fx.keepRun(t)
			dst := filepath.Join(fx.mnt, "c", "Elsewhere", "run-niagara-test-1")
			copyMarker(t, kept, dst)
			return `C:\Elsewhere\run-niagara-test-1`
		}, `is not under the scratch parent`},
		{"(b) the scratch parent itself", func(t *testing.T, fx *fixture) string {
			_, kept := fx.keepRun(t)
			copyMarker(t, kept, fx.temp)
			return tempWin
		}, `is not under the scratch parent`},
		{"(c) no marker", func(t *testing.T, fx *fixture) string {
			write(t, filepath.Join(fx.temp, "run-niagara-test-2", "home", "bin", "test.exe"), "x")
			return tempWin + `\run-niagara-test-2`
		}, `has no ` + markerFile},
		{"(d) run id not the kept run's log", func(t *testing.T, fx *fixture) string {
			win, kept := fx.keepRun(t)
			b, _ := os.ReadFile(filepath.Join(kept, markerFile))
			id := regexp.MustCompile(`(?m)^run id: (\S+)-[0-9a-f]{8}$`).FindStringSubmatch(string(b))
			if id == nil {
				t.Fatalf("marker run id is not <stamp>-<module>-<dir>-<nonce>:\n%s", b)
			}
			editMarker(t, kept, "run id", id[1]+"-00000000")
			return win
		}, `marker run id`},
		{"(e) install path", func(t *testing.T, fx *fixture) string {
			win, kept := fx.keepRun(t)
			editMarker(t, kept, "install", `C:\Niagara\Other`)
			return win
		}, `marker install`},
		{"(e) system.properties sha256", func(t *testing.T, fx *fixture) string {
			win, _ := fx.keepRun(t)
			write(t, filepath.Join(fx.install, "defaults", "system.properties"), "niagara.x=1\n")
			return win
		}, `marker system.properties sha256`},
		{"(f) D3: a real user home holding a copied marker", func(t *testing.T, fx *fixture) string {
			// %USERPROFILE% under %TEMP%, so the real user home lies under the scratch parent.
			prof := tempWin + `\prof`
			fx.f.on["profile"] = func(_ context.Context, c Cmd) (int, error) {
				_, err := c.Stdout.Write([]byte(prof + "\r\n"))
				return 0, err
			}
			home := filepath.Join(fx.temp, "prof", "Niagara4.15", "brand")
			for _, p := range []string{"registry/registry.db", "security/keystore"} {
				write(t, filepath.Join(home, p), "x")
			}
			_, kept := fx.keepRun(t)
			copyMarker(t, kept, home)
			return prof + `\Niagara4.15\brand`
		}, `is a real Niagara user home`},
		{"(f) D3: a copied user home holding a copied marker", func(t *testing.T, fx *fixture) string {
			_, kept := fx.keepRun(t)
			dst := filepath.Join(fx.temp, "copied-home")
			for _, p := range []string{"registry/registry.db", "security/keystore"} {
				write(t, filepath.Join(dst, p), "x")
			}
			copyMarker(t, kept, dst)
			return tempWin + `\copied-home`
		}, `is a real Niagara user home`},
		{"marker log empty: boot 1 did not finish with --keep-home", func(t *testing.T, fx *fixture) string {
			win, kept := fx.keepRun(t)
			editMarker(t, kept, "log", "")
			return win
		}, `marker log`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			dir := tc.setup(t, fx)
			code, out := fx.reuse(t, dir)
			if code != exitUsage || !strings.Contains(out, tc.want) {
				t.Fatalf("exit %d, want 2 with %q:\n%s", code, tc.want, out)
			}
			for _, k := range []string{"gradle", "test", "robocopy"} {
				if n := len(fx.f.of(k)); n != 0 {
					t.Errorf("a refused reuse ran %d %s call(s)", n, k)
				}
			}
		})
	}
}

// --reuse-home takes no copy flags: the kept home keeps the copy, exclusions and verification boot 1 made.
func TestRunReuseHomeRefusesCopyFlags(t *testing.T) {
	t.Parallel()
	for _, extra := range [][]string{{"--exclude-jar", "moduleA-*.jar"}, {"--module-verification", "low"}} {
		fx := newFixture(t)
		code, out := fx.reuse(t, tempWin+`\run-niagara-test-3`, extra...)
		if code != exitUsage || !strings.Contains(out, "--reuse-home takes no "+extra[0]) {
			t.Errorf("%v: exit %d:\n%s", extra, code, out)
		}
	}
}

// (g) Boot 2 deletes the kept home unless --keep-home is given again, and says which in a row.
func TestRunReuseHomeCleanup(t *testing.T) {
	t.Parallel()
	t.Run("deleted", func(t *testing.T) {
		t.Parallel()
		fx := newFixture(t)
		win, wsl := fx.keepRun(t)
		code, out := fx.reuse(t, win)
		if code != exitPass || !strings.Contains(out, "PASS  run-niagara-test  cleanup: deleted "+win) {
			t.Errorf("exit %d, no deleted row:\n%s", code, out)
		}
		if _, err := os.Stat(wsl); !os.IsNotExist(err) {
			t.Errorf("the kept home is still there: %v", err)
		}
	})
	t.Run("kept again, and reusable a third time", func(t *testing.T) {
		t.Parallel()
		fx := newFixture(t)
		win, wsl := fx.keepRun(t)
		before, _ := os.ReadFile(filepath.Join(wsl, markerFile))
		code, out := fx.reuse(t, win, "--keep-home")
		if code != exitPass || !strings.Contains(out, "PASS  run-niagara-test  cleanup: kept "+win) {
			t.Fatalf("exit %d, no kept row:\n%s", code, out)
		}
		after, _ := os.ReadFile(filepath.Join(wsl, markerFile))
		if string(after) == string(before) {
			t.Errorf("boot 2 did not rebind the marker to its own run:\n%s", after)
		}
		fx.resetCalls()
		if code, out := fx.reuse(t, win); code != exitPass {
			t.Errorf("boot 3: exit %d\n%s", code, out)
		}
	})
}

// A reuse run that dies before its log is written leaves a home no later run may take: the marker is rebound
// to the new run, its log empty, before anything is built.
func TestRunReuseHomeRebindsMarkerFirst(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	win, wsl := fx.keepRun(t)
	checked := false
	fx.f.on["gradle"] = func(context.Context, Cmd) (int, error) {
		b, _ := os.ReadFile(filepath.Join(wsl, markerFile))
		if !regexp.MustCompile(`(?m)^log: $`).Match(b) {
			t.Errorf("marker during boot 2 still names boot 1's log:\n%s", b)
		}
		checked = true
		return 0, nil
	}
	if code, out := fx.reuse(t, win); code != exitPass || !checked {
		t.Fatalf("boot 2: exit %d, marker checked %v:\n%s", code, checked, out)
	}
}
