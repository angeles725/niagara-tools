package platread

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const secret = "s3cr3t!pw"

// fakeHome is a Niagara home whose bin/plat prints its arguments, an NRE boot line, a module
// permission block and the answer, as the 4.14 plat does, then exits with code.
func fakeHome(t *testing.T, code int) (home, env string) {
	t.Helper()
	home = t.TempDir()
	bin := filepath.Join(home, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\necho \"ARGS $*\"\n" +
		"echo 'INFORMACIÓN [18:59:18 25-nov-26 CST][nre] Booting'\necho 'INFO [nre] Launching Niagara Runtime Environment'\n" +
		"echo 'Type: ACCESS_CLASS'\necho ' Purpose: clipboard'\necho ' Parameters: [ x ]'\necho ' Risk Level: SEVERE'\n" +
		"echo 'Platform summary for localhost'\necho 'Daemon Version:           4.14.0.162'\n" +
		"exit " + string(rune('0'+code)) + "\n"
	if err := os.WriteFile(filepath.Join(bin, "plat"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	env = filepath.Join(t.TempDir(), "plat.env")
	body := "# lab daemon\nNIAGARA_PLAT_HOST=localhost\nNIAGARA_PLAT_PORT=5011\nNIAGARA_PLAT_USER=admin\n" +
		"NIAGARA_PLAT_PASSWORD='" + secret + "'\n"
	if err := os.WriteFile(env, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return home, env
}

func run(t *testing.T, home string, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	env := map[string]string{"NIAGARA_HOME": home}
	code := Main(args, cli.Stdio{Out: &out, Err: &errb, Getenv: func(k string) string { return env[k] }})
	return code, out.String(), errb.String()
}

func TestDetailsRunsPlatOverTLSAndMasksThePassword(t *testing.T) {
	home, env := fakeHome(t, 0)
	code, out, errs := run(t, home, "--env", env, "details")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if strings.Contains(out+errs, secret) {
		t.Fatalf("password printed: %s", out)
	}
	for _, want := range []string{"ARGS details -h:ip:localhost -p:5011 -secure -noinput -usr:admin -pwd:<masked>",
		"Platform summary for localhost", "Daemon Version:           4.14.0.162"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
	for _, noise := range []string{"Booting", "Launching Niagara", "Type: ACCESS_CLASS", "Risk Level"} {
		if strings.Contains(out, noise) {
			t.Errorf("noise %q kept in\n%s", noise, out)
		}
	}
}

func TestPlatExitCodeIsPassedOn(t *testing.T) {
	home, env := fakeHome(t, 1)
	if code, _, _ := run(t, home, "--env", env, "liststations"); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
}

func TestAStateChangingVerbIsRefused(t *testing.T) {
	home, env := fakeHome(t, 0)
	for _, verb := range []string{"stopstation", "reboothost", "fput", "tellstation", "settime"} {
		code, out, errs := run(t, home, "--env", env, verb)
		if code != 3 || out != "" || !strings.Contains(errs, "only reads") {
			t.Errorf("%s: exit %d out %q err %q", verb, code, out, errs)
		}
	}
}

func TestDryRunShowsTheCallWithoutRunningIt(t *testing.T) {
	home, env := fakeHome(t, 0)
	code, out, _ := run(t, home, "--env", env, "--dry-run", "ipconfig")
	if code != 0 || strings.Contains(out, secret) || strings.Contains(out, "ARGS") ||
		!strings.Contains(out, "plat ipconfig -h:ip:localhost -p:5011 -secure -noinput -usr:admin -pwd:<masked>") {
		t.Fatalf("exit %d:\n%s", code, out)
	}
}

func TestAMissingOrOpenEnvFileIsRefused(t *testing.T) {
	home, env := fakeHome(t, 0)
	if code, _, errs := run(t, home, "--env", env+".none", "details"); code != 3 || !strings.Contains(errs, "env file") {
		t.Errorf("missing: exit %d %q", code, errs)
	}
	if err := os.Chmod(env, 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errs := run(t, home, "--env", env, "details"); code != 3 || !strings.Contains(errs, "0600") {
		t.Errorf("mode 0644: exit %d %q", code, errs)
	}
	for body, key := range map[string]string{
		"NIAGARA_PLAT_HOST=localhost\nNIAGARA_PLAT_USER=admin\n":                            "has no NIAGARA_PLAT_PASSWORD",
		"NIAGARA_PLAT_HOST=localhost\nNIAGARA_PLAT_USER=<admin>\nNIAGARA_PLAT_PASSWORD=x\n": "NIAGARA_PLAT_USER is still wrapped",
	} {
		if err := os.Chmod(env, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(env, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		code, _, errs := run(t, home, "--env", env, "details")
		if code != 3 || !strings.Contains(errs, key) || strings.Contains(errs, "admin") {
			t.Errorf("%q: exit %d %q", key, code, errs)
		}
	}
}
