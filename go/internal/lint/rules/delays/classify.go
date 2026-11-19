package delays

import (
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// classify reports the delay argument arg of the call on line lnum (1-based).
func (r *file) classify(lnum int, arg string) {
	arg = trimSpace(arg)

	// 1. An inline BRelTime literal.
	if v := literalMs(arg); v >= 0 {
		if v == 0 {
			r.emit(lint.Fail, lnum, "literal-zero  BRelTime(0) — Niagara rejects delay/period <= 0")
		}
		return
	}

	// 2. BRelTime.make(Math.max(x, k)).
	if loc := maxInMake.FindStringIndex(arg); loc != nil {
		seg := arg[loc[1]:]
		depth, comma := 0, -1
	scan:
		for i := 0; i < len(seg); i++ {
			switch seg[i] {
			case '(':
				depth++
			case ')':
				if depth == 0 {
					break scan
				}
				depth--
			case ',':
				if depth == 0 {
					comma = i
					break scan
				}
			}
		}
		if comma >= 0 {
			k := digits.FindString(strings.TrimLeft(seg[comma+1:], " \t\n\v\f\r"))
			if k == "" { // retro lint-dashboard-write-save Δ3: a bare literal first argument, Math.max(1L, x), is the floor too
				k = firstLiteral(seg[:comma])
			}
			if k != "" {
				if n, _ := strconv.ParseFloat(k, 64); n <= 0 {
					r.emit(lint.Fail, lnum, "zero-floor  Math.max floor="+num(n)+" (<=0) — use Math.max(x,1L)")
				}
				return
			}
		}
		r.emit(lint.Fail, lnum, "zero-floor  cannot determine Math.max floor in: "+arg)
		return
	}

	// 2.5 D2b: BRelTime.make(<method call>), the helper resolved one level across files.
	if loc := makeOpen.FindStringIndex(arg); loc != nil {
		if fn := helperName(strings.TrimLeft(arg[loc[1]:], " \t\n\v\f\r")); fn != "" {
			r.helper(lnum, fn, true)
			return
		}
	}

	// 3. BRelTime.make(VAR): a same-method guard, else the variable's nearest binding in the same body.
	if loc := makeVar.FindStringIndex(arg); loc != nil {
		seg := strings.Replace(arg[loc[0]:], "BRelTime.make(", "", 1)
		if k := strings.IndexByte(seg, ')'); k >= 0 {
			seg = seg[:k]
		}
		v := trimSpace(seg)
		if r.guarded(lnum, v) {
			return
		}
		for j := lnum - 1; j >= r.scopeStart(lnum); j-- {
			if r.binding(lnum, v, r.code[j-1]) {
				return
			}
		}
		r.emit(lint.Fail, lnum, "unfloored  cannot verify floor of variable: "+v)
		return
	}

	// 4. A slot getter: getXxx().
	if m := getterCall.FindString(arg); m != "" {
		g := getterArgs.ReplaceAllString(m, "")
		if r.guarded(lnum, g) {
			return
		}
		slot, ok := r.getterSlot[g]
		switch min, has := r.propMin[slot]; {
		case !ok:
			r.emit(lint.Fail, lnum, "unfloored  unknown getter "+g+" — cannot verify floor")
		case !has:
			r.emit(lint.Fail, lnum, "facet-min-zero  slot "+slot+" has no MIN facet — can schedule 0")
		case min >= 1:
			r.emit(lint.Warn, lnum, "facet-floor  slot "+slot+" facet MIN="+num(min)+"ms (advisory)")
		default:
			r.emit(lint.Fail, lnum, "facet-min-zero  slot "+slot+" facet MIN=0 — schedule can receive 0")
		}
		return
	}

	// 5. A known static final BRelTime constant.
	if v, ok := r.constMs[arg]; ok {
		if v <= 0 {
			r.emit(lint.Fail, lnum, "literal-zero  constant "+arg+"="+num(v)+"ms (<=0)")
		}
		return
	}

	// 6. Unresolved.
	r.emit(lint.Fail, lnum, "unfloored  cannot verify floor: "+arg)
	return
}

// helper reports a delay computed by method fn: proven floor, same-method guard, or FAIL. guard is false
// for a variable bound to the call, which awk never checked for a guard on fn.
func (r *file) helper(lnum int, fn string, guard bool) {
	if r.helpers[fn] {
		return
	}
	if guard && r.guarded(lnum, fn) {
		return
	}
	r.emit(lint.Fail, lnum, "zero-floor  helper "+fn+" has no visible strictly-positive floor")
	return
}

// scopeStart is the first line (1-based) of the method or lambda body holding line lnum (javasrc.Scopes),
// where a backward walk from the call stops; lnum itself when no body holds it (nothing is read).
func (r *file) scopeStart(lnum int) int {
	if m, ok := javasrc.Enclosing(r.scopes, lnum); ok {
		return m.Start
	}
	return lnum
}
