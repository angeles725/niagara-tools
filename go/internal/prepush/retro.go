package prepush

import "strings"

// retroGate is the continuity + retro gate (BUILD-LOOP.md §7): a range that changes build-relevant
// kit files must carry a ledger update + a pending retro, a `Retro: none (trivial: <reason>)` trailer,
// or a promotion trailer with a structural anchor. It returns the ref's final status.
func (h hook) retroGate(rng string, changed []string, log string) int {
	c := classify(changed)
	if !c.relevant { // nothing build-relevant: only the content half of changed ledger data
		return h.passEarly(c)
	}
	// the trivial waiver short-circuits the retro bookkeeping, not the content half of changed ledger data
	if trivialTrailer.MatchString(log) {
		return h.passEarly(c)
	}
	patch, ok := gitOut(h.s.Err, "diff", "--no-renames", rng, "--", retroPatchPathspec) // a rename would put two retros in one file diff
	if !ok {
		return refuse(h.s.Err, msgPatchFailed, rng)
	}
	// The promotion exit folds already-filed lessons into the core. It is valid ONLY with a structural
	// anchor: an in-range retro header flip pending -> folded (a full promotion), OR an in-range ledger
	// update (a partial promotion keeps its source retro pending and stamps the owed open_issue in a
	// ledger.d/ fragment). The trailer alone is no escape: with neither anchor it falls to the refusal.
	if promotionTrailer.MatchString(log) && (flipsRetroHeader(patch) || c.ledger) {
		return h.sweepStatus()
	}
	// the new-retro exit: the generated ledger + index must actually cohere
	if c.ledger && c.retro != "" {
		return h.sweepStatus()
	}
	printf(h.s.Err, msgNoRetro)
	return 1
}

// passEarly lets a range through without retro bookkeeping, after the content half when the range
// changed ledger data: the early exits never waive the coherence of the data a range changes.
func (h hook) passEarly(c pathClasses) int {
	if !c.ledgerData {
		return 0
	}
	rc, ran := h.runSweep()
	if rc != 0 && ran {
		printf(h.s.Err, msgIncoherent)
	}
	return rc
}

// sweepStatus is the status of the exits that end in the content half.
func (h hook) sweepStatus() int {
	rc, _ := h.runSweep()
	return rc
}

// flipsRetroHeader reports whether a patch flips a retro header pending -> folded: the removed
// pending line and the added folded line must sit in the SAME file's diff, so deleting one pending
// retro and adding another one born folded is not a flip.
func flipsRetroHeader(patch string) bool {
	file := ""
	pending, folded := map[string]bool{}, map[string]bool{}
	for _, line := range strings.Split(patch, "\n") {
		switch {
		case strings.HasPrefix(line, patchFileHeader):
			file = line
		case strings.HasPrefix(line, pendingRemoved):
			pending[file] = true
		case strings.HasPrefix(line, foldedAdded):
			folded[file] = true
		}
	}
	for f := range pending {
		if folded[f] {
			return true
		}
	}
	return false
}
