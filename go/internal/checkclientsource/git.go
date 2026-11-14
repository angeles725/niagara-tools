package checkclientsource

import (
	"io"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/gitcmd"
)

// checkout runs git from a module root. Every call reads local refs only; nothing is fetched.
type checkout struct {
	dir    string
	stderr io.Writer // where git's own diagnostics go when a call does not silence them
}

// git runs git from the module root and returns its stdout without the final newlines, and whether it
// exited 0. Git's stderr goes to errTo (nil drops it).
func (c checkout) git(errTo io.Writer, args ...string) (string, bool) {
	out, ok := gitcmd.Run(c.dir, errTo, args...)
	return strings.TrimRight(out, "\n"), ok
}

// insideWorkTree reports whether the module root sits in a git work tree.
func (c checkout) insideWorkTree() bool {
	out, _ := c.git(nil, "rev-parse", "--is-inside-work-tree")
	return out == "true"
}

// topLevel is the work tree's top-level directory.
func (c checkout) topLevel() string {
	out, _ := c.git(c.stderr, "rev-parse", "--show-toplevel")
	return out
}

// head is what `git rev-parse HEAD` prints: the commit id, or the word HEAD on a branch with no commit.
func (c checkout) head() string {
	out, _ := c.git(nil, "rev-parse", "HEAD")
	return out
}

// branch is the checked-out branch, "" on a detached HEAD.
func (c checkout) branch() string {
	out, _ := c.git(nil, "symbolic-ref", "--quiet", "--short", "HEAD")
	return out
}

// commit is the id rev names in this checkout, "" when it names no commit here.
func (c checkout) commit(rev string) string {
	out, _ := c.git(nil, "rev-parse", "--verify", "--quiet", rev+"^{commit}")
	return out
}

// isAncestorOfHead reports whether HEAD descends from the commit id.
func (c checkout) isAncestorOfHead(id string) bool {
	_, ok := c.git(nil, "merge-base", "--is-ancestor", id, "HEAD")
	return ok
}

// behindUpstream is how many commits HEAD is behind its upstream tracking ref as of the last fetch; 0
// when there is no upstream.
func (c checkout) behindUpstream() int {
	out, _ := c.git(nil, "rev-list", "--count", "HEAD..@{upstream}")
	n, _ := strconv.Atoi(out)
	return n
}
