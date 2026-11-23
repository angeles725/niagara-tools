// Package retrorows reads the delta rows of a retro: the table rows under a "delta" heading, whatever column layout
// the retro uses (kit retros: `| Δ | Delta | Target file / § | Token |`; research retros: `| # | Proposed change |
// Target | Evidence | Type | Priority |`). It also classifies a row as mcp-n4 or build-n4-module work and writes a
// GitHub issue number back onto a kit retro's row. Shared by retro-issues and the fold audit.
package retrorows

import (
	"regexp"
	"strconv"
	"strings"
)

// Row is one delta row of a retro.
type Row struct {
	Line     int    // 1-based line of the row in the retro
	ID       string // normalised: "Δ" + the id cell without its own Δ ("Δ1", "ΔB2")
	Text     string // the delta cell, without a trailing "#N"
	Target   string // the target column ("" when the table has none)
	Evidence string // the evidence (or token) column
	Heading  string // the heading of the section the row sits in
	Issue    int    // the trailing "#N" of the delta cell: the GitHub issue that tracks the row (0 = none)
	Done     string // APPLIED, DISMISSED, SHIPPED, REJECTED, FOLDED or PARKED when the delta text starts with it
}

var (
	idRe       = regexp.MustCompile(`^Δ?([A-Za-z]{0,2}[0-9]+)$`)
	headingRe  = regexp.MustCompile(`^#{2,6} +(.*)$`)
	deltaHead  = regexp.MustCompile(`(?i)delta`)
	notDelta   = regexp.MustCompile(`(?i)\b(not (a )?(kit )?delta|no delta|already|reinforced|rationale|verdict|summary|run quality|state, not)`)
	sepCell    = regexp.MustCompile(`^:?-+:?$`)
	trailIssue = regexp.MustCompile(`\s+#([0-9]+)\s*$`)
	doneRe     = regexp.MustCompile(`^\**(APPLIED|DISMISSED|SHIPPED|REJECTED|FOLDED|PARKED)\b`)
)

const pipe = "\x00"

// splitCells splits a table row on unescaped pipes, trimmed, without the outer pipes.
func splitCells(line string) []string {
	line = strings.TrimSpace(strings.TrimRight(line, "\r"))
	line = strings.ReplaceAll(line, `\|`, pipe)
	line = strings.TrimSuffix(strings.TrimPrefix(line, "|"), "|")
	cells := strings.Split(line, "|")
	for i, c := range cells {
		cells[i] = strings.TrimSpace(strings.ReplaceAll(c, pipe, "|"))
	}
	return cells
}

// Parse returns the delta rows of a retro, in file order. Only tables under a heading that names deltas count (not
// "Not a kit delta", "Already covered", "Summary"…).
func Parse(data string) []Row {
	var rows []Row
	heading, in := "", false
	target, evidence := -1, -1
	started := false
	for i, line := range strings.Split(data, "\n") {
		if m := headingRe.FindStringSubmatch(strings.TrimRight(line, "\r")); m != nil {
			heading = strings.TrimSpace(m[1])
			in = deltaHead.MatchString(heading) && !notDelta.MatchString(heading)
			started = false
			continue
		}
		if !in {
			continue
		}
		if !strings.HasPrefix(strings.TrimSpace(line), "|") {
			started = false
			continue
		}
		cells := splitCells(line)
		if !started {
			started, target, evidence = true, -1, -1
			for j, h := range cells {
				h = strings.ToLower(h)
				switch {
				case strings.Contains(h, "target"):
					target = j
				case strings.Contains(h, "evidence") || strings.Contains(h, "token"):
					evidence = j
				}
			}
			continue
		}
		m := idRe.FindStringSubmatch(cells[0])
		if m == nil || len(cells) < 2 {
			continue
		}
		r := Row{Line: i + 1, ID: "Δ" + m[1], Text: cells[1], Heading: heading}
		if mi := trailIssue.FindStringSubmatchIndex(r.Text); mi != nil {
			r.Issue, _ = strconv.Atoi(r.Text[mi[2]:mi[3]])
			r.Text = strings.TrimSpace(r.Text[:mi[0]])
		}
		if d := doneRe.FindStringSubmatch(r.Text); d != nil {
			r.Done = d[1]
		}
		r.Target, r.Evidence = cell(cells, target), cell(cells, evidence)
		rows = append(rows, r)
	}
	return rows
}

func cell(cells []string, i int) string {
	if i < 0 || i >= len(cells) {
		return ""
	}
	return cells[i]
}

// Stem is the retro file name without its directory, its leading YYYY-MM-DD- and its .md.
func Stem(path string) string {
	name := path[strings.LastIndex(path, "/")+1:]
	name = strings.TrimSuffix(name, ".md")
	if len(name) > 11 && name[4] == '-' && name[7] == '-' && name[10] == '-' {
		return name[11:]
	}
	return name
}
