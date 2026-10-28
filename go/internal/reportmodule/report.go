package reportmodule

import (
	"fmt"
	"io"
	"strconv"
)

// Exit codes of report-module (unchanged from the bash tool).
const (
	ExitClean      = 0 // zero FAIL rows and the floor checks ran (or --lint-only / --first-deploy)
	ExitFail       = 1 // any FAIL row
	ExitUsage      = 2
	ExitEnv        = 3 // a pre-check failed or a member had an env fault (an ERROR row)
	ExitIncomplete = 4 // a floor check (verify-module, schema-risk) SKIPped
)

// Row statuses the summary counts; any other status (ERROR, INFO, ...) is printed and not counted.
const (
	pass     = "PASS"
	failed   = "FAIL"
	warn     = "WARN"
	skip     = "SKIP"
	advisory = "ADVISORY"
	errored  = "ERROR"
)

// report is one run: the rows printed so far, their counts and the verdict inputs.
type report struct {
	out, errw io.Writer
	o         options
	x         runner
	getenv    func(string) string
	n         map[string]int
	hadFail   bool // a FAIL row
	hadEnv    bool // a member env fault (an ERROR row)
	floorSkip bool // verify-module or schema-risk did not run (BUILD-LOOP §5 floor)
}

// emit prints `<label>  <ST>  <check>  <detail>` and counts it.
func (r *report) emit(label, st, check, detail string) {
	fmt.Fprintf(r.out, "%s  %s  %s  %s\n", label, st, check, detail)
	r.count(st)
}

func (r *report) count(st string) {
	switch st {
	case pass, warn, skip, advisory:
		r.n[st]++
	case failed:
		r.n[st]++
		r.hadFail = true
	}
}

// envFault emits the ERROR row of a member env fault.
func (r *report) envFault(label, check, detail string) {
	r.emit(label, errored, check, detail)
	r.hadEnv = true
}

// summary prints the --lint-only floor note (stderr) and the summary line, and returns the exit code:
// env fault 3 over any FAIL 1 over INCOMPLETE 4, else 0. The verdict text can read ISSUES while the exit is 3.
// A FAIL with a skipped floor check reads ISSUES (+INCOMPLETE): the bash printed ISSUES alone, hiding that the
// floor was not run (retro r10-client-gate-retro Δ4). A member env fault with no FAIL reads INCOMPLETE, never
// CLEAN: the member did not run, so the run cannot vouch for it (the bash printed CLEAN there; the exit stays 3).
func (r *report) summary(artifacts int) int {
	verdict := "CLEAN"
	incomplete := r.floorSkip && !r.o.lintOnly
	switch {
	case r.hadFail && incomplete:
		verdict = "ISSUES (+INCOMPLETE)"
	case r.hadFail:
		verdict = "ISSUES"
	case incomplete || r.hadEnv:
		verdict = "INCOMPLETE"
	}
	if r.o.lintOnly && r.floorSkip {
		fmt.Fprint(r.errw, "report-module: --lint-only — floor checks (verify-module, schema-risk) were not run and are not part of this verdict\n")
	}
	plural := "s"
	if artifacts == 1 {
		plural = ""
	}
	fmt.Fprintf(r.out, "report-module: %d artifact%s · %d PASS · %d FAIL · %d WARN · %d SKIP · %d ADVISORY  ->  %s\n",
		artifacts, plural, r.n[pass], r.n[failed], r.n[warn], r.n[skip], r.n[advisory], verdict)
	switch {
	case r.hadEnv:
		return ExitEnv
	case r.hadFail:
		return ExitFail
	case verdict == "INCOMPLETE":
		return ExitIncomplete
	}
	return ExitClean
}

func itoa(n int) string { return strconv.Itoa(n) }
