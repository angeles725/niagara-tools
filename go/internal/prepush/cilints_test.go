package prepush

import (
	"os"
	"strings"
	"testing"
)

// The CI-lints step (retro coordination-stretch C3): the pushed range goes through the pushed tree's
// `scripts/ci-local.sh --changed --range <range>`, and a failing lint refuses the push.

func (r *repo) ciLocalCalls() string {
	b, err := os.ReadFile(r.ciCalls)
	if err != nil && !os.IsNotExist(err) {
		r.t.Fatal(err)
	}
	return string(b)
}

func TestCILintsGetThePushedRange(t *testing.T) {
	r := newRepo(t)
	sha := r.change("chore: x"+noChangelog+trivial, "tests/x.bats", "x\n")
	r.push(r.update(sha)).want(t, 0)
	if got, want := r.ciLocalCalls(), "--changed --range "+r.base+".."+sha+"\n"; got != want {
		t.Errorf("ci-local calls %q, want %q", got, want)
	}
}

func TestFailingCILintsRefuseThePushBeforeTheOtherGates(t *testing.T) {
	r := newRepo(t)
	t.Setenv("CI_LOCAL_RC", "1")
	t.Setenv("CI_LOCAL_OUT", "FAIL  bats-negation  tests/x.bats:3")
	sha := r.change("chore: x"+noChangelog+trivial, "tests/x.bats", "x\n")
	res := r.push(r.update(sha))
	res.want(t, 1, "FAIL  bats-negation  tests/x.bats:3",
		"pre-push: the changed files fail the CI lints (scripts/ci-local.sh --changed --range "+r.base+".."+sha+
			", exit 1, above); refusing to push\n",
		"pre-push: refused refs/heads/x -> refs/heads/x (exit 1)")
	if strings.Contains(r.stubCalls(), "sweep") {
		t.Error("a later gate ran after the CI lints refused")
	}
}

func TestCILintsStatusIsTheRefusalStatus(t *testing.T) {
	r := newRepo(t)
	t.Setenv("CI_LOCAL_RC", "4") // host budget exceeded: the push waits, never goes through unchecked
	sha := r.change("chore: x"+noChangelog+trivial, "README.md", "x\n")
	r.push(r.update(sha)).want(t, 4, "exit 4, above); refusing to push")
}

func TestATreeWithoutCILocalSkipsTheStepNamingIt(t *testing.T) {
	r := newRepo(t)
	r.run("rm", "-q", "scripts/ci-local.sh")
	r.commit("chore: drop the script" + noChangelog + trivial)
	r.run("update-ref", "refs/remotes/origin/main", "HEAD")
	r.base = r.run("rev-parse", "HEAD")
	sha := r.change("chore: x"+noChangelog, "README.md", "y\n")
	r.push(r.update(sha)).want(t, 0, "pre-push: CI lints SKIPPED — no scripts/ci-local.sh in the work tree\n")
}
