package mergewait

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

const head = "1111111111111111111111111111111111111111"

// fakeGh answers each gh call by the first words of its arguments; a key with several replies gives
// them in order and repeats the last. Every call is recorded.
type fakeGh struct {
	t       *testing.T
	replies map[string][]reply
	calls   []string
}

type reply struct {
	out string
	err error
}

func (f *fakeGh) gh(args ...string) (string, error) {
	call := strings.Join(args, " ")
	f.calls = append(f.calls, call)
	for key, rs := range f.replies {
		if strings.HasPrefix(call, key) {
			r := rs[0]
			if len(rs) > 1 {
				f.replies[key] = rs[1:]
			}
			return r.out, r.err
		}
	}
	f.t.Fatalf("unexpected gh call: %s", call)
	return "", nil
}

func ok(out string) reply { return reply{out: out} }

func view(draft bool, sha string) reply {
	return ok(fmt.Sprintf(`{"state":"OPEN","isDraft":%v,"headRefOid":%q,"baseRefName":"main"}`, draft, sha))
}

func behind(n int) reply { return ok(fmt.Sprintf(`{"behind_by":%d}`, n)) }

func run(id int, name, status, conclusion string) string {
	return fmt.Sprintf(`{"id":%d,"name":%q,"status":%q,"conclusion":%q,"details_url":"https://github.com/o/r/actions/runs/77/job/%d"}`,
		id, name, status, conclusion, id)
}

func checks(runs ...string) reply { return ok(`{"check_runs":[` + strings.Join(runs, ",") + `]}`) }

// newFake is a PR #5 whose head contains main, with the given replies added.
func newFake(t *testing.T, extra map[string][]reply) *fakeGh {
	f := &fakeGh{t: t, replies: map[string][]reply{
		"pr view 5":                         {view(false, head)},
		"api repos/{owner}/{repo}/compare/": {behind(0)},
		"pr merge 5":                        {ok("")},
	}}
	for k, v := range extra {
		f.replies[k] = v
	}
	return f
}

const checksKey = "api repos/{owner}/{repo}/commits/" + head + "/check-runs"

func waitWith(f *fakeGh) (int, string, string) {
	var out, errb bytes.Buffer
	clock := time.Unix(0, 0)
	code := Wait(Config{
		PR: "5", Method: "rebase", Interval: time.Second, Timeout: time.Minute, LogLines: 3, Gh: f.gh,
		Sleep: func(d time.Duration) { clock = clock.Add(d) }, Now: func() time.Time { return clock },
		Out: &out, Err: &errb,
	})
	return code, out.String(), errb.String()
}

func (f *fakeGh) called(prefix string) bool {
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

func TestMergesTheExactHeadOnceEveryCheckIsGreen(t *testing.T) {
	f := newFake(t, map[string][]reply{checksKey: {
		checks(), // no run yet
		checks(run(1, "go", "in_progress", ""), run(2, "bats", "completed", "success")),
		checks(run(1, "go", "completed", "success"), run(2, "bats", "completed", "success"), run(3, "x", "completed", "skipped")),
	}})
	code, out, errs := waitWith(f)
	if code != ExitMerged || out != "merge-when-green: merged #5 at "+head+" (--rebase)\n" {
		t.Fatalf("exit %d, out %q\n%s", code, out, errs)
	}
	if !f.called("api repos/{owner}/{repo}/compare/main..." + head) {
		t.Errorf("the base was not compared with the exact head: %v", f.calls)
	}
	if !f.called("pr merge 5 --rebase --match-head-commit " + head) {
		t.Errorf("no merge bound to the head: %v", f.calls)
	}
	if !strings.Contains(errs, "no check run yet") || !strings.Contains(errs, "1/2 check(s) completed") {
		t.Errorf("progress lines missing:\n%s", errs)
	}
}

func TestADraftIsNeverGreen(t *testing.T) {
	f := newFake(t, map[string][]reply{"pr view 5": {view(true, head)}})
	code, _, errs := waitWith(f)
	if code != ExitDraft || !strings.Contains(errs, "is a draft") || f.called("pr merge") {
		t.Errorf("exit %d, merged %v\n%s", code, f.called("pr merge"), errs)
	}
}

func TestABranchBehindItsBaseIsRefusedBeforeAndAfterTheWait(t *testing.T) {
	f := newFake(t, map[string][]reply{"api repos/{owner}/{repo}/compare/": {behind(2)}})
	if code, _, errs := waitWith(f); code != ExitBehind || !strings.Contains(errs, "2 commit(s) behind") || f.called(checksKey) {
		t.Errorf("before: exit %d, checks read %v\n%s", code, f.called(checksKey), errs)
	}
	// main moved while the checks ran: the merge is not attempted
	f = newFake(t, map[string][]reply{
		"api repos/{owner}/{repo}/compare/": {behind(0), behind(1)},
		checksKey:                           {checks(run(1, "go", "completed", "success"))},
	})
	if code, _, _ := waitWith(f); code != ExitBehind || f.called("pr merge") {
		t.Errorf("after: exit %d, merged %v", code, f.called("pr merge"))
	}
}

func TestAHeadThatMovesWhileWaitingIsNotMerged(t *testing.T) {
	f := newFake(t, map[string][]reply{
		"pr view 5": {view(false, head), view(false, "2222222222222222222222222222222222222222")},
		checksKey:   {checks(run(1, "go", "queued", ""))},
	})
	if code, _, errs := waitWith(f); code != ExitMoved || !strings.Contains(errs, "moved from 111111111111 to 222222222222") {
		t.Errorf("exit %d\n%s", code, errs)
	}
}

func TestAFailedCheckExitsWithItsLogTail(t *testing.T) {
	f := newFake(t, map[string][]reply{
		checksKey:  {checks(run(1, "go", "completed", "success"), run(9, "bats", "completed", "failure"))},
		"run view": {ok("l1\nl2\nnot ok 3 x\nl4\nl5\n")},
	})
	code, _, errs := waitWith(f)
	if code != ExitFailed || !strings.Contains(errs, `check "bats" failed (failure)`) ||
		!strings.Contains(errs, "last 3 line(s) of its log:\nnot ok 3 x\nl4\nl5\n") || f.called("pr merge") || f.called("run rerun") {
		t.Errorf("exit %d\n%s\ncalls %v", code, errs, f.calls)
	}
}

func TestATransientDownloadFailureIsReRunOnceThenMerged(t *testing.T) {
	f := newFake(t, map[string][]reply{
		checksKey: {
			checks(run(9, "bats", "completed", "failure")),
			checks(run(9, "bats", "completed", "failure")), // the API still lists the old attempt: pending
			checks(run(10, "bats", "completed", "success")),
		},
		"run view":  {ok("curl: (22) The requested URL returned error: 429\n")},
		"run rerun": {ok("")},
	})
	code, _, errs := waitWith(f)
	if code != ExitMerged || !f.called("run rerun 77 --failed") || !strings.Contains(errs, "re-running its failed jobs once") {
		t.Errorf("exit %d\n%s\ncalls %v", code, errs, f.calls)
	}
}

func TestATransientFailureIsReRunOnlyOnce(t *testing.T) {
	f := newFake(t, map[string][]reply{
		checksKey:   {checks(run(9, "bats", "completed", "failure")), checks(run(10, "bats", "completed", "failure"))},
		"run view":  {ok("curl: (22) The requested URL returned error: 503\n")},
		"run rerun": {ok("")},
	})
	if code, _, _ := waitWith(f); code != ExitFailed || strings.Count(strings.Join(f.calls, "\n"), "run rerun") != 1 {
		t.Errorf("exit %d, calls %v", code, f.calls)
	}
}

func TestTheLogDownloadIsRetriedOnce(t *testing.T) {
	f := newFake(t, map[string][]reply{
		checksKey:  {checks(run(9, "bats", "completed", "failure"))},
		"run view": {{err: errors.New("HTTP 502")}, ok("the real failure\n")},
	})
	if code, _, errs := waitWith(f); code != ExitFailed || !strings.Contains(errs, "the real failure") {
		t.Errorf("exit %d\n%s", code, errs)
	}
}

func TestAllSkippedIsNotGreenAndTimeoutAndRefusedMerge(t *testing.T) {
	f := newFake(t, map[string][]reply{checksKey: {checks(run(1, "go", "completed", "skipped"))}})
	if code, _, errs := waitWith(f); code != ExitFailed || !strings.Contains(errs, "no success") {
		t.Errorf("all skipped: exit %d\n%s", code, errs)
	}
	f = newFake(t, map[string][]reply{checksKey: {checks(run(1, "go", "in_progress", ""))}})
	if code, _, errs := waitWith(f); code != ExitTimeout || !strings.Contains(errs, "did not finish in 1m0s") {
		t.Errorf("timeout: exit %d\n%s", code, errs)
	}
	f = newFake(t, map[string][]reply{
		checksKey:    {checks(run(1, "go", "completed", "success"))},
		"pr merge 5": {{err: errors.New("gh pr merge: Head branch was modified")}},
	})
	if code, _, errs := waitWith(f); code != ExitRefused || !strings.Contains(errs, "Head branch was modified") {
		t.Errorf("refused: exit %d\n%s", code, errs)
	}
	f = newFake(t, map[string][]reply{"pr view 5": {{err: errors.New("could not resolve to a PullRequest")}}})
	if code, _, _ := waitWith(f); code != ExitGh {
		t.Errorf("gh failure: exit %d", code)
	}
}
