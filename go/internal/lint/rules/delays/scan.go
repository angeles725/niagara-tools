package delays

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// maxCont is the continuation-line window of one Clock.schedule*( call.
const maxCont = 12

var (
	relLiteral = re(`BRelTime\.make(Seconds|Minutes|Hours|Days)?\([0-9]+\)`)
	facetMin   = re(`BFacets\.MIN,[[:space:]]*BRelTime\.make(Seconds|Minutes|Hours|Days)?\([0-9]+\)`)
	constDecl  = re(`static[[:space:]]+final[[:space:]]+BRelTime[[:space:]]+`)
	propDecl   = re(`static[[:space:]]+final[[:space:]]+Property[[:space:]]+`)
	slotName   = re(`^[a-z][A-Za-z0-9_]*`)
	propName   = re(`name[[:space:]]*=[[:space:]]*"[^"]*"`)
	getterDecl = re(`public[[:space:]]+BRelTime[[:space:]]+get[A-Za-z0-9_]+[[:space:]]*\(`)
	getterMods = re(`public[[:space:]]+BRelTime[[:space:]]+`)
	getterName = re(`^get[A-Za-z0-9_]+`)
	getQuoted  = re(`get\("[^"]+"\)`)
	getField   = re(`get\([a-z][A-Za-z0-9_]*\)`)
	callSite   = re(`Clock\.schedule(Periodically)?\(`)
	maxInMake  = re(`BRelTime\.make\(Math\.max\(`)
	makeOpen   = re(`^BRelTime\.make\(`)
	qualCall   = re(`^[A-Za-z_][A-Za-z0-9_]*\.[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\(`)
	bareCall   = re(`^[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\(`)
	callTail   = re(`[[:space:]]*\($`)
	makeVar    = re(`BRelTime\.make\([^)]+\)`)
	maxBinding = re(`Math\.max\([^,]+,[[:space:]]*`)
	digits     = re(`^[0-9]+`)
	assign     = re(`^[[:space:]]*=[^=]`)
	getterCall = re(`^get[A-Za-z0-9_]+[[:space:]]*\([[:space:]]*\)`)
	getterArgs = re(`[[:space:]]*\(.*`)
)

// unitMs is the milliseconds per unit of a BRelTime.make<Unit>( factory ("" is make(ms)).
func unitMs(u string) float64 {
	switch u {
	case "Seconds":
		return 1000
	case "Minutes":
		return 60000
	case "Hours":
		return 3600000
	case "Days":
		return 86400000
	}
	return 1
}

// relMs reads one BRelTime.make<Unit>(N) match: its value in ms.
func relMs(seg string) float64 {
	open := strings.IndexByte(seg, '(')
	n, _ := strconv.ParseFloat(seg[open+1:len(seg)-1], 64)
	return n * unitMs(seg[len("BRelTime.make"):open])
}

// literalMs is the ms of the first BRelTime literal in text, -1 when there is none.
func literalMs(text string) float64 {
	if seg := relLiteral.FindString(text); seg != "" {
		return relMs(seg)
	}
	return -1
}

// facetMinMs is the BFacets.MIN of a property declaration in ms; -1 when BFacets.MIN is present but its
// value is not a BRelTime literal, -2 when there is no BFacets.MIN.
func facetMinMs(text string) float64 {
	if m := facetMin.FindString(text); m != "" {
		return relMs(m[strings.Index(m, "BRelTime.make"):])
	}
	if strings.Contains(text, "BFacets.MIN") {
		return -1
	}
	return -2
}

func trimSpace(s string) string { return strings.Trim(s, " \t\n\v\f\r") }

// callClosed reports whether s (the text after a call's opening paren) holds its closing paren.
func callClosed(s string) bool {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				return true
			}
			depth--
		}
	}
	return false
}

// secondArg is the second top-level argument of s, the text after a call's opening paren.
func secondArg(s string) string {
	depth, past := 0, false
	var out strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '(':
			depth++
		case c == ')':
			if depth == 0 {
				return out.String()
			}
			depth--
		case c == ',' && depth == 0:
			if past {
				return out.String()
			}
			past = true
			continue
		}
		if past {
			out.WriteByte(c)
		}
	}
	return out.String()
}

// helperName is the method a call text starts with (the last name of Class.fn( or fn( ), or "".
func helperName(s string) string {
	if m := qualCall.FindString(s); m != "" {
		m = callTail.ReplaceAllString(m, "")
		return m[strings.LastIndexByte(m, '.')+1:]
	}
	if m := bareCall.FindString(s); m != "" {
		return callTail.ReplaceAllString(m, "")
	}
	return ""
}

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

// declarations is pass 1: BRelTime constants, slot MIN facets (slotomatic and @NiagaraProperty) and the
// slot each BRelTime getter reads.
func (r *file) declarations() {
	inProp, buf, name := false, "", ""
	for i, ln := range r.lines {
		if loc := constDecl.FindStringIndex(ln); loc != nil {
			if c := ident.FindString(ln[loc[1]:]); c != "" {
				if v := literalMs(ln); v >= 0 {
					r.constMs[c] = v
				}
			}
		}
		if loc := propDecl.FindStringIndex(ln); loc != nil {
			if p := slotName.FindString(ln[loc[1]:]); p != "" {
				if v := facetMinMs(ln); v >= 0 {
					r.propMin[p] = v
				}
			}
		}
		if !inProp {
			if strings.Contains(ln, "@NiagaraProperty") {
				inProp, buf, name = true, ln, ""
			}
		} else {
			buf += " " + ln
		}
		if inProp {
			if name == "" {
				if loc := propName.FindStringIndex(buf); loc != nil {
					seg := buf[loc[0]:]
					seg = seg[strings.IndexByte(seg, '"')+1:]
					name = seg[:strings.IndexByte(seg, '"')]
				}
			}
			if strings.Count(buf, "(")-strings.Count(buf, ")") <= 0 {
				if name != "" {
					if v := facetMinMs(buf); v != -2 {
						r.propMin[name] = v
					}
				}
				inProp, buf, name = false, "", ""
			}
		}
		if !inProp {
			r.getter(i, ln)
		}
	}
}

// getter maps `public BRelTime getXxx(` on line i to the slot a get("slot") or get(slot) names on that
// line or one of the next three.
func (r *file) getter(i int, ln string) {
	loc := getterDecl.FindStringIndex(ln)
	if loc == nil {
		return
	}
	seg := ln[loc[0]:]
	seg = seg[getterMods.FindStringIndex(seg)[1]:]
	g := getterName.FindString(seg)
	if g == "" {
		return
	}
	for j := i; j <= i+3 && j < len(r.lines); j++ {
		if m := getQuoted.FindString(r.lines[j]); m != "" {
			r.getterSlot[g] = m[len(`get("`) : len(m)-2]
			return
		}
		if m := getField.FindString(r.lines[j]); m != "" {
			r.getterSlot[g] = m[len("get(") : len(m)-1]
			return
		}
	}
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
			if k := digits.FindString(strings.TrimLeft(seg[comma+1:], " \t\n\v\f\r")); k != "" {
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
		if k := digits.FindString(rhs[loc[1]:]); k != "" {
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

// guarded looks back to the start of the call's method or lambda body (scopeStart) for a positivity guard on
// expr (`expr > 0`, `>= 1`, `== 0`, or the same on `expr().getMillis()`), comments cut, and reports a PASS
// row at the call when one is found; a guard in another method never counts.
// expr is matched as text (the bash lint spliced it into a regular expression) and as a whole name.
func (r *file) guarded(lnum int, expr string) bool {
	if expr == "" {
		return false
	}
	var pats []*regexp.Regexp
	for j := lnum - 1; j >= r.scopeStart(lnum); j-- {
		ln := r.lines[j-1]
		if !strings.Contains(ln, expr) {
			continue
		}
		ln = javasrc.CutComments(ln)
		if !strings.Contains(ln, expr) {
			continue
		}
		if pats == nil {
			pats = guardPatterns(expr)
		}
		for _, p := range pats {
			if p.MatchString(ln) {
				r.emit(lint.Pass, lnum, "guarded at :"+strconv.Itoa(j))
				return true
			}
		}
	}
	return false
}

// guardPatterns are the guard regexps on the literal text expr, which no name character may precede.
func guardPatterns(expr string) []*regexp.Regexp {
	q := regexp.QuoteMeta(expr)
	if isNameByte(expr[0]) {
		q = `(^|[^A-Za-z0-9_$])` + q
	}
	var out []*regexp.Regexp
	for _, tail := range []string{
		`[[:space:]]*>[[:space:]]*0`,
		`[[:space:]]*>=[[:space:]]*1`,
		`[[:space:]]*==[[:space:]]*0`,
		`\(\)\.getMillis\(\)[[:space:]]*>[[:space:]]*0`,
		`\(\)\.getMillis\(\)[[:space:]]*>=[[:space:]]*1`,
		`\(\)\.getMillis\(\)[[:space:]]*==[[:space:]]*0`,
	} {
		out = append(out, re(q+tail))
	}
	return out
}
