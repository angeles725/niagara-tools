package prepush

import "testing"

func TestBehaviourChangeWithNoChangelogEntryRefuses(t *testing.T) {
	r := newRepo(t)
	// the retro trailer alone is not enough: the changelog gate runs before the retro exits
	sha := r.change("feat: change x"+trivial, "go/x.go", "package x\n")
	res := r.push(r.update(sha))
	res.want(t, 1, "changelog-gate: kit behaviour changed with no CHANGELOG entry:", "  go/x.go",
		"pre-push: refused refs/heads/x -> refs/heads/x (exit 1)")
}

func TestChangelogFragmentPassesAndTheGateReportsOnStdout(t *testing.T) {
	r := newRepo(t)
	sha := r.change("feat: change x"+trivial, "go/x.go", "package x\n",
		"changelog.d/2026-10-25-change-x.md", "### Changed — x\n\n- x changed\n")
	res := r.push(r.update(sha))
	res.want(t, 0)
	if want := "changelog-gate: PASS — fragment changelog.d/2026-10-25-change-x.md\n"; res.out != want {
		t.Errorf("stdout %q, want %q", res.out, want)
	}
}

func TestMalformedChangelogTrailerRefusesEvenADocsOnlyRange(t *testing.T) {
	r := newRepo(t)
	sha := r.change("docs: notes\n\nChangelog: nothing to say", "notes.txt", "notes\n")
	r.push(r.update(sha)).want(t, 1, "malformed changelog fragment or trailer in range")
}
