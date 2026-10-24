package rcscan

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

var (
	pageSection = re(`<section[^>]*[[:space:]]id=["']page-[A-Za-z0-9_-]+["']`)
	sectionID   = re(`^.*id=["']page-`)
	navEntry    = re(`data-page=["'][A-Za-z0-9_-]+["']`)
)

// orphanPages is the cross-file orphan-page pass over every scanned .html and .js file, in walk order: a
// <section id="page-<id>"> (in .html) with no data-page="<id>" nav entry anywhere, or a nav entry with no
// such section. A tree with no page section has nothing to check. Each id is reported at its first site.
func orphanPages(pages []*source, sev lint.Severity) []lint.Finding {
	type site struct {
		rel  string
		line int
	}
	sec, nav := map[string]site{}, map[string]site{}
	type ref struct {
		section bool
		id      string
	}
	var order []ref
	for _, src := range pages {
		for i, line := range src.lines {
			if commentOnly.MatchString(line) || strings.Contains(line, "rc-scan: allow orphan-page") {
				continue
			}
			at := site{src.rel, i + 1}
			if src.ext == "html" {
				for _, m := range pageSection.FindAllString(line, -1) {
					id := strings.TrimRight(sectionID.ReplaceAllString(m, ""), `"'`)
					if _, ok := sec[id]; !ok {
						sec[id] = at
						order = append(order, ref{true, id})
					}
				}
			}
			for _, m := range navEntry.FindAllString(line, -1) {
				id := m[11 : len(m)-1]
				if _, ok := nav[id]; !ok {
					nav[id] = at
					order = append(order, ref{false, id})
				}
			}
		}
	}
	if len(sec) == 0 {
		return nil
	}
	var out []lint.Finding
	add := func(at site, msg string) {
		out = append(out, lint.Finding{Severity: sev, Path: at.rel, Line: at.line, Msg: "orphan-page: " + msg})
	}
	for _, r := range order {
		if _, ok := nav[r.id]; r.section && !ok {
			add(sec[r.id], "section#page-"+r.id+" has no nav entry data-page=\""+r.id+"\"")
		}
		if _, ok := sec[r.id]; !r.section && !ok {
			add(nav[r.id], "nav entry data-page=\""+r.id+"\" has no section#page-"+r.id)
		}
	}
	return out
}
