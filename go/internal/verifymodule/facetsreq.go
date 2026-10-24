package verifymodule

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

var (
	nameAttrRe     = regexp.MustCompile(`name[[:space:]]*=[[:space:]]*"([^"]*)"`)
	typeAttrRe     = regexp.MustCompile(`type[[:space:]]*=[[:space:]]*"([^"]*)"`)
	facetsKeyRe    = regexp.MustCompile(`[[:space:],]facets[[:space:]]*=`)
	numericTypeRe  = regexp.MustCompile(`^(double|float|BDouble|BFloat|BInteger|BLong|BRelTime|BStatusNumeric)$`)
	setpointNameRe = regexp.MustCompile(`[Ss]etpoint|Temp|[Ll]imit|[Bb]and|[Pp]si`)
	countNameRe    = regexp.MustCompile(`demand|[Cc]ount|stages`)
	unitArgRe      = regexp.MustCompile(`[Uu]nit|UNIT`)
	nullFacetsRe   = regexp.MustCompile(`, *null *[);]`)
	upToPropertyRe = rx(`.*Property[[:space:]]+`)
	fromEqualsRe   = rx(`[[:space:]]*=.*`)
	spaceRe        = regexp.MustCompile(`[[:space:]]`)
	slotNameRe     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

func (r *run) facetPresence() bool {
	pd, ok := r.srcDirOrSkip("facets-req")
	if !ok {
		return true
	}
	warned := false
	for _, f := range r.javaFiles(pd + "/src") {
		v, ok := r.java(f)
		if !ok {
			return true
		}
		for _, hit := range facetHits(f, v.code) {
			r.row("WARN", "facets-req", hit)
			warned = true
		}
	}
	if !warned {
		r.row("PASS", "facets-req", "no missing required facets under "+pd+"/src")
	}
	return true
}

// facetHits is the facets-req scan over one file's javasrc code lines (comments blanked): pass 1 reads
// every @NiagaraProperty joined to its balancing ")" (javasrc.PropertyScan), pass 2 every old-style
// newProperty(... OPERATOR ..., null) slot no annotation declared. Each hit is "<file>:<line> <reason>".
// The bash awk read the raw text and joined an annotation only when its own line held no ")", up to a
// line that is only ")" (declared difference, truth-java-view).
func facetHits(file string, lines []string) []string {
	var out []string
	seen := map[string]bool{}
	type prop struct {
		s string
		l int
	}
	var props []prop
	javasrc.NiagaraProperties(lines, func(text string, line int) { props = append(props, prop{text, line - 1}) })
	for _, p := range props {
		s, l := p.s, p.l
		nm := submatch(nameAttrRe, s)
		if nm == "" || seen[nm] {
			continue
		}
		typ, reason := submatch(typeAttrRe, s), ""
		if strings.Contains(s, "OPERATOR") && !facetsKeyRe.MatchString(s) && (typ == "" || numericTypeRe.MatchString(typ)) {
			reason = "OPERATOR numeric without facets (slot=" + nm + ")"
		}
		if reason == "" && setpointNameRe.MatchString(nm) && !strings.Contains(s, "UNITS") && !makeNumericHasUnit(s) {
			reason = "setpoint-like slot " + nm + " missing UNITS"
		}
		if reason == "" && countNameRe.MatchString(nm) && !strings.Contains(s, "PRECISION") && !strings.Contains(s, "makeNumeric(") {
			reason = "count-like slot " + nm + " missing PRECISION"
		}
		seen[nm] = true
		if reason != "" {
			out = append(out, file+":"+itoa(l+1)+" "+reason)
		}
	}
	for l, ln := range lines {
		if !strings.Contains(ln, "newProperty(") || !strings.Contains(ln, "OPERATOR") {
			continue
		}
		s := ln
		for j := l + 1; j < len(lines) && j <= l+5 && !strings.Contains(s, ";"); j++ {
			s += " " + lines[j]
		}
		if !nullFacetsRe.MatchString(s) {
			continue
		}
		nm := replaceFirst(upToPropertyRe, s)
		nm = spaceRe.ReplaceAllString(replaceFirst(fromEqualsRe, nm), "")
		if !slotNameRe.MatchString(nm) || seen[nm] {
			continue
		}
		seen[nm] = true
		out = append(out, file+":"+itoa(l+1)+" OPERATOR numeric null facets (slot="+nm+")")
	}
	return out
}

// makeNumericHasUnit is the awk mn_has_unit: whether the first BFacets.makeNumeric( call in s carries a
// unit. makeNumeric(<int>) is the only one-argument overload, so a lone argument is a precision unless it
// names a unit (BUnit..., *UNIT*); a first argument starting with a letter followed by more arguments is
// the (<unit>, <precision>, ...) form.
func makeNumericHasUnit(s string) bool {
	i := strings.Index(s, "makeNumeric(")
	if i < 0 {
		return false
	}
	depth, commas, arg := 0, 0, []byte{}
scan:
	for _, c := range []byte(s[i+len("makeNumeric("):]) {
		switch {
		case c == '(':
			depth++
		case c == ')' && depth == 0:
			break scan
		case c == ')':
			depth--
		case c == ',' && depth == 0:
			commas++
		}
		if commas == 0 {
			arg = append(arg, c)
		}
	}
	first := string(arg)
	first = strings.TrimFunc(first, func(c rune) bool { return strings.ContainsRune(" \t\n\v\f\r", c) })
	if first == "" || !isAlpha(first[0]) && first[0] != '_' {
		return false
	}
	return unitArgRe.MatchString(first) || commas >= 1
}

// submatch is awk's match() then the sub()s that keep the quoted value: "" when re does not match.
func submatch(re *regexp.Regexp, s string) string {
	if m := re.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return ""
}

// replaceFirst is awk sub(re, ""): the leftmost-longest match removed.
func replaceFirst(re *regexp.Regexp, s string) string {
	if k := re.FindStringIndex(s); k != nil {
		return s[:k[0]] + s[k[1]:]
	}
	return s
}
