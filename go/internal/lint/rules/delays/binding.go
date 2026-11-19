package delays

import (
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// binding reads one earlier code line ln (comments removed) for an assignment of variable v (a whole name
// followed by "=", not "=="); false when ln assigns no v. The nearest assignment decides the call: a
// Math.max(., k) floor (FAIL when k <= 0), a BRelTime literal (FAIL when 0) or a helper call, and any other
// value FAILs unfloored; it never falls through to an earlier binding.
func (r *file) binding(lnum int, v, ln string) bool {
	rhs, ok := assigned(ln, v)
	if !ok {
		return false
	}
	if loc := maxBinding.FindStringIndex(rhs); loc != nil {
		k := digits.FindString(rhs[loc[1]:])
		if k == "" { // Δ3: Math.max(1L, x), the floor in the first argument
			head := rhs[loc[0]+len("Math.max(") : loc[1]]
			k = firstLiteral(head[:strings.LastIndexByte(head, ',')])
		}
		if k != "" {
			if n, _ := strconv.ParseFloat(k, 64); n <= 0 {
				r.emit(lint.Fail, lnum, "zero-floor  local "+v+"=Math.max(.,k) k="+num(n)+" (<=0)")
			}
			return true
		}
	}
	if ms := literalMs(rhs); ms >= 0 {
		if ms == 0 {
			r.emit(lint.Fail, lnum, "literal-zero  local "+v+"=BRelTime(0)")
		}
		return true
	}
	if fn := helperName(strings.TrimLeft(rhs, " \t\n\v\f\r")); fn != "" {
		r.helper(lnum, fn, false)
		return true
	}
	r.emit(lint.Fail, lnum, "unfloored  cannot verify floor of variable: "+v)
	return true
}

// assigned is the text after the "=" of the first whole-name v in s that is assigned (v, blanks, "=" not
// followed by "="), ok false when s assigns no v.
func assigned(s, v string) (rhs string, ok bool) {
	for pos := nameAt(s, v); pos >= 0; pos = nameFrom(s, v, pos+1) {
		after := s[pos+len(v):]
		if loc := assign.FindStringIndex(after); loc != nil {
			return after[loc[1]-1:], true
		}
	}
	return "", false
}

func isNameByte(c byte) bool {
	return c == '_' || c == '$' || c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z'
}

// nameAt is the byte offset of the first whole-name occurrence of v in s (no name character glued to an
// end of v that is itself a name character), -1 when there is none.
func nameAt(s, v string) int { return nameFrom(s, v, 0) }

// nameFrom is nameAt from byte offset off of s on.
func nameFrom(s, v string, off int) int {
	if v == "" {
		return -1
	}
	for {
		i := strings.Index(s[off:], v)
		if i < 0 {
			return -1
		}
		i += off
		end := i + len(v)
		if (i == 0 || !isNameByte(v[0]) || !isNameByte(s[i-1])) &&
			(end == len(s) || !isNameByte(v[len(v)-1]) || !isNameByte(s[end])) {
			return i
		}
		off = i + 1
	}
}
