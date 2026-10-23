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
	helpers    map[string]bool
	constMs    map[string]float64
	propMin    map[string]float64
	getterSlot map[string]string
	out        []lint.Finding
	err        error
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

// helper reports a delay computed by method fn: proven floor, same-method guard, or FAIL. guard is false
// for a variable bound to the call, which awk never checked for a guard on fn.
func (r *file) helper(lnum int, fn string, guard bool) error {
	if r.helpers[fn] {
		return nil
	}
	if guard {
		if done, err := r.guarded(lnum, fn); done || err != nil {
			return err
		}
	}
	r.emit(lint.Fail, lnum, "zero-floor  helper "+fn+" has no visible strictly-positive floor")
	return nil
}

// binding reads one earlier line ln for the binding of variable v; true when it decided the call.
func (r *file) binding(lnum int, v, ln string) bool {
	if awkIndex(ln, v) == 0 {
		return false
	}
	if loc := maxBinding.FindStringIndex(ln); loc != nil {
		if k := digits.FindString(ln[loc[1]:]); k != "" {
			if n, _ := strconv.ParseFloat(k, 64); n <= 0 {
				r.emit(lint.Fail, lnum, "zero-floor  local "+v+"=Math.max(.,k) k="+num(n)+" (<=0)")
			}
			return true
		}
	}
	if ms := literalMs(ln); ms >= 0 {
		if ms == 0 {
			r.emit(lint.Fail, lnum, "literal-zero  local "+v+"=BRelTime(0)")
		}
		return true
	}
	after := ln[awkIndex(ln, v)-1+len(v):]
	if loc := assign.FindStringIndex(after); loc != nil {
		if fn := helperName(strings.TrimLeft(after[loc[1]:], " \t\n\v\f\r")); fn != "" {
			_ = r.helper(lnum, fn, false)
			return true
		}
	}
	return false
}

// guarded looks up to 80 lines back for a positivity guard on expr (`expr > 0`, `>= 1`, `== 0`, or the
// same on `expr().getMillis()`), comments cut, and reports a PASS row at the call when one is found.
func (r *file) guarded(lnum int, expr string) (bool, error) {
	if expr == "" {
		return false, nil
	}
	var pats []*regexp.Regexp
	for j := lnum - 1; j >= 1 && j >= lnum-80; j-- {
		ln := r.lines[j-1]
		if !strings.Contains(ln, expr) {
			continue
		}
		ln = javasrc.CutComments(ln)
		if !strings.Contains(ln, expr) {
			continue
		}
		if pats == nil {
			var err error
			if pats, err = guardPatterns(expr); err != nil {
				return false, fmt.Errorf("cannot scan %s: guard pattern for %q at line %d is not a valid regular expression (awk died on it): %v", r.path, expr, lnum, err)
			}
		}
		for _, p := range pats {
			if p.MatchString(ln) {
				r.emit(lint.Pass, lnum, "guarded at :"+strconv.Itoa(j))
				return true, nil
			}
		}
	}
	return false, nil
}

// guardPatterns are awk's dynamic guard regexps: expr is spliced in as a regular expression.
func guardPatterns(expr string) ([]*regexp.Regexp, error) {
	var out []*regexp.Regexp
	for _, tail := range []string{
		`[[:space:]]*>[[:space:]]*0`,
		`[[:space:]]*>=[[:space:]]*1`,
		`[[:space:]]*==[[:space:]]*0`,
		`\(\)\.getMillis\(\)[[:space:]]*>[[:space:]]*0`,
		`\(\)\.getMillis\(\)[[:space:]]*>=[[:space:]]*1`,
		`\(\)\.getMillis\(\)[[:space:]]*==[[:space:]]*0`,
	} {
		p, err := regexp.CompilePOSIX(expr + tail)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}
