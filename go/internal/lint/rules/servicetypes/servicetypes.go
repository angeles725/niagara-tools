// Package servicetypes is the lint-service-types rule: a service whose getServiceTypes() registers no type
// (an empty array or null) FAILs, and one whose returned array does not hold its own TYPE WARNs. Port of the
// awk in toolbelt/lint-service-types.sh (the wrapper keeps the user-facing contract and its header documents
// the rule and its mutation pins). Per-file rule: every verdict is read from the one file that declares the
// method, as the awk did.
package servicetypes

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

const id = "lint-service-types"

// Spec registers the rule with the argument contract of toolbelt/lint-service-types.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.One,
		Usage:  "usage: lint-service-types.sh [--strict] <src-root>",
		Select: lint.Selection{Names: []string{"*.java"}},
		// Reason: bash used awk_lint_list, then scanned the partial list (env line, then rows, exit 3).
		ContinueOnWalkError: true,
	}
}

type rule struct{}

func (rule) ID() string              { return id }
func (rule) Severity() lint.Severity { return lint.Fail }

// verdict is what one return registers, in rank order: the worst return of a method wins.
type verdict int

const (
	none       verdict = iota // no return seen yet
	own                       // the array holds the service's own type
	unresolved                // the return cannot be resolved in this file
	other                     // a non-empty array without the own type
	empty                     // an empty array or null: the service is never registered
)

const space = " \t\n\v\f\r" // awk's [[:space:]] in the C locale

var (
	decl      = regexp.MustCompile(`Type[[:space:]]*\[[[:space:]]*\][[:space:]]+getServiceTypes[[:space:]]*\(`)
	classDecl = regexp.MustCompile(`(^|[^A-Za-z0-9_])class[[:space:]]+([A-Za-z_][A-Za-z0-9_]*)`)
	spaceRun  = regexp.MustCompile(`[[:space:]]+`)
	returnSt  = regexp.MustCompile(`(^|[^A-Za-z0-9_])return[[:space:]][^;]*;`)
	returnKw  = regexp.MustCompile(`^.*return[[:space:]]`)
	ownPart   = regexp.MustCompile(`^(this\.)?(TYPE|getType\(\))$`)
	ident     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	// nested scopes opened by the "{" that follows the text before it (see nestedOpener)
	localType = regexp.MustCompile(`(^|[^A-Za-z0-9_])(class|interface|enum)[[:space:]]+[A-Za-z_]`)
	anonNew   = regexp.MustCompile(`(^|[^A-Za-z0-9_])new[[:space:]]+[A-Za-z_][A-Za-z0-9_.]*[[:space:]]*(<[^;{}()]*>)?[[:space:]]*$`)
	arrowEnd  = regexp.MustCompile(`[[:space:]]*->$`)
	caseLabel = regexp.MustCompile(`^case[[:space:]]`)
	compareOp = regexp.MustCompile(`==|!=|<=|>=`)
)

const failMsg = "getServiceTypes() registers no type (empty array or null) -- the service is never registered; " +
	"return new Type[] { TYPE } (METHODOLOGY.md § rt)"

// Check never fails: every getServiceTypes() declaration with a body gets the verdict of its worst return.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	code := f.Stripped()
	cls := ownClass(code)
	var out []lint.Finding
	for i, ln := range code {
		if !decl.MatchString(ln) || strings.Contains(untilChar(code, i, 0, '{'), ";") {
			continue // not the method, or an abstract / interface declaration
		}
		if fd, ok := finding(methodVerdict(code, i, cls), cls); ok {
			fd.Line = i + 1
			out = append(out, fd)
		}
	}
	return out, nil
}

// finding is the row of a verdict; an own-type verdict has none.
func finding(v verdict, cls string) (lint.Finding, bool) {
	switch v {
	case empty:
		return lint.Finding{Msg: failMsg}, true
	case other:
		return lint.Finding{Severity: lint.Warn, Msg: fmt.Sprintf("getServiceTypes() does not return the service's own TYPE "+
			"-- Sys.getService(%s.TYPE) will not find it; add TYPE unless that is intended", cls)}, true
	case unresolved:
		return lint.Finding{Severity: lint.Warn, Msg: "getServiceTypes() return not resolved in this file " +
			"-- check it returns a non-empty array holding TYPE"}, true
	}
	return lint.Finding{}, false
}

// ownClass is the name of the first `class X` in the file ("" when there is none).
func ownClass(code []string) string {
	for _, ln := range code {
		if m := classDecl.FindStringSubmatch(ln); m != nil {
			return m[2]
		}
	}
	return ""
}

// methodVerdict is the worst verdict over every `return <expr>;` of the body that opens on line from (nested
// scopes are blanked by blockBody); a body with no return is unresolved.
func methodVerdict(code []string, from int, cls string) verdict {
	rest := spaceRun.ReplaceAllString(blockBody(code, from), " ")
	worst := none
	for {
		loc := returnSt.FindStringIndex(rest)
		if loc == nil {
			break
		}
		e := returnKw.ReplaceAllString(rest[loc[0]:loc[1]], "")
		rest = rest[loc[1]:]
		e = noSpace(strings.TrimSuffix(e, ";"))
		v := classify(e, cls)
		if v == unresolved {
			v = resolve(e, code, cls)
		}
		worst = max(worst, v)
	}
	if worst == none {
		return unresolved
	}
	return worst
}

func noSpace(s string) string { return spaceRun.ReplaceAllString(s, "") }

// untilChar is the raw text from byte col of line from up to (not including) the first stop byte, across
// lines joined with no separator.
func untilChar(code []string, from, col int, stop byte) string {
	var b strings.Builder
	for i := from; i < len(code); i++ {
		ln := code[i]
		if i != from {
			col = 0
		}
		if k := strings.IndexByte(ln[col:], stop); k >= 0 {
			b.WriteString(ln[col : col+k])
			return b.String()
		}
		b.WriteString(ln[col:])
	}
	return b.String()
}

// classify reads a whitespace-free array expression; anything but an array literal or null is unresolved.
func classify(e, cls string) verdict {
	for len(e) >= 2 && e[0] == '(' && e[len(e)-1] == ')' {
		e = e[1 : len(e)-1] // outer parentheses, matched or not, as the awk dropped them
	}
	var inner string
	switch {
	case e == "null" || e == "{}" || e == "newType[0]" || e == "newType[]{}":
		return empty
	case strings.HasPrefix(e, "newType[]{") && strings.HasSuffix(e, "}"):
		inner = e[len("newType[]{") : len(e)-1]
	case len(e) >= 2 && e[0] == '{' && e[len(e)-1] == '}':
		inner = e[1 : len(e)-1]
	default:
		return unresolved
	}
	for _, p := range strings.Split(inner, ",") {
		if ownPart.MatchString(p) || p == cls+".TYPE" {
			return own
		}
	}
	return other
}

// resolve follows a returned field (`a`, `Own.a`, `this.a`, `(Type[]) a.clone()`) through every `a = …;` in
// the file: any empty -> empty, any other -> other, all own -> own, no assignment found -> unresolved.
func resolve(e string, code []string, cls string) verdict {
	e = strings.TrimSuffix(strings.TrimPrefix(e, "(Type[])"), ".clone()")
	if cls != "" && strings.HasPrefix(e, cls+".") {
		e = e[len(cls)+1:]
	}
	e = strings.TrimPrefix(e, "this.")
	if !ident.MatchString(e) {
		return unresolved
	}
	assign := regexp.MustCompile(`(^|[^A-Za-z0-9_])` + e + `[[:space:]]*=([^=]|$)`)
	out, seen := own, false
	for i, ln := range code {
		loc := assign.FindStringIndex(ln)
		if loc == nil {
			continue
		}
		init := noSpace(untilChar(code, i, loc[0], ';'))
		v := classify(init[strings.IndexByte(init, '=')+1:], cls)
		seen = true
		if v == empty {
			return empty
		}
		if v != own {
			if v == other || out == other {
				out = other
			} else {
				out = unresolved
			}
		}
	}
	if !seen {
		return unresolved
	}
	return out
}
