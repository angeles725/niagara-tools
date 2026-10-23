package foldaudit

import (
	"fmt"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/retrocells"
)

// Exit codes of sweep-fold-audit (unchanged from the bash tool).
const (
	ExitOK     = 0 // clean, or WARN-only without --strict
	ExitStrict = 1 // uncited rows (or uncited Δ) under --strict
	ExitEnv    = 3 // usage or environment error
)

const usage = "usage: sweep-fold-audit.sh [--strict] [--deltas-since <YYYY-MM-DD>] <INDEX.md> <kit-root>"

// Main implements `sweep-fold-audit [--strict] [--deltas-since <YYYY-MM-DD>] <INDEX.md> <kit-root>`.
func Main(args []string, s cli.Stdio) int {
	o, usageErr := parseArgs(args)
	if usageErr != "" {
		fmt.Fprint(s.Err, usageErr)
		return ExitEnv
	}
	if st, err := os.Stat(o.index); err != nil || !st.Mode().IsRegular() {
		fmt.Fprintf(s.Err, "fold-audit: INDEX not found: %s\n", o.index)
		return ExitEnv
	}
	if st, err := os.Stat(o.root); err != nil || !st.IsDir() {
		fmt.Fprintf(s.Err, "fold-audit: kit-root not found: %s\n", o.root)
		return ExitEnv
	}
	h := harvestCorpus(o.root, o.since != "")
	tokens := h.sortedTokens()

	// An unreadable INDEX is an environment error: auditing it as empty would pass under --strict.
	data, err := os.ReadFile(o.index)
	if err != nil {
		fmt.Fprintf(s.Err, "fold-audit: cannot read INDEX: %v\n", err)
		return ExitEnv
	}
	t := audit{o: o, h: h, tokens: tokens, s: s}
	for _, line := range indexLines(data) {
		t.row(line)
	}
	t.summary()
	if t.warned && o.strict {
		return ExitStrict
	}
	return ExitOK
}

// indexLines splits the INDEX into lines; the final newline ends the last row, and an unterminated
// last row is still read.
func indexLines(data []byte) []string {
	lines := strings.Split(string(data), "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// audit is one sweep over the INDEX rows: the corpus citations and the running counts.
type audit struct {
	o      options
	h      *harvest
	tokens []string
	s      cli.Stdio

	nFolded, nCited, nUncited, dChecked, dMissing, dUnreadable int
	warned                                                     bool
}

// row audits one INDEX line: a folded dated row must be cited, and since --deltas-since each of its
// deltas too.
func (t *audit) row(line string) {
	line = strings.ReplaceAll(line, "\x00", "") // `read` drops NUL bytes
	if !strings.HasPrefix(line, "|") || !datedRow(line) {
		return
	}
	file := rowFile(line)
	if file == "" || rowStatus(line) != "folded" {
		return
	}
	t.nFolded++
	st := stem(file)
	hits := citedBy(st, t.tokens)

	if t.o.since != "" && !(prefix(file, 10) < t.o.since) {
		t.deltas(line, file, st)
	}

	if len(hits) == 0 {
		fmt.Fprintf(t.s.Out, "fold-audit: WARN %s folded with no [ev: retro …] citation\n", file)
		t.nUncited++
		t.warned = true
	} else {
		t.nCited++
		if len(hits) > 1 {
			fmt.Fprintf(t.s.Out, "fold-audit: NOTE ambiguous citation token credits %s: %s\n", file, strings.Join(hits, " "))
		}
	}
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
func (t *audit) summary() {
	fmt.Fprintf(t.s.Out, "fold-audit: %d folded, %d cited, %d uncited\n", t.nFolded, t.nCited, t.nUncited)
	if t.o.since != "" {
		fmt.Fprintf(t.s.Out, "fold-audit: deltas: %d checked, %d not cited (folded rows since %s)\n", t.dChecked, t.dMissing, t.o.since)
		// Mutation: R2-unreadable -- folding dUnreadable back into dMissing hides a corrupt cell as a citation gap.
		if t.dUnreadable > 0 {
			fmt.Fprintf(t.s.Out, "fold-audit: deltas: %d unreadable cells\n", t.dUnreadable)
		}
	}
}

func prefix(s string, n int) string {
	if len(s) < n {
		return s
	}
	return s[:n]
}
