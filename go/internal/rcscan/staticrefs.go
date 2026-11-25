package rcscan

import (
	"regexp"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/jssrc"
)

// Ref is one URL an rc file names: its 1-based line and the text as written.
type Ref struct {
	Line int
	URL  string
	col  int // the byte column, to order the references of one line
}

var (
	// attrRef is a src= or href= attribute with a quoted value (the value is group 1 or 2).
	attrRef = regexp.MustCompile(`(?i)(?:^|[\s"'<])(?:src|href)\s*=\s*(?:"([^"]*)"|'([^']*)')`)
	// urlRef is a CSS url(...) (group 1 double-quoted, 2 single-quoted, 3 bare) or an @import "..." (4, 5).
	urlRef = regexp.MustCompile(`(?i)url\(\s*(?:"([^"]*)"|'([^']*)'|([^)\s"']*))\s*\)|@import\s+(?:"([^"]*)"|'([^']*)')`)
)

// fileRefs is every reference of an .html, .css or .js file read as its markup, style and script: attributes
// and CSS url()/@import in markup and stylesheets (comments blanked), and the string arguments of fetch(,
// new URL(, import in JavaScript (a .js file or the <script> elements of an .html file), in report order.
func fileRefs(ext string, lines []string) []Ref {
	var refs []Ref
	if ext != "js" {
		markup := lines
		if ext == "html" {
			scripts := jssrc.HTMLScripts(lines)
			markup = make([]string, len(lines))
			for i, l := range lines {
				markup[i] = outside(l, scripts[i])
			}
		}
		for i, l := range blankComments(markup, ext == "html") {
			refs = append(refs, textRefs(l, i+1)...)
		}
	}
	if ext != "css" {
		js := lines
		if ext == "html" {
			js = jssrc.HTMLScripts(lines)
		}
		refs = append(refs, jsRefs(jssrc.Tokens(js))...)
	}
	sort.SliceStable(refs, func(a, b int) bool {
		if refs[a].Line != refs[b].Line {
			return refs[a].Line < refs[b].Line
		}
		return refs[a].col < refs[b].col
	})
	return refs
}

// outside is line with every byte that script (the line with all but the <script> contents blanked) keeps
// replaced by a blank: the markup and style of the page.
func outside(line, script string) string {
	b := []byte(line)
	for k := range b {
		if k < len(script) && script[k] == b[k] && b[k] != ' ' {
			b[k] = ' '
		}
	}
	return string(b)
}

// blankComments blanks /* ... */ comments and, for html, <!-- ... --> ones across lines (line breaks kept).
func blankComments(lines []string, html bool) []string {
	out := make([]string, len(lines))
	closer := ""
	for i, l := range lines {
		b := []byte(l)
		for k := 0; k < len(b); {
			if closer != "" {
				if strings.HasPrefix(l[k:], closer) {
					for j := 0; j < len(closer); j++ {
						b[k+j] = ' '
					}
					k += len(closer)
					closer = ""
				} else {
					b[k] = ' '
					k++
				}
				continue
			}
			switch {
			case strings.HasPrefix(l[k:], "/*"):
				closer = "*/"
			case html && strings.HasPrefix(l[k:], "<!--"):
				closer = "-->"
			default:
				k++
				continue
			}
			b[k], b[k+1] = ' ', ' '
			k += 2
		}
		out[i] = string(b)
	}
	return out
}

// textRefs are the attribute and url()/@import references of one line of markup or CSS.
func textRefs(text string, line int) []Ref {
	var refs []Ref
	for _, m := range attrRef.FindAllStringSubmatchIndex(text, -1) {
		if v, ok := group(text, m, 1, 2); ok {
			refs = append(refs, Ref{Line: line, URL: v, col: m[0]})
		}
	}
	for _, m := range urlRef.FindAllStringSubmatchIndex(text, -1) {
		if v, ok := group(text, m, 1, 2, 3, 4, 5); ok {
			refs = append(refs, Ref{Line: line, URL: v, col: m[0]})
		}
	}
	return refs
}

// group is the text of the first of the numbered groups of match m that took part in it.
func group(text string, m []int, groups ...int) (string, bool) {
	for _, g := range groups {
		if m[2*g] >= 0 {
			return text[m[2*g]:m[2*g+1]], true
		}
	}
	return "", false
}
