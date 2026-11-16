package niagaratest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

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
	// --reuse-home: under the scratch parent, no real user home (D3), its marker bound to the kept run's log and
	// to this install (reusehome.go). Guard 3 and every later gate still apply.
	if cfg.reuseHome != "" {
		if why := h.checkReuse(p); why != "" {
			return refuse("%s", why)
		}
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
