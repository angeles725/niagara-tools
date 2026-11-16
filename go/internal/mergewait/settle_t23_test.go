package mergewait

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

// Review advisories of #707 (retro 2026-11-15 coordination-stretch T23).

// waitTimed runs Wait on a fake clock with the given timeout and records the clock at each pr view call.
func waitTimed(f *fakeGh, timeout time.Duration) (code int, errs string, views []time.Duration) {
	var out, errb bytes.Buffer
	clock := time.Unix(0, 0)
	gh := func(args ...string) (string, error) {
		if strings.HasPrefix(strings.Join(args, " "), "pr view") {
			views = append(views, clock.Sub(time.Unix(0, 0)))
		}
		return f.gh(args...)
	}
	code = Wait(Config{
		PR: "5", Method: "rebase", Interval: time.Second, Timeout: timeout, LogLines: 3, Retries: 3, Gh: gh,
		Sleep: func(d time.Duration) { clock = clock.Add(d) }, Now: func() time.Time { return clock },
		Out: &out, Err: &errb,
	})
	return code, errb.String(), views
}

// R4-refusal-latency: a merge GitHub refused outright is decided on one state read, not 1+Retries reads.
func TestAMergeGitHubRefusedIsDecidedOnOneRead(t *testing.T) {
	f := newFake(t, map[string][]reply{
		checksKey:    {green1},
		"pr merge 5": {{err: errors.New("gh pr merge 5: GraphQL: Pull Request is not mergeable (mergePullRequest)")}},
	})
	f.lagless = false
	code, errs, views := waitTimed(f, time.Minute)
	if code != ExitRefused || len(views) != 2 || views[1] != 0 {
		t.Errorf("exit %d, pr view at %v (want the start and one read at once)\n%s", code, views, errs)
	}
}

// R4-unsettled-message-any-state, R3-003: the queue wording is for an OPEN PR only, and says how many reads
// answered; a PR closed after the merge call is not open (6), not pending.
func TestTheUnsettledMessageFitsTheStateItReports(t *testing.T) {
	closed := ok(`{"state":"CLOSED","isDraft":false,"headRefOid":"` + head + `","baseRefName":"main"}`)
	f := newFake(t, map[string][]reply{checksKey: {green1}, "pr view 5": {view(false, head), closed}})
	f.lagless = false
	if code, errs, _ := waitTimed(f, time.Minute); code != ExitMoved || !strings.Contains(errs, "#5 is CLOSED after the merge call") || strings.Contains(errs, "merge queue") {
		t.Errorf("closed: exit %d\n%s", code, errs)
	}
	bad := reply{err: errors.New("gh pr view 5: HTTP 502: Bad Gateway")}
	f = newFake(t, map[string][]reply{checksKey: {green1}, "pr view 5": {view(false, head), view(false, head), bad}})
	f.lagless = false
	if code, errs, _ := waitTimed(f, time.Minute); code != ExitPending || !strings.Contains(errs, "still OPEN after 4 read(s) (1 answered)") {
		t.Errorf("partly read: exit %d\n%s", code, errs)
	}
}

// R3-002: --timeout bounds the wait for green checks; the authorized merge call keeps its network retry.
func TestAMergeCallNearTheTimeoutKeepsItsNetworkRetry(t *testing.T) {
	queued := checks(run(1, "go", "queued", ""))
	f := newFake(t, map[string][]reply{checksKey: {queued, queued, green1}, "pr merge 5": {netErr(), ok("")}})
	code, errs, _ := waitTimed(f, 2*time.Second) // green at the deadline
	if code != ExitMerged || strings.Count(strings.Join(f.calls, "\n"), "pr merge 5") != 2 {
		t.Errorf("exit %d, calls %v\n%s", code, f.calls, errs)
	}
}

// R3-001 (declined, pinned): failed pr view polls are an interval apart, each loop pausing before its view.
func TestFailedViewPollsAreAnIntervalApart(t *testing.T) {
	f := newFake(t, map[string][]reply{
		checksKey:   {checks(run(1, "go", "queued", ""))},
		"pr view 5": {view(false, head), {err: errors.New("gh pr view 5: HTTP 404: Not Found")}},
	})
	code, _, views := waitTimed(f, time.Minute)
	if code != ExitGh || len(views) != 5 {
		t.Fatalf("exit %d, views at %v", code, views)
	}
	for i := 2; i < len(views); i++ {
		if views[i]-views[i-1] < time.Second {
			t.Errorf("views %d and %d %s apart, want at least the 1s interval: %v", i-1, i, views[i]-views[i-1], views)
		}
	}
}
