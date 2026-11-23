package foldaudit

import (
	"fmt"
	"sort"
	"strings"
)

// aliases reports every corpus tag whose token is not an exact stem but is a segment-aligned part
// of one (#419, CONTRIBUTING.md:261): an advisory WARN row by default, a FAIL row that fails
// --strict under --exact-stems. A token that matches no stem is left alone, as before.
func (t *audit) aliases(stems map[string]bool) {
	sorted := make([]string, 0, len(stems))
	for s := range stems {
		sorted = append(sorted, s)
	}
	sort.Strings(sorted)
	kind, memo := "WARN", map[string][]string{}
	if t.o.exactStems {
		kind = "FAIL"
	}
	for _, c := range t.h.cites {
		cands, ok := memo[c.token]
		if !ok {
			cands = aliasOf(c.token, stems, sorted)
			memo[c.token] = cands
		}
		if len(cands) == 0 {
			continue
		}
		t.nAlias++
		where := fmt.Sprintf("fold-audit: %s %s:%d retro alias %q", kind, c.path, c.line, c.token)
		if len(cands) == 1 {
			fmt.Fprintf(t.s.Out, "%s — use the exact stem %q (CONTRIBUTING.md:261)\n", where, cands[0])
		} else {
			t.nAmbiguous++
			quoted := make([]string, len(cands))
			for i, s := range cands {
				quoted[i] = fmt.Sprintf("%q", s)
			}
			fmt.Fprintf(t.s.Out, "%s is ambiguous — use one exact stem of %s (CONTRIBUTING.md:261)\n", where, strings.Join(quoted, " "))
		}
		// Mutation: F21 -- treating a FAIL alias row as advisory lets --exact-stems --strict exit 0.
		// Mutation: F20 -- setting warned for the default WARN rows turns the CI run (--strict) red.
		if t.o.exactStems {
			t.warned = true
		}
	}
}
