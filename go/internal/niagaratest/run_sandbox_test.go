package niagaratest

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func envValue(env []string, prefix string) string { return argValue(env, prefix) }

// D1: a client gradle.properties may name a Windows JDK; the build always gets a WSL JDK 8.
func TestRunBuildUsesWSLJDK8(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	code, out := fx.run(t, append([]string{"run"}, fx.args("--jdk8", fx.jdk)...)...)
	if code != exitPass {
		t.Fatalf("exit %d\n%s", code, out)
	}
	g := fx.f.of("gradle")[0]
	if got := argValue(g.Args, "-Porg.gradle.java.installations.paths="); got != fx.jdk {
		t.Errorf("installations.paths %q, want %q", got, fx.jdk)
	}
	if got := envValue(g.Env, "JAVA_HOME="); got != fx.jdk {
		t.Errorf("JAVA_HOME %q, want %q", got, fx.jdk)
	}
	code, out = fx.run(t, append([]string{"run"}, fx.args("--jdk8", fx.gradle)...)...)
	if want := "--jdk8 " + fx.gradle + " has no bin/javac"; code != exitUsage || !strings.Contains(out, want) {
		t.Errorf("exit %d, want 2 with %q:\n%s", code, want, out)
	}
}

// D2: gradle keeps the caller's distribution cache; only user.home (the signing profile) is scratch.
func TestRunBuildKeepsGradleUserHome(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		extra []string
		want  string
	}{{"default", nil, "/nonexistent-home/.gradle"}, {"flag", []string{"--gradle-user-home", "/srv/gradle"}, "/srv/gradle"}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			code, out := fx.run(t, append([]string{"run"}, fx.args(tc.extra...)...)...)
			if code != exitPass {
				t.Fatalf("exit %d\n%s", code, out)
			}
			g := fx.f.of("gradle")[0]
			if got := envValue(g.Env, "GRADLE_USER_HOME="); got != tc.want {
				t.Errorf("GRADLE_USER_HOME %q, want %q (env %q)", got, tc.want, g.Env)
			}
		})
	}
}

// D3: test.exe may ignore the scratch niagara_user_home; a write to a real user home is a FAIL.
func TestRunRealUserHomeChangedIsFail(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	fx.f.on["test"] = func(_ context.Context, c Cmd) (int, error) {
		fx.f.testExe(c, greenXML, greenLog)
		fx.f.useNUH(c)
		write(t, filepath.Join(fx.userHome, "security", "signing", "signers", "s.pem"), "x")
		return 0, nil
	}
	code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
	want := `FAIL  run-niagara-test  real user home changed during the run: added ` + profWin + `\Niagara4.15\brand\security\signing`
	if code != exitFail || !strings.Contains(out, want) {
		t.Errorf("exit %d, want 1 with %q:\n%s", code, want, out)
	}
}

// The scratch home is used only when the NRE created its etc\ and security\ there; a build's leftovers are not enough.
func TestRunScratchUserHomeUnusedIsFail(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, leftover string }{{"empty", ""}, {"no security", "etc/nre.properties"}, {"other files only", "registry/registry.db"}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			fx.f.on["test"] = func(_ context.Context, c Cmd) (int, error) {
				fx.f.testExe(c, greenXML, greenLog)
				if tc.leftover != "" {
					write(t, filepath.Join(fx.f.wsl(argValue(c.Env, "niagara_user_home=")), tc.leftover), "x")
				}
				return 0, nil
			}
			code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
			if want := "FAIL  run-niagara-test  scratch niagara_user_home was not used (no etc/ and security/ in it)"; code != exitFail || !strings.Contains(out, want) {
				t.Errorf("exit %d, want 1 with %q:\n%s", code, want, out)
			}
		})
	}
}

func TestRunUserProfileUnreadableIsEnv(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	fx.f.on["profile"] = func(context.Context, Cmd) (int, error) { return 1, nil }
	code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
	if code != exitEnv || !strings.Contains(out, "cannot read %USERPROFILE%") || len(fx.f.of("robocopy")) != 0 {
		t.Errorf("exit %d, want 3 before any copy:\n%s", code, out)
	}
}

// D4: a non-elevated probe reads no path or command line of a service's station.exe; it is attributed
// through the niagarad service that started it, and refused when it cannot be.
func TestRunPreflightAttributesServiceStations(t *testing.T) {
	t.Parallel()
	const unread = "UNREAD\t100\t50\tstation.exe\r\n"
	svc := func(path string) string { return "SVC\t50\tNiagara\t" + path + "\r\n" }
	for _, tc := range []struct {
		name, rows string
		code       int
		want       string
	}{
		{"another install's service", unread + svc(`"C:\Other\N4\bin\niagarad.exe" -s`), exitPass,
			`PASS  run-niagara-test  preflight: station.exe 100 belongs to C:\Other\N4 (service Niagara, pid 50)`},
		{"the install's service", unread + svc(`"C:\Niagara\N4\bin\niagarad.exe"`), exitUsage,
			`a running process uses the install: station.exe 100 (service Niagara, pid 50)`},
		{"a service containing the install", unread + svc(`C:\Niagara\bin\niagarad.exe`), exitUsage,
			`a running process uses the install: station.exe 100 (service Niagara, pid 50)`},
		{"the service itself unreadable", "UNREAD\t50\t4\tniagarad.exe\r\n" + svc(`"C:\Niagara\N4\bin\niagarad.exe"`), exitUsage,
			`a running process uses the install: niagarad.exe 50 (service Niagara, pid 50)`},
		{"no service", unread, exitUsage, "cannot attribute running station process 100 to an install"},
		{"a service path that names no niagara_home", unread + svc(`C:\Windows\svchost.exe`), exitUsage,
			"cannot attribute running station process 100 to an install"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			fx.f.on["preflight"] = func(_ context.Context, c Cmd) (int, error) {
				fmt.Fprint(c.Stdout, tc.rows+"DONE\r\n")
				return 0, nil
			}
			code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
			if code != tc.code || !strings.Contains(out, tc.want) {
				t.Errorf("exit %d, want %d with %q:\n%s", code, tc.code, tc.want, out)
			}
		})
	}
}

func TestInUseScriptReadsServices(t *testing.T) {
	t.Parallel()
	for _, w := range []string{"Win32_Service", "niagarad.exe", "ParentProcessId", "'UNREAD'", "'SVC'"} {
		if !strings.Contains(inUseScript, w) {
			t.Errorf("inUseScript lacks %q", w)
		}
	}
}

// The scratch niagara_user_home reaches test.exe through the WSL interop environment (WSLENV .../w), never a
// cmd.exe `set`: WSL re-quotes the inner quotes of a cmd line, so `set "niagara_user_home=..."` set another name.
func TestRunTestExeGetsUserHomeThroughWSLENV(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
	if code != exitPass {
		t.Fatalf("exit %d\n%s", code, out)
	}
	c := fx.f.of("test")[0]
	if w := envValue(c.Env, "WSLENV="); !slices.Contains(strings.Split(w, ":"), "niagara_user_home/w") {
		t.Errorf("WSLENV %q lacks niagara_user_home/w", w)
	}
	if nuh := envValue(c.Env, "niagara_user_home="); !strings.HasPrefix(nuh, tempWin+`\run-niagara-test-`) || !strings.HasSuffix(nuh, `\nuh`) {
		t.Errorf("niagara_user_home %q is not the scratch home in Windows form", nuh)
	}
	if !strings.HasSuffix(c.Dir, "/home") || c.Name != c.Dir+"/bin/test.exe" {
		t.Errorf("test.exe %q in %q, want the copy's bin/test.exe with the copy as cwd", c.Name, c.Dir)
	}
	for _, c := range fx.f.calls {
		if j := strings.Join(append([]string{c.Name}, c.Args...), " "); strings.Contains(j, `set "niagara_user_home`) {
			t.Errorf("a command sets niagara_user_home through cmd.exe: %s", j)
		}
	}
}
