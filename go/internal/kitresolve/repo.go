package kitresolve

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/gitcmd"
)

// repo asks git about the work tree at top; git's own diagnostics go to errTo.
type repo struct {
	top   string
	errTo io.Writer
}

// topLevel is the work tree holding kit, or false when kit is not in one.
func topLevel(kit string) (string, bool) {
	out, ok := gitcmd.Run(kit, nil, "rev-parse", "--show-toplevel")
	return strings.TrimRight(out, "\n"), ok
}

// noPrompt keeps git from asking for credentials on a terminal.
const noPrompt = "GIT_TERMINAL_PROMPT=0"

// fetch runs `git fetch -q origin main` with no credential prompt and reports whether it succeeded;
// its output is dropped (the caller prints the offline WARN).
func (r repo) fetch() bool {
	return gitcmd.Command{Dir: r.top, Env: []string{noPrompt}}.Run("fetch", "-q", "origin", "main").OK()
}

// hasFreshRef reports whether origin/main exists locally.
func (r repo) hasFreshRef() bool {
	_, ok := gitcmd.Run(r.top, r.errTo, "rev-parse", "-q", "--verify", freshRef)
	return ok
}

// dirtyPaths is the number of `git status --porcelain` lines. A status that fails is an error: the
// bash tool read it as a clean tree.
func (r repo) dirtyPaths() (int, error) {
	out, ok := gitcmd.Run(r.top, r.errTo, "status", "--porcelain")
	if !ok {
		return 0, fmt.Errorf("git status failed in %s", r.top)
	}
	return strings.Count(out, "\n"), nil
}

// aheadBehind counts the commits only on HEAD and only on origin/main. HEAD is an ancestor of
// origin/main exactly when ahead is 0 (`git merge-base --is-ancestor HEAD origin/main`), and behind is
// `git rev-list --count HEAD..origin/main`.
func (r repo) aheadBehind() (ahead, behind int, err error) {
	out, ok := gitcmd.Run(r.top, r.errTo, "rev-list", "--left-right", "--count", "HEAD..."+freshRef)
	fields := strings.Fields(out)
	if ok && len(fields) == 2 {
		var errA, errB error
		ahead, errA = strconv.Atoi(fields[0])
		behind, errB = strconv.Atoi(fields[1])
		if errA == nil && errB == nil {
			return ahead, behind, nil
		}
	}
	return 0, 0, fmt.Errorf("git rev-list HEAD...%s failed in %s", freshRef, r.top)
}

// short is `git rev-parse --short[=n] <rev>`; n 0 leaves the length to git. "" when git fails, which the
// callers print as it is (a message, never a decision).
func (r repo) short(rev string, n int) string {
	flag := "--short"
	if n > 0 {
		flag += "=" + strconv.Itoa(n)
	}
	out, _ := gitcmd.Run(r.top, r.errTo, "rev-parse", flag, rev)
	return strings.TrimRight(out, "\n")
}

// checkoutFresh detaches HEAD at origin/main.
func (r repo) checkoutFresh() bool {
	_, ok := gitcmd.Run(r.top, r.errTo, "checkout", "-q", "--detach", freshRef)
	return ok
}
