package retroissues

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/clientdenylist"
	"github.com/angeles725/niagara-tools/go/internal/retrodup"
	"github.com/angeles725/niagara-tools/go/internal/retrogh"
)

var markerLine = regexp.MustCompile(`(?m)^<!-- retro-delta: .* -->\s*$`)

// scrub names the field of text that matches the denylist, never the matched text: "<field> matches denylist rule #N".
func scrub(rules clientdenylist.Rules, field, text string) string {
	if hits := rules.Match(text); len(hits) > 0 {
		return fmt.Sprintf("%s matches denylist rule #%d", field, hits[0])
	}
	return ""
}

// refusal scrubs everything a group would publish: its title and body, and the comment each member would become on an
// existing issue. Nothing is searched, created or commented before this passes.
func refusal(rules clientdenylist.Rules, g *group) string {
	if r := scrub(rules, "title", g.primary().title); r != "" {
		return r
	}
	if r := scrub(rules, "body", g.body()); r != "" {
		return r
	}
	for _, m := range g.members {
		if r := scrub(rules, "comment", comment(m)); r != "" {
			return r
		}
	}
	return ""
}

// forceNew reports whether --new names the delta.
func (a *app) forceNew(d delta) bool { return a.o.renew[d.row.ID] || a.o.renew[d.src+":"+d.row.ID] }

// decide settles every delta against the issues that already exist: an exact marker (body or comment of a
// retro-delta issue) means it exists; otherwise the deltas that describe one change become one group, and a group
// whose key terms match an existing issue (open or closed) at or above the threshold is a probable duplicate (open) or
// already fixed (closed), unless --new names it. notes carries what could not be checked.
func (a *app) decide(ds []delta) (gs []*group, notes []string, err error) {
	var rest []delta
	for _, d := range ds {
		if is, ok := a.markers[d.key()]; ok {
			gs = append(gs, &group{members: []delta{d}, kind: "exists", issue: is, number: is.Number})
		} else {
			rest = append(rest, d)
		}
	}
	skipSearch := a.noRules && len(rest) > 0
	if skipSearch { // terms that cannot be scrubbed never leave the machine
		notes = append(notes, "dedup search skipped: no denylist on this machine, so no search term is sent to GitHub")
	}
	for _, g := range groupDeltas(rest, a.o) {
		gs = append(gs, g)
		if r := refusal(a.rules, g); r != "" {
			g.kind, g.reason = "refused", r
			continue
		}
		p := g.primary()
		if a.remoteErr != nil || a.forceNew(p) {
			continue
		}
		if skipSearch {
			a.match(g, a.issues)
			continue
		}
		query := retrodup.Terms(p.tok, 3)
		if r := scrub(a.rules, "search query", query); r != "" {
			g.kind, g.reason = "refused", r
			continue
		}
		found, serr := retrogh.Search(a.run, query+" in:title,body")
		if serr != nil {
			if a.o.apply {
				return nil, nil, serr
			}
			notes = append(notes, "dedup unchecked: "+serr.Error())
		}
		a.match(g, append(append([]retrogh.Issue{}, a.issues...), found...))
	}
	return gs, notes, nil
}

// match sets g to dup or fixed when an existing issue describes the same change.
func (a *app) match(g *group, candidates []retrogh.Issue) {
	p := g.primary()
	best, bestScore := retrogh.Issue{}, 0.0
	for _, c := range candidates {
		body := markerLine.ReplaceAllString(c.Body, "")
		if len(body) > 800 {
			body = body[:800]
		}
		if s := retrodup.Score(p.tok, retrodup.Tokens(c.Title, body)); s > bestScore {
			best, bestScore = c, s
		}
		// An issue that holds a whole retro as a table: each row is one change, scored with the title.
		for _, row := range tableRows(c.Body) {
			if s := retrodup.Score(p.tok, retrodup.Tokens(c.Title, row)); s > bestScore {
				best, bestScore = c, s
			}
		}
	}
	if bestScore < a.o.threshold {
		return
	}
	g.kind, g.issue, g.score, g.number = "dup", best, bestScore, best.Number
	if best.Closed() {
		g.kind = "fixed"
	}
}

// tableRows lists the data rows of the markdown tables in body (lines that start with "|", minus the separator).
func tableRows(body string) []string {
	var rows []string
	for _, l := range strings.Split(body, "\n") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "|") && strings.Trim(l, "|-: ") != "" {
			rows = append(rows, l)
		}
	}
	return rows
}

// summary is the closing line.
func summary(gs []*group, sk skipped, apply bool) string {
	n := map[string]int{}
	for _, g := range gs {
		if apply && g.kind == "create" && g.number == 0 {
			continue // planned, but gh stopped before creating it
		}
		n[g.kind]++
	}
	verb := "to create"
	if apply {
		verb = "created"
	}
	parts := []string{
		fmt.Sprintf("%d %s", n["create"], verb), fmt.Sprintf("%d exist", n["exists"]),
		fmt.Sprintf("%d probable duplicates", n["dup"]), fmt.Sprintf("%d already fixed", n["fixed"]),
		fmt.Sprintf("%d refused", n["refused"]), fmt.Sprintf("%d tracked", len(sk.tracked)),
		fmt.Sprintf("%d skipped (applied)", sk.applied), fmt.Sprintf("%d skipped (no area)", sk.noArea),
	}
	return "retro-issues: " + strings.Join(parts, ", ")
}
