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

// usage names every subcommand.
const usage = `usage: run-niagara-test verdict [<log>|-]
       run-niagara-test verdict --testng <testng-results.xml> [--allow-ignored-count <n> --allow-ignored-reason <why>]
                                [<log>|-]
       run-niagara-test run --install <dir> --gradle-root <dir> --part <gradle-part> --module <test-module>
                            [--timeout <dur>] [--keep-home] [--scratch <dir>] [--jdk8 <dir>]
                            [--gradle-user-home <dir>] [--module-verification low] [--exclude-jar <glob>]...
                            [--restore-gradle-root] [--log-dir <dir>] [--reuse-home <dir>]
                            [--allow-ignored <Class.method>... | --allow-ignored-count <n>] [--allow-ignored-reason <why>]
  verdict  decide PASS or FAIL from the captured output of a Niagara test run (bin\test.exe <module> -v:2,
           or a gradle niagaraTest log); no <log> or - reads standard input. PASS only when every
           "Total tests run: N" line has N > 0 and 0 failures; a stray Failures count, "invalid test
           target(s)", "failed signature validation", a malformed summary or no summary is FAIL.
           The exit code and BUILD SUCCESSFUL of the run are never read.
  --testng decide from the TestNG results file of test.exe -output:<dir> instead, the authoritative record
           (test.exe prints its summaries on stderr; its stdout holds only thread dumps). PASS only when
           total > 0, failed = 0, skipped = 0, no test-method (config included) has status FAIL and the
           root counts match the test-methods, and ignored = 0; a missing, malformed or over 64 MiB file is FAIL.
           Ignored methods (@Test(enabled = false) or excluded) are only a count in the file, never named:
           --allow-ignored-count <n> --allow-ignored-reason <why> (both or neither, n from 1, the reason one
           printable line) waives exactly n and prints the reason; more ignored than n (a new disable) or
           fewer (a stale allowance) is FAIL. run names them from the source instead: every method-level
           @Test(enabled = false) a registered type of <part>/moduleTest-include.xml declares, or inherits from
           a srcTest/ superclass, is <SimpleName>.<method> (an abstract type counts on its concrete subclasses);
           their count must equal the root ignored ("source and TestNG disagree" is FAIL), and --allow-ignored
           <Class.method> (repeatable, with --allow-ignored-reason) waives exactly that set: a disabled method not
           named (new) or a name no source disables (stale) is FAIL, so swapping one disable for another fails
           although the count holds. A class-level @Test(enabled = false), an enabled value that is no literal or
           a registered source the scanner cannot read is exit 3, before the build. Every run prints the names
           the scan found. run's count pair is the fallback for a source that cannot name the set: the scan still
           runs, and the count is judged only when the scan refused (case (i): a NOTE row, not exit 3) or its
           count differs from the root ignored (case (ii)); when it names every ignored method, the count is FAIL
           ("use --allow-ignored names"), whatever the reason. Mixing it with --allow-ignored is exit 2. A <log>,
           when given (- for standard input; capture stderr, 2>&1), is a cross-check: its rejections and
           failure counts fail the run, its missing summary line does not.
  run      from WSL, copy the install (X:\... or /mnt/x/...) into a new dir under %TEMP% (or --scratch, under
           %TEMP%), build :<part>:jar and :moduleTestJar against the copy, run bin\test.exe <module> on it with
           a scratch niagara_user_home, judge with verdict --testng (stderr as the console log; a non-zero
           exit with green results is FAIL), then kill this run's processes and delete the dir unless
           --keep-home. Refused: an install under a user home, without bin/test.exe, or used by a running
           station (a service's unreadable station.exe is attributed through its niagarad service, or
           refused); a scratch outside %TEMP%. The install is never written; a change to it is FAIL. The
           NRE must use the scratch niagara_user_home: a change to a real %USERPROFILE%\Niagara4.*\<brand>
           registry\ or security\, or no etc\ and security\ in the scratch home, is FAIL. test.exe gets it
           as niagara_user_home through WSLENV (niagara_user_home/w). The build uses the WSL JDK 8
           --jdk8 (default /usr/lib/jvm/java-8-openjdk-amd64, JAVA_HOME and installations.paths) and the
           gradle cache --gradle-user-home (default ~/.gradle). --timeout (default 30m) kills the run.
           --module-verification low (TEST ONLY, off by default) appends niagara.moduleVerificationMode=low
           to the copy's defaults\system.properties, never the install's (4.15 defaults to medium, which
           rejects a self-signed test jar; 4.14 ships low); the install's file is hashed before and after.
           --exclude-jar <glob> (repeatable; a jar basename of letters, digits, _ . - * ?, case-insensitive)
           leaves the matching top-level modules\*.jar out of the copy (a stale <module>-ux.jar beside the
           freshly built jars); a glob matching no jar is FAIL, exit 2, before anything is copied.
           --gradle-root must be a git worktree (else exit 3, before any copy): its git status, every listed
           file hashed, is taken before the copy and after cleanup, and any change is FAIL (plugin 7.3.40's
           compileModuleTestJava appends the test types to the production module-include.xml).
           --restore-gradle-root then checks out only the files that were clean before and the build
           modified; it never touches a file already dirty, never deletes an untracked file (it lists them),
           and the run stays FAIL.
           A failed run (exit 1, 3 or 4 once its scratch dir exists) keeps a run log before the dir is deleted:
           --log-dir (default $XDG_STATE_HOME/run-niagara-test, else ~/.local/state/run-niagara-test, made
           0700) gets <UTC timestamp>-<module>.log (0600) with the runner's rows, the last 200 stderr and 50
           stdout lines of test.exe (each cut at 4 KiB), the -output: dir listing and the exit status, and a
           "diagnostics kept: <path>" row. The scratch dir and its home copy read <scratch> and <copy>, the
           %USERPROFILE% and the WSL home <you>; install paths stay. A green run writes one only with an
           explicit --log-dir or --keep-home. A failed run also keeps, redacted the same way, beside the
           log: <stem>.testng-results.xml (a copy) and <stem>.traces.txt (the whole full-stacktrace of every
           FAIL test-method), each cut at a line end under 4 MiB with a "(truncated: ...)" row; a missing or
           unreadable results file is a "testng-results.xml unavailable" row in the log instead.
           --reuse-home <dir> boots again on the dir a --keep-home run kept (its "cleanup: kept" row), to prove
           what survives a restart. Refused (exit 2): a dir not under the scratch parent, one that is, holds or
           sits in a real %USERPROFILE%\Niagara4.*\<brand> or has registry\ or security\ at its top, and one
           whose run-niagara-test.marker (written by every run in its dir: run id, install, sha256 of the
           install's defaults\system.properties, and the run log a --keep-home run wrote) does not match: the
           run id must be the one the kept run's log in --log-dir records, the install and the sha256 this run's.
           Nothing is copied again (the home copy, its registry and the scratch niagara_user_home stay); the
           module jars are rebuilt and out\ starts empty. Every other gate runs; --exclude-jar and
           --module-verification are exit 2 with it. The dir is deleted at the end ("cleanup: deleted") unless
           --keep-home again.
exit: verdict 0 PASS, 1 FAIL, 2 usage, 3 unreadable log
      run 0 PASS, 1 FAIL, 2 usage or refused, 3 environment (cmd.exe, git, robocopy, gradle, cleanup), 4 timeout
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

// verdictFlags are the value flags of verdict, each with the name of its value.
var verdictFlags = map[string]string{"--testng": "<testng-results.xml>", "--allow-ignored-count": "<n>", "--allow-ignored-reason": "<why>"}

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
