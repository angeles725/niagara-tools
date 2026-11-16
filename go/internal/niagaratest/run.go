package niagaratest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

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
	copyStep := h.copyHome
	if p.reuseHome == "" {
		dir, err := os.MkdirTemp(p.parent.wsl, "run-niagara-test-")
		if err != nil {
			fmt.Fprintf(s.Err, "%s: run: %v\n", name, err)
			return exitEnv
		}
		win, _ := h.toWin(dir)
		p.dir = winDir{dir, win}
	} else {
		copyStep = h.keptHome
	}
	defer func() { code = h.cleanup(p, snap, users, s, code) }()
	if err := h.takeHome(p); err != nil {
		row(s, "FAIL", "marker: cannot bind "+p.dir.win+" to this run: "+err.Error())
		return exitEnv
	}
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	for _, step := range []struct {
		name string
		do   func(context.Context, *plan, cli.Stdio) (int, error)
	}{{"copy", copyStep}, {"module verification", h.lowerVerification}, {"build", h.build}, {"run", h.testExe}} {
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
