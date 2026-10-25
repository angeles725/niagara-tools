// Package niagaratest decides whether a Niagara test run passed, from its captured output alone: the
// stdout and stderr of `bin\test.exe <module> -v:2` or a gradle `niagaraTest` log. The exit code and
// "BUILD SUCCESSFUL" are never evidence: both are seen on runs that ran no test or had their test module
// rejected (types/moduleTest.md § Known gotchas). The rule is spec C40 of the kit improvement candidates.
package niagaratest

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// MaxLine is the longest line Verdict reads. A longer line stops the read and fails the run: what follows it
// is unread, and a failure there must not turn into a pass.
const MaxLine = 1 << 20

// The markers Verdict looks for. The two rejections are matched in any case (a missed one could hide a
// failed run); the summary keys are TestNG's own spelling.
const (
	summaryKey    = "Total tests run:"
	failuresKey   = "Failures:"
	invalidTarget = "invalid test target(s)"
	badSignature  = "failed signature validation"
)

// noSummary is the reason of a log with no summary line. In --testng mode it is no reason: the results file
// is the verdict, and a captured stdout of `test.exe` holds no summary (it goes to stderr).
const noSummary = "no test summary (no Total tests run line)"

// Summary is one `Total tests run: N, Failures: F, Skips: S` line. Failures and Skips are -1 when the line
// has no such count. Line numbers start at 1.
type Summary struct {
	Line, Run, Failures, Skips int
}

// Problem says why the summary fails the run, "" when it passes: tests ran and none failed.
func (s Summary) Problem() string {
	switch {
	case s.Run == 0:
		return "no tests ran (Total tests run: 0)"
	case s.Failures < 0:
		return "no Failures count"
	case s.Failures == 1:
		return "1 failure"
	case s.Failures > 1:
		return fmt.Sprintf("%d failures", s.Failures)
	}
	return ""
}

// Reason is one cause of a FAIL, at its line (0 for the log as a whole).
type Reason struct {
	Line int
	Text string
}

// Result is the verdict on one log. Reasons holds every cause of a FAIL, the summaries' problems included,
// in line order. Err is a read error: the log was not read to its end, and Pass is false.
type Result struct {
	Pass      bool
	Summaries []Summary
	Reasons   []Reason
	Err       error
}

// Verdict reads the log line by line, keeping one line at a time. The run passes only when the log holds at
// least one test summary, every summary has run tests with 0 failures, and no line reports a failure count
// or a rejected test module.
func Verdict(r io.Reader) Result {
	var res Result
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), MaxLine)
	n, sawSummary := 0, false
	for sc.Scan() {
		n++
		line := strings.TrimSuffix(sc.Text(), "\r")
		lower := strings.ToLower(line)
		if strings.Contains(lower, invalidTarget) {
			res.fail(n, invalidTarget+": the test NRE rejected the test module")
		}
		if strings.Contains(lower, badSignature) {
			res.fail(n, badSignature+": the test NRE does not trust the test module")
		}
		if i := strings.Index(line, summaryKey); i >= 0 {
			sawSummary = true
			s, ok := parseSummary(n, line[i+len(summaryKey):])
			if !ok {
				res.fail(n, "malformed test summary")
				continue
			}
			res.Summaries = append(res.Summaries, s)
			if p := s.Problem(); p != "" {
				res.fail(n, p)
			}
		} else {
			res.strayFailures(n, line)
		}
	}
	if err := sc.Err(); errors.Is(err, bufio.ErrTooLong) {
		res.fail(n+1, fmt.Sprintf("line longer than %d bytes: the log is not read past it", MaxLine))
	} else if err != nil {
		res.Err = err
		return res
	}
	if !sawSummary {
		res.fail(0, noSummary)
	}
	res.Pass = len(res.Reasons) == 0
	return res
}

func (r *Result) fail(line int, text string) { r.Reasons = append(r.Reasons, Reason{line, text}) }

// strayFailures fails the run on every "Failures:" key of a line that is no summary: a count above 0, or a
// value that is no count, is a reason (a value that cannot be read must never read as 0 failures).
func (r *Result) strayFailures(n int, line string) {
	for rest := line; ; {
		i := strings.Index(rest, failuresKey)
		if i < 0 {
			return
		}
		rest = rest[i+len(failuresKey):]
		if f, ok := count(rest); !ok {
			r.fail(n, "malformed Failures count")
		} else if f > 0 {
			r.fail(n, fmt.Sprintf("%d failures (a Failures line outside a test summary)", f))
		}
	}
}

// parseSummary reads what follows "Total tests run:": the run count, then ", Key: value" fields, of which
// Failures and Skips are kept and any other key is ignored.
func parseSummary(line int, rest string) (Summary, bool) {
	s := Summary{Line: line, Failures: -1, Skips: -1}
	fields := strings.Split(rest, ",")
	run, ok := count(fields[0])
	if !ok {
		return s, false
	}
	s.Run = run
	for _, f := range fields[1:] {
		key, val, found := strings.Cut(strings.TrimSpace(f), ":")
		if !found {
			continue
		}
		dst := map[string]*int{"Failures": &s.Failures, "Skips": &s.Skips}[key]
		if dst == nil {
			continue
		}
		if *dst, ok = count(val); !ok {
			return s, false
		}
	}
	return s, true
}

// count reads the count token at the start of s: leading blanks, decimal digits, then the end or any byte
// that cannot continue a number (a blank or punctuation such as , ; ) ]). So "6 (took 3 s)", "5, Skips: 0"
// and "1)" are counts, while "6x", "1.5", "-1", "six" and a number past int32 are malformed.
func count(s string) (int, bool) {
	s = strings.TrimLeft(s, " \t")
	end := 0
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if end == 0 || (end < len(s) && continuesNumber(s[end])) {
		return 0, false
	}
	v, err := strconv.ParseUint(s[:end], 10, 31)
	if err != nil {
		return 0, false
	}
	return int(v), true
}

// continuesNumber says whether b, right after a digit run, would make it part of a longer word or number.
func continuesNumber(b byte) bool {
	return b == '_' || b == '.' || (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
