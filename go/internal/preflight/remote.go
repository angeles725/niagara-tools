package preflight

import (
	"io"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/gitcmd"
)

// remoteBackup WARNs when the gradle root's git checkout keeps its commits on this disk only: no remote, or
// a branch with no upstream. Pushing or backing them up is the operator's decision; the row only names it.
// Silent outside a git work tree, on a detached HEAD and when the branch tracks an upstream
// [ev: retro delivery-guards-and-ui-contrast Δ3].
func (r *run) remoteBackup() {
	if r.win {
		return
	}
	git := func(args ...string) (string, bool) {
		out, ok := gitcmd.Run(r.gr, io.Discard, args...)
		return strings.TrimSpace(out), ok
	}
	top, ok := git("rev-parse", "--show-toplevel")
	if !ok || top == "" {
		return
	}
	if remotes, _ := git("remote"); remotes == "" {
		r.row("WARN", "remote-backup", top+" has no git remote: its commits exist on this disk only "+
			"(push or back them up: the operator's call)")
		return
	}
	branch, ok := git("symbolic-ref", "--quiet", "--short", "HEAD")
	if !ok || branch == "" {
		return
	}
	if _, ok := git("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"); !ok {
		r.row("WARN", "remote-backup", top+" branch "+branch+" has no upstream: its commits are not pushed "+
			"(git push -u origin "+branch+", or the operator's backup)")
	}
}
