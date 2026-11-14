// Package checkclientsource answers "is this module root the declared SOURCE OF TRUTH checkout?"
// (scripts/check-client-source.sh): BUILD-STATE declares, per module,
// `source_of_truth: <path>@<branch>@<commit>` (or `unknown`), and the check compares a module root's
// checkout against it. It is the version-control half of the build preflight; local refs only, never a
// fetch. [ev: retro client-source-of-truth Δ1] [ev: retro client-source-of-truth Δ2]
//
// One file per concern: args.go (the command line), envelope.go (the BUILD-STATE read), declared.go (the
// declared value), git.go (the checkout), check.go (the rules and their rows). This file composes them.
package checkclientsource

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Exit codes of check-client-source (unchanged from the bash tool).
const (
	ExitOK    = 0 // no FAIL row
	ExitFail  = 1 // any FAIL row
	ExitUsage = 2 // usage, including a malformed declared value
	ExitEnv   = 3 // module root missing, BUILD-STATE file missing, git absent
)

const tool = "check-client-source"

//go:embed usage.txt
var helpText string

const usageText = "usage: check-client-source.sh [--declared <path>@<branch>@<commit>|unknown] <module-root>\n" +
	"       check-client-source.sh --build-state <BUILD-STATE.md> --module <MOD> <module-root>\n"

// Main implements `check-client-source [--declared <value>] <module-root>` and
// `check-client-source --build-state <BUILD-STATE.md> --module <MOD> <module-root>`.
func Main(args []string, s cli.Stdio) int {
	usage := func() int {
		fmt.Fprint(s.Err, usageText)
		return ExitUsage
	}
	o, ok := parseArgs(args)
	switch {
	case o.help:
		fmt.Fprint(s.Out, helpText)
		return ExitOK
	case !ok:
		return usage()
	}
	if st, err := os.Stat(o.root); err != nil || !st.IsDir() {
		fmt.Fprintf(s.Err, "%s: module root not found: %s\n", tool, o.root)
		return ExitEnv
	}
	if _, err := exec.LookPath("git"); err != nil {
		fmt.Fprintf(s.Err, "%s: git not found in PATH\n", tool)
		return ExitEnv
	}
	raw := o.declared
	if !o.haveDeclared && o.buildState != "" {
		if o.module == "" {
			return usage()
		}
		text, found := readBuildState(o.buildState, s)
		if !found {
			fmt.Fprintf(s.Err, "%s: BUILD-STATE not found: %s\n", tool, o.buildState)
			return ExitEnv
		}
		raw = SourceOfTruth(text, o.module)
	}
	decl, shape := parseDeclared(raw)
	switch shape {
	case shapeUsage:
		return usage()
	case shapeMalformed:
		fmt.Fprintf(s.Err, "%s: malformed source_of_truth (want <path>@<branch>@<commit> or unknown): %s\n", tool, raw)
		return ExitUsage
	}
	r := &report{out: s.Out}
	check(o.root, decl, checkout{dir: o.root, stderr: s.Err}, r)
	if r.failed {
		return ExitFail
	}
	return ExitOK
}

// readBuildState reads a BUILD-STATE file; found is false when path is not a regular file. A file that
// exists but cannot be read is named on stderr and read as empty (no field: the module is undeclared),
// as the bash tool's awk did.
func readBuildState(path string, s cli.Stdio) (text string, found bool) {
	if st, err := os.Stat(path); err != nil || !st.Mode().IsRegular() {
		return "", false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(s.Err, "%s: cannot read %s: %v\n", tool, path, err)
	}
	return string(b), true
}
