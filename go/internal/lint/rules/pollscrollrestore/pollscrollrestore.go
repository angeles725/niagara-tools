// Package pollscrollrestore is lint-poll-scroll-restore (#909): a poll repaint that clears a container and a
// file with no scroll restore anywhere. A page that rebuilds a scrolling view on every poll puts the operator
// back at the top unless the repaint saves and restores the scroll position (types/frontend-standard.md §5).
// toolbelt/lint-poll-scroll-restore.sh is the wrapper and its header is the user-facing contract.
//
// One lexical, single-file rule, so it is advisory and biased to the false negative: it WARNs once per
// *.html/*.js file, at the first clear, when all hold: a function named as the callback of a setInterval/
// setTimeout poll (or a function that poll calls, one level) clears a container (innerHTML/textContent/
// innerText = "", replaceChildren(), jQuery .empty() or .html("")); the file shows a scrolling container
// (CSS overflow: auto|scroll, or style.overflow(Y) set to one); and the file reads or sets no scroll position
// at all (scrollTop, scrollLeft, scrollTo, scrollBy, scrollIntoView, scroll). A page whose scroller is styled
// in a separate stylesheet shows no container here and is not judged.
package pollscrollrestore

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/jssrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-poll-scroll-restore"

// Spec registers the rule: [--strict] [--] <ux-src-root>, WARN rows (exit 1 under --strict).
func Spec() lint.Spec {
	return lint.Spec{
		Rule:          rule{},
		Mode:          lint.First,
		Usage:         "usage: lint-poll-scroll-restore.sh [--strict] <ux-src-root>",
		Select:        lint.Selection{Names: []string{"*.html", "*.js"}, Prune: []string{"node_modules", "vendor", "dist"}},
		EmptyError:    lint.EmptyRow{Msg: "no files to scan (K20: wrong path or empty scaffold)"},
		OptionalInAll: true, // a -rt root has no *.html/*.js
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

const name = `[A-Za-z_$][A-Za-z0-9_$]*`

// minified is the length above which a line is bundled code, not the page's own poll loop.
const minified = 1000

var (
	pollCall = regexp.MustCompile(`(setInterval|setTimeout)\(\s*(` + name + `)\s*,`)
	clearing = regexp.MustCompile(`\.(innerHTML|textContent|innerText)\s*=\s*(''|""|` + "``" + `)|\.replaceChildren\(\s*\)|\.empty\(\s*\)|\.html\(\s*(''|"")\s*\)`)
	restore  = regexp.MustCompile(`\bscroll(Top|Left|To|By|IntoView)\b|\.scroll\(`)
	call     = regexp.MustCompile(`(^|[^.\w$])(` + name + `)\s*\(`)
	decl     = regexp.MustCompile(`function\s+(` + name + `)\s*\(|(const|let|var)\s+(` + name + `)\s*=\s*(async\s*)?(function\s*\(|\([^)]*\)\s*=>|` + name + `\s*=>)`)
	scroller = regexp.MustCompile(`(?i)overflow(-y|-x)?\s*:\s*(auto|scroll)\b|style\.overflow[XY]?\s*=\s*['"](auto|scroll)['"]`)
)

const tail = " clears a container on every repaint, the file has a scrolling container (overflow auto|scroll) and it " +
	"never reads or sets a scroll position (scrollTop/scrollTo/scrollIntoView): the repaint puts the operator back at " +
	"the top; save the position before the clear and restore it after the render, after an async render and on later " +
	"frames, and let wheel/touch/key input cancel the restore (types/frontend-standard.md section 5)"

// Check WARNs once, at the first clear a poll repaint runs, when the file shows a scroller and no scroll code.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	if !scroller.MatchString(strings.Join(f.Raw, "\n")) {
		return nil, nil
	}
	src := f.Raw
	if strings.HasSuffix(f.Path, ".html") {
		src = jssrc.HTMLScripts(src)
	}
	lines := jssrc.Code(src)
	for _, ln := range lines {
		if restore.MatchString(ln) {
			return nil, nil
		}
	}
	decls := declarations(lines)
	for _, fn := range polls(lines) {
		decl, ok := decls[fn]
		if !ok {
			continue
		}
		end := bodyEnd(lines, decl)
		if at := clearAt(lines, decl, end); at >= 0 {
			return []lint.Finding{{Line: at + 1, Msg: "poll repaint '" + fn + "()'" + tail}}, nil
		}
		for _, callee := range callees(lines, decl, end, decls) {
			d := decls[callee]
			if at := clearAt(lines, d, bodyEnd(lines, d)); at >= 0 {
				return []lint.Finding{{Line: at + 1, Msg: "poll repaint '" + callee + "()' (run by '" + fn + "()')" + tail}}, nil
			}
		}
	}
	return nil, nil
}

// polls is the names of the callbacks of every setInterval/setTimeout(<name>, ...) call, first seen first.
func polls(lines []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, ln := range lines {
		if len(ln) > minified {
			continue
		}
		for _, m := range pollCall.FindAllStringSubmatch(ln, -1) {
			if !seen[m[2]] {
				seen[m[2]] = true
				out = append(out, m[2])
			}
		}
	}
	return out
}

// clearAt is the first line in [from, to] that clears a container, or -1.
func clearAt(lines []string, from, to int) int {
	for i := from; i <= to && i < len(lines); i++ {
		if clearing.MatchString(lines[i]) {
			return i
		}
	}
	return -1
}

// callees is the distinct names called as `name(` in lines [from, to] that the file declares, in call order.
func callees(lines []string, from, to int, decls map[string]int) []string {
	var out []string
	seen := map[string]bool{}
	for i := from; i <= to && i < len(lines); i++ {
		for _, m := range call.FindAllStringSubmatch(lines[i], -1) {
			if _, ok := decls[m[2]]; ok && !seen[m[2]] {
				seen[m[2]] = true
				out = append(out, m[2])
			}
		}
	}
	return out
}

// declarations maps a function name to the line that declares it (`function fn(`, or
// `const|let|var fn = [async] function(`/`(...) =>`/`x =>`); the first declaration of a name wins.
func declarations(lines []string) map[string]int {
	out := map[string]int{}
	for i, ln := range lines {
		if len(ln) > minified {
			continue
		}
		if m := decl.FindStringSubmatch(ln); m != nil {
			fn := m[1] + m[3]
			if _, ok := out[fn]; !ok {
				out[fn] = i
			}
		}
	}
	return out
}

// bodyEnd is the line where the braces counted from the declaration line return to depth 0 (the declaration
// line itself for a concise arrow with no brace there or on the next line; the last line when they never close).
func bodyEnd(lines []string, decl int) int {
	depth, opened := 0, false
	for end := decl; end < len(lines); end++ {
		depth += strings.Count(lines[end], "{") - strings.Count(lines[end], "}")
		opened = opened || strings.Contains(lines[end], "{")
		if opened && depth <= 0 {
			return end
		}
		if !opened && end > decl {
			return decl
		}
	}
	return len(lines) - 1
}
