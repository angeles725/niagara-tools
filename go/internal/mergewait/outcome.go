// Package mergewait is merge-when-green (retro coordination-stretch C5): wait for the CI checks of a pull
// request's exact head commit and merge that commit, or say why not, with one exit code per outcome.
// Three merge waits were hand-written in one session; one died on a shell syntax error and one on a
// close/reopen. The rules it enforces:
//
//   - merge only on green checks of the exact head SHA, with `gh pr merge --match-head-commit <sha>`;
//   - the branch must contain the current base branch (rebase or update first, then run it again);
//   - a draft is never green (its checks may be skipped), so a draft is refused;
//   - a check that failed on a transient download error (a 429, a 5xx, a reset connection) is re-run once;
//   - a failed check is reported with the last lines of its log;
//   - every gh error is printed; a network error (connect, DNS, TLS handshake) is retried a bounded number
//     of times, and after the merge call the PR state is read again: MERGED at the checked head is the
//     success, whatever the merge call returned (a session script once reported "OPEN" for a merge that
//     had gone through, its gh stderr hidden); a merge call that returned while the PR stays OPEN (a merge
//     queue, auto-merge, lag) is exit 9, not a refusal;
//   - no retry starts after --timeout, and each failing call (check runs, pr view) has its own budget.
//
// outcome.go holds the exit codes and messages, gh.go the gh calls and replies, checks.go how check runs
// are judged, transient.go the transient-failure patterns, retry.go the gh retries, wait.go the loop,
// settle.go the state read that decides a merge, command.go the flags.
package mergewait

// Exit codes, one per outcome.
const (
	ExitMerged  = 0 // merged at the head that was checked
	ExitFailed  = 1 // a check failed (its log tail printed)
	ExitUsage   = 2 // usage error
	ExitGh      = 3 // gh missing, a gh call failed or its reply could not be read
	ExitDraft   = 4 // the PR is a draft: never green
	ExitBehind  = 5 // the branch does not contain the current base branch
	ExitMoved   = 6 // the PR is not open, or its head changed while waiting
	ExitTimeout = 7 // the checks did not finish in time
	ExitRefused = 8 // gh pr merge refused the merge
	ExitPending = 9 // the merge call returned but the PR is not MERGED yet (a merge queue, auto-merge, or lag)
)

const (
	msgPrefix      = "merge-when-green: "
	msgDraft       = msgPrefix + "#%s is a draft; a draft is never green (mark it ready: gh pr ready %s)\n"
	msgNotOpen     = msgPrefix + "#%s is %s, not OPEN\n"
	msgBehind      = msgPrefix + "%.12s does not contain the current %s (%d commit(s) behind); rebase or update the branch, push, and run this again\n"
	msgMoved       = msgPrefix + "the head of #%s moved from %.12s to %.12s while waiting; nothing merged (check the new head, then run this again)\n"
	msgWaiting     = msgPrefix + "waiting on %.12s: %d/%d check(s) completed\n"
	msgNoChecks    = msgPrefix + "waiting on %.12s: no check run yet\n"
	msgFailed      = msgPrefix + "check %q failed (%s): %s\n"
	msgLogTail     = msgPrefix + "last %d line(s) of its log:\n"
	msgNoLog       = msgPrefix + "its log could not be read (%v)\n"
	msgRerun       = msgPrefix + "check %q failed on a transient download error (%s); re-running its failed jobs once\n"
	msgNotGreen    = msgPrefix + "the checks of %.12s completed with no success (every one skipped or neutral); nothing merged\n"
	msgTimeout     = msgPrefix + "the checks of %.12s did not finish in %s; nothing merged\n"
	msgMergeFail   = msgPrefix + "gh pr merge failed for #%s at %.12s (above) and the PR is still %s; nothing merged\n"
	msgUnsettled   = msgPrefix + "gh pr merge returned for #%s at %.12s, but the PR is still OPEN after %d read(s) (%d answered): a merge queue or auto-merge holds it, or GitHub has not caught up; check it by hand\n"
	msgClosedAfter = msgPrefix + "#%s is %s after the merge call at %.12s; nothing merged\n"
	msgUnread      = msgPrefix + "gh pr merge returned for #%s at %.12s, but its state could not be read in %d read(s) (above); check it by hand\n"
	msgMerged      = msgPrefix + "merged #%s at %s (%s), merge commit %s\n"
	msgAlready     = msgPrefix + "#%s is already merged at %s, merge commit %s\n"
	msgOtherHead   = msgPrefix + "#%s was merged at another head %.12s, not the checked %.12s\n"
	msgGhError     = msgPrefix + "%v\n"
	msgGhRetry     = msgPrefix + "a network error; retry %d/%d in %s\n"
	msgNoRetry     = msgPrefix + "a network error; no retry: --timeout %s ends first\n"
)
