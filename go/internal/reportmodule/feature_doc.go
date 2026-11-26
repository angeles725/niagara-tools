package reportmodule

import (
	"os"
	"regexp"
	"strings"
)

// featureDocCheck is the check column of the rows --feature-doc adds: ADVISORY rows for the steps of the
// build-n4-module skill that no gate read before (skill/SKILL.md § Steps no tool checks). They are counted as
// ADVISORY and never change the exit code or the verdict.
const featureDocCheck = "feature-doc"

// featureDoc reads the document named by --feature-doc and emits one ADVISORY row per item it lacks (featureDocGaps),
// or one row when the file cannot be read. Without the option nothing is read and nothing printed.
func (r *report) featureDoc() {
	path := r.o.featureDoc
	if path == "" {
		return
	}
	b, err := os.ReadFile(path)
	if err != nil {
		r.emit("(module)", advisory, featureDocCheck,
			"feature doc not readable: "+path+" ("+errText(err)+"); tier, Behavior decisions and Checks skipped not checked")
		return
	}
	for _, gap := range featureDocGaps(string(b)) {
		r.emit("(module)", advisory, featureDocCheck, path+": "+gap)
	}
}

var (
	// headingRe is a level-2 heading: "## " then the title (a "###" heading is not one).
	headingRe = regexp.MustCompile(`^##[ \t]+(\S.*?)[ \t]*$`)
	// tierRe is a tier line (BUILD-LOOP.md §0.c): the word "tier" then P0-P3 (the template's `<P0|P1|P2|P3>`
	// placeholder is not one), or a line that starts with the tier.
	tierRe = regexp.MustCompile("(?i)\\btier\\b[\\s:*_`=-]*P[0-3]\\b|^[\\s*_`]*P[0-3]\\b")
	// noneRe is the answer "none" (optionally followed by a reason).
	noneRe = regexp.MustCompile(`(?i)^\W*none\b`)
	// sepRe is a table separator cell.
	sepRe = regexp.MustCompile(`^:?-+:?$`)
)

// featureDocGaps are the items of the feature document text that the skill asks for and that are not there, in
// order: a tier line before the first `## ` heading; a `## Behavior decisions` heading with content (an answered
// table row, a row with any cell besides its id filled, or "none"); a `## Checks skipped` heading (its text may be
// "none", the heading is the item).
func featureDocGaps(text string) []string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	var gaps []string
	if !hasTier(lines) {
		gaps = append(gaps, "no tier line (tier P0-P3 before the first `## ` heading, BUILD-LOOP.md §0.c)")
	}
	if body, found := section(lines, "behavior decisions"); !found {
		gaps = append(gaps, "no `## Behavior decisions` heading (BUILD-LOOP.md §1)")
	} else if !hasDecision(body) {
		gaps = append(gaps, "`## Behavior decisions` has no row and no \"none\" (BUILD-LOOP.md §1)")
	}
	if _, found := section(lines, "checks skipped"); !found {
		gaps = append(gaps, "no `## Checks skipped` heading (BUILD-LOOP.md §5; the text may be \"none\")")
	}
	return gaps
}

// hasTier reports whether a line before the first level-2 heading is a tier line.
func hasTier(lines []string) bool {
	for _, ln := range lines {
		if headingRe.MatchString(ln) {
			return false
		}
		if tierRe.MatchString(ln) {
			return true
		}
	}
	return false
}

// section is the lines under the first level-2 heading titled title (case-insensitive) up to the next one.
func section(lines []string, title string) (body []string, found bool) {
	for _, ln := range lines {
		m := headingRe.FindStringSubmatch(ln)
		switch {
		case m != nil && found:
			return body, true
		case m != nil:
			found = strings.EqualFold(m[1], title)
		case found:
			body = append(body, ln)
		}
	}
	return body, found
}

// hasDecision reports whether the Behavior decisions body holds a "none" line or a table data row (not the header
// row, nor the separator) with a filled cell besides the first.
func hasDecision(body []string) bool {
	for i, ln := range body {
		if noneRe.MatchString(strings.TrimSpace(ln)) {
			return true
		}
		cells := tableCells(ln)
		if cells == nil || (i+1 < len(body) && isSeparator(tableCells(body[i+1]))) || isSeparator(cells) {
			continue
		}
		for _, c := range cells[1:] {
			if c != "" {
				return true
			}
		}
	}
	return false
}

// tableCells are the trimmed cells of a markdown table row; nil for a line that is not one.
func tableCells(ln string) []string {
	ln = strings.TrimSpace(ln)
	if !strings.HasPrefix(ln, "|") {
		return nil
	}
	cells := strings.Split(strings.TrimSuffix(ln[1:], "|"), "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

// isSeparator reports whether cells is a table's `|---|---|` row.
func isSeparator(cells []string) bool {
	if cells == nil {
		return false
	}
	for _, c := range cells {
		if !sepRe.MatchString(c) {
			return false
		}
	}
	return true
}
