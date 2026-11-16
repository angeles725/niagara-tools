// Package vendorfloor implements lint-vendor-floor: every vendored browser library under
// <rc-dir>/vendor is parsed at the Chromium 83 panel floor and its tokens are scanned for APIs above
// that floor. node + acorn are external tools (driver.js); everything else is decided here.
//
// One responsibility per file: apis.go the API table, messages.go every printed text, walk.go the
// vendored-file list, tools.go node/acorn resolution, driver.go + driver.js the acorn protocol,
// classify.go the verdicts, command.go the command line and exit codes.
package vendorfloor

import (
	"bufio"
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// Exit codes of lint-vendor-floor (unchanged from the bash tool).
const (
	ExitOK          = 0 // no FAIL row (WARN-only is still 0)
	ExitFail        = 1 // any FAIL row
	ExitEnv         = 3 // usage, a root that is not a directory, a vendor/ that cannot be looked for or listed whole, or a run that could not finish
	ExitUnavailable = 4 // node or acorn is missing: one SKIP row, never a pass (K20)
)

// Main implements `lint-vendor-floor <rc-dir> [--strict]` with the shipped API table.
func Main(args []string, s cli.Stdio) int { return run(args, s, Default) }

// run is Main over any API table. An rc dir with no vendored library needs no tool and is clean; a
// vendor/ that could not be looked for or listed whole (an rc dir the process may not search, a link that
// leads nowhere, an unreadable sub-directory) is ExitEnv before any tool is asked; a missing tool is one SKIP row and ExitUnavailable.
func run(args []string, s cli.Stdio, apis APIs) int {
	a, err := lint.ParseValued(lint.Valued, tool, usage, lint.ValueOptions{Strict: true}, dropLeadingEmpty(args))
	if err != nil {
		fmt.Fprint(s.Err, err.Error())
		return ExitEnv
	}
	rc := a.Roots[0]
	vendor := rc + "/vendor"
	switch present, fault := vendorDir(rc, vendor); {
	case fault != "":
		fmt.Fprintf(s.Err, envFaultLine, fault)
		return ExitEnv
	case !present:
		fmt.Fprintf(s.Err, noVendorNote, rc)
		return ExitOK
	}
	files, walkErrs := listVendored(vendor)
	for _, e := range walkErrs {
		fmt.Fprintf(s.Err, envFaultLine, fmt.Sprintf(incompleteWalk, vendor, e))
	}
	if len(walkErrs) > 0 {
		// Mutation: VF-unreadable -- scanning on over the files that were listed passes libraries nobody parsed
		return ExitEnv
	}
	if len(files) == 0 {
		fmt.Fprintf(s.Err, noFilesNote, rc)
		return ExitOK
	}
	node, ok := nodeBinary(s)
	if !ok {
		// Mutation: VF3 -- treat a missing node as a clean pass (exit 0) so VF3 no longer sees exit 4
		fmt.Fprintln(s.Out, skipRow(rc, nodeMissing))
		return ExitUnavailable
	}
	if err := apis.Validate(); err != nil {
		fmt.Fprintf(s.Err, envFaultLine, err)
		return ExitEnv
	}
	return scan(scanJob{rc: rc, files: files, node: node, acorn: acornSpec(s), apis: apis, strict: a.Strict}, s)
}

// scanJob is one resolved run: what to scan, with which tools and table.
type scanJob struct {
	rc, node, acorn string
	files           []string
	apis            APIs
	strict          bool
}

// scan runs the driver and prints each file's rows as they arrive. Row paths are relative to the rc dir.
func scan(j scanJob, s cli.Stdio) int {
	out := bufio.NewWriter(s.Out)
	defer out.Flush()
	failed := false
	req := request{Acorn: j.acorn, ECMAVersion: FloorECMAVersion, Names: j.apis.Names()}
	end, err := drive(j.node, req, j.files, s.Err, func(rec record) {
		if rec.Index < 0 || rec.Index >= len(j.files) {
			return
		}
		for _, f := range classify(rec, j.files[rec.Index][len(j.rc)+1:], j.apis, j.strict) {
			fmt.Fprintln(out, f.Row())
			failed = failed || f.Severity == lint.Fail
		}
	})
	out.Flush() // rows first: a caller that merges the streams reads them before the stderr line
	switch {
	case err != nil:
		fmt.Fprintf(s.Err, envFaultLine, err)
		return ExitEnv
	case end.Type == recFault:
		fmt.Fprintf(s.Err, envFaultLine, end.Message)
		return ExitEnv
	case end.Type == recUnavailable:
		fmt.Fprintln(out, skipRow(j.rc, fmt.Sprintf(acornMissing, j.acorn)))
		return ExitUnavailable
	case failed:
		return ExitFail
	}
	return ExitOK
}

// dropLeadingEmpty drops the empty arguments before the first directory, as the bash tool's
// `[ -z "$RC" ] || usage; RC="$1"` did: an empty argument leaves the directory unset.
func dropLeadingEmpty(args []string) []string {
	var kept []string
	seen := false
	for _, arg := range args {
		if arg == "" && !seen {
			continue
		}
		seen = seen || (arg != "" && arg[0] != '-')
		kept = append(kept, arg)
	}
	return kept
}
