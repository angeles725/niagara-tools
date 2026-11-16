package mergewait

import "fmt"

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
// times, Interval apart, whatever the call returned (GitHub may show MERGED a moment later); a read that
// fails is printed and counts as one of them. (retro 2026-11-15 coordination-stretch T16)
func (c Config) merge(sha string, base string) int {
	if rc := c.containsBase(base, sha); rc != ExitMerged {
		return rc
	}
	_, mergeErr := c.Gh("pr", "merge", c.PR, "--"+c.Method, "--match-head-commit", sha)
	reads := 1 + c.Retries
	state := "" // the last state read; "" when no read succeeded
	for i := 0; i < reads; i++ {
		if i > 0 {
			c.Sleep(c.Interval)
		}
		p, err := viewPR(c.Gh, c.PR)
		if err != nil {
			c.report(err)
			continue
		}
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
	case mergeErr != nil && state == "":
		return ExitGh // the call and every read failed, each printed above
	case mergeErr != nil:
		fmt.Fprintf(c.Err, msgMergeFail, c.PR, sha, state)
		return ExitRefused
	case state == "":
		fmt.Fprintf(c.Err, msgUnread, c.PR, sha, reads)
		return ExitPending
	}
	fmt.Fprintf(c.Err, msgUnsettled, c.PR, sha, state, reads)
	return ExitPending
}
