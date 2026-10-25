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
	defaultJDK8    = "/usr/lib/jvm/java-8-openjdk-amd64"
)

// copiedDirs are the install dirs the home copy takes whole; modules/ is copied jar by jar (copyJars).
var copiedDirs = []string{"bin", "etc", "jre", "lib", "security", "defaults"}

var (
	partRe    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*(:[A-Za-z0-9][A-Za-z0-9_.-]*)*$`)
	moduleRe  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	versionRe = regexp.MustCompile(`^[0-9]+(\.[0-9]+)+$`)
)

// runConfig is the parsed command line of run.
type runConfig struct {
	install, gradleRoot, part, module, scratch string
	jdk8, gradleUserHome, logDir               string
	timeout                                    time.Duration
	keepHome, verifyLow, restoreRoot           bool
	excludeJars                                []string // --exclude-jar globs, in flag order
	allow                                      IgnoredAllowance
}

// plan is a validated run: the install, %TEMP%, %USERPROFILE%, the scratch parent and, once created, the
// scratch dir.
type plan struct {
	runConfig
	install, temp, profile, parent, dir winDir
	windows                             bool   // a Windows-side step started: cleanup must look for its processes
	sysPropsSum                         string // --module-verification low: sha256 of the install's system.properties
	excluded                            []jarExclusion
	root                                rootSnapshot // --gradle-root's git status before the copy
	rootTop                             string       // its worktree top, where the status paths start
	rows                                *lineRing    // the last rows the run printed, for the run log
	logDirSet                           bool         // --log-dir was given: a green run keeps a log too
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
	return host{run: execRunner{}, mnt: "/mnt", home: home, jdk8: defaultJDK8, state: stateHome(home)}.runMain(ctx, append([]string{"run"}, args...), s)
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
	logDirSet := cfg.logDir != ""
	if !logDirSet && h.state != "" {
		cfg.logDir = filepath.Join(h.state, "run-niagara-test")
	} else if logDirSet {
		if cfg.logDir, err = filepath.Abs(cfg.logDir); err != nil {
			return usageError(s, "run: --log-dir: "+err.Error())
		}
	}
	rows := newLineRing(diagRowLines, diagLineCap)
	s = tee(s, rows)
	if cfg.jdk8 == "" {
		cfg.jdk8 = h.jdk8
	}
	if cfg.gradleUserHome == "" && h.home != "" {
		cfg.gradleUserHome = filepath.Join(h.home, ".gradle")
	}
	p, code := h.validate(ctx, cfg, s)
	if code >= 0 {
		return code
	}
	p.rows, p.logDirSet = rows, logDirSet
	return h.execute(ctx, p, s)
}

// stateHome is $XDG_STATE_HOME when it is absolute, else ~/.local/state; "" when neither is known.
func stateHome(home string) string {
	if v := os.Getenv("XDG_STATE_HOME"); filepath.IsAbs(v) {
		return v
	}
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".local", "state")
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
	fs.StringVar(&c.jdk8, "jdk8", "", "")
	fs.StringVar(&c.gradleUserHome, "gradle-user-home", "", "")
	fs.StringVar(&c.logDir, "log-dir", "", "")
	fs.DurationVar(&c.timeout, "timeout", defaultTimeout, "")
	fs.BoolVar(&c.keepHome, "keep-home", false, "")
	fs.BoolVar(&c.restoreRoot, "restore-gradle-root", false, "")
	fs.Var(verificationFlag{&c.verifyLow}, "module-verification", "")
	fs.Var(excludeFlag{&c.excludeJars}, "exclude-jar", "")
	var count, why string
	var names []string
	fs.Var(namesFlag{&names}, "allow-ignored", "")
	fs.StringVar(&count, "allow-ignored-count", "", "")
	fs.StringVar(&why, "allow-ignored-reason", "", "")
	if err := fs.Parse(args); err != nil {
		return c, fmt.Errorf("run: %v", err)
	}
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	var err error
	if c.allow, err = parseRunAllowance(count, set["allow-ignored-count"], why, set["allow-ignored-reason"], names); err != nil {
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
	// The gradle-root gate runs git -C on it: an absolute path, never one resolved against a later cwd.
	if p.gradleRoot, err = filepath.Abs(cfg.gradleRoot); err != nil {
		return refuse("--gradle-root %s: %v", cfg.gradleRoot, err)
	}
	if info, err := os.Stat(filepath.Join(cfg.jdk8, "bin", "javac")); err != nil || !info.Mode().IsRegular() {
		return refuse("--jdk8 %s has no bin/javac (a WSL JDK 8)", cfg.jdk8)
	}
	if info, err := os.Stat(p.install.wsl + "/" + sysPropsRel); cfg.verifyLow && (err != nil || !info.Mode().IsRegular()) {
		return refuse("--module-verification low: --install %s has no %s to lower in its copy", p.install.win, sysPropsRel)
	}
	if len(cfg.excludeJars) > 0 {
		if p.excluded, err = matchExclusions(p.install.wsl+"/modules", cfg.excludeJars); err != nil {
			return refuse("--install %s: %v", p.install.win, err)
		}
		for _, x := range p.excluded {
			if len(x.jars) == 0 {
				row(s, "FAIL", fmt.Sprintf(`copy: --exclude-jar %s matched no jar in %s\modules`, x.glob, p.install.win))
				return nil, exitUsage
			}
		}
	}
	if !filepath.IsAbs(cfg.gradleUserHome) {
		return refuse("--gradle-user-home %q is not an absolute WSL path", cfg.gradleUserHome)
	}
	// A LocalSigningProfile whose refresh outlasts the key it mints re-mints it at every signing: the jars then
	// fail signature verification and the NRE drops them (signingprofile.go). Refused before anything is copied.
	if why := signingProfileRefusal(p.gradleRoot, cfg.gradleUserHome); why != "" {
		return refuse("%s", why)
	}
	for _, v := range []struct {
		name string
		dst  *winDir
	}{{"%TEMP%", &p.temp}, {"%USERPROFILE%", &p.profile}} {
		out := &capped{max: maxCaptured}
		if code, err := h.run.Run(ctx, Cmd{Name: "cmd.exe", Args: []string{"/c", "echo " + v.name}, Stdout: out}); err != nil || code != 0 {
			fmt.Fprintf(s.Err, "%s: run: cannot read %s through cmd.exe (exit %d, %v)\n", name, v.name, code, err)
			return nil, exitEnv
		}
		if *v.dst, err = h.canonical(v.name, strings.TrimSpace(out.buf.String())); err != nil {
			fmt.Fprintf(s.Err, "%s: run: %v\n", name, err)
			return nil, exitEnv
		}
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
	svcs := services(rows)
	for _, r := range rows {
		switch {
		case strings.HasPrefix(r, "SVC\t"):
		case strings.HasPrefix(r, "UNREAD\t"):
			pass, refusal := attribute(r, svcs, p.install.win)
			if refusal != "" {
				return refuse("%s", refusal)
			}
			row(s, "PASS", pass)
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

// scanIgnored names the disabled test-methods of the part's registered types from the source before the build,
// under every allowance, and prints them. A refusal stops the run (exit 3) unless the count fallback is given:
// then it is a NOTE row and the count is judged as case (i). -1 goes on.
func scanIgnored(p *plan, s cli.Stdio) int {
	src, err := disabledInSource(filepath.Join(p.gradleRoot, filepath.FromSlash(strings.ReplaceAll(p.part, ":", "/"))))
	switch {
	case err != nil && p.allow.Count == 0:
		row(s, "FAIL", "source: "+err.Error())
		return exitEnv
	case err != nil:
		row(s, "NOTE", "source: "+err.Error()+"; the count fallback applies (case (i): the source scan refused)")
		p.allow.Refused = err.Error()
	default:
		names := ""
		if len(src) > 0 {
			names = ": " + named(src)
		}
		row(s, "PASS", "source: the registered types disable "+plural(len(src), "test-method", "test-methods")+names)
		p.allow.Source, p.allow.Scanned = src, true
	}
	return -1
}

// execute snapshots the gradle root, the install and the real user homes, creates the scratch dir, then copies,
// builds, runs and judges; cleanup runs on every path out, a panic included (the panic goes on after it).
func (h host) execute(ctx context.Context, p *plan, s cli.Stdio) (code int) {
	if code := scanIgnored(p, s); code >= 0 {
		return code
	}
	var ok bool
	if p.rootTop, ok = h.rootTop(ctx, p.gradleRoot); !ok {
		row(s, "FAIL", "gradle root is not a git worktree: cannot verify it")
		return exitEnv
	}
	root, err := h.rootStatus(ctx, p.gradleRoot, p.rootTop)
	if err != nil {
		row(s, "FAIL", "gradle root: cannot snapshot it: "+err.Error())
		return exitEnv
	}
	p.root = root
	snap, err := takeSnapshot(p.install.wsl)
	if err != nil {
		fmt.Fprintf(s.Err, "%s: run: cannot snapshot the install: %v\n", name, err)
		return exitEnv
	}
	users, err := userHomeSnapshot(p.profile)
	if err != nil {
		fmt.Fprintf(s.Err, "%s: run: cannot snapshot the real user homes: %v\n", name, err)
		return exitEnv
	}
	if p.verifyLow {
		if p.sysPropsSum, err = fileSum(p.install.wsl + "/" + sysPropsRel); err != nil {
			fmt.Fprintf(s.Err, "%s: run: cannot hash the install's %s: %v\n", name, sysPropsRel, err)
			return exitEnv
		}
	}
	dir, err := os.MkdirTemp(p.parent.wsl, "run-niagara-test-")
	if err != nil {
		fmt.Fprintf(s.Err, "%s: run: %v\n", name, err)
		return exitEnv
	}
	win, _ := h.toWin(dir)
	p.dir = winDir{dir, win}
	defer func() { code = h.cleanup(p, snap, users, s, code) }()
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	for _, step := range []struct {
		name string
		do   func(context.Context, *plan, cli.Stdio) (int, error)
	}{{"copy", h.copyHome}, {"module verification", h.lowerVerification}, {"build", h.build}, {"run", h.testExe}} {
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
// modules\ but those an --exclude-jar glob matched and those whose module name has a dot (a stray
// `<name>.77-rt.jar` makes every boot rebuild the registry). The source is only read; licenses travel untouched.
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
		if e.IsDir() || !strings.HasSuffix(n, ".jar") || isExcluded(p.excluded, n) {
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
	for _, x := range p.excluded {
		row(s, "PASS", fmt.Sprintf("copy: excluded %s matching %s: %s", plural(len(x.jars), "jar", "jars"), x.glob, listed(x.jars, maxExcludedListed)))
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
// every home the signing plugin writes (niagara_user_home, user.home) is scratch. Gradle's own cache is the
// caller's (GRADLE_USER_HOME: no 7.6 download per run), and its JDK the WSL JDK 8 (-P overrides a client
// gradle.properties that names a Windows JDK path, which WSL gradle cannot parse).
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
			"-Pniagara_user_home=" + p.sub("nuh"), "-PniagaraPluginVersion=" + v, "-Porg.gradle.java.installations.paths=" + p.jdk8,
			":" + p.part + ":jar", ":" + p.part + ":moduleTestJar"},
		Env: []string{"JAVA_HOME=" + p.jdk8, "GRADLE_USER_HOME=" + p.gradleUserHome, "JAVA_TOOL_OPTIONS=-Duser.home=" + p.sub("gradle-uh")}})
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

// testExe runs the copy's bin\test.exe straight from WSL, with the copy as cwd and its streams in files, then
// judges testng-results.xml with stderr as the console cross-check; the exit code is a third check only (WSL
// reports the Windows exit -1 as 255). The scratch niagara_user_home reaches the NRE through the interop
// environment (WSLENV niagara_user_home/w): a cmd.exe `set "…"` does not, as WSL re-quotes its inner quotes.
// D3 still gates it: the NRE must have created etc\ and security\ in the scratch home, and cleanup compares
// the real user homes.
func (h host) testExe(ctx context.Context, p *plan, s cli.Stdio) (int, error) {
	files := map[string]*os.File{}
	for _, n := range []string{"stdout", "stderr"} {
		f, err := os.Create(p.sub("out/" + n + ".txt"))
		if err != nil {
			return -1, err
		}
		defer f.Close()
		files[n] = f
	}
	wslenv := "niagara_user_home/w"
	if v := os.Getenv("WSLENV"); v != "" {
		wslenv = v + ":" + wslenv
	}
	exit, err := h.run.Run(ctx, Cmd{Name: p.home().wsl + "/bin/test.exe", Dir: p.home().wsl,
		Args:   []string{p.module, "-v:2", "-output:" + p.subWin("out/testng")},
		Env:    []string{"WSLENV=" + wslenv, "niagara_user_home=" + p.subWin("nuh")},
		Stdout: files["stdout"], Stderr: files["stderr"]})
	if err != nil {
		return -1, err
	}
	code := testngVerdict(p.sub("out/testng/testng-results.xml"), []string{p.sub("out/stderr.txt")}, p.allow, s, nil)
	if code != exitPass {
		signatureHint(p, s, p.sub("out/stderr.txt"), p.sub("out/stdout.txt"))
	}
	switch {
	case !isDir(p.sub("nuh/etc")) || !isDir(p.sub("nuh/security")):
		row(s, "FAIL", "scratch niagara_user_home was not used (no etc/ and security/ in it)")
	case exit != 0 && code == exitPass:
		row(s, "FAIL", fmt.Sprintf("run: exit %d but results green", exit))
	default:
		return code, nil
	}
	return exitFail, nil
}

// cleanup kills the Windows processes of this run, compares the install with its snapshot, deletes the
// scratch dir the run created (only that dir, unless --keep-home), compares the real user homes with theirs,
// then the gradle root's git status with its own (gradleroot.go), and last writes the run log (diagnostics.go)
// from the evidence read before the delete. It can only make code worse.
func (h host) cleanup(p *plan, snap, users snapshot, s cli.Stdio, code int) int {
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
	if p.verifyLow && !checkInstallSysProps(p, s) {
		worse(exitFail)
	}
	ev := gatherEvidence(p) // before the scratch dir goes: the run log is written last, with the final code
	if p.keepHome {
		row(s, "PASS", "cleanup: kept "+p.dir.win)
	} else if err := os.RemoveAll(p.dir.wsl); err != nil {
		row(s, "FAIL", "cleanup: cannot delete "+p.dir.win+": "+err.Error())
		worse(exitEnv)
	}
	if now, err := userHomeSnapshot(p.profile); err != nil {
		row(s, "FAIL", "real user homes unchanged: cannot re-read them: "+err.Error())
		worse(exitFail)
	} else if d := users.diff(now); d != "" {
		row(s, "FAIL", "real user home changed during the run: "+d)
		worse(exitFail)
	} else {
		row(s, "PASS", fmt.Sprintf("real user homes unchanged (%d entries checked)", len(users)))
	}
	if !h.checkGradleRoot(ctx, p, s) {
		worse(exitFail)
	}
	if !h.writeRunLog(p, ev, code, s) {
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
