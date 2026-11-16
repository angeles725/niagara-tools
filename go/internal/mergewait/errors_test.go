package mergewait

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Every error is printed exactly once: a gh error by withRetries, an error built after a gh call that
// succeeded (a reply without the fields the wait needs, a reply that is not JSON) by the caller.
// (#696 native review: both used to end in exit 3 with an empty stderr.)

func TestAPRReplyWithoutHeadOrBaseIsExit3WithItsLine(t *testing.T) {
	f := newFake(t, map[string][]reply{"pr view 5": {ok(`{"state":"OPEN","isDraft":false}`)}})
	code, _, errs := waitWith(f)
	if code != ExitGh || strings.Count(errs, "merge-when-green: gh pr view 5: no head commit or base branch in the reply\n") != 1 {
		t.Errorf("exit %d, stderr %q", code, errs)
	}
}

func TestAReplyThatIsNotJSONIsExit3WithItsLine(t *testing.T) {
	f := newFake(t, map[string][]reply{"pr view 5": {ok("<html>rate limited</html>")}})
	code, _, errs := waitWith(f)
	if code != ExitGh || strings.Count(errs, "merge-when-green: gh pr view 5 --json state,isDraft,headRefOid,baseRefName,mergeCommit: unreadable reply") != 1 {
		t.Errorf("exit %d, stderr %q", code, errs)
	}
}

func TestAGhErrorIsPrintedOnceNotTwice(t *testing.T) {
	f := newFake(t, map[string][]reply{"pr view 5": {netErr(), {err: errors.New("gh pr view 5: HTTP 404: Not Found")}}})
	code, _, errs := waitWith(f)
	if code != ExitGh || strings.Count(errs, "network is unreachable") != 1 || strings.Count(errs, "HTTP 404") != 1 {
		t.Errorf("exit %d, stderr %q", code, errs)
	}
}

// R1: the check-runs API pages its reply; a commit with more runs than one page must not read as green
// from the first page alone.
func TestCheckRunsAreReadAcrossPages(t *testing.T) {
	f := newFake(t, map[string][]reply{
		checksKey + "?filter=latest&per_page=100&page=1": {ok(`{"total_count":2,"check_runs":[` + run(1, "go", "completed", "success") + `]}`)},
		checksKey + "?filter=latest&per_page=100&page=2": {
			ok(`{"total_count":2,"check_runs":[` + run(2, "bats", "in_progress", "") + `]}`),
			ok(`{"total_count":2,"check_runs":[` + run(2, "bats", "completed", "failure") + `]}`),
		},
		"run view": {ok("not ok 1\n")},
	})
	code, _, errs := waitWith(f)
	if code != ExitFailed || f.called("pr merge") {
		t.Errorf("exit %d, merged %v\n%s", code, f.called("pr merge"), errs)
	}
}

// R4: a poll that fails is printed and tried again on the next poll, up to the retry budget.
func TestAFailedPollIsPrintedAndPolledAgain(t *testing.T) {
	f := newFake(t, map[string][]reply{checksKey: {
		{err: errors.New("gh api: HTTP 502: Bad Gateway")},
		checks(run(1, "go", "completed", "success")),
	}})
	code, out, errs := waitWith(f)
	if code != ExitMerged || !strings.Contains(out, "merged #5") || strings.Count(errs, "HTTP 502") != 1 {
		t.Errorf("exit %d, out %q\n%s", code, out, errs)
	}
	f = newFake(t, map[string][]reply{checksKey: {{err: errors.New("gh api: HTTP 502: Bad Gateway")}}})
	if code, _, errs := waitWith(f); code != ExitGh || strings.Count(errs, "HTTP 502") != 4 {
		t.Errorf("budget: exit %d, %d prints\n%s", code, strings.Count(errs, "HTTP 502"), errs)
	}
}

// R3: a gh call that hangs is ended at its deadline, as an error naming it.
func TestAGhCallThatHangsEndsAtItsDeadline(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte("#!/bin/sh\nexec sleep 10\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	start := time.Now()
	_, err := ExecGh("", os.Environ(), 200*time.Millisecond)("pr", "view", "5")
	if err == nil || !strings.Contains(err.Error(), "timed out after 200ms") || time.Since(start) > 5*time.Second {
		t.Errorf("err %v after %s", err, time.Since(start))
	}
	if !isNetworkError(err) {
		t.Error("a call that hit its deadline must be retried like a network error")
	}
}
