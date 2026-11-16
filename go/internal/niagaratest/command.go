package niagaratest

import (
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Main is the run-niagara-test command (toolbelt/run-niagara-test.sh); the log on stdin is the process's.
func Main(args []string, s cli.Stdio) int { return run(args, s, os.Stdin) }

func run(args []string, s cli.Stdio, stdin io.Reader) int {
	if len(args) == 0 {
		return usageError(s, "no subcommand")
	}
	switch args[0] {
	case "-h", "--help", "help":
		_, _ = io.WriteString(s.Out, usage)
		return exitPass
	case "verdict":
		return verdict(args[1:], s, stdin)
	case "run":
		return runCmd(args[1:], s)
	}
	return usageError(s, "unknown subcommand: "+args[0])
}

func usageError(s cli.Stdio, msg string) int {
	fmt.Fprintf(s.Err, "%s: %s\n%s", name, msg, usage)
	return exitUsage
}

// verdict parses `[--testng <xml> [--allow-ignored-count <n> --allow-ignored-reason <why>]] [<log>|-]` and judges
// the console log, the TestNG results file, or both.
func verdict(args []string, s cli.Stdio, stdin io.Reader) int {
	var logs []string
	vals := map[string]string{}
	for i := 0; i < len(args); i++ {
		value, isFlag := verdictFlags[args[i]]
		if !isFlag {
			logs = append(logs, args[i])
			continue
		}
		if _, twice := vals[args[i]]; twice {
			return usageError(s, args[i]+" given twice")
		}
		if i+1 == len(args) {
			return usageError(s, args[i]+" needs a "+value)
		}
		vals[args[i]] = args[i+1]
		i++
	}
	if len(logs) > 1 {
		return usageError(s, "verdict takes one log at most")
	}
	count, countSet := vals["--allow-ignored-count"]
	why, whySet := vals["--allow-ignored-reason"]
	allow, err := parseAllowance(count, countSet, why, whySet)
	if err != nil {
		return usageError(s, err.Error())
	}
	xmlPath, testng := vals["--testng"]
	if testng {
		return testngVerdict(xmlPath, logs, allow, s, stdin)
	}
	if countSet {
		return usageError(s, "--allow-ignored-count needs --testng (a console log has no ignored count)")
	}
	res, code := readLog(logs, s, stdin)
	if code >= 0 {
		return code
	}
	ran := testsRun(res.Summaries)
	consoleRows(s, res, "")
	if res.Pass {
		row(s, "PASS", fmt.Sprintf("verdict: %s, %s, 0 failures",
			plural(len(res.Summaries), "test summary", "test summaries"), plural(ran, "test run", "tests run")))
		return exitPass
	}
	row(s, "FAIL", "verdict: "+plural(len(res.Reasons), "FAIL reason", "FAIL reasons"))
	return exitFail
}

// testsRun is the run count of the last grand-total summary, else the sum of the summaries: a grand total
// already holds the per-suite counts before it. Each summary still passes or fails on its own.
func testsRun(sums []Summary) int {
	sum, grand := 0, -1
	for _, sm := range sums {
		sum += sm.Run
		if sm.GrandTotal {
			grand = sm.Run
		}
	}
	if grand >= 0 {
		return grand
	}
	return sum
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
func testngVerdict(xmlPath string, logs []string, allow IgnoredAllowance, s cli.Stdio, stdin io.Reader) int {
	var console Result
	if len(logs) == 1 {
		var code int
		if console, code = readLog(logs, s, stdin); code >= 0 {
			return code
		}
		console.Reasons = slices.DeleteFunc(console.Reasons, func(r Reason) bool { return r.Line == 0 && r.Text == noSummary })
	}
	res := TestNGFile(xmlPath, allow)
	if res.Root {
		status := "FAIL"
		if res.Pass {
			status = "PASS"
		}
		ignored := ""
		if res.Ignored != 0 {
			ignored = ", ignored " + shown(res.Ignored)
		}
		row(s, status, fmt.Sprintf("testng: total %s, passed %s, failed %s, skipped %s%s",
			shown(res.Total), shown(res.Passed), shown(res.Failed), shown(res.Skipped), ignored))
	}
	if res.Allowed != "" {
		row(s, "PASS", "testng: "+res.Allowed)
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
	row(s, "PASS", fmt.Sprintf("verdict: %s, 0 failed, 0 skipped (testng-results)", plural(res.Total-res.Ignored, "test run", "tests run")))
	return exitPass
}
