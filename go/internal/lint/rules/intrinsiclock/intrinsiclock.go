// Package intrinsiclock is lint-intrinsic-lock: locks on a Niagara object's own monitor, and nested locks (corpus
// B1242 §1242.7 K3, from B1238 §1238.7). The framework takes the same monitor: every property set runs
// synchronized (this.instance) in the slot map and BComponent has synchronized methods, so a module that holds it
// while waiting on another thread can close a deadlock cycle. toolbelt/lint-intrinsic-lock.sh is the wrapper and its
// header is the user-facing contract.
//
// A per-file rule over *.java, read as javasrc tokens (comments blanked, a literal one token). A Niagara class is a
// named class whose own name or superclass name is B followed by an upper-case letter (BFoo, extends BComponent;
// Buffer and BatchQueue are not). Rows: IL1 a synchronized instance method of a Niagara class; IL2 a
// synchronized (this) or synchronized (<Class>.this) block whose this is a Niagara class (an anonymous class's plain
// this is its own object, not judged; <Class>.this inside it is judged); IL3 a synchronized block inside another
// synchronized block or synchronized method, in any class (the course rule: never nest locks), unless a lambda body
// or an anonymous class's method lies between them: that body runs later, not under the outer lock. A static
// synchronized method locks the Class object and is not judged.
package intrinsiclock

import (
	"sort"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-intrinsic-lock"

// Spec registers the rule: [--strict] [--] <src-root>, WARN rows (exit 1 under --strict).
func Spec() lint.Spec {
	return lint.Spec{
		Rule:       rule{},
		Mode:       lint.One,
		Usage:      "usage: lint-intrinsic-lock.sh [--strict] <src-root>",
		Select:     lint.Selection{Names: []string{"*.java"}},
		EmptyError: lint.EmptyRow{Msg: "no *.java to scan (K20: wrong path or empty scaffold)"},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

const (
	where = " (types/logic.md § Tridium rt idioms)"
	fix   = ": lock a private final Object or use Atomic*" + where
	il1   = "IL1 synchronized method locks the component, which the framework also locks" + fix
	il2   = "IL2 synchronized (this) locks the component, which the framework also locks" + fix
	il3   = "IL3 synchronized block nested in another lock: nested locks can deadlock; take one lock at a time" + where
)

// niagara: B followed by an upper-case letter.
func niagara(name string) bool {
	return len(name) > 1 && name[0] == 'B' && name[1] >= 'A' && name[1] <= 'Z'
}

// region is the token range [from, to) a lock is held over: a method body (from is its first token) or the braces of
// a synchronized block (from is its "{", after the block's own keyword).
type region struct{ from, to int }

// Check reports each finding at the line of its synchronized keyword, in line order.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	toks := javasrc.Tokens(f.Stripped())
	nb := map[string]bool{}
	for _, c := range javasrc.Classes(toks) {
		if niagara(c.Name) || niagara(c.Extends) {
			nb[c.Name] = true
		}
	}
	decls := javasrc.Decls(toks)
	var out []lint.Finding
	var held []region
	for _, d := range decls {
		if k := syncModifier(toks, d); k >= 0 {
			held = append(held, region{d.Body[0], d.Body[1]})
			if !d.Static && !d.Anonymous && nb[d.Class] {
				out = append(out, lint.Finding{Line: toks[k].Line + 1, Msg: il1})
			}
		}
	}
	type block struct {
		at int
		r  region
	}
	var blocks []block
	for k, t := range toks {
		if t.Text != "synchronized" || t.Kind != javasrc.TokIdent || k+1 >= len(toks) || toks[k+1].Text != "(" {
			continue
		}
		closeParen := match(toks, k+1, "(", ")")
		if closeParen < 0 || closeParen+1 >= len(toks) || toks[closeParen+1].Text != "{" {
			continue
		}
		closeBrace := match(toks, closeParen+1, "{", "}")
		if closeBrace < 0 {
			continue
		}
		blocks = append(blocks, block{k, region{closeParen + 1, closeBrace}})
		if d, ok := enclosing(decls, k); thisOf(toks[k+2:closeParen], d, ok, nb) {
			out = append(out, lint.Finding{Line: t.Line + 1, Msg: il2})
		}
	}
	for _, b := range blocks {
		held = append(held, b.r)
	}
	deferred := deferredBodies(toks, decls)
	for _, b := range blocks {
		for _, r := range held {
			if r.from <= b.at && b.at < r.to && !runsLater(deferred, r, b.at) {
				out = append(out, lint.Finding{Line: toks[b.at].Line + 1, Msg: il3})
				break
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Line < out[j].Line })
	return out, nil
}

// syncModifier is the index of a synchronized modifier among d's modifiers, or -1. The walk goes back from the "("
// of the parameter list over the name, the return type, type parameters, annotations (their arguments skipped
// whole) and modifiers, and stops at the end of the previous member.
func syncModifier(toks []javasrc.Token, d javasrc.Decl) int {
	open := paramsOpen(toks, d.Body[0]-1)
	for k, depth := open-1, 0; k >= 0; k-- {
		switch toks[k].Text {
		case ")":
			depth++
		case "(":
			depth--
		case ";", "{", "}":
			if depth == 0 {
				return -1
			}
		case "synchronized":
			if depth == 0 {
				return k
			}
		}
	}
	return -1
}

// paramsOpen is the "(" of the parameter list of the method whose body opens at toks[brace]: the last ")" before
// the brace (after any throws clause) matched back.
func paramsOpen(toks []javasrc.Token, brace int) int {
	k := brace - 1
	for k >= 0 && toks[k].Text != ")" {
		k--
	}
	for depth := 0; k >= 0; k-- {
		switch toks[k].Text {
		case ")":
			depth++
		case "(":
			if depth--; depth == 0 {
				return k
			}
		}
	}
	return 0
}

// match is the index of the bracket closing the one at toks[k], or -1 when unbalanced.
func match(toks []javasrc.Token, k int, open, close string) int {
	depth := 0
	for ; k < len(toks); k++ {
		switch toks[k].Text {
		case open:
			depth++
		case close:
			if depth--; depth == 0 {
				return k
			}
		}
	}
	return -1
}

// enclosing is the innermost declaration whose body holds token k.
func enclosing(decls []javasrc.Decl, k int) (javasrc.Decl, bool) {
	best, found := javasrc.Decl{}, false
	for _, d := range decls {
		if d.Body[0] <= k && k < d.Body[1] && (!found || d.Body[0] > best.Body[0]) {
			best, found = d, true
		}
	}
	return best, found
}

// thisOf: the lock expression is this or <Name>.this, and the object it names is a Niagara class. A plain this in
// an anonymous class body (d.Anonymous) is that anonymous object and is not judged; <Name>.this names its class
// wherever it stands, an anonymous body included.
func thisOf(expr []javasrc.Token, d javasrc.Decl, inDecl bool, nb map[string]bool) bool {
	switch {
	case len(expr) == 1 && expr[0].Text == "this":
		return inDecl && !d.Anonymous && nb[d.Class]
	case len(expr) == 3 && expr[1].Text == "." && expr[2].Text == "this":
		// Mutation: IL2-outer-this-anon -- judging only non-anonymous bodies skips BFoo.this in an anonymous class
		return nb[expr[0].Text]
	}
	return false
}

// deferredBodies are the bodies that run later, not where they are written: each block lambda body (its "{" after
// "->") and each method body of an anonymous class. A lock taken inside one is not held under a lock around it.
func deferredBodies(toks []javasrc.Token, decls []javasrc.Decl) []region {
	var out []region
	for _, d := range decls {
		if d.Anonymous {
			out = append(out, region{d.Body[0], d.Body[1]})
		}
	}
	for k := 0; k+2 < len(toks); k++ {
		if toks[k].Text == "-" && toks[k+1].Text == ">" && toks[k+2].Text == "{" && toks[k].Kind == javasrc.TokPunct {
			if end := match(toks, k+2, "{", "}"); end >= 0 {
				out = append(out, region{k + 2, end})
			}
		}
	}
	return out
}

// runsLater: a deferred body that starts inside lock region r holds token at, so the code at at does not run under r.
// The strict r.from < d.from keeps an anonymous synchronized method's own lock: its body is both r and d.
func runsLater(deferred []region, r region, at int) bool {
	for _, d := range deferred {
		// Mutation: IL3-deferred -- a lambda or anonymous method inside synchronized counts as a nested lock again
		if r.from < d.from && d.to <= r.to && d.from <= at && at < d.to {
			return true
		}
	}
	return false
}
