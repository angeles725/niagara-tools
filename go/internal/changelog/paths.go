package changelog

import (
	"regexp"
	"strings"
)

// behaviourRoots are the trees whose changes change kit behaviour. notBehaviour are the retro gate's own
// exclusions inside them (.githooks/pre-push `relevant`): built binaries and the ledger/retro bookkeeping.
var (
	behaviourRoots = []string{"go/", "build-n4-module-kit/", "tools/", "scripts/"}
	notBehaviour   = regexp.MustCompile(`^(go/bin/|build-n4-module-kit/(BUILD-STATE(\.src)?\.md$|ledger\.d/|retros/))`)
)

// IsBehaviour reports whether a changed path changes kit behaviour (the paths the gate applies to).
func IsBehaviour(path string) bool {
	for _, r := range behaviourRoots {
		if strings.HasPrefix(path, r) {
			return !notBehaviour.MatchString(path)
		}
	}
	return false
}
