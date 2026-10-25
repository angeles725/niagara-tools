package niagaratest

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Exit codes of run, beyond verdict's.
const (
	exitEnv     = 3
	exitTimeout = 4
)

const (
	defaultTimeout = 30 * time.Minute
	maxTimeout     = 24 * time.Hour
	cleanupTimeout = 2 * time.Minute
)

// copiedDirs are the install dirs the home copy takes whole; modules/ is copied jar by jar (copyJars).
var copiedDirs = []string{"bin", "etc", "jre", "lib", "security", "defaults"}

var (
	partRe    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*(:[A-Za-z0-9][A-Za-z0-9_.-]*)*$`)
	moduleRe  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	versionRe = regexp.MustCompile(`^[0-9]+(\.[0-9]+)+$`)
	exitRe    = regexp.MustCompile(`EXIT=(-?[0-9]+)`)
)

// runConfig is the parsed command line of run.
type runConfig struct {
	install, gradleRoot, part, module, scratch string
	timeout                                    time.Duration
	keepHome                                   bool
}

// plan is a validated run: the install, %TEMP%, the scratch parent and, once created, the scratch dir.
type plan struct {
	runConfig
	install, temp, parent, dir winDir
	windows                    bool // a Windows-side step started: cleanup must look for its processes
}

func (p *plan) home() winDir        { return winDir{p.dir.wsl + "/home", p.dir.win + `\home`} }
func (p *plan) sub(n string) string { return p.dir.wsl + "/" + n }
func (p *plan) subWin(n string) string {
	return p.dir.win + `\` + strings.ReplaceAll(n, "/", `\`)
}

// runCmd is `run` on this host: real processes, the /mnt drive mount, Ctrl-C or SIGTERM cancelling the run.
func runCmd(args []string, s cli.Stdio) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	home, _ := os.UserHomeDir()
	return host{run: execRunner{}, mnt: "/mnt", home: home}.runMain(ctx, append([]string{"run"}, args...), s)
}

// runMain parses `run …`, validates every path and probe, then runs the test with its cleanup.
func (h host) runMain(ctx context.Context, args []string, s cli.Stdio) int {
	if len(args) == 0 || args[0] != "run" {
		return usageError(s, "runMain: not a run command")
	}
	cfg, err := parseRun(args[1:])
	if err != nil {
		return usageError(s, err.Error())
	}
	p, code := h.validate(ctx, cfg, s)
	if code >= 0 {
		return code
	}
	return h.execute(ctx, p, s)
}

func parseRun(args []string) (runConfig, error) {
	var c runConfig
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&c.install, "install", "", "")
	fs.StringVar(&c.gradleRoot, "gradle-root", "", "")
	fs.StringVar(&c.part, "part", "", "")
	fs.StringVar(&c.module, "module", "", "")
	fs.StringVar(&c.scratch, "scratch", "", "")
	fs.DurationVar(&c.timeout, "timeout", defaultTimeout, "")
	fs.BoolVar(&c.keepHome, "keep-home", false, "")
	if err := fs.Parse(args); err != nil {
		return c, fmt.Errorf("run: %v", err)
	}
	if fs.NArg() > 0 {
		return c, fmt.Errorf("run: unexpected argument %q", fs.Arg(0))
	}
	for _, f := range []struct{ name, v string }{{"install", c.install}, {"gradle-root", c.gradleRoot}, {"part", c.part}, {"module", c.module}} {
		if f.v == "" {
			return c, fmt.Errorf("run: --%s is required", f.name)
		}
	}
	switch {
	case !partRe.MatchString(c.part):
		return c, fmt.Errorf("run: --part %q is not a gradle project name", c.part)
	case !moduleRe.MatchString(c.module):
		return c, fmt.Errorf("run: --module %q is not a module name", c.module)
	case c.timeout <= 0 || c.timeout > maxTimeout:
		return c, fmt.Errorf("run: --timeout %s is not in (0, %s]", c.timeout, maxTimeout)
	}
	return c, nil
}

// validate checks everything before anything is copied, built or run; a refusal is exit 2, an environment
// that cannot answer (no cmd.exe) exit 3. code is -1 when the run may go ahead.
func (h host) validate(ctx context.Context, cfg runConfig, s cli.Stdio) (*plan, int) {
	p := &plan{runConfig: cfg}
	refuse := func(format string, a ...any) (*plan, int) {
		return nil, usageError(s, "run: "+fmt.Sprintf(format, a...))
	}
	var err error
	if p.install, err = h.canonical("--install", cfg.install); err != nil {
		return refuse("%v", err)
	}
	if underUsers(p.install.win) || (h.home != "" && strings.HasPrefix(p.install.wsl+"/", strings.TrimSuffix(h.home, "/")+"/")) {
		return refuse("--install %s is under a user home; point it at a niagara_home", p.install.win)
	}
	if info, err := os.Stat(p.install.wsl + "/bin/test.exe"); err != nil || !info.Mode().IsRegular() {
		return refuse("--install %s is not a niagara_home (no bin/test.exe)", p.install.win)
	}
	if info, err := os.Stat(filepath.Join(cfg.gradleRoot, "gradlew")); err != nil || !info.Mode().IsRegular() {
		return refuse("--gradle-root %s has no gradlew", cfg.gradleRoot)
	}
	out := &capped{max: maxCaptured}
	if code, err := h.run.Run(ctx, Cmd{Name: "cmd.exe", Args: []string{"/c", "echo %TEMP%"}, Stdout: out}); err != nil || code != 0 {
		fmt.Fprintf(s.Err, "%s: run: cannot read %%TEMP%% through cmd.exe (exit %d, %v)\n", name, code, err)
		return nil, exitEnv
	}
	if p.temp, err = h.canonical("%TEMP%", strings.TrimSpace(out.buf.String())); err != nil {
		fmt.Fprintf(s.Err, "%s: run: %v\n", name, err)
		return nil, exitEnv
	}
	p.parent = p.temp
	if cfg.scratch != "" {
		if p.parent, err = h.canonical("--scratch", cfg.scratch); err != nil {
			return refuse("%v", err)
		}
		if !within(p.parent.win, p.temp.win) {
			return refuse("--scratch is not under %%TEMP%% (%s resolves to %s)", p.temp.win, p.parent.win)
		}
	}
	if strings.ContainsAny(p.parent.win, "  ") {
		return refuse("the scratch parent %s holds whitespace; pass a --scratch under %%TEMP%% without any", p.parent.win)
	}
	// Guard 1: the build's niagara_home is the copy under the scratch parent; it must never be the install.
	if within(p.parent.win, p.install.win) || within(p.install.win, p.parent.win) {
		return refuse("the build's niagara_home would be inside --install (scratch %s, install %s)", p.parent.win, p.install.win)
	}
	// Guard 3: refuse an install a running station, NRE or Workbench uses.
	rows, ok := h.powershell(ctx, fmt.Sprintf(inUseScript, p.install.win))
	if !ok {
		return refuse("cannot verify that no station uses the install (the PowerShell probe failed)")
	}
	for _, r := range rows {
		switch {
		case strings.HasPrefix(r, "PROC "):
			return refuse("a running process uses the install: %s", r)
		case strings.HasPrefix(r, "LOCK "):
			return refuse("a station holds a live lock: %s", strings.TrimPrefix(r, "LOCK "))
		default:
			return refuse("cannot verify that no station uses the install: %s", r)
		}
	}
	return p, -1
}

// execute creates the scratch dir, then copies, builds, runs and judges; cleanup runs on every path out,
// a panic included (the panic goes on after it).
func (h host) execute(ctx context.Context, p *plan, s cli.Stdio) (code int) {
	snap, err := takeSnapshot(p.install.wsl)
	if err != nil {
		fmt.Fprintf(s.Err, "%s: run: cannot snapshot the install: %v\n", name, err)
		return exitEnv
	}
	dir, err := os.MkdirTemp(p.parent.wsl, "run-niagara-test-")
	if err != nil {
		fmt.Fprintf(s.Err, "%s: run: %v\n", name, err)
		return exitEnv
	}
	win, _ := h.toWin(dir)
	p.dir = winDir{dir, win}
	defer func() { code = h.cleanup(p, snap, s, code) }()
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	for _, step := range []struct {
		name string
		do   func(context.Context, *plan, cli.Stdio) (int, error)
	}{{"copy", h.copyHome}, {"build", h.build}, {"run", h.testExe}} {
		code, err := step.do(ctx, p, s)
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			row(s, "FAIL", fmt.Sprintf("timeout: %s killed after %s", step.name, p.timeout))
			return exitTimeout
		case errors.Is(err, context.Canceled):
			row(s, "FAIL", step.name+": interrupted")
			return exitEnv
		case err != nil:
			row(s, "FAIL", fmt.Sprintf("%s: %v", step.name, err))
			return exitEnv
		case code >= 0:
			return code
		}
	}
	return exitPass // not reached: the run step always returns its verdict
}

// copyHome copies the install into <scratch>\home with robocopy: the dirs whole, then the top-level jars of
// modules\ but those whose module name has a dot (a stray `<name>.77-rt.jar` makes every boot rebuild the
// registry). The source is only read; licenses travel untouched.
func (h host) copyHome(ctx context.Context, p *plan, s cli.Stdio) (int, error) {
	p.windows = true
	src, dst := p.install.win, p.home().win
	for _, d := range copiedDirs {
		if _, err := os.Stat(p.install.wsl + "/" + d); err != nil {
			continue
		}
		if err := h.robocopy(ctx, src+`\`+d, dst+`\`+d, "/E"); err != nil {
			return -1, err
		}
	}
	es, err := os.ReadDir(p.install.wsl + "/modules")
	if err != nil {
		return -1, err
	}
	var jars, skipped []string
	for _, e := range es {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".jar") {
			continue
		}
		if strings.Contains(strings.TrimSuffix(n, ".jar"), ".") || strings.ContainsAny(n, unsafeChars+" ") {
			skipped = append(skipped, n)
			continue
		}
		jars = append(jars, n)
	}
	// cmd.exe takes 8191 characters a line: copy the jars in batches well under it.
	for len(jars) > 0 {
		n, size := 0, 0
		for n < len(jars) && size+len(jars[n]) < 6000 {
			size += len(jars[n]) + 1
			n++
		}
		if err := h.robocopy(ctx, src+`\modules`, dst+`\modules`, jars[:max(n, 1)]...); err != nil {
			return -1, err
		}
		jars = jars[max(n, 1):]
	}
	if len(skipped) > 0 {
		row(s, "PASS", fmt.Sprintf("copy: %s (dotted module name): %s", plural(len(skipped), "jar skipped", "jars skipped"), strings.Join(skipped, " ")))
	}
	return -1, nil
}

// robocopy fails on its exit 8 and above (0-7 report what was copied).
func (h host) robocopy(ctx context.Context, src, dst string, what ...string) error {
	args := append([]string{"/c", "robocopy", src, dst}, what...)
	code, err := h.run.Run(ctx, Cmd{Name: "cmd.exe", Args: append(args, "/R:0", "/W:0", "/NFL", "/NDL", "/NJH", "/NJS", "/NP")})
	if err != nil {
		return err
	}
	if code >= 8 || code < 0 {
		return fmt.Errorf("robocopy %s exit %d", src, code)
	}
	return nil
}

// pluginVersion is the one com.tridium.niagara gradle plugin version the copy's etc/m2 ships.
func pluginVersion(home string) (string, error) {
	d := home + "/etc/m2/repository/com/tridium/niagara/com.tridium.niagara.gradle.plugin"
	es, err := os.ReadDir(d)
	if err != nil {
		return "", fmt.Errorf("no niagara gradle plugin in etc/m2: %v", err)
	}
	var vs []string
	for _, e := range es {
		if e.IsDir() && versionRe.MatchString(e.Name()) {
			vs = append(vs, e.Name())
		}
	}
	if len(vs) != 1 {
		return "", fmt.Errorf("etc/m2 ships %d niagara gradle plugin versions %q, want exactly 1", len(vs), vs)
	}
	return vs[0], nil
}

// build runs gradle in WSL against the copy: :jar and :moduleTestJar install into the copy's modules/, and
// every home gradle or the signing plugin writes (niagara_user_home, user.home) is scratch.
func (h host) build(ctx context.Context, p *plan, s cli.Stdio) (int, error) {
	v, err := pluginVersion(p.home().wsl)
	if err != nil {
		return -1, err
	}
	for _, d := range []string{"nuh", "gradle-uh", "out/testng"} {
		if err := os.MkdirAll(p.sub(d), 0o755); err != nil {
			return -1, err
		}
	}
	log, err := os.Create(p.sub("out/build.log"))
	if err != nil {
		return -1, err
	}
	defer log.Close()
	code, err := h.run.Run(ctx, Cmd{Name: "bash", Dir: p.gradleRoot, Stdout: log, Stderr: log,
		Args: []string{"./gradlew", "--no-daemon", "--max-workers=4", "-Pniagara_home=" + p.home().wsl,
			"-Pniagara_user_home=" + p.sub("nuh"), "-PniagaraPluginVersion=" + v, ":" + p.part + ":jar", ":" + p.part + ":moduleTestJar"},
		Env: []string{"JAVA_TOOL_OPTIONS=-Duser.home=" + p.sub("gradle-uh")}})
	if err != nil || code == 0 {
		return -1, err
	}
	row(s, "FAIL", fmt.Sprintf("build: gradle exit %d (last lines of the build log on stderr)", code))
	fmt.Fprint(s.Err, tail(p.sub("out/build.log"), 4096))
	return exitEnv, nil
}

// tail is the last n bytes of a file, read without loading the rest.
func tail(path string, n int64) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil && info.Size() > n {
		_, _ = f.Seek(-n, io.SeekEnd)
	}
	b, _ := io.ReadAll(io.LimitReader(f, n))
	return string(b)
}

// testExe runs bin\test.exe on the copy with a scratch niagara_user_home, its streams in files, then judges
// testng-results.xml with stderr as the console cross-check; the exit code is a third check only.
func (h host) testExe(ctx context.Context, p *plan, s cli.Stdio) (int, error) {
	line := fmt.Sprintf(`set "niagara_user_home=%s" && cd /d %s && bin\test.exe %s -v:2 -output:%s 1>%s 2>%s & echo EXIT=!ERRORLEVEL!`,
		p.subWin("nuh"), p.home().win, p.module, p.subWin("out/testng"), p.subWin("out/stdout.txt"), p.subWin("out/stderr.txt"))
	out := &capped{max: maxCaptured}
	if _, err := h.run.Run(ctx, Cmd{Name: "cmd.exe", Args: []string{"/v:on", "/c", line}, Stdout: out}); err != nil {
		return -1, err
	}
	code := testngVerdict(p.sub("out/testng/testng-results.xml"), []string{p.sub("out/stderr.txt")}, s, nil)
	m := exitRe.FindStringSubmatch(out.buf.String())
	switch {
	case m == nil:
		row(s, "FAIL", "run: no exit status read")
	case m[1] != "0" && code == exitPass:
		row(s, "FAIL", "run: exit "+m[1]+" but results green")
	default:
		return code, nil
	}
	return exitFail, nil
}

// cleanup kills the Windows processes of this run, compares the install with its snapshot and deletes the
// scratch dir the run created (only that dir, unless --keep-home). It can only make code worse.
func (h host) cleanup(p *plan, snap snapshot, s cli.Stdio, code int) int {
	worse := func(c int) {
		if code == exitPass {
			code = c
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
	defer cancel()
	if p.windows {
		if c := h.killRun(ctx, p, s); c != exitPass {
			worse(c)
		}
	}
	if now, err := takeSnapshot(p.install.wsl); err != nil {
		row(s, "FAIL", "install unchanged: cannot re-read the install: "+err.Error())
		worse(exitFail)
	} else if d := snap.diff(now); d != "" {
		row(s, "FAIL", "install changed during the run (the run already happened): "+d)
		worse(exitFail)
	} else {
		row(s, "PASS", fmt.Sprintf("install unchanged (%d entries checked)", len(snap)))
	}
	if p.keepHome {
		row(s, "PASS", "cleanup: kept "+p.dir.win)
		return code
	}
	if err := os.RemoveAll(p.dir.wsl); err != nil {
		row(s, "FAIL", "cleanup: cannot delete "+p.dir.win+": "+err.Error())
		worse(exitEnv)
	}
	return code
}

// killRun finds the processes of this run by the scratch dir name, kills each tree by its PID
// (taskkill /PID <pid> /T /F, never by image name) and lists again to confirm none is left.
func (h host) killRun(ctx context.Context, p *plan, s cli.Stdio) int {
	script := fmt.Sprintf(procsScript, filepath.Base(p.dir.wsl))
	rows, ok := h.powershell(ctx, script)
	if !ok {
		row(s, "FAIL", "cleanup: cannot list the processes of this run")
		return exitEnv
	}
	pids := pidsOf(rows)
	if len(pids) == 0 {
		return exitPass
	}
	for _, pid := range pids {
		_, _ = h.run.Run(ctx, Cmd{Name: "cmd.exe", Args: []string{"/c", "taskkill", "/PID", pid, "/T", "/F"}})
	}
	rows, ok = h.powershell(ctx, script)
	if left := pidsOf(rows); !ok || len(left) > 0 {
		row(s, "FAIL", fmt.Sprintf("cleanup: %s of this run still running: %s", plural(len(left), "process", "processes"), strings.Join(left, " ")))
		return exitEnv
	}
	row(s, "PASS", fmt.Sprintf("cleanup: %s of this run killed", plural(len(pids), "process", "processes")))
	return exitPass
}

func pidsOf(rows []string) []string {
	var pids []string
	for _, r := range rows {
		if pid, ok := strings.CutPrefix(r, "PID "); ok {
			if _, err := strconv.Atoi(pid); err == nil {
				pids = append(pids, pid)
			}
		}
	}
	return pids
}
