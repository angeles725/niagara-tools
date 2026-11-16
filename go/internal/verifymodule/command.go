// Package verifymodule is toolbelt/verify-module.sh, THE gate for built N4 module jars: one row per check
// per jar and a summary line, the coverage subcommand and the --plano mode. It follows the bash it replaces
// byte for byte except where that bash failed open or misread its input (the phase-B list in the
// go-verify-module retro): each such fix is a declared difference in a golden (testdata/truth-*.txt and a
// "declared difference:" line) or a test comment. Its text model is shell.go, it reads XML through xmlsrc,
// gradle.kts through ktssrc and Java through javasrc, and each check family has a file of its own.
package verifymodule

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// usage is what `sed -n '2,20p' "$0" | sed 's/^# \{0,1\}//'` printed: lines 2-20 of the bash script. The
// --expected-signer lines under `signed` were added to it with the option.
//
//go:embed usage.txt
var usage string

const usageCoverage = "usage: verify-module.sh coverage <npass> <nfail> <nwarn> <nskip>\n"

// run is one gate run: its flags, the jar being checked and the counts the summary line prints.
type run struct {
	s                          cli.Stdio
	target, src                string
	expected                   string // --expected-signer: the alias the jar must be signed with, "" for any
	stored, strict             bool
	jar                        *jarFile
	failed                     bool
	npass, nfail, nskip, nwarn int
	walkErrs                   []string // the walk error of every "cannot list every file" line printed
	unread                     string   // the "cannot read" line of an input no check may give a verdict on
	texts                      map[string]grepText
	views                      map[string]*javaView
}

// check is one check_* function: false is its `return 1` (a FAIL that fails the run).
type check func(*run) bool

// Main is verify-module.sh. `coverage` and `--plano` are read before any flag.
func Main(args []string, s cli.Stdio) int {
	if len(args) >= 1 && args[0] == "coverage" {
		return coverage(args[1:], s)
	} else if len(args) >= 1 && args[0] == "--plano" {
		return plano(args[1:], s)
	}
	return gate(args, s)
}

// gate is the jar loop: flags, argument checks, every check on every jar, then the summary line.
func gate(args []string, s cli.Stdio) int {
	r := &run{s: s, texts: map[string]grepText{}, views: map[string]*javaView{}}
	jars, code := r.parseFlags(args)
	if code != proceed {
		return code
	}
	if len(jars) == 0 {
		fmt.Fprint(s.Err, usage)
		return 2
	}
	if r.src != "" && !lint.IsDir(r.src) {
		fmt.Fprintf(s.Err, "verify-module: --src is not a directory: %s\n", r.src)
		return 2
	}
	for _, j := range jars {
		if !strings.HasSuffix(j, ".jar") {
			fmt.Fprintf(s.Err, "verify-module: not a jar: %s\n", j)
			return 2
		}
	}
	for _, j := range jars {
		jf, msg := openJar(j)
		if msg != "" {
			fmt.Fprintf(s.Err, "verify-module: %s\n", msg)
			return 3
		}
		r.jar = jf
		for _, c := range checks {
			if !c.run(r) {
				r.failed = true
			}
			if r.unread != "" {
				fmt.Fprintln(s.Err, r.unread)
				jf.close()
				return 3
			}
		}
		jf.close()
	}
	verdict := "ALL PASS"
	if r.failed {
		verdict = "FAILED"
	}
	if len(r.walkErrs) > 0 {
		verdict = "INCOMPLETE (a --src walk failed)"
	}
	fmt.Fprintf(s.Out, "verify-module: %d passed, %d failed, %d skipped, %d warned -> %s\n",
		r.npass, r.nfail, r.nskip, r.nwarn, verdict)
	switch {
	case len(r.walkErrs) > 0:
		return 3
	case r.failed:
		return 1
	}
	return 0
}

// cannotRead ends the run after the current check: an input the gate cannot read gets no verdict, exit 3,
// as an unreadable jar does (the bash read it as empty or grepped it into a verdict). what names the input.
func (r *run) cannotRead(what, reason string) { r.envError("cannot read " + what + ": " + reason) }

// envError ends the run after the current check with "verify-module: <msg>", exit 3: an input the gate
// cannot read or parse (a malformed XML file names its xmlsrc position).
func (r *run) envError(msg string) {
	if r.unread == "" {
		r.unread = "verify-module: " + msg
	}
}

// row prints one report row for the current jar and counts it.
func (r *run) row(status, check, detail string) {
	fmt.Fprintf(r.s.Out, "%-4s  %-9s  %s  %s\n", status, check, r.jar.path, detail)
	switch status {
	case "PASS":
		r.npass++
	case "FAIL":
		r.nfail++
	case "SKIP":
		r.nskip++
	case "WARN":
		r.nwarn++
	}
}

// coverage is the coverage subcommand: applicable = pass + fail + warn, covered = pass, integer-tenths
// rounding. Every count is a decimal integer; one too large for int64 is a usage error naming it.
func coverage(args []string, s cli.Stdio) int {
	if len(args) != 4 {
		fmt.Fprint(s.Err, usageCoverage)
		return 2
	}
	for _, a := range args {
		if a == "" || strings.Trim(a, "0123456789") != "" {
			fmt.Fprint(s.Err, usageCoverage)
			return 2
		}
	}
	var v [4]int64
	for i, a := range args {
		n, ok := intOperand(a)
		if !ok {
			fmt.Fprintf(s.Err, "verify-module: coverage: not an integer: %s\n%s", a, usageCoverage)
			return 2
		}
		v[i] = n
	}
	if a := v[0] + v[1] + v[2]; a == 0 {
		fmt.Fprintln(s.Out, "N/A")
	} else {
		t := (1000*v[0] + a/2) / a
		fmt.Fprintf(s.Out, "%d.%d\n", t/10, t%10)
	}
	return 0
}
