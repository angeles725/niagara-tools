// Package verifymodule is toolbelt/verify-module.sh, THE gate for built N4 module jars: one row per check
// per jar and a summary line, the coverage subcommand and the --plano mode. It follows the bash it replaces
// byte for byte, including where that bash fails open (the phase-B list in the go-verify-module retro); its
// text model is shell.go and each check family has a file of its own.
package verifymodule

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// usage is what `sed -n '2,20p' "$0" | sed 's/^# \{0,1\}//'` printed: lines 2-20 of the bash script.
//
//go:embed usage.txt
var usage string

const usageCoverage = "usage: verify-module.sh coverage <npass> <nfail> <nwarn> <nskip>\n"

// run is one gate run: its flags, the jar being checked and the counts the summary line prints.
type run struct {
	s                          cli.Stdio
	target, src                string
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

// checks is the bash loop's order.
var checks = []check{
	(*run).bytecode, (*run).signed, (*run).types, (*run).baja, (*run).storedCheck, (*run).typeCount,
	(*run).facets, (*run).facetPresence, (*run).ordLiteral, (*run).rcBackup, (*run).palette, (*run).wbScaffold,
	(*run).phantomDep, (*run).moduleTest, (*run).crossModuleType, (*run).transientOperator,
	(*run).compact3Imports, (*run).subscriptionLeak, (*run).slotWall,
}

// Main is verify-module.sh. `coverage` and `--plano` are read before any flag. When bash arithmetic
// failed inside either block (an operand such as 08 or 2.5), bash abandoned that block and went on to
// parse the rest of the arguments as the gate's, so this does too.
func Main(args []string, s cli.Stdio) int {
	if len(args) >= 1 && args[0] == "coverage" {
		if code, ok := coverage(args[1:], s); ok {
			return code
		}
		args = args[1:]
	} else if len(args) >= 1 && args[0] == "--plano" {
		if code, ok := plano(args[1:], s); ok {
			return code
		}
		args = args[1:]
	}
	return gate(args, s)
}

// gate is the jar loop: flags, argument checks, every check on every jar, then the summary line.
func gate(args []string, s cli.Stdio) int {
	r := &run{s: s, texts: map[string]grepText{}, views: map[string]*javaView{}}
	var jars []string
	for len(args) > 0 {
		switch a := args[0]; {
		case a == "--target-version" || a == "--src":
			if len(args) < 2 {
				fmt.Fprint(s.Err, usage)
				return 2
			}
			if a == "--src" {
				r.src = args[1]
			} else {
				r.target = args[1]
			}
			args = args[1:]
		case a == "--stored":
			r.stored = true
		case a == "--strict":
			r.strict = true
		case a == "-h" || a == "--help":
			fmt.Fprint(s.Out, usage)
			return 0
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(s.Err, "verify-module: unknown flag %s\n%s", a, usage)
			return 2
		default:
			jars = append(jars, a)
		}
		args = args[1:]
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
			if !c(r) {
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
// rounding. ok is false when bash arithmetic failed on an operand (the gate then runs on the rest).
func coverage(args []string, s cli.Stdio) (code int, ok bool) {
	if len(args) != 4 {
		fmt.Fprint(s.Err, usageCoverage)
		return 2, true
	}
	for _, a := range args {
		if a == "" || strings.Trim(a, "0123456789") != "" {
			fmt.Fprint(s.Err, usageCoverage)
			return 2, true
		}
	}
	var v [3]int64
	for i := range v {
		n, err := bashInt(args[i])
		if err != nil {
			fmt.Fprintf(s.Err, "verify-module: %s\n", err.msg)
			return 0, false
		}
		v[i] = n
	}
	if a := v[0] + v[1] + v[2]; a == 0 {
		fmt.Fprintln(s.Out, "N/A")
	} else {
		t := (1000*v[0] + a/2) / a
		fmt.Fprintf(s.Out, "%d.%d\n", t/10, t%10)
	}
	return 0, true
}
