package citeresolve

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/citeparse"
)

// anchorWindow is how many lines before the first and after the last line of a cited range an anchor
// may sit on: a re-render that shifts the text by a line or two still holds the citation.
const anchorWindow = 2

// maxAnchorLines is how many other lines holding the anchor a DRIFT detail lists.
const maxAnchorLines = 4

// guidePage reports whether path is a rendered guide page (a guides-clean/ copy of niagara-help), the one
// tree whose lines shift on a re-render and so the one tree an anchor is checked in. Elsewhere a quoted
// phrase after a pointer is often a paraphrase of the claim, not the cited text.
func guidePage(path string) bool { return strings.Contains(filepath.ToSlash(path), "/guides-clean/") }

// checkAnchor verifies the quoted anchor of p is on a line of a cited range, give or take anchorWindow
// lines; otherwise DRIFT, naming the lines that hold it now, or saying no line does. A pointer without an
// anchor, or outside a guide page, passes. Whitespace (and zero-width spaces) are not significant.
func checkAnchor(p citeparse.Pointer, path string, lines []string) (Result, bool) {
	if p.Anchor == "" || !guidePage(path) {
		return Result{}, true
	}
	want := squash(p.Anchor)
	var at []string
	for i, l := range lines {
		if !strings.Contains(squash(l), want) {
			continue
		}
		if near(p.Lines, i+1) {
			return Result{}, true
		}
		at = append(at, strconv.Itoa(i+1))
	}
	quoted := fmt.Sprintf("anchor %q", p.Anchor)
	if len(at) == 0 {
		return Result{Status: Drift, File: path, Detail: quoted + " not in " + filepath.Base(path)}, false
	}
	where := "line " + at[0]
	if len(at) > 1 {
		if len(at) > maxAnchorLines {
			at = append(at[:maxAnchorLines], fmt.Sprintf("+%d more", len(at)-maxAnchorLines))
		}
		where = "lines " + strings.Join(at, ", ")
	}
	return Result{Status: Drift, File: path, Detail: fmt.Sprintf("%s not within %d lines of :%s; at %s",
		quoted, anchorWindow, citeparse.FormatRanges(p.Lines), where)}, false
}

// near reports whether line n lies in one of rs widened by anchorWindow on each side.
func near(rs []citeparse.Range, n int) bool {
	for _, r := range rs {
		if n >= r.From-anchorWindow && n <= r.To+anchorWindow {
			return true
		}
	}
	return false
}

// squash collapses runs of whitespace to one space and drops zero-width spaces.
func squash(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(s, "​", "")), " ")
}
