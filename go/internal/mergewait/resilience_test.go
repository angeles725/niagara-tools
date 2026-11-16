package mergewait

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// GitHub calls from this host fail now and then with `dial tcp ...:443: connect: network is unreachable`; a
// session script hid that stderr and reported "OPEN" for a PR whose merge had gone through. Every gh error is
// printed, a network error is retried a bounded number of times, and the PR state is read again before the
// outcome is decided. (retro 2026-11-15 coordination-stretch C5, coordinator update 2026-11-16)

const unreachable = "gh api: exit status 1: dial tcp 140.82.112.6:443: connect: network is unreachable"

const mergeSHA = "9999999999999999999999999999999999999999"

func merged(sha string) reply {
	return ok(fmt.Sprintf(`{"state":"MERGED","isDraft":false,"headRefOid":%q,"baseRefName":"main","mergeCommit":{"oid":%q}}`, sha, mergeSHA))
}

func netErr() reply { return reply{err: errors.New(unreachable)} }

func TestANetworkErrorIsPrintedAndRetried(t *testing.T) {
	f := newFake(t, map[string][]reply{
		checksKey:   {netErr(), netErr(), checks(run(1, "go", "completed", "success"))},
		"pr view 5": {view(false, head), view(false, head), merged(head)},
	})
	code, out, errs := waitWith(f)
	if code != ExitMerged || !strings.Contains(out, mergeSHA) {
		t.Fatalf("exit %d, out %q\n%s", code, out, errs)
	}
	if strings.Count(errs, "network is unreachable") != 2 || !strings.Contains(errs, "retry 1/3") || !strings.Contains(errs, "retry 2/3") {
		t.Errorf("every network error must be printed with its retry:\n%s", errs)
	}
}

func TestNetworkErrorsBeyondTheRetryBudgetStop(t *testing.T) {
	f := newFake(t, map[string][]reply{"pr view 5": {netErr()}})
	code, _, errs := waitWith(f)
	if code != ExitGh || len(f.calls) != 4 || strings.Count(errs, "network is unreachable") != 4 {
		t.Errorf("exit %d, %d calls (want 1 + 3 retries)\n%s", code, len(f.calls), errs)
	}
}

func TestAnErrorThatIsNotANetworkErrorIsPrintedAndNotRetried(t *testing.T) {
	f := newFake(t, map[string][]reply{"pr view 5": {{err: errors.New("gh pr view 5: HTTP 404: Not Found")}}})
	code, _, errs := waitWith(f)
	if code != ExitGh || len(f.calls) != 1 || !strings.Contains(errs, "HTTP 404") {
		t.Errorf("exit %d, %d calls\n%s", code, len(f.calls), errs)
	}
}

func TestAMergeThatWentThroughDespiteAnErrorIsASuccess(t *testing.T) {
	// the merge call reports a failure, yet GitHub merged: the state read afterwards decides
	f := newFake(t, map[string][]reply{
		checksKey:    {checks(run(1, "go", "completed", "success"))},
		"pr merge 5": {{err: errors.New("gh pr merge 5: read tcp: connection reset by peer")}},
		"pr view 5":  {view(false, head), merged(head)},
	})
	code, out, errs := waitWith(f)
	if code != ExitMerged || !strings.Contains(out, "merge commit "+mergeSHA) || !strings.Contains(errs, "connection reset by peer") {
		t.Errorf("exit %d, out %q\n%s", code, out, errs)
	}
}

func TestAMergeReportedOKIsConfirmedByTheState(t *testing.T) {
	f := newFake(t, map[string][]reply{
		checksKey:   {checks(run(1, "go", "completed", "success"))},
		"pr view 5": {view(false, head), view(false, head), merged(head)}, // GitHub shows MERGED one read later
	})
	f.lagless = false
	if code, out, errs := waitWith(f); code != ExitMerged || !strings.Contains(out, mergeSHA) {
		t.Errorf("exit %d, out %q\n%s", code, out, errs)
	}
	f = newFake(t, map[string][]reply{checksKey: {checks(run(1, "go", "completed", "success"))}}) // never MERGED
	f.lagless = false
	if code, _, errs := waitWith(f); code != ExitPending || !strings.Contains(errs, "still OPEN") { // T16: pending, not refused
		t.Errorf("unconfirmed: exit %d\n%s", code, errs)
	}
}

func TestAnAlreadyMergedPRIsASuccessAndOneMergedAtAnotherHeadIsNot(t *testing.T) {
	f := newFake(t, map[string][]reply{"pr view 5": {merged(head)}})
	code, out, _ := waitWith(f)
	if code != ExitMerged || !strings.Contains(out, "already merged") || !strings.Contains(out, mergeSHA) || f.called("pr merge") {
		t.Errorf("exit %d, out %q, calls %v", code, out, f.calls)
	}
	f = newFake(t, map[string][]reply{
		checksKey:   {checks(run(1, "go", "queued", ""))},
		"pr view 5": {view(false, head), merged("2222222222222222222222222222222222222222")},
	})
	if code, _, errs := waitWith(f); code != ExitMoved || !strings.Contains(errs, "merged at another head") {
		t.Errorf("other head: exit %d\n%s", code, errs)
	}
}
