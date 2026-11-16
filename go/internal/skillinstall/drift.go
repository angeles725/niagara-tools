package skillinstall

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/installtools"
)

const driftTool = "check-skill-drift"

// DriftMain implements `check-skill-drift [--skill <name>] [--harness <name|all> | --all] [--home <dir>]`:
// one PASS or FAIL row per installed file saying whether it matches the tracked launcher (or, for a
// prompt file, its rendered template); an absent optional harness is a SKIP row. It never writes.
func DriftMain(args []string, s cli.Stdio) int {
	fail := func(code int, format string, a ...any) int {
		fmt.Fprintf(s.Err, driftTool+": "+format+"\n", a...)
		return code
	}
	usage := func() int {
		fmt.Fprint(s.Err, "usage: check-skill-drift.sh [--skill <name>] [--harness <"+harnessNames("|")+"> | --all] [--home <dir>]\n")
		return exitUsage
	}
	launcher, err := installtools.LauncherFrom(s)
	if err != nil {
		return fail(exitEnv, "%v", err)
	}
	o, ok, message := parseDrift(args, s.Env("HOME"))
	switch {
	case !ok:
		if message != "" {
			fail(exitUsage, "%s", message)
		}
		return usage()
	case o.help:
		header, err := launcher.Header()
		if err != nil {
			return fail(exitEnv, "cannot read %s: %s", launcher.Path, installtools.Reason(err))
		}
		fmt.Fprint(s.Out, header)
		return exitOK
	}
	installer := launcher.Sibling(installerScript)
	if isFile, _ := installtools.IsFile(installer); !isFile {
		return fail(exitEnv, "installer not found: %s", installer)
	}
	table, err := readTable(installer)
	if err != nil {
		return fail(exitEnv, "cannot read %s: %s", installer, installtools.Reason(err))
	}
	if len(table) == 0 {
		return fail(exitEnv, "no SKILLS table in %s", installer)
	}
	rel, known := lookup(table, o.skill)
	if !known {
		fail(exitUsage, "unknown skill: %s", o.skill)
		return usage()
	}
	selected, known := selectHarnesses(o.harness)
	if !known {
		fail(exitUsage, "unknown harness: %s (expected %s)", o.harness, harnessChoices())
		return usage()
	}
	tracked := launcher.RepoRoot() + "/" + rel
	if isFile, _ := installtools.IsFile(tracked); !isFile {
		return fail(exitEnv, "tracked launcher not found: %s", tracked)
	}
	if o.home == "" {
		return fail(exitEnv, "no home directory (pass --home)")
	}
	code := exitOK
	for _, h := range selected {
		c := checkHarness(driftCheck{o: o, h: h, rel: rel, tracked: tracked}, s)
		if c == exitEnv {
			return exitEnv
		}
		if c != exitOK {
			code = c
		}
	}
	return code
}
