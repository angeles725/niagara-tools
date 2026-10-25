package niagaratest

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const (
	greenXML = `<testng-results skipped="0" failed="0" ignored="0" total="1" passed="1"><suite name="S"><test name="T">` +
		`<class name="test.com.example.ModuleA.BExampleStationTest"><test-method status="PASS" name="reads_value"/></class></test></suite></testng-results>`
	redXML = `<testng-results skipped="0" failed="1" ignored="0" total="1" passed="0"><suite name="S"><test name="T">` +
		`<class name="test.com.example.ModuleA.BExampleStationTest"><test-method status="FAIL" name="reads_value"/></class></test></suite></testng-results>`
	greenLog = "Total tests run: 1, Failures: 0, Skips: 0\r\n"
	tempWin  = `C:\Users\you\AppData\Local\Temp`
)

// fake is the command runner seam: it records every call and answers with a per-kind handler.
type fake struct {
	t     *testing.T
	fx    *fixture
	mu    sync.Mutex
	calls []Cmd
	on    map[string]func(context.Context, Cmd) (int, error)
}

// fixture is a fake Windows drive (mnt/c) holding an install and %TEMP%, plus a gradle root.
type fixture struct {
	mnt, install, temp, gradle string
	f                          *fake
	h                          host
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	fx := &fixture{mnt: t.TempDir(), gradle: t.TempDir()}
	fx.install = filepath.Join(fx.mnt, "c", "Niagara", "N4")
	fx.temp = filepath.Join(fx.mnt, "c", "Users", "you", "AppData", "Local", "Temp")
	for _, p := range []string{"bin/test.exe", "etc/m2/repository/com/tridium/niagara/com.tridium.niagara.gradle.plugin/7.6.17/x.pom",
		"modules/moduleA-rt.jar", "modules/stray.77-rt.jar", "security/keys", "lib/l.jar", "jre/bin/java.exe"} {
		write(t, filepath.Join(fx.install, p), "x")
	}
	write(t, filepath.Join(fx.gradle, "gradlew"), "#!/bin/sh\n")
	if err := os.MkdirAll(fx.temp, 0o755); err != nil {
		t.Fatal(err)
	}
	fx.f = &fake{t: t, fx: fx, on: map[string]func(context.Context, Cmd) (int, error){}}
	fx.h = host{run: fx.f, mnt: fx.mnt, home: "/nonexistent-home"}
	return fx
}

func write(t *testing.T, p, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func (fx *fixture) args(extra ...string) []string {
	return append([]string{"--install", `C:\Niagara\N4`, "--gradle-root", fx.gradle, "--part", "ModuleA-rt", "--module", "ModuleA"}, extra...)
}

func (fx *fixture) run(t *testing.T, args ...string) (int, string) {
	t.Helper()
	var o, e bytes.Buffer
	code := fx.h.runMain(context.Background(), args, cli.Stdio{Out: &o, Err: &e})
	return code, o.String() + e.String()
}

// kindOf names a call: temp, preflight, procs, robocopy, taskkill, gradle or test.
func kindOf(c Cmd) string {
	switch {
	case c.Name == "bash":
		return "gradle"
	case c.Name == "powershell.exe":
		if strings.Contains(decodePS(c.Args), "config.bog.lock") {
			return "preflight"
		}
		return "procs"
	case c.Name == "cmd.exe" && len(c.Args) > 1 && c.Args[0] == "/v:on":
		return "test"
	case c.Name == "cmd.exe" && len(c.Args) > 1 && c.Args[1] == "echo %TEMP%":
		return "temp"
	case c.Name == "cmd.exe" && len(c.Args) > 1:
		return c.Args[1]
	}
	return "?"
}

func decodePS(args []string) string {
	b, err := base64.StdEncoding.DecodeString(args[len(args)-1])
	if err != nil || len(b)%2 != 0 {
		return ""
	}
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = uint16(b[2*i]) | uint16(b[2*i+1])<<8
	}
	return string(utf16.Decode(u))
}

func (f *fake) Run(ctx context.Context, c Cmd) (int, error) {
	f.mu.Lock()
	f.calls = append(f.calls, c)
	h := f.on[kindOf(c)]
	f.mu.Unlock()
	if h != nil {
		return h(ctx, c)
	}
	return f.std(c)
}

// std is the green route: robocopy copies for real, gradle succeeds, test.exe writes a green XML.
func (f *fake) std(c Cmd) (int, error) {
	switch kindOf(c) {
	case "temp":
		fmt.Fprint(c.Stdout, tempWin+"\r\n")
	case "preflight", "procs":
		fmt.Fprint(c.Stdout, "DONE\r\n")
	case "robocopy":
		f.robocopy(c.Args[2:])
		return 1, nil
	case "test":
		f.testExe(c, greenXML, greenLog, "0")
	}
	return 0, nil
}

func (f *fake) wsl(win string) string {
	p, ok := f.fx.h.toWSL(win)
	if !ok {
		f.t.Fatalf("not a drive path: %s", win)
	}
	return p
}

func (f *fake) robocopy(args []string) {
	src, dst := f.wsl(args[0]), f.wsl(args[1])
	var files []string
	for _, a := range args[2:] {
		if !strings.HasPrefix(a, "/") {
			files = append(files, a)
		}
	}
	_ = filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if files != nil && !slices.Contains(files, rel) {
			return nil
		}
		b, _ := os.ReadFile(p)
		write(f.t, filepath.Join(dst, rel), string(b))
		return nil
	})
}

var outRe, errRe = regexp.MustCompile(`-output:(\S+)`), regexp.MustCompile(`2>(\S+)`)

func (f *fake) testExe(c Cmd, xml, log, exit string) {
	line := c.Args[len(c.Args)-1]
	write(f.t, filepath.Join(f.wsl(outRe.FindStringSubmatch(line)[1]), "testng-results.xml"), xml)
	write(f.t, f.wsl(errRe.FindStringSubmatch(line)[1]), log)
	if exit != "" {
		fmt.Fprintf(c.Stdout, "EXIT=%s\r\n", exit)
	}
}

func (f *fake) kinds() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var ks []string
	for _, c := range f.calls {
		if k := kindOf(c); len(ks) == 0 || ks[len(ks)-1] != k {
			ks = append(ks, k)
		}
	}
	return ks
}

func (f *fake) of(kind string) []Cmd {
	f.mu.Lock()
	defer f.mu.Unlock()
	var cs []Cmd
	for _, c := range f.calls {
		if kindOf(c) == kind {
			cs = append(cs, c)
		}
	}
	return cs
}

// leftovers lists what the run left in %TEMP%; every run but --keep-home leaves nothing.
func (fx *fixture) leftovers(t *testing.T) []string {
	t.Helper()
	es, err := os.ReadDir(fx.temp)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range es {
		names = append(names, e.Name())
	}
	return names
}

// noImageKill is guard 4: no call ever kills by image name.
func (fx *fixture) noImageKill(t *testing.T) {
	t.Helper()
	for _, c := range fx.f.calls {
		for _, a := range c.Args {
			if strings.EqualFold(a, "/IM") || strings.Contains(strings.ToUpper(a), " /IM ") {
				t.Errorf("a call kills by image name: %q", c.Args)
			}
		}
	}
}

func TestRunGreenRoute(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
	if code != exitPass {
		t.Fatalf("exit %d\n%s", code, out)
	}
	want := "temp preflight robocopy gradle test procs"
	if got := strings.Join(fx.f.kinds(), " "); got != want {
		t.Errorf("steps %q, want %q", got, want)
	}
	for _, w := range []string{"PASS  run-niagara-test  verdict: 1 test run, 0 failed, 0 skipped (testng-results)",
		"PASS  run-niagara-test  install unchanged (", "copy: 1 jar skipped (dotted module name): stray.77-rt.jar"} {
		if !strings.Contains(out, w) {
			t.Errorf("no %q in\n%s", w, out)
		}
	}
	g := fx.f.of("gradle")[0]
	joined := strings.Join(g.Args, " ")
	for _, w := range []string{"./gradlew --no-daemon --max-workers=4", "-PniagaraPluginVersion=7.6.17", ":ModuleA-rt:jar :ModuleA-rt:moduleTestJar"} {
		if !strings.Contains(joined, w) {
			t.Errorf("gradle args %q lack %q", joined, w)
		}
	}
	// Guard 1: the build's niagara_home is the copy, never --install.
	home := argValue(g.Args, "-Pniagara_home=")
	if !strings.HasPrefix(home, fx.temp+"/run-niagara-test-") || !strings.HasSuffix(home, "/home") {
		t.Errorf("build niagara_home %q is not the copy under %%TEMP%%", home)
	}
	if uh := argValue(g.Args, "-Pniagara_user_home="); !strings.HasSuffix(uh, "/nuh") || !strings.HasPrefix(uh, fx.temp) {
		t.Errorf("niagara_user_home %q is not scratch", uh)
	}
	if len(g.Env) != 1 || !strings.HasPrefix(g.Env[0], "JAVA_TOOL_OPTIONS=-Duser.home="+fx.temp) {
		t.Errorf("gradle env %q", g.Env)
	}
	if l := fx.leftovers(t); len(l) != 0 {
		t.Errorf("left in %%TEMP%%: %q", l)
	}
	fx.noImageKill(t)
}

func argValue(args []string, prefix string) string {
	for _, a := range args {
		if strings.HasPrefix(a, prefix) {
			return strings.TrimPrefix(a, prefix)
		}
	}
	return ""
}

func TestRunRefusals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		setup func(fx *fixture) []string
		want  string
	}{
		{"install under a user home", func(fx *fixture) []string {
			write(t, filepath.Join(fx.mnt, "c", "Users", "you", "N4", "bin", "test.exe"), "x")
			return replace(fx.args(), `C:\Niagara\N4`, `C:\Users\you\N4`)
		}, "under a user home"},
		{"not a niagara_home", func(fx *fixture) []string {
			_ = os.Remove(filepath.Join(fx.install, "bin", "test.exe"))
			return fx.args()
		}, "not a niagara_home (no bin/test.exe)"},
		{"install in WSL form, not on a drive", func(fx *fixture) []string {
			return replace(fx.args(), `C:\Niagara\N4`, fx.gradle)
		}, "not on a Windows drive"},
		{"scratch outside %TEMP%", func(fx *fixture) []string {
			_ = os.MkdirAll(filepath.Join(fx.mnt, "c", "scratch"), 0o755)
			return fx.args("--scratch", `C:\scratch`)
		}, "--scratch is not under %TEMP%"},
		{"guard 1: %TEMP% inside --install", func(fx *fixture) []string {
			_ = os.MkdirAll(filepath.Join(fx.install, "tmp"), 0o755)
			fx.f.on["temp"] = func(_ context.Context, c Cmd) (int, error) { fmt.Fprint(c.Stdout, `C:\Niagara\N4\tmp`); return 0, nil }
			return fx.args()
		}, "the build's niagara_home would be inside --install"},
		{"guard 1: scratch symlinked into --install", func(fx *fixture) []string {
			if err := os.Symlink(fx.install, filepath.Join(fx.temp, "link")); err != nil {
				t.Fatal(err)
			}
			return fx.args("--scratch", tempWin+`\link`)
		}, "--scratch is not under %TEMP%"},
		{"guard 3: a running station uses the install", func(fx *fixture) []string {
			fx.f.on["preflight"] = func(_ context.Context, c Cmd) (int, error) {
				fmt.Fprint(c.Stdout, "PROC 4242 station.exe\r\nDONE\r\n")
				return 0, nil
			}
			return fx.args()
		}, "a running process uses the install: PROC 4242 station.exe"},
		{"guard 3: a live station lock", func(fx *fixture) []string {
			fx.f.on["preflight"] = func(_ context.Context, c Cmd) (int, error) {
				fmt.Fprint(c.Stdout, `LOCK C:\Niagara\N4\stations\s1\config.bog.lock`+"\r\nDONE\r\n")
				return 0, nil
			}
			return fx.args()
		}, `a station holds a live lock: C:\Niagara\N4\stations\s1\config.bog.lock`},
		{"guard 3: the probe cannot tell", func(fx *fixture) []string {
			fx.f.on["preflight"] = func(context.Context, Cmd) (int, error) { return 1, nil }
			return fx.args()
		}, "cannot verify that no station uses the install"},
		{"unsafe character", func(fx *fixture) []string {
			return replace(fx.args(), `C:\Niagara\N4`, `C:\Niagara\N4&calc`)
		}, "unsafe character"},
		{"bad part", func(fx *fixture) []string { return replace(fx.args(), "ModuleA-rt", "a b") }, "--part"},
		{"missing --module", func(fx *fixture) []string { return fx.args()[:6] }, "--module is required"},
		{"bad timeout", func(fx *fixture) []string { return fx.args("--timeout", "0s") }, "--timeout"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			code, out := fx.run(t, append([]string{"run"}, tc.setup(fx)...)...)
			if code != exitUsage || !strings.Contains(out, tc.want) {
				t.Errorf("exit %d, want 2 with %q:\n%s", code, tc.want, out)
			}
			if n := len(fx.f.of("robocopy")) + len(fx.f.of("gradle")) + len(fx.f.of("test")); n != 0 {
				t.Errorf("%d copy/build/run calls after a refusal", n)
			}
			if l := fx.leftovers(t); len(l) > 1 || (len(l) == 1 && l[0] != "link") {
				t.Errorf("left in %%TEMP%%: %q", l)
			}
		})
	}
}

func replace(args []string, old, repl string) []string {
	out := slices.Clone(args)
	for i, a := range out {
		if a == old {
			out[i] = repl
		}
	}
	return out
}

func TestRunVerdictWiredToXML(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, xml, exit, want string }{
		{"red XML", redXML, "-1", "FAIL  run-niagara-test  testng: test-method reads_value FAIL"},
		{"exit -1 but results green", greenXML, "-1", "FAIL  run-niagara-test  run: exit -1 but results green"},
		{"no exit status read", greenXML, "", "FAIL  run-niagara-test  run: no exit status read"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			fx.f.on["test"] = func(_ context.Context, c Cmd) (int, error) { fx.f.testExe(c, tc.xml, greenLog, tc.exit); return 0, nil }
			code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
			if code != exitFail || !strings.Contains(out, tc.want) {
				t.Errorf("exit %d, want 1 with %q:\n%s", code, tc.want, out)
			}
			if l := fx.leftovers(t); len(l) != 0 {
				t.Errorf("left in %%TEMP%%: %q", l)
			}
		})
	}
}

// Guard 2: a write into --install during the run is a FAIL, never exit 2.
func TestRunInstallChangedIsFail(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	fx.f.on["gradle"] = func(context.Context, Cmd) (int, error) {
		write(t, filepath.Join(fx.install, "modules", "ModuleA-rtTest.jar"), "x")
		return 0, nil
	}
	code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
	want := "FAIL  run-niagara-test  install changed during the run (the run already happened): added modules/ModuleA-rtTest.jar"
	if code != exitFail || !strings.Contains(out, want) {
		t.Errorf("exit %d, want 1 with %q:\n%s", code, want, out)
	}
}

func TestRunBuildFailureCleansUp(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	fx.f.on["gradle"] = func(_ context.Context, c Cmd) (int, error) { fmt.Fprint(c.Stdout, "BUILD FAILED\n"); return 1, nil }
	code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
	if code != exitEnv || !strings.Contains(out, "FAIL  run-niagara-test  build: gradle exit 1") || !strings.Contains(out, "BUILD FAILED") {
		t.Errorf("exit %d, want 3:\n%s", code, out)
	}
	if len(fx.f.of("test")) != 0 || len(fx.f.of("procs")) != 1 {
		t.Errorf("steps %q", fx.f.kinds())
	}
	if l := fx.leftovers(t); len(l) != 0 {
		t.Errorf("left in %%TEMP%%: %q", l)
	}
}

func TestRunPanicCleansUp(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	fx.f.on["gradle"] = func(context.Context, Cmd) (int, error) { panic("boom") }
	func() {
		defer func() {
			if r := recover(); r != "boom" {
				t.Errorf("recovered %v, want the panic to propagate", r)
			}
		}()
		fx.run(t, append([]string{"run"}, fx.args()...)...)
	}()
	if len(fx.f.of("procs")) != 1 {
		t.Errorf("no process cleanup after a panic: %q", fx.f.kinds())
	}
	if l := fx.leftovers(t); len(l) != 0 {
		t.Errorf("left in %%TEMP%%: %q", l)
	}
}

func TestRunTimeoutKillsTree(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	fx.f.on["test"] = func(ctx context.Context, _ Cmd) (int, error) { <-ctx.Done(); return -1, ctx.Err() }
	listed := 0
	fx.f.on["procs"] = func(_ context.Context, c Cmd) (int, error) {
		if listed++; listed == 1 {
			fmt.Fprint(c.Stdout, "PID 4242\r\nPID 4343\r\n")
		}
		fmt.Fprint(c.Stdout, "DONE\r\n")
		return 0, nil
	}
	start := time.Now()
	code, out := fx.run(t, append([]string{"run"}, fx.args("--timeout", "200ms")...)...)
	if code != exitTimeout || !strings.Contains(out, "FAIL  run-niagara-test  timeout: run killed after 200ms") {
		t.Errorf("exit %d, want 4:\n%s", code, out)
	}
	if time.Since(start) > 10*time.Second {
		t.Errorf("the timeout took %s", time.Since(start))
	}
	var kills []string
	for _, c := range fx.f.of("taskkill") {
		kills = append(kills, strings.Join(c.Args[1:], " "))
	}
	if got := strings.Join(kills, "; "); got != "taskkill /PID 4242 /T /F; taskkill /PID 4343 /T /F" {
		t.Errorf("kills %q", got)
	}
	if !strings.Contains(out, "PASS  run-niagara-test  cleanup: 2 processes of this run killed") {
		t.Errorf("no kill row:\n%s", out)
	}
	if l := fx.leftovers(t); len(l) != 0 {
		t.Errorf("left in %%TEMP%%: %q", l)
	}
	fx.noImageKill(t)
}

func TestRunSurvivingProcessIsEnvFailure(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	fx.f.on["procs"] = func(_ context.Context, c Cmd) (int, error) {
		fmt.Fprint(c.Stdout, "PID 4242\r\nDONE\r\n")
		return 0, nil
	}
	code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
	if code != exitEnv || !strings.Contains(out, "FAIL  run-niagara-test  cleanup: 1 process of this run still running: 4242") {
		t.Errorf("exit %d, want 3:\n%s", code, out)
	}
	fx.noImageKill(t)
}

func TestRunKeepHome(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	code, out := fx.run(t, append([]string{"run"}, fx.args("--keep-home")...)...)
	l := fx.leftovers(t)
	if code != exitPass || len(l) != 1 || !strings.Contains(out, "kept "+tempWin+`\`+l[0]) {
		t.Errorf("exit %d, left %q:\n%s", code, l, out)
	}
	if _, err := os.Stat(filepath.Join(fx.temp, l[0], "home", "bin", "test.exe")); err != nil {
		t.Errorf("no kept home copy: %v", err)
	}
}

func TestWithinIsCaseInsensitive(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		child, parent string
		want          bool
	}{{`C:\NIAGARA\n4\tmp`, `C:\Niagara\N4`, true}, {`c:\niagara\n4`, `C:\Niagara\N4`, true}, {`C:\Niagara\N4-old`, `C:\Niagara\N4`, false}} {
		if got := within(tc.child, tc.parent); got != tc.want {
			t.Errorf("within(%q, %q) = %v", tc.child, tc.parent, got)
		}
	}
}
