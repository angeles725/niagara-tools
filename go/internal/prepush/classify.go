package prepush

// pathClasses is what the retro gate needs to know about the paths a range changes.
type pathClasses struct {
	relevant   bool   // a build-relevant path changed: the range owes retro bookkeeping
	ledgerData bool   // a file the content half checks changed
	ledger     bool   // a ledger update: a ledger.d/ fragment or a BUILD-STATE.src.md edit
	retro      string // the first changed retro file ("" when none)
}

// classify sorts the changed paths of a range into the retro gate's classes (paths.go).
func classify(changed []string) pathClasses {
	var c pathClasses
	for _, path := range changed {
		c.relevant = c.relevant || isBuildRelevant(path)
		c.ledgerData = c.ledgerData || ledgerData.MatchString(path)
		c.ledger = c.ledger || ledgerUpdate.MatchString(path)
		if c.retro == "" && retroFile.MatchString(path) && !retroIndex.MatchString(path) {
			c.retro = path
		}
	}
	return c
}

func isBuildRelevant(path string) bool {
	if !relevantRoots.MatchString(path) {
		return false
	}
	for _, not := range notRelevant {
		if not.MatchString(path) {
			return false
		}
	}
	return true
}
