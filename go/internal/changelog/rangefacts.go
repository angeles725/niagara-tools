package changelog

import (
	"fmt"
	"strings"
)

// rangeFacts is what the gate reads from git for one range: the changed paths, the changelog.d/ paths
// added or modified, and the commit messages.
type rangeFacts struct {
	rng, head      string
	changed, added []string
	log            string
}

// malformedRange is a range that is not <base>..<head>; it prints the usage, not a git error.
type malformedRange struct{ rng string }

func (m malformedRange) Error() string { return fmt.Sprintf("malformed range %q", m.rng) }

func readRange(rng string, git gitRunner) (*rangeFacts, error) {
	ends := strings.Split(rng, "..")
	if len(ends) != 2 || ends[0] == "" || ends[1] == "" || strings.HasPrefix(ends[1], ".") {
		return nil, malformedRange{rng}
	}
	for _, e := range ends {
		if _, err := git("rev-parse", "--verify", "--quiet", e+"^{commit}"); err != nil {
			return nil, fmt.Errorf("range end %q is not a commit", e)
		}
	}
	changed, err := git("diff", "--name-only", "--no-renames", rng)
	if err != nil {
		return nil, err
	}
	added, err := git("diff", "--name-only", "--no-renames", "--diff-filter=AM", rng, "--", FragmentDir+"/")
	if err != nil {
		return nil, err
	}
	log, err := git("log", "--format=%B", rng)
	if err != nil {
		return nil, err
	}
	return &rangeFacts{rng: rng, head: ends[1], changed: nonEmpty(changed), added: nonEmpty(added), log: log}, nil
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
