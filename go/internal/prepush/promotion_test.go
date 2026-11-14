package prepush

import "testing"

// promotionRepo holds a pending retro in the base commit, so a range can flip its header.
func promotionRepo(t *testing.T) *repo {
	r := newRepo(t)
	r.base = r.change("docs: retro", retro, pending)
	return r
}

func TestPromotionWithAHeaderFlipPassesThroughTheSweep(t *testing.T) {
	r := promotionRepo(t)
	sha := r.change("docs: fold"+promotion+noChangelog, kitFile, "echo\n", retro, folded)
	r.push(r.update(sha)).want(t, 0)
	if !swept(r) {
		t.Error("the content half did not run")
	}
}

func TestPromotionWithALedgerUpdatePasses(t *testing.T) {
	r := promotionRepo(t)
	sha := r.change("docs: fold"+promotion+noChangelog, kitFile, "echo\n", fragment, "log\n")
	r.push(r.update(sha)).want(t, 0)
	if !swept(r) {
		t.Error("the content half did not run")
	}
}

func TestPromotionSweepFailureRefusesWithItsStatus(t *testing.T) {
	r := promotionRepo(t)
	t.Setenv("SWEEP_RC", "1")
	sha := r.change("docs: fold"+promotion+noChangelog, kitFile, "echo\n", retro, folded)
	r.push(r.update(sha)).want(t, 1, "pre-push: refused refs/heads/x -> refs/heads/x (exit 1)")
}

func TestPromotionTrailerAloneIsNoEscape(t *testing.T) {
	r := promotionRepo(t)
	sha := r.change("docs: fold"+promotion+noChangelog, kitFile, "echo\n")
	r.push(r.update(sha)).want(t, 1, "build-relevant kit files changed")
}

func TestPromotionWithABodyOnlyRetroEditRefuses(t *testing.T) {
	r := promotionRepo(t)
	sha := r.change("docs: fold"+promotion+noChangelog, kitFile, "echo\n", retro, pending+"more body\n")
	r.push(r.update(sha)).want(t, 1, "build-relevant kit files changed")
}

// One pending retro deleted and another added already folded is not a flip: the two header lines must
// sit in the same file's diff.
func TestPendingAndFoldedInDifferentFilesIsNoFlip(t *testing.T) {
	r := promotionRepo(t)
	r.run("rm", "-q", retro)
	sha := r.change("docs: fold"+promotion+noChangelog, kitFile, "echo\n",
		"build-n4-module-kit/retros/2026-09-05-new.md", "<!-- review-status: folded -->\n# another retro, born folded\n")
	r.push(r.update(sha)).want(t, 1, "build-relevant kit files changed")
}

func TestFailingRetroPatchDiffRefuses(t *testing.T) {
	r := promotionRepo(t)
	sha := r.change("feat: change"+noChangelog, kitFile, "echo\n")
	r.tool("git", `[ "$1" = diff ] && [ "$2" != --name-only ] && [ "$2" != --quiet ] && exit 128; exec `+r.git+` "$@"`)
	r.push(r.update(sha)).want(t, 1, "pre-push: git diff "+r.base+".."+sha+" -- retros failed; refusing to push unchecked")
}
