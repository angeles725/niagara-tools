package niagaratest

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// name is the check column of every row and the prefix of every error.
const name = "run-niagara-test"

// usage names only the subcommands that exist. Phase 2 (copy the home, build, run test.exe, back up and
// restore the reference jar) adds its own subcommand here when it lands.
const usage = `usage: run-niagara-test verdict [<log>|-]
       run-niagara-test verdict --testng <testng-results.xml> [<log>|-]
  verdict  decide PASS or FAIL from the captured output of a Niagara test run (bin\test.exe <module> -v:2,
           or a gradle niagaraTest log); no <log> or - reads standard input. PASS only when every
           "Total tests run: N" line has N > 0 and 0 failures; a stray Failures count, "invalid test
           target(s)", "failed signature validation", a malformed summary or no summary is FAIL.
           The exit code and BUILD SUCCESSFUL of the run are never read.
  --testng decide from the TestNG results file of test.exe -output:<dir> instead, the authoritative record
           (test.exe prints its summaries on stderr; its stdout holds only thread dumps). PASS only when
           total > 0, failed = 0, skipped = 0, no test-method (config included) has status FAIL and the
           root counts match the test-methods; a missing, malformed or over 64 MiB file is FAIL. A <log>,
           when given (- for standard input; capture stderr, 2>&1), is a cross-check: its rejections and
           failure counts fail the run, its missing summary line does not.
exit: 0 PASS, 1 FAIL, 2 usage, 3 unreadable log
`

// maxListed is how many failed test-methods get a row of their own; the rest are one "and N more" row.
const maxListed = 20

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

// verdict parses `[--testng <xml>] [<log>|-]` and judges the console log, the TestNG results file, or both.
func verdict(args []string, s cli.Stdio, stdin io.Reader) int {
	var xmlPath string
	var logs []string
	testng := false
	for i := 0; i < len(args); i++ {
		if args[i] != "--testng" {
			logs = append(logs, args[i])
			continue
		}
		if testng {
			return usageError(s, "--testng given twice")
		}
		if i+1 == len(args) {
			return usageError(s, "--testng needs a <testng-results.xml>")
		}
		testng, xmlPath = true, args[i+1]
		i++
	}
	if len(logs) > 1 {
		return usageError(s, "verdict takes one log at most")
	}
	if testng {
		return testngVerdict(xmlPath, logs, s, stdin)
	}
	res, code := readLog(logs, s, stdin)
	if code >= 0 {
		return code
	}
	ran := 0
	for _, sm := range res.Summaries {
		ran += sm.Run
	}
	consoleRows(s, res, "")
	if res.Pass {
		row(s, "PASS", fmt.Sprintf("verdict: %s, %s, 0 failures",
			plural(len(res.Summaries), "test summary", "test summaries"), plural(ran, "test run", "tests run")))
		return exitPass
	}
	row(s, "FAIL", "verdict: "+plural(len(res.Reasons), "FAIL reason", "FAIL reasons"))
	return exitFail
}

// readLog judges the log named by logs (none or "-": stdin). code is -1, or the exit of an unreadable log.
func readLog(logs []string, s cli.Stdio, stdin io.Reader) (Result, int) {
	in, label := stdin, "standard input"
	if len(logs) == 1 && logs[0] != "-" {
		f, err := os.Open(logs[0])
		if err != nil {
			return Result{}, unreadable(s, logs[0], err)
		}
		defer f.Close()
		in, label = f, logs[0]
	}
	res := Verdict(in)
	if res.Err != nil {
		return res, unreadable(s, label, res.Err)
	}
	return res, -1
}

// testngVerdict judges the results file, then the console log when one is named (only then is stdin read),
// and prints the file's rows, the log's rows and the verdict.
func testngVerdict(xmlPath string, logs []string, s cli.Stdio, stdin io.Reader) int {
	var console Result
	if len(logs) == 1 {
		var code int
		if console, code = readLog(logs, s, stdin); code >= 0 {
			return code
		}
		console.Reasons = slices.DeleteFunc(console.Reasons, func(r Reason) bool { return r.Line == 0 && r.Text == noSummary })
	}
	res := TestNGFile(xmlPath)
	if res.Root {
		status := "FAIL"
		if res.Pass {
			status = "PASS"
		}
		row(s, status, fmt.Sprintf("testng: total %s, passed %s, failed %s, skipped %s",
			shown(res.Total), shown(res.Passed), shown(res.Failed), shown(res.Skipped)))
	}
	for _, r := range res.Reasons {
		row(s, "FAIL", "testng: "+r)
	}
	for i, m := range res.Failing {
		if i == maxListed {
			row(s, "FAIL", fmt.Sprintf("testng: and %d more failed test-methods", len(res.Failing)-maxListed))
			break
		}
		config := ""
		if m.Config {
			config = " (config)"
		}
		row(s, "FAIL", "testng: test-method "+m.Name+" FAIL"+config)
	}
	consoleRows(s, console, "console ")
	if n := len(res.Reasons) + len(console.Reasons); n > 0 {
		row(s, "FAIL", "verdict: "+plural(n, "FAIL reason", "FAIL reasons"))
		return exitFail
	}
	row(s, "PASS", fmt.Sprintf("verdict: %s, 0 failed, 0 skipped (testng-results)", plural(res.Total, "test run", "tests run")))
	return exitPass
}

// consoleRows prints a row per summary line of res, then a row per other reason, each detail after prefix.
func consoleRows(s cli.Stdio, res Result, prefix string) {
	for _, sm := range res.Summaries {
		detail := fmt.Sprintf("%sline %d: run %d, failures %s, skips %s", prefix, sm.Line, sm.Run, shown(sm.Failures), shown(sm.Skips))
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
