package platread

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setPlat(t *testing.T, home, script string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(home, "bin", "plat"), []byte("#!/bin/sh\n"+script), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(home, "bin", "plat"), mode); err != nil {
		t.Fatal(err)
	}
}

func setEnv(t *testing.T, env, body string) {
	t.Helper()
	if err := os.WriteFile(env, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// Only INFO boot lines and the registry's missing-class lines are noise: an error plat logs in the NRE
// format (an auth or connection failure) is the answer and stays (independent verify, #850).
func TestTheNoiseFilterKeepsErrorLines(t *testing.T) {
	home, env := fakeHome(t, 1)
	setPlat(t, home, "echo 'INFO [12:00:00 25-nov-26 CST][nre] Booting'\n"+
		"echo 'GRAVE [12:00:01 25-nov-26 CST][sys.registry] Missing class for \"x:Y\"'\n"+
		"echo 'SEVERE [12:00:02 25-nov-26 CST][platform] Authentication failed'\n"+
		"echo 'ERROR [12:00:03 25-nov-26 CST][platform] java.net.ConnectException'\nexit 1\n", 0o755)
	code, out, _ := run(t, home, "--env", env, "details")
	if code != 1 || strings.Contains(out, "Booting") || strings.Contains(out, "Missing class") ||
		!strings.Contains(out, "Authentication failed") || !strings.Contains(out, "ConnectException") {
		t.Fatalf("exit %d:\n%s", code, out)
	}
}

func TestNoVerbBypassesTheAllowList(t *testing.T) {
	home, env := fakeHome(t, 0)
	for _, args := range [][]string{{"Details"}, {"details;x"}, {"details", "-x"}, {"details", "extra"},
		{"stopstation", "details"}, {"--env"}} {
		code, out, _ := run(t, home, append([]string{"--env", env}, args...)...)
		if code != 3 || out != "" {
			t.Errorf("%q: exit %d out %q", args, code, out)
		}
	}
}

func TestAnEmptyOrTemplatePasswordIsRefused(t *testing.T) {
	home, env := fakeHome(t, 0)
	for _, pw := range []string{"''", "<pw>"} {
		setEnv(t, env, "NIAGARA_PLAT_HOST=localhost\nNIAGARA_PLAT_USER=admin\nNIAGARA_PLAT_PASSWORD="+pw+"\n")
		if code, _, errs := run(t, home, "--env", env, "details"); code != 3 || !strings.Contains(errs, "NIAGARA_PLAT_PASSWORD") {
			t.Errorf("%s: exit %d %q", pw, code, errs)
		}
	}
}

func TestAPasswordEchoedOnStderrOrAnExecFailureIsMasked(t *testing.T) {
	home, env := fakeHome(t, 0)
	setPlat(t, home, "echo \"$*\" >&2\nexit 7\n", 0o755)
	code, out, errs := run(t, home, "--env", env, "details")
	if code != 7 || strings.Contains(out+errs, secret) || !strings.Contains(out, "-pwd:<masked>") {
		t.Fatalf("exit %d out %q err %q", code, out, errs)
	}
	setPlat(t, home, "exit 0\n", 0o644)
	code, out, errs = run(t, home, "--env", env, "details")
	if code != 1 || out != "" || strings.Contains(errs, secret) {
		t.Fatalf("exec failure: exit %d out %q err %q", code, out, errs)
	}
}

func TestThePortDefaultsTo5011(t *testing.T) {
	home, env := fakeHome(t, 0)
	setEnv(t, env, "NIAGARA_PLAT_HOST=localhost\nNIAGARA_PLAT_USER=admin\nNIAGARA_PLAT_PASSWORD="+secret+"\n")
	if code, out, _ := run(t, home, "--env", env, "--dry-run", "details"); code != 0 || !strings.Contains(out, "-p:5011 ") {
		t.Fatalf("exit %d: %s", code, out)
	}
}

// plat exits 0 after an AuthenticationException (live, 4.14.0.162): plat-read says why and exits 1.
func TestAStackTraceUnderExitZeroIsAFailure(t *testing.T) {
	home, env := fakeHome(t, 0)
	setPlat(t, home, "echo 'javax.baja.security.AuthenticationException'\n"+
		"printf '\\tat com.tridium.platform.daemon.BDaemonSession.connect(BDaemonSession.java:305)\\n'\nexit 0\n", 0o755)
	code, out, errs := run(t, home, "--env", env, "liststations")
	if code != 1 || !strings.Contains(errs, "refused the user or password") || !strings.Contains(out, "AuthenticationException") {
		t.Fatalf("exit %d out %q err %q", code, out, errs)
	}
	setPlat(t, home, "echo 'java.lang.NullPointerException'\nexit 0\n", 0o755)
	if code, _, errs = run(t, home, "--env", env, "liststations"); code != 1 || !strings.Contains(errs, "plat failed: java.lang.NullPointerException") {
		t.Fatalf("NPE: exit %d err %q", code, errs)
	}
}
