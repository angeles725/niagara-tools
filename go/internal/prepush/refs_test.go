package prepush

import (
	"os"
	"strings"
	"testing"
)

const ungated = "build-relevant kit files changed without a ledger.d/ fragment"

// ungatedKitWork commits a kit toolbelt file with nothing that closes the retro gate.
func ungatedKitWork(r *repo) string {
	return r.change("feat: new toolbelt script"+noChangelog, "build-n4-module-kit/toolbelt/new.sh", "echo hi\n")
}

func TestDeletedRefIsNotGated(t *testing.T) {
	r := newRepo(t)
	sha := ungatedKitWork(r)
	r.push("(delete) "+zeroSha+" refs/heads/x "+sha).want(t, 0)
}

func TestExistingRemoteBranchGatesRemoteToLocal(t *testing.T) {
	r := newRepo(t)
	first := ungatedKitWork(r)
	second := r.change("docs: notes", "notes.txt", "notes\n")
	// only the docs commit is in <remote sha>..<local sha>: the ungated commit before it is not re-gated
	r.push("refs/heads/x "+second+" refs/heads/x "+first).want(t, 0)
	r.push(r.update(second)).want(t, 1, ungated)
}

func TestNewBranchGatesEverythingSinceOriginMain(t *testing.T) {
	r := newRepo(t)
	sha := ungatedKitWork(r)
	res := r.push("refs/heads/feat " + sha + " refs/heads/feat " + zeroSha)
	res.want(t, 1, ungated, "pre-push: refused refs/heads/feat -> refs/heads/feat (exit 1)")
}

func TestNewBranchFallsBackToOriginHEAD(t *testing.T) {
	r := newRepo(t)
	sha := ungatedKitWork(r)
	r.run("update-ref", "refs/remotes/origin/trunk", r.base)
	r.run("update-ref", "-d", "refs/remotes/origin/main")
	r.run("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/trunk")
	res := r.push("refs/heads/feat " + sha + " refs/heads/feat " + zeroSha)
	res.want(t, 1, ungated)
	if strings.Contains(res.err, "cannot find") {
		t.Errorf("origin/HEAD fallback not used: %s", res.err)
	}
}

func TestNewBranchWithNoBaseRefuses(t *testing.T) {
	r := newRepo(t)
	sha := r.change("docs: notes", "notes.txt", "notes\n")
	r.run("update-ref", "-d", "refs/remotes/origin/main")
	res := r.push("refs/heads/feat " + sha + " refs/heads/feat " + zeroSha)
	res.want(t, 1, "pre-push: cannot find where "+sha+" left origin/main (git fetch origin first); refusing to push unchecked")
}

func TestEveryRefLineIsGated(t *testing.T) {
	r := newRepo(t)
	docs := r.change("docs: notes", "notes.txt", "notes\n")
	bad := ungatedKitWork(r)
	res := r.push(
		"(delete) "+zeroSha+" refs/heads/old "+docs,
		"refs/heads/docs "+docs+" refs/heads/docs "+zeroSha,
		"refs/heads/feat "+bad+" refs/heads/feat "+zeroSha,
	)
	res.want(t, 1, ungated, "pre-push: refused refs/heads/feat -> refs/heads/feat (exit 1)")
	if strings.Contains(res.err, "refused refs/heads/docs") {
		t.Errorf("the passing ref is named as refused: %s", res.err)
	}
}

func TestSeveralPassingRefsPass(t *testing.T) {
	r := newRepo(t)
	one := r.change("docs: notes", "notes.txt", "notes\n")
	two := r.change("docs: more", "notes.txt", "more\n")
	res := r.push("refs/heads/a "+one+" refs/heads/a "+zeroSha, "refs/heads/b "+two+" refs/heads/b "+zeroSha)
	res.want(t, 0)
	if strings.Contains(res.err, "refused") {
		t.Errorf("a passing push names a refusal: %s", res.err)
	}
}

func TestBlankAndShortRefLinesAreSkipped(t *testing.T) {
	r := newRepo(t)
	r.push("", "refs/heads/only-a-ref-name").want(t, 0)
}

// The remote sha is the rest of the line, as the shell's `read` took it: a stray fifth field is part
// of it, never dropped, so the range is not a commit range and the push is refused.
func TestRemoteShaIsTheRestOfTheRefLine(t *testing.T) {
	r := newRepo(t)
	sha := r.change("docs: notes", "notes.txt", "notes\n")
	res := r.push("refs/heads/x\t" + sha + "  refs/heads/x  " + r.base + " extra ")
	res.want(t, 1, "pre-push: git diff "+r.base+" extra.."+sha+" failed; refusing to push unchecked")
	r.push("  refs/heads/x\t"+sha+"  refs/heads/x  "+r.base+" \t").want(t, 0)
}

func TestFailingGitDiffRefuses(t *testing.T) {
	r := newRepo(t)
	sha := r.change("docs: notes", "notes.txt", "notes\n")
	missing := strings.Repeat("1", 40)
	res := r.push("refs/heads/x " + sha + " refs/heads/x " + missing)
	res.want(t, 1, "pre-push: git diff "+missing+".."+sha+" failed; refusing to push unchecked",
		"pre-push: refused refs/heads/x -> refs/heads/x (exit 1)")
}

func TestFailingGitLogRefuses(t *testing.T) {
	r := newRepo(t)
	sha := r.change("docs: notes"+trivial, "notes.txt", "notes\n")
	r.tool("git", `[ "$1" = log ] && exit 128; exec `+r.git+` "$@"`)
	r.push(r.update(sha)).want(t, 1, "pre-push: git log "+r.base+".."+sha+" failed; refusing to push unchecked")
}

// Outside a work tree there is nothing the hook can check: `git rev-parse --show-toplevel` failing refuses
// the push with git's own error, never a silent pass.
func TestOutsideAWorkTreeRefusesWithTheGitError(t *testing.T) {
	r := newRepo(t)
	sha := ungatedKitWork(r)
	t.Chdir(t.TempDir())
	t.Setenv("GIT_CEILING_DIRECTORIES", os.TempDir())
	res := r.push(r.update(sha))
	if res.code != 1 || !strings.Contains(res.err, "pre-push: git rev-parse --show-toplevel failed: fatal: not a git repository") ||
		!strings.HasSuffix(res.err, "; refusing to push unchecked\n") {
		t.Errorf("exit %d, stderr %q; want 1 and the git error in the refusal", res.code, res.err)
	}
}

// A bare repository has no work tree to gate: the push goes on, and a one-line notice says why.
func TestABareRepositoryIsNamedInANotice(t *testing.T) {
	r := newRepo(t)
	sha := ungatedKitWork(r)
	bare := t.TempDir() + "/bare.git"
	r.run("init", "-q", "--bare", bare)
	t.Chdir(bare)
	res := r.push(r.update(sha))
	if res.code != 0 || res.out != "" || res.err != "pre-push: bare repository "+bare+": no work tree, nothing to gate\n" {
		t.Errorf("exit %d, stdout %q, stderr %q; want 0 and the one-line notice", res.code, res.out, res.err)
	}
}
