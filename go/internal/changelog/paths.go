package changelog

import (
	"fmt"
	"os/exec"
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

type gitRunner func(args ...string) (string, error)

func realGit(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	if ee, ok := err.(*exec.ExitError); ok {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(ee.Stderr)))
	}
	return string(out), err
}

func nonEmpty(out string) []string {
	var r []string
	for _, l := range strings.Split(out, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			r = append(r, l)
		}
	}
	return r
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
