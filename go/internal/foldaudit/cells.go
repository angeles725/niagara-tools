package foldaudit

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/retrocells"
)

// pending reports a pending row since --deltas-since whose every delta is cited (or deferred) as landed: the fold
// is done and only the flip to folded is missing, which fails --strict (retro build-kit-bench-session Δ19). A
// pending row with a delta not yet cited, a row with no delta and a row before the cutoff are left alone.
func (t *audit) pending(line, file string) {
	if t.o.since == "" || prefix(file, 10) < t.o.since {
		return
	}
	deltas, err := retrocells.ParseDeltasCell(cellAfter(line, "pending"))
	if err != nil || len(deltas) == 0 {
		return
	}
	st := stem(file)
	for _, d := range deltas {
		if !t.h.deltas[st+" "+d.String()] {
			return
		}
	}
	fmt.Fprintf(t.s.Out, "fold-audit: WARN %s pending, but all %d deltas are cited: set `<!-- review-status: folded -->` in the retro\n", file, len(deltas))
	t.nPendingDone++
	t.warned = true
}

// deltas checks that every Δk of a folded row is cited as "<stem> Δk".
func (t *audit) deltas(line, file, st string) {
	cell := deltasCell(line)
	// Decimal (bash arithmetic read a leading zero as octal); an unreadable or oversize cell is
	// counted on its own, never as a "not cited" Δ, and still fails --strict (fail closed).
	deltas, perr := retrocells.ParseDeltasCell(cell)
	if perr != nil {
		fmt.Fprintf(t.s.Out, "fold-audit: WARN %s deltas cell unreadable (%s)\n", file, cell)
		t.dUnreadable++
		t.warned = true
	}
	for _, d := range deltas {
		t.dChecked++
		// Mutation: F9 -- substring matching credits another stem's Δk, or Δ10 for Δ1.
		if !t.h.deltas[st+" "+d.String()] {
			fmt.Fprintf(t.s.Out, "fold-audit: WARN %s %s not cited\n", file, d)
			t.dMissing++
			t.warned = true
		}
	}
}

// summary prints the folded/cited totals and, since --deltas-since, the deltas totals.

// deltasCell is the cell right after the first "folded" cell, with every space removed (tabs kept);
// "" when there is no such cell.
func deltasCell(line string) string { return cellAfter(line, "folded") }

// cellAfter is the cell right after the first cell that reads status, with every space removed (tabs kept);
// "" when there is no such cell.
func cellAfter(line, status string) string {
	cs := cells(line)
	for i, c := range cs {
		if trimSpaces(c) == status {
			if i+1 < len(cs) {
				return strings.ReplaceAll(cs[i+1], " ", "")
			}
			return ""
		}
	}
	return ""
}

// stem is the filename without its leading YYYY-MM-DD- date and trailing .md.
