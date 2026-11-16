package mergewait

import (
	"fmt"
	"time"
)

// mergedState decides a PR that is MERGED: success when it was merged at the checked head sha (or at any
// head when sha is ""), ExitMoved when another head was merged. done is false for a PR not merged.
func (c Config) mergedState(p pr, sha string) (rc int, done bool) {
	if p.State != "MERGED" {
		return 0, false
	}
	if sha != "" && p.HeadRefOid != sha {
		fmt.Fprintf(c.Err, msgOtherHead, c.PR, p.HeadRefOid, sha)
		return ExitMoved, true
	}
	fmt.Fprintf(c.Out, msgAlready, c.PR, p.HeadRefOid, p.mergeOid())
	return ExitMerged, true
}

// merge checks the base once more (it may have moved while the checks ran), asks gh to merge exactly sha
// and lets the PR state decide: what the merge call returned is only printed. The state is read 1+Retries
// times, Interval apart (GitHub may show MERGED a moment later), or once when GitHub refused the merge outright
// on its only send (a refusal of a call sent again after a network error may answer a first send that merged);
// a read that fails is printed and counts as one of them. The merge call and these reads keep their network
// retries past --timeout: the green checks authorized the merge. (retro 2026-11-15 coordination-stretch T16, T23, T27)
func (c Config) merge(sha string, base string) int {
	if rc := c.containsBase(base, sha); rc != ExitMerged {
		return rc
	}
	sent := 0
	send := c.withRetries(func(args ...string) (string, error) { sent++; return c.rawGh(args...) }, time.Time{})
	_, mergeErr := send("pr", "merge", c.PR, "--"+c.Method, "--match-head-commit", sha)
	reads := 1 + c.Retries
	if mergeErr != nil && sent == 1 && isRefusal(mergeErr) {
		reads = 1
	}
	state, answered := "", 0 // the last state read, and how many reads answered
	for i := 0; i < reads; i++ {
		if i > 0 {
			c.Sleep(c.Interval)
		}
		p, err := viewPR(c.settleGh, c.PR)
		if err != nil {
			c.report(err)
			continue
		}
		answered++
		if p.State == "MERGED" {
			if p.HeadRefOid != sha {
				fmt.Fprintf(c.Err, msgOtherHead, c.PR, p.HeadRefOid, sha)
				return ExitMoved
			}
			fmt.Fprintf(c.Out, msgMerged, c.PR, sha, "--"+c.Method, p.mergeOid())
			return ExitMerged
		}
		state = p.State
	}
	switch {
	case mergeErr != nil && answered == 0:
		return ExitGh // the call and every read failed, each printed above
	case mergeErr != nil:
		fmt.Fprintf(c.Err, msgMergeFail, c.PR, sha, state)
		return ExitRefused
	case answered == 0:
		fmt.Fprintf(c.Err, msgUnread, c.PR, sha, reads)
		return ExitPending
	case state != "OPEN":
		fmt.Fprintf(c.Err, msgClosedAfter, c.PR, state, sha)
		return ExitMoved
	}
	fmt.Fprintf(c.Err, msgUnsettled, c.PR, sha, reads, answered)
	return ExitPending
}
