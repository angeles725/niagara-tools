package mergewait

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Review advisories of #696's final slice (retro 2026-11-15 coordination-stretch T16).

var green1 = checks(run(1, "go", "completed", "success"))

// R3-merge-error-single-read: a merge call that errored gets the same 1+Retries state reads as one that
// returned: GitHub may show MERGED a moment later either way.
func TestAMergeErrorIsFollowedByTheSameLaggingReads(t *testing.T) {
	f := newFake(t, map[string][]reply{
		checksKey:    {green1},
		"pr merge 5": {{err: errors.New("gh pr merge 5: read tcp: connection reset by peer")}},
		"pr view 5":  {view(false, head), view(false, head), merged(head)},
	})
	f.lagless = false
	if code, out, errs := waitWith(f); code != ExitMerged || !strings.Contains(out, mergeSHA) {
		t.Errorf("exit %d, out %q\n%s", code, out, errs)
	}
}

// R3-merge-queue-unsettled: a merge call that returned while the PR stays OPEN (a merge queue, auto-merge, or
// GitHub's lag) is its own outcome, not "gh pr merge refused".
func TestAMergeRequestedButNotMergedIsPending(t *testing.T) {
	f := newFake(t, map[string][]reply{checksKey: {green1}}) // pr view stays OPEN
	f.lagless = false
	code, _, errs := waitWith(f)
	if code != ExitPending || !strings.Contains(errs, "still OPEN after 4 read(s)") || !strings.Contains(errs, "merge queue or auto-merge") {
		t.Errorf("exit %d\n%s", code, errs)
	}
}

// R3-post-merge-read-error-exit3: after a merge call that returned, a state read that fails (not only on a
// network error) is printed and read again; it never turns a merged PR into exit 3.
func TestAStateReadErrorAfterTheMergeIsReadAgain(t *testing.T) {
	bad := reply{err: errors.New("gh pr view 5: HTTP 502: Bad Gateway")}
	f := newFake(t, map[string][]reply{checksKey: {green1}, "pr view 5": {view(false, head), bad, merged(head)}})
	f.lagless = false
	code, out, errs := waitWith(f)
	if code != ExitMerged || !strings.Contains(out, mergeSHA) || strings.Count(errs, "HTTP 502") != 1 {
		t.Errorf("exit %d, out %q\n%s", code, out, errs)
	}
	// no read succeeds: the merge was requested and its state is unknown
	f = newFake(t, map[string][]reply{checksKey: {green1}, "pr view 5": {view(false, head), bad}})
	f.lagless = false
	if code, _, errs := waitWith(f); code != ExitPending || !strings.Contains(errs, "could not be read") {
		t.Errorf("unreadable: exit %d\n%s", code, errs)
	}
}

// R4-poll-budget-reset: a pr view that keeps failing stops the wait after its own budget, even while the
// check-runs call succeeds every time.
func TestAViewThatKeepsFailingStopsTheWaitWhileChecksSucceed(t *testing.T) {
	f := newFake(t, map[string][]reply{
		checksKey:   {checks(run(1, "go", "queued", ""))},
		"pr view 5": {view(false, head), {err: errors.New("gh pr view 5: HTTP 404: Not Found")}},
	})
	code, _, errs := waitWith(f)
	if code != ExitGh || strings.Count(errs, "HTTP 404") != 4 {
		t.Errorf("exit %d (want %d after 1 + 3 failed views), %d prints\n%s", code, ExitGh, strings.Count(errs, "HTTP 404"), errs)
	}
}

// R4-timeout-overrun: a network retry that would start after --timeout is not made.
func TestNetworkRetriesStopAtTheTimeout(t *testing.T) {
	f := newFake(t, map[string][]reply{"pr view 5": {netErr()}})
	var out, errb bytes.Buffer
	clock := time.Unix(0, 0)
	code := Wait(Config{
		PR: "5", Method: "rebase", Interval: time.Second, Timeout: 2 * time.Second, LogLines: 3, Retries: 3, Gh: f.gh,
		Sleep: func(d time.Duration) { clock = clock.Add(d) }, Now: func() time.Time { return clock },
		Out: &out, Err: &errb,
	})
	if code != ExitGh || len(f.calls) != 3 || clock.After(time.Unix(2, 0)) || !strings.Contains(errb.String(), "no retry: --timeout 2s ends first") {
		t.Errorf("exit %d, %d calls, %s elapsed\n%s", code, len(f.calls), clock.Sub(time.Unix(0, 0)), errb.String())
	}
}

// R3-commandcontext-no-waitdelay: a gh that leaves a child holding its output pipes still ends near its
// deadline.
func TestAGhCallWhoseChildHoldsThePipesEndsNearItsDeadline(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte("#!/bin/sh\nsleep 20 &\nexec sleep 20\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	start := time.Now()
	_, err := ExecGh("", os.Environ(), 200*time.Millisecond)("pr", "view", "5")
	if err == nil || time.Since(start) > 8*time.Second {
		t.Errorf("err %v after %s", err, time.Since(start))
	}
}
