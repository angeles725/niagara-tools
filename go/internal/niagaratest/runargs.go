package niagaratest

import (
	"flag"
	"fmt"
	"io"
)

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
	fs.StringVar(&c.reuseHome, "reuse-home", "", "")
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
	case set["reuse-home"] && c.reuseHome == "":
		return c, fmt.Errorf("run: --reuse-home needs a dir")
	case c.reuseHome != "" && len(c.excludeJars) > 0:
		return c, fmt.Errorf("run: --reuse-home takes no --exclude-jar: the kept home keeps the copy the run that made it took")
	case c.reuseHome != "" && c.verifyLow:
		return c, fmt.Errorf("run: --reuse-home takes no --module-verification: the kept copy keeps the verification mode the run that made it set")
	}
	return c, nil
}
