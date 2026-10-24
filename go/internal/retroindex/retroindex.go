// Package retroindex derives the retro promotion registry (build-n4-module-kit/retros/INDEX.md)
// from the header comments at the top of each retro. It is the Go port of the original awk/bash
// gen-retro-index.sh and keeps its output byte for byte. [ev: retro generated-retro-index]
//
// Header contract (column-0 comments at the top of every retro):
//
//	<!-- review-status: pending|folded -->  within lines 1-5; the first word after the colon is the status
//	<!-- module: <Module|kit> -->           within lines 1-10, exactly one line
//	<!-- deltas: <N> -->                    within lines 1-10, N a non-negative integer
//	<!-- index-comment: <text> -->          optional, within lines 1-10; emitted as a trailing
//	                                        `<!-- <text> -->` on the row (e.g. a sweep --age disposition)
//
// The date is the first YYYY-MM-DD in the filename. Rows sort by date, then filename (bytewise).
package retroindex

import (
	"bufio"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/gen"
)

// Entry is one retro's index row and its sort keys.
type Entry struct {
	Date, Name, Row string
}

// Problem is a retro that yields no row: a header-contract violation, or (IO) an unreadable file.
type Problem struct {
	Name, Reason string
	IO           bool
}

var datePattern = regexp.MustCompile(`[0-9]{4}-[0-9]{2}-[0-9]{2}`)

// headerValue returns the value of a `<!-- key: value -->` line and whether the line is one.
func headerValue(line, key string) (string, bool) {
	prefix := "<!-- " + key + ":"
	if !strings.HasPrefix(line, prefix) || !strings.HasSuffix(line[len(prefix):], "-->") {
		return "", false
	}
	v := strings.TrimLeft(line[len(prefix):], " \t")
	return strings.TrimRight(strings.TrimSuffix(v, "-->"), " \t"), true
}

// statusWord returns the first word after `review-status:` when line is a status marker.
func statusWord(line string) (string, bool) {
	rest, ok := strings.CutPrefix(line, "<!--")
	if !ok {
		return "", false
	}
	if rest, ok = strings.CutPrefix(strings.TrimLeft(rest, " \t"), "review-status:"); !ok {
		return "", false
	}
	rest = strings.TrimLeft(rest, " \t")
	if i := strings.IndexAny(rest, " \t"); i >= 0 {
		rest = rest[:i]
	}
	return rest, true
}

func digits(s string) bool {
	return s != "" && strings.Trim(s, "0123456789") == ""
}

// ParseHeader reads the first 10 lines of the retro called name and returns its entry, or the
// contract violation that keeps it out of the index ("" reason = ok), or the read error that
// stopped it (a read error is never a short header: the caller must not render a partial index).
func ParseHeader(name string, r io.Reader) (Entry, string, error) {
	return parseHeader(name, r)
}

func parseHeader(name string, r io.Reader) (Entry, string, error) {
	var status, module, deltas, comment string
	var nmod, ndel int
	br := bufio.NewReader(r)
	for n := 1; n <= 10; n++ {
		line, err := br.ReadString('\n')
		if err != nil && err != io.EOF {
			return Entry{}, "", err
		}
		if line == "" && err == io.EOF {
			break
		}
		line = strings.TrimSuffix(line, "\n")
		if w, ok := statusWord(line); ok && n <= 5 && status == "" {
			status = w
		}
		if v, ok := headerValue(line, "module"); ok {
			nmod, module = nmod+1, v
		}
		if v, ok := headerValue(line, "deltas"); ok {
			ndel, deltas = ndel+1, v
		}
		if v, ok := headerValue(line, "index-comment"); ok {
			comment = v
		}
		if err == io.EOF {
			break
		}
	}
	date := datePattern.FindString(name)
	switch {
	case status != "pending" && status != "folded":
		return Entry{}, "no <!-- review-status: pending|folded --> marker in lines 1-5", nil
	case nmod != 1 || module == "" || strings.Contains(module, "|"):
		return Entry{}, "needs exactly one <!-- module: <Module|kit> --> header in lines 1-10", nil
	case ndel != 1 || !digits(deltas):
		return Entry{}, "needs exactly one <!-- deltas: <N> --> header (N an integer) in lines 1-10", nil
	case date == "":
		return Entry{}, "no YYYY-MM-DD date in the filename", nil
	}
	row := "| " + name + " | " + module + " | " + date + " | " + status + " | " + deltas + " |"
	if comment != "" {
		// Mutation: GRI-comment -- dropping the index-comment cell loses the disposition that sweep --age reads
		row += " <!-- " + comment + " -->"
	}
	return Entry{Date: date, Name: name, Row: row}, "", nil
}

// Scan parses every retro (*.md regular file, INDEX.md and dotfiles excluded) in dir, in
// bytewise name order. Problems come back in that order; entries are unsorted by date.
// It fails closed: an unlistable dir is an error, and a retro that cannot be stat'ed or read is
// an IO problem, so no partial index is rendered. Only an entry that vanished or a dangling
// symlink is skipped, as the `[ -f ]` test of the shell glob it replaces did.
func Scan(dir string) ([]Entry, []Problem, error) {
	files, err := os.ReadDir(dir) // sorted by filename
	if err != nil {
		return nil, nil, err
	}
	var entries []Entry
	var problems []Problem
	for _, de := range files {
		name := de.Name()
		if !strings.HasSuffix(name, ".md") || name == "INDEX.md" || name[0] == '.' {
			continue
		}
		path := filepath.Join(dir, name)
		st, err := os.Stat(path)
		if errors.Is(err, fs.ErrNotExist) || (err == nil && !st.Mode().IsRegular()) {
			continue
		}
		var e Entry
		var reason string
		if err == nil {
			e, reason, err = parseFile(name, path)
		}
		switch {
		case err != nil:
			problems = append(problems, Problem{Name: name, Reason: "cannot read the header: " + err.Error(), IO: true})
		case reason != "":
			problems = append(problems, Problem{Name: name, Reason: reason})
		default:
			entries = append(entries, e)
		}
	}
	return entries, problems, nil
}

func parseFile(name, path string) (Entry, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return Entry{}, "", err
	}
	defer f.Close()
	return parseHeader(name, f)
}

const header = gen.GeneratedLine + `
<!-- Regenerate: bash toolbelt/kit-state.sh --write · print: bash toolbelt/kit-state.sh retro-index · validate: bash toolbelt/gen-retro-index.sh --check -->
# Retro index — the promotion registry for the build-n4-module kit

One row per retro in this directory, GENERATED from the header comments at the top of each retro
(` + "`<!-- review-status: pending|folded -->`, `<!-- module: X -->`, `<!-- deltas: N -->`, optional\n" +
	"`<!-- index-comment: … -->`) and validated by `toolbelt/sweep-build-state.sh` + `tests/build-retro-sync.bats`.\n" +
	"Rows sort by date (the first YYYY-MM-DD in the filename), then filename. To add or flip a row, edit the\n" +
	"retro header; this file is not committed (.gitignore): toolbelt/kit-state.sh regenerates it (.githooks: on every pull).\n" +
	"`review-status`: `pending` = a PROPOSED delta not yet folded into the kit core; `folded` = promoted.\n" +
	"`deltas` = number of Δ<n>-numbered proposed deltas the retro enumerates (0 = not enumerated).\n" +
	`
| Retro file | Module | Date | review-status | deltas |
|---|---|---|---|---|`

// Render returns the index text without its final newline: the header, then one row per entry
// sorted by date, then filename.
func Render(entries []Entry) string {
	sorted := append([]Entry(nil), entries...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Date != sorted[j].Date {
			return sorted[i].Date < sorted[j].Date
		}
		return sorted[i].Name < sorted[j].Name
	})
	var b strings.Builder
	b.WriteString(header)
	for _, e := range sorted {
		b.WriteString("\n" + e.Row)
	}
	return b.String()
}
