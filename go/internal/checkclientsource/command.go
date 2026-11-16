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
	ExitEnv   = 3 // module root missing, BUILD-STATE file missing or unreadable, git absent
)

const tool = "check-client-source"

// Why a BUILD-STATE file could not be read (stderr, after the tool prefix).
const (
	buildStateMissing    = "BUILD-STATE not found: %s"
	buildStateUnreadable = "cannot read %s: %v"
)

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
	if !o.haveDeclared && o.haveState {
		// Mutation: CS-empty-path -- reading only a non-empty path takes `--build-state ""` for no declaration (WARN undeclared, exit 0)
		if o.module == "" || o.buildState == "" {
			return usage()
		}
		text, fault := readBuildState(o.buildState)
		if fault != "" {
			fmt.Fprintf(s.Err, "%s: %s\n", tool, fault)
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

// readBuildState reads a BUILD-STATE file; fault is the reason it could not be read, "" when text holds
// its content. A file that exists and cannot be read is a fault like a missing one: read as empty it
// would declare nothing, and the check would WARN "undeclared" over a checkout nobody compared.
func readBuildState(path string) (text, fault string) {
	if st, err := os.Stat(path); err != nil || !st.Mode().IsRegular() {
		return "", fmt.Sprintf(buildStateMissing, path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		// Mutation: CS-unreadable -- returning the empty text with no fault reads an unreadable BUILD-STATE as undeclared (WARN, exit 0)
		return "", fmt.Sprintf(buildStateUnreadable, path, err)
	}
	if text, fault = decodeBuildState(path, b); fault != "" {
		return "", fault
	}
	return text, strayBOM(path, text)
}
