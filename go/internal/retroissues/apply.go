package retroissues

import (
	"fmt"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/atomicfile"
	"github.com/angeles725/niagara-tools/go/internal/retrogh"
	"github.com/angeles725/niagara-tools/go/internal/retrorows"
)

func state(is retrogh.Issue) string {
	if is.Closed() {
		return "closed"
	}
	return "open"
}

func labelsOf(g *group) string { return strings.Join(g.labels(), ", ") }

// report prints what every group would do (or, under --apply, is about to do). It returns ExitFail when a group was
// refused by the scrub.
func (a *app) report(gs []*group) int {
	code := ExitOK
	for _, g := range gs {
		switch g.kind {
		case "create":
			a.outf("create: %s\n  labels: %s\n  ---\n", g.primary().title, labelsOf(g))
			for _, l := range strings.Split(strings.TrimRight(g.body(), "\n"), "\n") {
				a.outf("  %s\n", l)
			}
			a.outf("  ---\n")
		case "refused":
			code = ExitFail
			names := make([]string, len(g.members))
			for i, m := range g.members {
				names[i] = m.label()
			}
			a.outf("refused: %s: %s\n", strings.Join(names, ", "), g.reason)
		default:
			for _, m := range g.members {
				switch g.kind {
				case "exists":
					if g.issue.Closed() {
						a.outf("exists #%d (closed): %s\n", g.number, m.label())
					} else {
						a.outf("exists #%d: %s\n", g.number, m.label())
					}
				case "dup":
					a.outf("probable duplicate of #%d (open) %q (score %.2f): %s\n", g.number, g.issue.Title, g.score, m.label())
				case "fixed":
					a.outf("already fixed in #%d (closed) %q: %s\n", g.number, g.issue.Title, m.label())
				}
			}
		}
	}
	return code
}

// execute files the issues and the comments of every group, in order, and points every row at its issue. It stops at the
// first gh failure: what ran stays done (every marker makes a rerun skip it).
func (a *app) execute(gs []*group) int {
	labelled := false
	for _, g := range gs {
		if g.kind == "create" || g.kind == "dup" || g.kind == "fixed" {
			if !labelled { // a comment adds the retro-delta label too: ensure the labels before any write
				if err := retrogh.EnsureLabels(a.run, map[string]string{
					retrogh.Label:          "A delta proposed by a retro",
					"area:mcp-n4":          "retro delta for mcp-n4",
					"area:build-n4-module": "retro delta for build-n4-module",
				}); err != nil {
					a.errf("cannot create the labels: %v", err)
					return ExitFail
				}
				labelled = true
			}
		}
		switch g.kind {
		case "create":
			n, err := retrogh.Create(a.run, g.primary().title, g.body(), g.labels())
			if err != nil {
				a.errf("cannot create %q: %v", g.primary().title, err)
				return ExitFail
			}
			g.number, g.done = n, true
			a.outf("created #%d: %s\n", n, g.primary().title)
		case "dup", "fixed":
			for _, m := range g.members {
				if err := retrogh.Comment(a.run, g.number, comment(m)); err != nil {
					a.errf("cannot comment on #%d: %v", g.number, err)
					return ExitFail
				}
				a.outf("commented on #%d: %s\n", g.number, m.label())
			}
			g.done = true
		}
		if g.number == 0 {
			continue
		}
		for _, m := range g.members {
			if err := a.point(m, g.number); err != nil {
				a.errf("%v", err)
				return ExitFail
			}
		}
	}
	return ExitOK
}

// point records which issue tracks a delta: written onto the row of a kit retro under --write-back, printed as a
// mapping for a retro outside the kit (which this tool never rewrites).
func (a *app) point(d delta, n int) error {
	if !d.inKit {
		a.outf("mapping: %s %s -> #%d\n", d.src, d.row.ID, n)
		return nil
	}
	if !a.o.writeBack {
		return nil
	}
	b, err := os.ReadFile(d.path)
	if err != nil {
		return fmt.Errorf("cannot read %s to write #%d back: %v", d.path, n, err)
	}
	out, err := retrorows.AppendIssue(string(b), d.row.Line, n)
	if err != nil {
		return fmt.Errorf("%s: %v", d.path, err)
	}
	if out != string(b) {
		if err := atomicfile.Write(d.path, []byte(out)); err != nil {
			return fmt.Errorf("cannot write %s: %v", d.path, err)
		}
	}
	a.outf("written back: %s -> #%d\n", d.label(), n)
	return nil
}
