package delays

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

type file struct {
	path       string
	lines      []string
	code       []string         // the lines with comments removed (javasrc.Strip): where calls are read
	scopes     []javasrc.Method // the method and class-level lambda bodies (javasrc.Scopes)
	helpers    map[string]bool
	constMs    map[string]float64
	propMin    map[string]float64
	getterSlot map[string]string
	out        []lint.Finding
}

func scanFile(f *javasrc.File, helpers map[string]bool) []lint.Finding {
	r := &file{path: f.Path, lines: f.Raw, code: f.Stripped(), scopes: f.Scopes(), helpers: helpers,
		constMs: map[string]float64{}, propMin: map[string]float64{}, getterSlot: map[string]string{}}
	r.declarations()
	r.calls()
	return r.out
}

func (r *file) emit(sev lint.Severity, line int, msg string) {
	r.out = append(r.out, lint.Finding{Check: check, Severity: sev, Path: r.path, Line: line, Msg: msg})
}

// calls is pass 2: every Clock.schedule*( call in code (not in a comment or a literal), its argument
// list read from comment-free lines up to the closing paren over at most maxCont continuation lines,
// and its delay argument classified.
func (r *file) calls() {
	for i, ln := range r.code {
		loc := codeCall(ln)
		if loc == nil {
			continue
		}
		rest := ln[loc[1]:]
		for j := i + 1; !callClosed(rest) && j < len(r.code) && j <= i+maxCont; j++ {
			rest += " " + r.code[j]
		}
		if !callClosed(rest) {
			r.emit(lint.Fail, i+1, fmt.Sprintf("unclosed-call  Clock.schedule( not closed within %d lines -- cannot read the delay; wrap the call tighter", maxCont))
			continue
		}
		r.classify(i+1, secondArg(rest))
	}
}

// codeCall is the span of the first Clock.schedule*( of a comment-free line that is not inside a
// string or char literal, nil when there is none.
func codeCall(ln string) []int {
	for _, loc := range callSite.FindAllStringIndex(ln, -1) {
		if _, open := javasrc.BlankLiterals([]string{ln[:loc[0]]}, nil); !open {
			return loc
		}
	}
	return nil
}
