package niagaratest

import (
	"fmt"
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// consoleRows prints a row per summary line of res, then a row per other reason, each detail after prefix.
func consoleRows(s cli.Stdio, res Result, prefix string) {
	for _, sm := range res.Summaries {
		detail := fmt.Sprintf("%sline %d: run %d, failures %s, skips %s", prefix, sm.Line, sm.Run, shown(sm.Failures), shown(sm.Skips))
		if sm.GrandTotal {
			detail += " (all run tests)"
		}
		if p := sm.Problem(); p != "" {
			row(s, "FAIL", detail+"; "+p)
		} else {
			row(s, "PASS", detail)
		}
	}
	for _, r := range res.Reasons {
		if ofSummary(res.Summaries, r) {
			continue // its summary row above already says it
		}
		if r.Line == 0 {
			row(s, "FAIL", prefix+r.Text)
		} else {
			row(s, "FAIL", fmt.Sprintf("%sline %d: %s", prefix, r.Line, r.Text))
		}
	}
}

func unreadable(s cli.Stdio, label string, err error) int {
	fmt.Fprintf(s.Err, "%s: cannot read %s: %s\n", name, label, strerror.Text(err))
	return exitUnreadable
}

// ofSummary says whether r is the problem of one of the summaries.
func ofSummary(sums []Summary, r Reason) bool {
	for _, sm := range sums {
		if sm.Line == r.Line && sm.Problem() == r.Text {
			return true
		}
	}
	return false
}

func row(s cli.Stdio, status, detail string) {
	fmt.Fprintf(s.Out, "%s  %s  %s\n", status, name, detail)
}

// shown spells a count, "-" when the line had none.
func shown(n int) string {
	if n < 0 {
		return "-"
	}
	return strconv.Itoa(n)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}
