package mergewait

import (
	"fmt"
	"io"
	"time"
)

// Config is one wait: the PR, how to merge it, the pace, and the seams the tests replace.
type Config struct {
	PR                string
	Method            string // rebase, squash or merge: the gh pr merge flag without its dashes
	Interval, Timeout time.Duration
	LogLines          int
	Retries           int // how many times a gh call that failed on a network error is sent again
	Gh                Gh
	Sleep             func(time.Duration)
	Now               func() time.Time
	Out, Err          io.Writer
	until             time.Time // Wait's deadline: Now at its start plus Timeout; no gh retry starts after it
	settleGh          Gh        // gh with network retries that ignore until: the reads of the merge outcome
	rawGh             Gh        // gh with no retries: the merge call wraps it the same way and counts its sends
}

// Wait checks the PR, waits for the checks of its head commit and merges that commit when they are
// green. It returns one of the Exit codes.
func Wait(c Config) int {
	c.until = c.Now().Add(c.Timeout)
	c.rawGh = c.Gh
	c.settleGh = c.withRetries(c.Gh, time.Time{})
	c.Gh = c.withRetries(c.Gh, c.until)
	p, err := viewPR(c.Gh, c.PR)
	if err != nil {
		return c.ghFailed(err)
	}
	if rc, done := c.mergedState(p, ""); done {
		return rc
	}
	if rc := c.mergeable(p); rc != ExitMerged {
		return rc
	}
	sha := p.HeadRefOid
	if rc := c.containsBase(p.BaseRefName, sha); rc != ExitMerged {
		return rc
	}
	deadline := c.until
	rerun := map[int64]bool{} // the failed runs re-run once: their old entry counts as pending
	last := ""
	// consecutive failures of each poll call: each is printed, and the wait gives up past c.Retries. Each call
	// has its own count, so a check-runs call that succeeds does not hide a pr view that keeps failing.
	checkErrs, viewErrs := 0, 0
	pollFailed := func(n *int, err error) bool {
		c.report(err)
		*n++
		return *n > c.Retries
	}
	for {
		runs, err := checkRuns(c.Gh, sha)
		if err != nil {
			if pollFailed(&checkErrs, err) {
				return ExitGh
			}
			if rc, stop := c.pause(deadline, sha); stop {
				return rc
			}
			continue
		}
		checkErrs = 0
		for i := range runs {
			if rerun[runs[i].ID] {
				runs[i].Status = "queued"
			}
		}
		v, done, bad := judge(runs)
		switch v {
		case green:
			return c.merge(sha, p.BaseRefName)
		case notGreen:
			fmt.Fprintf(c.Err, msgNotGreen, sha)
			return ExitFailed
		case failed:
			if rc := c.failure(bad, rerun); rc != ExitMerged {
				return rc
			}
		default:
			msg := fmt.Sprintf(msgWaiting, sha, done, len(runs))
			if len(runs) == 0 {
				msg = fmt.Sprintf(msgNoChecks, sha)
			}
			if msg != last { // one line per change of state, not one per poll
				fmt.Fprint(c.Err, msg)
				last = msg
			}
		}
		if rc, stop := c.pause(deadline, sha); stop {
			return rc
		}
		now, err := viewPR(c.Gh, c.PR)
		if err != nil {
			if pollFailed(&viewErrs, err) {
				return ExitGh
			}
			continue
		}
		viewErrs = 0
		if rc, done := c.mergedState(now, sha); done { // merged while we waited
			return rc
		}
		if now.HeadRefOid != sha {
			fmt.Fprintf(c.Err, msgMoved, c.PR, sha, now.HeadRefOid)
			return ExitMoved
		}
		if rc := c.mergeable(now); rc != ExitMerged {
			return rc
		}
	}
}

// pause ends the wait at its deadline, else sleeps one interval.
func (c Config) pause(deadline time.Time, sha string) (rc int, stop bool) {
	if !c.Now().Before(deadline) {
		fmt.Fprintf(c.Err, msgTimeout, sha, c.Timeout)
		return ExitTimeout, true
	}
	c.Sleep(c.Interval)
	return 0, false
}

// mergeable refuses a PR that is not open or is a draft.
func (c Config) mergeable(p pr) int {
	if p.State != "OPEN" {
		fmt.Fprintf(c.Err, msgNotOpen, c.PR, p.State)
		return ExitMoved
	}
	if p.IsDraft {
		fmt.Fprintf(c.Err, msgDraft, c.PR, c.PR)
		return ExitDraft
	}
	return ExitMerged
}

// containsBase refuses a head that lacks commits of the current base branch.
func (c Config) containsBase(base, sha string) int {
	n, err := behindBy(c.Gh, base, sha)
	if err != nil {
		return c.ghFailed(err)
	}
	if n > 0 {
		fmt.Fprintf(c.Err, msgBehind, sha, base, n)
		return ExitBehind
	}
	return ExitMerged
}
