package niagaratest

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// The logs below are synthetic: each one is built from a line shape quoted from a real test.exe or gradle
// niagaraTest run (retros testing-parity-btestng and r10-dr5-driver-exemplar), not copied from a client log.
func TestVerdict(t *testing.T) {
	t.Parallel()
	huge := strings.Repeat("x", MaxLine+1)
	cases := []struct {
		name      string
		log       string
		pass      bool
		summaries []Summary
		reasons   []Reason
	}{
		{name: "all pass", pass: true,
			log:       "Running moduleATest\nTotal tests run: 6, Failures: 0, Skips: 0\nBUILD SUCCESSFUL\n",
			summaries: []Summary{{Line: 2, Run: 6, Failures: 0, Skips: 0}}},
		{name: "every failed", log: "Total tests run: 6, Failures: 6\n",
			summaries: []Summary{{Line: 1, Run: 6, Failures: 6, Skips: -1}},
			reasons:   []Reason{{1, "6 failures"}}},
		{name: "a Failures line outside a summary", log: "Failures: 5\nTotal tests run: 6, Failures: 0, Skips: 0\n",
			summaries: []Summary{{Line: 2, Run: 6, Failures: 0, Skips: 0}},
			reasons:   []Reason{{1, "5 failures (a Failures line outside a test summary)"}}},
		{name: "zero tests with BUILD SUCCESSFUL", log: "Total tests run: 0\nBUILD SUCCESSFUL\n",
			summaries: []Summary{{Line: 1, Run: 0, Failures: -1, Skips: -1}},
			reasons:   []Reason{{1, "no tests ran (Total tests run: 0)"}}},
		{name: "invalid test target", log: "invalid test target(s)\nTotal tests run: 6, Failures: 0, Skips: 0\n",
			summaries: []Summary{{Line: 2, Run: 6, Failures: 0, Skips: 0}},
			reasons:   []Reason{{1, "invalid test target(s): the test NRE rejected the test module"}}},
		{name: "failed signature validation",
			log:       "Module moduleATest failed signature validation: Could not validate cert chain\nTotal tests run: 0\n",
			summaries: []Summary{{Line: 2, Run: 0, Failures: -1, Skips: -1}},
			reasons: []Reason{{1, "failed signature validation: the test NRE does not trust the test module"},
				{2, "no tests ran (Total tests run: 0)"}}},
		{name: "several modules, one fails",
			log:       "Total tests run: 6, Failures: 0, Skips: 0\nTotal tests run: 4, Failures: 1, Skips: 0\n",
			summaries: []Summary{{Line: 1, Run: 6, Failures: 0, Skips: 0}, {Line: 2, Run: 4, Failures: 1, Skips: 0}},
			reasons:   []Reason{{2, "1 failure"}}},
		{name: "no summary line", log: "BUILD SUCCESSFUL in 12s\n",
			reasons: []Reason{{0, "no test summary (no Total tests run line)"}}},
		{name: "empty log", reasons: []Reason{{0, "no test summary (no Total tests run line)"}}},
		{name: "CRLF line endings and no Skips", pass: true, log: "Total tests run: 3, Failures: 0\r\nBUILD SUCCESSFUL\r\n",
			summaries: []Summary{{Line: 1, Run: 3, Failures: 0, Skips: -1}}},
		{name: "log prefix and timestamp", pass: true,
			log:       "2026-10-15 12:00:01.123 [testng] INFO   Total tests run: 2, Failures: 0, Skips: 1 (took 3 s)\n",
			summaries: []Summary{{Line: 1, Run: 2, Failures: 0, Skips: 1}}},
		{name: "no Failures count", log: "Total tests run: 6\n",
			summaries: []Summary{{Line: 1, Run: 6, Failures: -1, Skips: -1}},
			reasons:   []Reason{{1, "no Failures count"}}},
		{name: "malformed numbers", log: "Total tests run: six, Failures: 0\nTotal tests run: 6, Failures: -1\n" +
			"Total tests run: 99999999999999999999, Failures: 0\nTotal tests run: 6x, Failures: 0\n",
			reasons: []Reason{{1, "malformed test summary"}, {2, "malformed test summary"},
				{3, "malformed test summary"}, {4, "malformed test summary"}}},
		{name: "a huge line stops the read", log: "Total tests run: 6, Failures: 0, Skips: 0\n" + huge + "\nTotal tests run: 1, Failures: 1\n",
			summaries: []Summary{{Line: 1, Run: 6, Failures: 0, Skips: 0}},
			reasons:   []Reason{{2, "line longer than 1048576 bytes: the log is not read past it"}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r := Verdict(strings.NewReader(c.log))
			if r.Err != nil {
				t.Fatalf("Err = %v", r.Err)
			}
			if r.Pass != c.pass {
				t.Errorf("Pass = %v, want %v", r.Pass, c.pass)
			}
			if !reflect.DeepEqual(r.Summaries, c.summaries) {
				t.Errorf("Summaries = %+v, want %+v", r.Summaries, c.summaries)
			}
			if !reflect.DeepEqual(r.Reasons, c.reasons) {
				t.Errorf("Reasons = %+v, want %+v", r.Reasons, c.reasons)
			}
		})
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestVerdictReadErrorIsNeverAPass(t *testing.T) {
	t.Parallel()
	r := Verdict(failingReader{})
	if r.Pass || r.Err == nil || r.Err.Error() != "boom" {
		t.Fatalf("Verdict = %+v, want Pass false and Err boom", r)
	}
}
