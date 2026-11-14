package prepush

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const card = "build-n4-module-kit/types/logic.md"

// liveCardPush commits a change to a live card that every other gate lets through, and pushes it with
// the resolver stub set to exit rc after printing out.
func liveCardPush(t *testing.T, r *repo, rc, out string) result {
	t.Helper()
	sha := r.change("docs: card"+trivial+noChangelog, card, "# logic v2\n")
	return resolverPush(t, r, sha, rc, out)
}

func resolverPush(t *testing.T, r *repo, sha, rc, out string) result {
	t.Helper()
	file := filepath.Join(filepath.Dir(r.dir), "cite-out")
	if err := os.WriteFile(file, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CITE_OUT", file)
	t.Setenv("CITE_RC", rc)
	return r.push(r.update(sha))
}

func TestLiveCardFindingsRefuseAndNameTheFindingRows(t *testing.T) {
	r := newRepo(t)
	finding := card + "\t3\tpath\tnope.java\t\tMISSING_FILE\tno file\n"
	res := liveCardPush(t, r, "1", finding+
		card+"\t4\tretro\tok\tΔ1\tRESOLVED\tfine\n"+
		card+"\t5\tretro\told\tΔ1\tHISTORICAL_DRIFT\tgone\n"+
		card+"\t6\turl\tx\t\tUNVERIFIABLE_KIND\turl\n"+
		card+"\t7\tretro\t<stem>\t\tSKIPPED_TEMPLATE\tplaceholder\n"+
		"kit-cite-resolve: strict findings: LIVE 1, HISTORY 0\n")
	want := finding + "pre-push: live doctrine has unresolved pointers (kit-cite-resolve --strict; rows above, first 20); refusing to push unchecked\n" +
		"pre-push: refused refs/heads/x -> refs/heads/x (exit 1)\n"
	if res.code != 1 || res.err != want {
		t.Errorf("exit %d, stderr\n%s\nwant 1 and\n%s", res.code, res.err, want)
	}
}

func TestLiveCardFindingsPrintTheFirstTwentyRows(t *testing.T) {
	r := newRepo(t)
	var rows strings.Builder
	for i := 1; i <= 25; i++ {
		fmt.Fprintf(&rows, "%s\t%d\tpath\tf%d\t\tMISSING_FILE\tno file\n", card, i, i)
	}
	res := liveCardPush(t, r, "1", rows.String())
	res.want(t, 1, "\tf20\t")
	if strings.Contains(res.err, "\tf21\t") {
		t.Errorf("more than 20 finding rows printed: %s", res.err)
	}
}

func TestLiveCardThatResolvesChecksTheStrictLiveSetOfThePushedTree(t *testing.T) {
	r := newRepo(t)
	liveCardPush(t, r, "0", "").want(t, 0)
	want := "resolve --kit build-n4-module-kit --format tsv --strict README.md " + card + "\n"
	if got := r.stubCalls(); got != want {
		t.Errorf("resolver calls %q, want %q", got, want)
	}
}

func TestAbsentRootIsTheOneNamedSkip(t *testing.T) {
	r := newRepo(t)
	res := liveCardPush(t, r, "3", "kit-cite-resolve: cannot read research root /no/research: open /no/research: no such file or directory\n")
	want := "pre-push: live cite check SKIPPED — roots absent (research /no/research: open /no/research)\n"
	if res.code != 0 || res.err != want {
		t.Errorf("exit %d, stderr %q; want 0 and %q", res.code, res.err, want)
	}
}

func TestUnreadableRootRefusesNamingTheCause(t *testing.T) {
	r := newRepo(t)
	res := liveCardPush(t, r, "3", "kit-cite-resolve: cannot read memory root /m: open /m: permission denied\n")
	res.want(t, 1, "permission denied",
		"pre-push: kit-cite-resolve failed (exit 3; above), a tool failure is not a zero result; refusing to push unchecked")
	if strings.Contains(res.err, "SKIPPED") {
		t.Errorf("an unreadable root is skipped: %s", res.err)
	}
}

func TestAnyOtherResolverExitRefusesWithItsLastFiveLines(t *testing.T) {
	r := newRepo(t)
	res := liveCardPush(t, r, "7", "l1\nl2\nl3\nl4\nl5\nl6\n")
	want := "l2\nl3\nl4\nl5\nl6\n" +
		"pre-push: kit-cite-resolve failed (exit 7; above), a tool failure is not a zero result; refusing to push unchecked\n" +
		"pre-push: refused refs/heads/x -> refs/heads/x (exit 1)\n"
	if res.code != 1 || res.err != want {
		t.Errorf("exit %d, stderr\n%s\nwant 1 and\n%s", res.code, res.err, want)
	}
}

func TestRangeWithNoLiveFileNeverCallsTheResolver(t *testing.T) {
	r := newRepo(t)
	sha := r.change("feat: script"+trivial+noChangelog, "scripts/ng-deploy.sh", "echo\n", "build-n4-module-kit/toolbelt/build.sh", "echo\n")
	resolverPush(t, r, sha, "1", "").want(t, 0)
	if calls := r.stubCalls(); calls != "" {
		t.Errorf("the resolver ran: %s", calls)
	}
}

func TestResolverChangeRechecksTheLiveSet(t *testing.T) {
	for _, path := range []string{
		"go/internal/citeresolve/resolve.go", "go/internal/citeparse/parse.go",
		"go/cmd/kit-cite-resolve/main.go", "build-n4-module-kit/toolbelt/kit-cite-resolve.sh",
	} {
		r := newRepo(t)
		sha := r.change("feat: resolver"+trivial+noChangelog, path, stubResolver+"# changed\n")
		resolverPush(t, r, sha, "1", "").want(t, 1, "live doctrine has unresolved pointers")
	}
}

func TestWorkingTreeThatDiffersFromThePushedCommitRefuses(t *testing.T) {
	r := newRepo(t)
	sha := r.change("docs: card"+trivial+noChangelog, card, "# logic v2\n")
	r.write(card, "# not what is pushed\n")
	res := resolverPush(t, r, sha, "0", "")
	res.want(t, 1, "pre-push: the working tree differs from the pushed "+sha+
		" on the live doctrine set or the resolver (check out the pushed commit); refusing to push unchecked")
	if calls := r.stubCalls(); calls != "" {
		t.Errorf("the resolver ran on other bytes: %s", calls)
	}
}

func TestFailingGitLsTreeRefuses(t *testing.T) {
	r := newRepo(t)
	sha := r.change("docs: card"+trivial+noChangelog, card, "# logic v2\n")
	r.tool("git", `[ "$1" = ls-tree ] && exit 128; exec `+r.git+` "$@"`)
	resolverPush(t, r, sha, "0", "").want(t, 1, "pre-push: git ls-tree "+sha+" failed; refusing to push unchecked")
}
