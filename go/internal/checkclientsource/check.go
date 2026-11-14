package checkclientsource

import (
	"fmt"
	"io"
)

// report prints the rows of a run and remembers whether one was a FAIL.
type report struct {
	out    io.Writer
	failed bool
}

// row prints `<STATUS>  source-of-truth  <detail>`. The detail never holds a status word.
func (r *report) row(status, detail string) {
	fmt.Fprintf(r.out, "%-4s  source-of-truth  %s\n", status, detail)
	r.failed = r.failed || status == "FAIL"
}

// check compares the checkout holding root with the declared source of truth, one row per finding.
func check(root string, d declaration, c checkout, r *report) {
	if !c.insideWorkTree() {
		r.row("WARN", root+" is not inside a git work tree — it cannot answer which commit is deployed; put the module in a repository")
		return
	}
	top, head, branch := c.topLevel(), c.head(), c.branch()
	if branch == "" {
		at := head
		if at == "" {
			at = "?"
		}
		r.row("FAIL", "detached HEAD at "+at+" in "+top+" — check out the source-of-truth branch before building")
	}
	switch {
	case d.raw == "":
		r.row("WARN", "source_of_truth undeclared for this module — record <path>@<branch>@<commit> in its BUILD-STATE.src.md envelope")
	case d.path == "":
		r.row("WARN", "source_of_truth is unknown — honest, but this checkout ("+top+") is unchecked")
	default:
		checkDeclared(d, c, r, top, head, branch)
	}
	// Advisory only: behind the upstream tracking ref as of the last fetch (no network).
	if branch == "" {
		return
	}
	if n := c.behindUpstream(); n > 0 {
		r.row("WARN", fmt.Sprintf("%s is behind its upstream tracking ref by %d commit(s) as of the last fetch — pull before building", branch, n))
	}
}

// checkDeclared holds the three comparisons of a declared <path>@<branch>@<commit>.
func checkDeclared(d declaration, c checkout, r *report, top, head, branch string) {
	on := ""
	if branch != "" {
		on = " on " + branch
	}
	if !d.names(top) {
		r.row("FAIL", "work tree "+top+" is not the declared source of truth "+d.path+" — build from the declared checkout")
	}
	if branch != "" && d.branch != unknown && branch != d.branch {
		r.row("FAIL", "checked-out branch "+branch+" is not the declared source-of-truth branch "+d.branch)
	}
	if d.commit == unknown {
		if !r.failed {
			r.row("PASS", top+on+" matches the declared path/branch (commit unknown)")
		}
		return
	}
	switch id := c.commit(d.commit); {
	case id == "":
		r.row("FAIL", "declared commit "+d.commit+" is not in this checkout — a stale clone (fetch it) or the wrong tree")
	case id == head:
		r.row("PASS", top+" at the declared commit "+d.commit+on)
	case c.isAncestorOfHead(id):
		r.row("PASS", top+" is ahead of the declared commit "+d.commit+" (new local work) — update source_of_truth at close")
	default:
		r.row("FAIL", "HEAD "+first(head, 12)+" is behind or diverged from the declared commit "+d.commit+" — stale checkout")
	}
}

// first is the first n bytes of s, or all of it when shorter.
func first(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
