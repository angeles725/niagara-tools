// Package setnullord is the lint-set-null-ord rule: a getSlotPathOrd() result (null on an unmounted
// component) passed to a setter or set() with no null guard in the same method. Port of the awk in
// toolbelt/lint-set-null-ord.sh (the wrapper keeps the user-facing contract and its header documents the
// rule and its mutation pins). Matching keeps gawk's POSIX leftmost-longest semantics (regexp Longest).
package setnullord

import (
	"fmt"
	"regexp"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-set-null-ord"

// Spec registers the rule with the argument contract of toolbelt/lint-set-null-ord.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.One,
		Usage:  "usage: lint-set-null-ord.sh [--strict] <src-root>",
		Select: lint.Selection{Names: []string{"*.java"}},
		// Reason: bash used awk_lint_run, which scans the partial list (env line, then rows, exit 3).
		ContinueOnWalkError: true,
	}
}

const (
	nb   = `(^|[^A-Za-z0-9_])` // no identifier character before
	sp   = `[[:space:]]*`
	call = `getSlotPathOrd\(\)`
	tail = " -- null on an unmounted component and BComplex.set NPEs; "
	ref  = " (types/issues-and-gotchas.md §H3)"
)

func rx(s string) *regexp.Regexp { re := regexp.MustCompile(s); re.Longest(); return re }

var (
	setter     = rx(nb + `set[A-Za-z0-9_]*` + sp + `\(`)
	setterTail = rx(sp + `\($`)
	direct     = rx(nb + `set[A-Za-z0-9_]*` + sp + `\([^;]*` + call + sp + `[,)]`)
	directArg  = rx(`[A-Za-z0-9_.]*` + call + sp + `[,)]`)
	guardAfter = rx(`[A-Za-z0-9_.]*` + call + sp + `[!=]=` + sp + `null`)
	guardFirst = rx(`null` + sp + `[!=]=` + sp + `[A-Za-z0-9_.]*` + call)
	recvChain  = rx(`^([A-Za-z_][A-Za-z0-9_]*\.)*$`)
	assign     = rx(`[A-Za-z_][A-Za-z0-9_]*` + sp + `=` + sp + `[^=;][^;]*` + call + sp + `;`)
	ident      = rx(`^[A-Za-z_][A-Za-z0-9_]*`)
	recvEnd    = rx(call + `.*$`)
	nullFirst  = rx(`^null` + sp + `[!=]=` + sp)
)

// local is an SNO2 variable assigned from getSlotPathOrd(): its guard and set-call patterns.
type local struct {
	name       string
	line       int
	done       bool
	guard, set *regexp.Regexp
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	code := f.Stripped()
	var out []lint.Finding
	for _, m := range f.Methods() {
		guarded := map[string]bool{}
		var locals []*local
		for i := m.Start; i <= m.End; i++ {
			c := code[i-1]
			guardRecvs(c, guarded)
			if direct.MatchString(c) && unguardedDirect(c, guarded) {
				out = append(out, lint.Finding{Line: i, Msg: "getSlotPathOrd() passed straight to " + setterName(c) +
					"(...)" + tail + "assign it, guard `if (ord == null || ord.isNull())`" + ref})
			}
			if s := assign.FindString(c); s != "" {
				locals = append(locals, newLocal(ident.FindString(s), i))
				continue
			}
			for _, v := range locals {
				switch {
				case v.done:
				case v.guard.MatchString(c):
					v.done = true
				case v.set.MatchString(c):
					out = append(out, lint.Finding{Line: i, Msg: fmt.Sprintf("'%s' (getSlotPathOrd() at line %d) reaches %s(...) with no null guard"+
						tail+"guard `if (%s == null || %s.isNull())`"+ref, v.name, v.line, setterName(c), v.name, v.name)})
					v.done = true
				}
			}
		}
	}
	return out, nil
}

// newLocal builds the awk guards_var test (one alternation of its four patterns) and set-call test for v.
func newLocal(v string, line int) *local {
	return &local{name: v, line: line,
		guard: rx(nb + v + sp + `[!=]=` + sp + `null|null` + sp + `[!=]=` + sp + v + `([^A-Za-z0-9_]|$)|` +
			nb + v + `\.isNull\(|requireNonNull\(` + sp + v + `([^A-Za-z0-9_]|$)`),
		set: rx(nb + `set[A-Za-z0-9_]*` + sp + `\(([^;]*[,[:space:]])?` + v + sp + `[,)]`)}
}

// setterName is the leftmost `set<Name>(` on the line, without its boundary character and parenthesis.
func setterName(s string) string {
	t := setter.FindString(s)
	if t == "" {
		return ""
	}
	for len(t) > 0 && t[0] != 's' {
		t = t[1:]
	}
	return setterTail.ReplaceAllString(t, "")
}

// guardRecvs records every receiver (`a.`, `x.y.`, or "" for this) whose getSlotPathOrd() is compared with
// null on s. A call-chain or indexed receiver is not a plain identifier chain and is never recorded.
func guardRecvs(s string, guarded map[string]bool) {
	each(s, guardAfter, func(m string) bool { record(recvEnd.ReplaceAllString(m, ""), guarded); return true })
	each(s, guardFirst, func(m string) bool {
		record(nullFirst.ReplaceAllString(m[:len(m)-len("getSlotPathOrd()")], ""), guarded)
		return true
	})
}

func record(r string, guarded map[string]bool) {
	if recvChain.MatchString(r) {
		guarded[r] = true
	}
}

// unguardedDirect reports whether a getSlotPathOrd() argument on s has a receiver no null check guarded.
func unguardedDirect(s string, guarded map[string]bool) (found bool) {
	each(s, directArg, func(m string) bool { found = !guarded[recvEnd.ReplaceAllString(m, "")]; return !found })
	return found
}

// each is the awk loop `while (match(t, re)) { m = the match; t = the rest }`; fn returns false to stop.
func each(t string, re *regexp.Regexp, fn func(m string) bool) {
	for {
		loc := re.FindStringIndex(t)
		if loc == nil || !fn(t[loc[0]:loc[1]]) {
			return
		}
		t = t[loc[1]:]
	}
}
