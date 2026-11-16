package mergewait

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// Review advisories of #711 (retro 2026-11-15 coordination-stretch T27).

const notMergeable = "gh pr merge 5: exit status 1: GraphQL: Pull Request is not mergeable (mergePullRequest)"

// settleReads runs a wait whose merge call gets mergeReplies and whose state reads after it are views,
// and returns the exit code, stdout, stderr and how many state reads followed the merge call.
func settleReads(t *testing.T, mergeReplies []reply, views ...reply) (code int, out, errs string, reads int) {
	t.Helper()
	f := newFake(t, map[string][]reply{
		checksKey:    {green1},
		"pr merge 5": mergeReplies,
		"pr view 5":  append([]reply{view(false, head)}, views...),
	})
	f.lagless = false
	code, out, errs = waitWith(f)
	all := strings.Join(f.calls, "\n")
	if i := strings.LastIndex(all, "pr merge 5"); i >= 0 {
		reads = strings.Count(all[i:], "pr view 5")
	}
	return code, out, errs, reads
}

// T27-retried-merge-refused: a merge call sent again after a network error may be refused because the first
// one merged; the refusal is then no proof that nothing happened, so the state gets its 1+Retries reads and
// MERGED at the checked head is the success.
func TestAMergeRefusedAfterANetworkRetryGetsEveryRead(t *testing.T) {
	code, out, errs, reads := settleReads(t, []reply{netErr(), {err: errors.New(notMergeable)}},
		view(false, head), view(false, head), merged(head))
	if code != ExitMerged || reads != 3 || !strings.Contains(out, "merged #5 at "+head) {
		t.Errorf("exit %d after %d read(s), out %q\n%s", code, reads, out, errs)
	}
	// the safety rule holds on this path too: MERGED at another head is not a success
	other := strings.Repeat("2", 40)
	code, out, errs, _ = settleReads(t, []reply{netErr(), {err: errors.New(notMergeable)}}, view(false, head), merged(other))
	if code != ExitMoved || out != "" || !strings.Contains(errs, "merged at another head") {
		t.Errorf("other head: exit %d, out %q\n%s", code, out, errs)
	}
	// and a PR that stays OPEN through every read is still a refusal: nothing merged
	code, _, errs, reads = settleReads(t, []reply{netErr(), {err: errors.New(notMergeable)}})
	if code != ExitRefused || reads != 4 || !strings.Contains(errs, "nothing merged") {
		t.Errorf("stays open: exit %d after %d read(s)\n%s", code, reads, errs)
	}
}

// T27-refusal-too-broad: a merge error that is not GitHub's verdict on the merge (a rate limit, a request
// timeout, a GraphQL server error that may follow a merge that went through) is not a refusal: every read.
func TestAMergeErrorThatIsNotAVerdictGetsEveryRead(t *testing.T) {
	for _, msg := range []string{
		"gh pr merge 5: exit status 1: GraphQL: Something went wrong while executing your query. This may be the result of a timeout, or it could be a GitHub bug.",
		"gh pr merge 5: exit status 1: HTTP 429: Too Many Requests",
		"gh pr merge 5: exit status 1: HTTP 408: Request Timeout",
		"gh pr merge 5: exit status 1: GraphQL: API rate limit exceeded for user",
		"gh pr merge 5: exit status 1: GraphQL: http 429 too many requests", // the status matched case-insensitively
	} {
		code, _, errs, reads := settleReads(t, []reply{{err: errors.New(msg)}}, view(false, head), merged(head))
		if code != ExitMerged || reads != 2 {
			t.Errorf("%s: exit %d after %d read(s)\n%s", msg, code, reads, errs)
		}
	}
	// GitHub's own refusals keep the one-read path (T23)
	for _, msg := range []string{notMergeable, "gh pr merge 5: exit status 1: HTTP 405: Head branch was modified. Review and try the merge again."} {
		if code, _, errs, reads := settleReads(t, []reply{{err: errors.New(msg)}}); code != ExitRefused || reads != 1 {
			t.Errorf("%s: exit %d after %d read(s)\n%s", msg, code, reads, errs)
		}
	}
}

// T27-settle-read-retry: a state read after the merge call that fails on a network error is sent again, even
// past --timeout (the green checks authorized the merge), and its retry is printed.
func TestASettleReadNetworkErrorIsRetriedPastTheTimeout(t *testing.T) {
	queued := checks(run(1, "go", "queued", ""))
	f := newFake(t, map[string][]reply{
		checksKey:   {queued, queued, green1},
		"pr view 5": {view(false, head), view(false, head), view(false, head), netErr(), merged(head)},
	})
	f.lagless = false
	code, errs, views := waitTimed(f, 2*time.Second) // green at the deadline
	if code != ExitMerged || len(views) != 5 || !strings.Contains(errs, "a network error; retry 1/3") || strings.Contains(errs, "no retry") {
		t.Errorf("exit %d, views at %v\n%s", code, views, errs)
	}
}
