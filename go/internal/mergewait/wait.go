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
	Gh                Gh
	Sleep             func(time.Duration)
	Now               func() time.Time
	Out, Err          io.Writer
}

// Wait checks the PR, waits for the checks of its head commit and merges that commit when they are
// green. It returns one of the Exit codes.
func Wait(c Config) int {
	p, err := viewPR(c.Gh, c.PR)
	if err != nil {
		return c.ghFailed(err)
	}
	if rc := c.mergeable(p); rc != ExitMerged {
		return rc
	}
	sha := p.HeadRefOid
	if rc := c.containsBase(p.BaseRefName, sha); rc != ExitMerged {
		return rc
	}
	deadline := c.Now().Add(c.Timeout)
	rerun := map[int64]bool{} // the failed runs re-run once: their old entry counts as pending
	last := ""
	for {
		runs, err := checkRuns(c.Gh, sha)
		if err != nil {
			return c.ghFailed(err)
		}
		for i := range runs {
			if rerun[runs[i].ID] {
				runs[i].Status = "queued"
			}
		}
		v, done, bad := judge(runs)
		switch v {
		case green:
			return c.merge(p, sha)
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
		if !c.Now().Before(deadline) {
			fmt.Fprintf(c.Err, msgTimeout, sha, c.Timeout)
			return ExitTimeout
		}
		c.Sleep(c.Interval)
		now, err := viewPR(c.Gh, c.PR)
		if err != nil {
			return c.ghFailed(err)
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

// merge checks the base once more (it may have moved while the checks ran) and merges exactly sha.
func (c Config) merge(p pr, sha string) int {
	if rc := c.containsBase(p.BaseRefName, sha); rc != ExitMerged {
		return rc
	}
	if _, err := c.Gh("pr", "merge", c.PR, "--"+c.Method, "--match-head-commit", sha); err != nil {
		fmt.Fprintln(c.Err, err)
		fmt.Fprintf(c.Err, msgMergeFail, c.PR, sha)
		return ExitRefused
	}
	fmt.Fprintf(c.Out, msgMerged, c.PR, sha, "--"+c.Method)
	return ExitMerged
}

func (c Config) ghFailed(err error) int {
	fmt.Fprintf(c.Err, "%s%v\n", msgPrefix, err)
	return ExitGh
}
