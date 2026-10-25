package niagaratest

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// name is the check column of every row and the prefix of every error.
const name = "run-niagara-test"

// usage names only the subcommands that exist. Phase 2 (copy the home, build, run test.exe, back up and
// restore the reference jar) adds its own subcommand here when it lands.
const usage = `usage: run-niagara-test verdict [<log>|-]
  verdict  decide PASS or FAIL from the captured output of a Niagara test run (bin\test.exe <module> -v:2,
           or a gradle niagaraTest log); no <log> or - reads standard input. PASS only when every
           "Total tests run: N" line has N > 0 and 0 failures; a stray Failures count, "invalid test
           target(s)", "failed signature validation", a malformed summary or no summary is FAIL.
           The exit code and BUILD SUCCESSFUL of the run are never read.
exit: 0 PASS, 1 FAIL, 2 usage, 3 unreadable log
`

// Exit codes.
const (
	exitPass       = 0
	exitFail       = 1
	exitUsage      = 2
	exitUnreadable = 3
)

// Main is the run-niagara-test command (toolbelt/run-niagara-test.sh); the log on stdin is the process's.
func Main(args []string, s cli.Stdio) int { return run(args, s, os.Stdin) }

func run(args []string, s cli.Stdio, stdin io.Reader) int {
	if len(args) == 0 {
		return usageError(s, "no subcommand")
	}
	switch args[0] {
	case "-h", "--help", "help":
		fmt.Fprint(s.Out, usage)
		return exitPass
	case "verdict":
		return verdict(args[1:], s, stdin)
	}
	return usageError(s, "unknown subcommand: "+args[0])
}

func usageError(s cli.Stdio, msg string) int {
	fmt.Fprintf(s.Err, "%s: %s\n%s", name, msg, usage)
	return exitUsage
}

// verdict reads one log and prints a row per summary line, a row per other FAIL reason, then the verdict.
func verdict(args []string, s cli.Stdio, stdin io.Reader) int {
	if len(args) > 1 {
		return usageError(s, "verdict takes one log at most")
	}
	in, label := stdin, "standard input"
	if len(args) == 1 && args[0] != "-" {
		f, err := os.Open(args[0])
		if err != nil {
			return unreadable(s, args[0], err)
		}
		defer f.Close()
		in, label = f, args[0]
	}
	res := Verdict(in)
	if res.Err != nil {
		return unreadable(s, label, res.Err)
	}
	ran := 0
	for _, sm := range res.Summaries {
		ran += sm.Run
		detail := fmt.Sprintf("line %d: run %d, failures %s, skips %s", sm.Line, sm.Run, shown(sm.Failures), shown(sm.Skips))
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
			row(s, "FAIL", r.Text)
		} else {
			row(s, "FAIL", fmt.Sprintf("line %d: %s", r.Line, r.Text))
		}
	}
	if res.Pass {
		row(s, "PASS", fmt.Sprintf("verdict: %s, %s, 0 failures",
			plural(len(res.Summaries), "test summary", "test summaries"), plural(ran, "test run", "tests run")))
		return exitPass
	}
	row(s, "FAIL", "verdict: "+plural(len(res.Reasons), "FAIL reason", "FAIL reasons"))
	return exitFail
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
