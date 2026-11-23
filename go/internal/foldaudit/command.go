package foldaudit

import (
	"fmt"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/retrogh"
	"github.com/angeles725/niagara-tools/go/internal/retrorows"
)

// Exit codes of sweep-fold-audit (unchanged from the bash tool).
const (
	ExitOK     = 0 // clean, or WARN-only without --strict
	ExitStrict = 1 // uncited rows (or uncited Δ) under --strict
	ExitEnv    = 3 // usage or environment error
)

const usage = "usage: sweep-fold-audit.sh [--strict] [--exact-stems] [--deltas-since <YYYY-MM-DD>] [--external-root <dir>]... [--alias <stem>=<ALIAS>]... <INDEX.md> <kit-root>"

// Main implements `sweep-fold-audit [--strict] [--exact-stems] [--deltas-since <YYYY-MM-DD>] [--external-root <dir>]...
// [--alias <stem>=<ALIAS>]... <INDEX.md> <kit-root>` over the real gh.
func Main(args []string, s cli.Stdio) int { return Command(retrogh.Exec)(args, s) }

// Command is sweep-fold-audit with an injectable gh runner (used only for rows that carry an issue number and for
// retros outside the kit).
func Command(run retrogh.Runner) cli.Main {
	return func(args []string, s cli.Stdio) int { return audits(run, args, s) }
}

func audits(run retrogh.Runner, args []string, s cli.Stdio) int {
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
	if msg := o.addConfig(s.Env("HOME")); msg != "" {
		fmt.Fprint(s.Err, msg)
		return ExitEnv
	}
	h := harvestCorpus(o.root, o.since != "" || len(o.externals) > 0)
	tokens := h.sortedTokens()

	// An unreadable INDEX is an environment error: auditing it as empty would pass under --strict.
	data, err := os.ReadFile(o.index)
	if err != nil {
		fmt.Fprintf(s.Err, "fold-audit: cannot read INDEX: %v\n", err)
		return ExitEnv
	}
	t := audit{o: o, h: h, tokens: tokens, s: s, gh: &lookup{run: run}}
	lines := indexLines(data)
	for _, line := range lines {
		t.row(line)
	}
	t.aliases(indexStems(lines))
	t.summary()
	if msg := t.external(); msg != "" {
		fmt.Fprint(s.Err, msg)
		return ExitEnv
	}
	t.issueSummary()
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
	nPendingDone                                               int
	nAlias, nAmbiguous                                         int
	gh                                                         *lookup
	retroRows                                                  map[string][]retrorows.Row
	warned                                                     bool
}

// indexRow returns the retro filename of a dated INDEX row ("" for any other line).
func indexRow(line string) string {
	line = strings.ReplaceAll(line, "\x00", "") // `read` drops NUL bytes
	if !strings.HasPrefix(line, "|") || !datedRow(line) {
		return ""
	}
	return rowFile(line)
}

// indexStems is the stem of every dated INDEX row, whatever its status: the names a tag may use.
func indexStems(lines []string) map[string]bool {
	stems := map[string]bool{}
	for _, line := range lines {
		if file := indexRow(line); file != "" {
			stems[stem(file)] = true
		}
	}
	return stems
}

// row audits one INDEX line: a folded dated row must be cited, and since --deltas-since each of its
// deltas too.
func (t *audit) row(line string) {
	file := indexRow(line)
	line = strings.ReplaceAll(line, "\x00", "")
	if file == "" {
		return
	}
	if rowStatus(line) == "pending" {
		t.pending(line, file)
		return
	}
	if rowStatus(line) != "folded" {
		return
	}
	t.nFolded++
	st := stem(file)
	hits := citedBy(st, t.tokens, t.o.exactStems)

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

func (t *audit) summary() {
	fmt.Fprintf(t.s.Out, "fold-audit: %d folded, %d cited, %d uncited\n", t.nFolded, t.nCited, t.nUncited)
	if t.o.since != "" {
		fmt.Fprintf(t.s.Out, "fold-audit: deltas: %d checked, %d not cited (folded rows since %s)\n", t.dChecked, t.dMissing, t.o.since)
		// Mutation: R2-unreadable -- folding dUnreadable back into dMissing hides a corrupt cell as a citation gap.
		if t.dUnreadable > 0 {
			fmt.Fprintf(t.s.Out, "fold-audit: deltas: %d unreadable cells\n", t.dUnreadable)
		}
		if t.nPendingDone > 0 {
			fmt.Fprintf(t.s.Out, "fold-audit: pending: %d fully cited\n", t.nPendingDone)
		}
	}
	if t.nAlias > 0 {
		fmt.Fprintf(t.s.Out, "fold-audit: aliases: %d tokens are not an exact stem (%d ambiguous)\n", t.nAlias, t.nAmbiguous)
	}
}

func prefix(s string, n int) string {
	if len(s) < n {
		return s
	}
	return s[:n]
}
