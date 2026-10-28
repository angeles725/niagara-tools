// Package retrodeltas is count-retro-deltas (G39): a retro's delta totals counted by script, never summed by hand
// (METHODOLOGY K34). For each retro it counts the `| Δ<n> |` / `| <n> |` rows of its delta section ("## Proposed kit
// deltas", "## Deltas" or "## Proposed delta table")
// and compares them with its `**Delta count**` line, its `<!-- deltas: N -->` header and its row of the generated
// retros/INDEX.md beside it. The contract is the deferred G39 spec
// (build-n4-module-kit/ledger.d/2026-10-23-r8-p1-process.md, C51); toolbelt/count-retro-deltas.sh is the wrapper.
//
// Output: one line per retro, `<file>  rows=<n>  count=<n>  header=<n>  index=<n>  OK|MISMATCH` (`-` for a value
// the retro or the index does not state, which is a mismatch), then `TOTAL  rows=<sum>  files=<n>  mismatches=<n>`.
// Exit 0 every retro agrees, 1 any mismatch, 3 usage or an unreadable retro or INDEX.md.
package retrodeltas

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const usage = "usage: count-retro-deltas.sh <retro.md>...\n"

var (
	deltaRow = regexp.MustCompile(`^\| *Δ?[0-9]+ *\|`)
	countRe  = regexp.MustCompile(`^\*\*Delta count\*\*: *([0-9]+)`)
	headerRe = regexp.MustCompile(`^<!-- deltas: *([0-9]+) *-->`)
	// the delta table's section: "## Proposed kit deltas ...", "## Deltas" or "## Proposed delta table"
	deltaSection = regexp.MustCompile(`^## (Proposed kit deltas|Deltas|Proposed delta table)\b`)
)

// Main runs count-retro-deltas.
func Main(args []string, s cli.Stdio) int {
	if len(args) == 0 {
		fmt.Fprint(s.Err, usage)
		return 3
	}
	indexes := map[string]map[string]string{} // retros dir -> retro file name -> INDEX deltas cell
	total, mismatches := 0, 0
	for _, p := range args {
		if strings.HasPrefix(p, "-") {
			fmt.Fprint(s.Err, usage)
			return 3
		}
		b, err := os.ReadFile(p)
		if err != nil {
			fmt.Fprintf(s.Err, "count-retro-deltas: cannot read %s: %v\n", p, err)
			return 3
		}
		dir := filepath.Dir(p)
		idx, ok := indexes[dir]
		if !ok {
			if idx, err = readIndex(filepath.Join(dir, "INDEX.md")); err != nil {
				fmt.Fprintf(s.Err, "count-retro-deltas: no INDEX.md beside %s (%v): generate it with toolbelt/kit-state.sh --write\n", p, err)
				return 3
			}
			indexes[dir] = idx
		}
		rows, count, header := scan(string(b))
		index := idx[filepath.Base(p)]
		if index == "" {
			index = "-"
		}
		verdict := "OK"
		// Mutation: CRD-count -- ignoring the Delta count line passes a retro that declares 3 and lists 2
		// Mutation: CRD-index -- skipping the INDEX row passes a generated index that says 1
		// Mutation: CRD-header -- dropping the header from the comparison passes a retro whose header alone says 3
		if n := strconv.Itoa(rows); count != n || header != n || index != n {
			verdict = "MISMATCH"
			mismatches++
		}
		// Mutation: CRD-total -- keeping only the last retro's rows prints TOTAL rows=1
		total += rows
		fmt.Fprintf(s.Out, "%s  rows=%d  count=%s  header=%s  index=%s  %s\n", p, rows, count, header, index, verdict)
	}
	fmt.Fprintf(s.Out, "TOTAL  rows=%d  files=%d  mismatches=%d\n", total, len(args), mismatches)
	if mismatches > 0 {
		return 1
	}
	return 0
}

// scan counts the delta rows of the "Proposed kit deltas" section and reads the stated counts ("-" when absent).
func scan(text string) (rows int, count, header string) {
	count, header = "-", "-"
	in := false
	for _, l := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		switch {
		case strings.HasPrefix(l, "## "):
			// Mutation: CRD-heading -- reading only "## Proposed kit deltas" counts 0 rows in a retro headed "## Deltas"
			in = deltaSection.MatchString(l)
		case in && deltaRow.MatchString(l):
			rows++
		}
		if m := countRe.FindStringSubmatch(l); m != nil && count == "-" {
			count = m[1]
		}
		if m := headerRe.FindStringSubmatch(l); m != nil && header == "-" {
			header = m[1]
		}
	}
	return rows, count, header
}

// readIndex reads the Deltas cell of every `| <retro>.md | ... |` row of a retros/INDEX.md: the column its table header
// names "Deltas" (a later column may hold a note), else the fifth.
func readIndex(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out, col := map[string]string{}, 4
	for _, l := range strings.Split(string(b), "\n") {
		c := strings.Split(strings.Trim(strings.TrimSpace(l), "|"), "|")
		for i := range c {
			c[i] = strings.TrimSpace(c[i])
		}
		for i, h := range c {
			if h == "Deltas" {
				col = i
			}
		}
		if len(c) > col && strings.HasSuffix(c[0], ".md") {
			out[c[0]] = c[col]
		}
	}
	return out, nil
}
