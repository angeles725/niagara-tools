package ntplmanifest

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rowpath"
)

// linkRows are the WARN ntpl-link rows of the links a walk did not follow because their real target is
// outside the module root (lint.WalkFollow): one row per link, in path order, the same link met by the
// template walk and the build-file walk once.
func linkRows(walks ...[]string) []lint.Finding {
	seen := map[string]bool{}
	var links []string
	for _, w := range walks {
		for _, l := range w {
			if !seen[l] {
				seen[l] = true
				links = append(links, l)
			}
		}
	}
	sort.Strings(links)
	out := make([]lint.Finding, 0, len(links))
	for _, l := range links {
		target, err := filepath.EvalSymlinks(l)
		if err != nil {
			target = "?"
		}
		out = append(out, lint.Finding{Check: checkLink, Severity: lint.Warn, Path: rowpath.Display(l),
			Msg: fmt.Sprintf(linkOutside, rowpath.Display(target))})
	}
	return out
}
