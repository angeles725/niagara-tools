// Package spapoll is the lint-spa-poll-no-recovery rule: a browser poll loop (the first setInterval/
// setTimeout(<name>, ...) of a .html/.js file) whose function has a catch path WARNs when the file has no
// location.reload(/location.href recovery, or when that recovery is not keyed on the time since the last
// success (a lastOk* timestamp set from a clock in the poll body and compared). Port of the awk in
// toolbelt/lint-spa-poll-no-recovery.sh (the wrapper keeps the user-facing contract and its header documents
// the rule and its mutation pins).
package spapoll

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-spa-poll-no-recovery"

// Spec registers the rule with the argument contract of toolbelt/lint-spa-poll-no-recovery.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:       rule{},
		Mode:       lint.First,
		Usage:      "usage: lint-spa-poll-no-recovery.sh [--strict] <ux-src-root>",
		Select:     lint.Selection{Names: []string{"*.html", "*.js"}},
		EmptyError: lint.EmptyRow{Msg: "no files to scan (K20: wrong path or empty scaffold)"},
		// Reason: bash used awk_lint_run, which scans the partial list (env line, then rows, exit 3).
		ContinueOnWalkError: true,
		// A -rt root has no *.html/*.js: under --all that is not an error for this -ux rule.
		OptionalInAll: true,
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

func rx(s string) *regexp.Regexp { re := regexp.MustCompile(s); re.Longest(); return re }

const name = `[A-Za-z_][A-Za-z0-9_]*`

var (
	pollCall  = rx(`(setInterval|setTimeout)\([[:space:]]*` + name + `[[:space:]]*,`)
	callName  = rx(name + `[[:space:]]*,$`)
	blank     = rx(`^[[:space:]]*$`)
	openBrace = rx(`^[[:space:]]*\{`)
	catchCall = rx(`catch[[:space:]]*\(`)
	hrefSet   = rx(`location\.href[[:space:]]*=`)
	okSet     = rx(`lastOk[A-Za-z0-9_]*[[:space:]]*=[[:space:]]*(\+[[:space:]]*)?(Date\.now\(|performance\.now\(|new[[:space:]]+Date)`)
	okMinus   = rx(`-[[:space:]]*lastOk[A-Za-z0-9_]*`)
	okCompare = rx(`lastOk[A-Za-z0-9_]*[[:space:]]*[<>]`)
)

const (
	tailNoRecovery = " has a catch path with no location.reload(/location.href recovery anywhere in the file --" +
		" kiosk/HMI will stay blank after a station restart; add a watchdog on time since the last success" +
		" (lastOkAt) -> fetchT probe of the app marker -> location.reload()"
	tailNotKeyed = " reloads, but the recovery is not keyed on time since the last success (no lastOk* timestamp set" +
		" from a clock in the poll body and compared) -- a failure-count or reload-on-error watchdog is blind to a" +
		" hung request; key it on lastOkAt (types/dashboard.md Poll loop reliability)"
)

// Check WARNs once at the first poll call when its function has a catch path and the file's recovery is
// missing or not keyed on the last success.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	lines := make([]string, len(f.Raw)) // each line cut at its first "//", as the awk's sub(/\/\/.*$/, "")
	for i, ln := range f.Raw {
		lines[i], _, _ = strings.Cut(ln, "//")
	}
	call, fn := -1, ""
	for i, ln := range lines {
		if m := pollCall.FindString(ln); m != "" {
			fn = strings.TrimRight(callName.FindString(m), " \t\n\v\f\r,")
			call = i
			break
		}
	}
	if fn == "" {
		return nil, nil
	}
	decl, end, ok := body(lines, fn)
	if !ok || !anyMatch(lines[decl:end+1], catchCall) {
		return nil, nil
	}
	prefix := "setInterval/setTimeout poll loop '" + fn + "()'"
	recovery := false
	for _, ln := range lines {
		if strings.Contains(ln, "location.reload(") || hrefSet.MatchString(ln) {
			recovery = true
			break
		}
	}
	if !recovery {
		return []lint.Finding{{Line: call + 1, Msg: prefix + tailNoRecovery}}, nil
	}
	set, read := false, false
	for i, ln := range lines {
		set = set || i >= decl && i <= end && okSet.MatchString(ln)
		read = read || okMinus.MatchString(ln) || okCompare.MatchString(ln)
	}
	if set && read {
		return nil, nil
	}
	return []lint.Finding{{Line: call + 1, Msg: prefix + tailNotKeyed}}, nil
}

// body finds fn's declaration (`function fn(`, or `const|let|var fn = [async] function(` / `(...) =>` /
// `x =>`) and the last line of its body: a concise arrow (no "{" after its "=>") ends at the first line
// holding a ";" within 20 lines, any other body where its braces, counted from the declaration line,
// return to depth 0 (the last line when they never do).
func body(lines []string, fn string) (decl, end int, ok bool) {
	q := regexp.QuoteMeta(fn)
	fnDecl := rx(`function[[:space:]]+` + q + `[[:space:]]*\(`)
	binding := rx(`(const|let|var)[[:space:]]+` + q + `[[:space:]]*=[[:space:]]*(async[[:space:]]*)?(function[[:space:]]*\(|\([^)]*\)[[:space:]]*=>|` + name + `[[:space:]]*=>)`)
	var loc []int
	for decl = range lines {
		if loc = fnDecl.FindStringIndex(lines[decl]); loc == nil {
			loc = binding.FindStringIndex(lines[decl])
		}
		if loc != nil {
			break
		}
	}
	if loc == nil {
		return 0, 0, false
	}
	arrow := strings.HasSuffix(lines[decl][:loc[1]], "=>")
	concise := false
	if arrow {
		next := lines[decl][loc[1]:]
		for i := decl + 1; blank.MatchString(next) && i < len(lines); i++ {
			next = lines[i]
		}
		concise = !openBrace.MatchString(next)
	}
	last := len(lines) - 1
	if concise {
		i := decl
		for ; i <= last && i < decl+20; i++ {
			if strings.Contains(lines[i], ";") {
				break
			}
		}
		return decl, min(i, last), true
	}
	depth, started := 0, false
	for i := decl; i <= last; i++ {
		for _, c := range []byte(lines[i]) {
			switch c {
			case '{':
				depth, started = depth+1, true
			case '}':
				depth--
			}
		}
		if started && depth == 0 {
			return decl, i, true
		}
	}
	return decl, last, true
}

func anyMatch(lines []string, re *regexp.Regexp) bool {
	for _, ln := range lines {
		if re.MatchString(ln) {
			return true
		}
	}
	return false
}
