package prepush

import (
	"strings"
	"testing"
)

const (
	kitFile  = "build-n4-module-kit/toolbelt/build.sh"
	fragment = "build-n4-module-kit/ledger.d/2026-09-05-demo.md"
	source   = "build-n4-module-kit/BUILD-STATE.src.md"
	retro    = "build-n4-module-kit/retros/2026-09-04-demo.md"
	pending  = "<!-- review-status: pending -->\n# demo\n"
	folded   = "<!-- review-status: folded -->\n# demo\n"
	noRetro  = "pre-push: build-relevant kit files changed without a ledger.d/ fragment (or a BUILD-STATE.src.md edit) + a pending retro,\n" +
		"          and no 'Retro: none (trivial: <reason>)' trailer.\n" +
		"          See BUILD-LOOP.md §7.\n" +
		"pre-push: refused refs/heads/x -> refs/heads/x (exit 1)\n"
	promotion = "\n\nRetro: promotion (folds demo Δ1)"
)

// swept reports whether the content half ran.
func swept(r *repo) bool { return strings.Contains(r.stubCalls(), "sweep ") }

func TestBuildRelevantChangeWithNoRetroBookkeepingRefuses(t *testing.T) {
	for _, path := range []string{kitFile, "scripts/ng-deploy.sh", "go/internal/javasrc/javasrc.go"} {
		r := newRepo(t)
		sha := r.change("feat: change"+noChangelog, path, "echo\n")
		if res := r.push(r.update(sha)); res.code != 1 || res.err != noRetro {
			t.Errorf("%s: exit %d, stderr\n%s\nwant 1 and\n%s", path, res.code, res.err, noRetro)
		}
	}
}

func TestPathsOutsideTheBuildRelevantSetPassWithoutASweep(t *testing.T) {
	for _, path := range []string{"README.md", "Group/Mod/src/A.java", "go/bin/n4lint", "tools/x.sh"} {
		r := newRepo(t)
		sha := r.change("chore: change"+noChangelog, path, "x\n")
		r.push(r.update(sha)).want(t, 0)
		if swept(r) {
			t.Errorf("%s: the content half ran on a range with no ledger data", path)
		}
	}
}

func TestTrivialTrailerWaivesTheRetroBookkeeping(t *testing.T) {
	r := newRepo(t)
	sha := r.change("chore: typo"+trivial+noChangelog, kitFile, "echo\n")
	r.push(r.update(sha)).want(t, 0)
	if swept(r) {
		t.Error("the content half ran on a code-only trivial range")
	}
}

func TestLedgerUpdatePlusARetroPassesThroughTheSweep(t *testing.T) {
	for _, ledger := range []string{fragment, source} {
		r := newRepo(t)
		sha := r.change("feat: change"+noChangelog, kitFile, "echo\n", ledger, "log\n", retro, pending)
		r.push(r.update(sha)).want(t, 0)
		if !swept(r) {
			t.Errorf("%s: the content half did not run", ledger)
		}
	}
}

func TestLedgerUpdateWithoutARetroRefuses(t *testing.T) {
	r := newRepo(t)
	sha := r.change("feat: change"+noChangelog, kitFile, "echo\n", fragment, "log\n")
	r.push(r.update(sha)).want(t, 1, "build-relevant kit files changed")
}

func TestRetroWithoutALedgerUpdateRefuses(t *testing.T) {
	r := newRepo(t)
	// retros/INDEX.md is no retro, and a nested ledger.d file is no fragment
	sha := r.change("feat: change"+noChangelog, kitFile, "echo\n", retro, pending,
		"build-n4-module-kit/retros/INDEX.md", "x\n", "build-n4-module-kit/ledger.d/sub/x.md", "x\n")
	r.push(r.update(sha)).want(t, 1, "build-relevant kit files changed")
}

func TestIndexFileIsNotARetro(t *testing.T) {
	r := newRepo(t)
	sha := r.change("feat: change"+noChangelog, kitFile, "echo\n", fragment, "log\n", "build-n4-module-kit/retros/INDEX.md", "x\n")
	r.push(r.update(sha)).want(t, 1, "build-relevant kit files changed")
}

func TestIncoherentLedgerDataFailsTheNewRetroExitWithTheSweepStatus(t *testing.T) {
	r := newRepo(t)
	t.Setenv("SWEEP_RC", "3")
	sha := r.change("feat: change"+noChangelog, kitFile, "echo\n", fragment, "log\n", retro, pending)
	res := r.push(r.update(sha))
	res.want(t, 3, "pre-push: refused refs/heads/x -> refs/heads/x (exit 3)")
	if strings.Contains(res.err, "incoherent") {
		t.Errorf("the early-exit message is printed at the new-retro exit: %s", res.err)
	}
}

func TestDataOnlyRangeRunsTheContentHalf(t *testing.T) {
	for _, path := range []string{retro, fragment, source, "build-n4-module-kit/BUILD-STATE.md"} {
		r := newRepo(t)
		sha := r.change("docs: data", path, pending)
		r.push(r.update(sha)).want(t, 0)
		if !swept(r) {
			t.Errorf("%s: the content half did not run", path)
		}
	}
}

func TestIncoherentDataOnlyRangeRefuses(t *testing.T) {
	r := newRepo(t)
	t.Setenv("SWEEP_RC", "1")
	sha := r.change("docs: retro", retro, pending)
	r.push(r.update(sha)).want(t, 1, "pre-push: ledger data in range is incoherent (BUILD-STATE.src.md, ledger.d/ or "+
		"retros/; sweep-build-state.sh above); refusing to push\n")
}

func TestTrivialTrailerDoesNotWaiveIncoherentLedgerData(t *testing.T) {
	r := newRepo(t)
	t.Setenv("SWEEP_RC", "1")
	sha := r.change("chore: typo"+trivial+noChangelog, kitFile, "echo\n", retro, pending)
	r.push(r.update(sha)).want(t, 1, "ledger data in range is incoherent")
}
