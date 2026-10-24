// Package configsanity is the lint-config-sanity rule: four unsafe @NiagaraProperty default patterns
// (CS1 interval <= duration, CS2 zero OPERATOR setpoint, CS3 comment-only flag rule, CS4 permanent floor
// beside a disabled cutout). Port of the four gawk passes of toolbelt/lint-config-sanity.sh (the wrapper
// keeps the user-facing contract and its header documents the checks and their mutation pins). Per-file
// rule: every pass reads the RAW lines of one file (comments are not stripped, as the awk did not), and
// a file's rows come pass by pass: CS1 unreadable WARNs, CS1 FAILs, CS2, CS3, CS4.
//
// The matching keeps gawk's semantics: POSIX leftmost-longest matches in the C locale, "." matching any
// byte, and the `for (k in a)` order of the CS1 pairs (awkorder.Order).
package configsanity

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/awkorder"
)

const id = "lint-config-sanity"

// Spec registers the rule with the argument contract of toolbelt/lint-config-sanity.sh: no flag is parsed
// and only the first argument is read (Raw), and an incomplete walk stops the lint (scan_files_or_exit).
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.Raw,
		Usage:  "usage: lint-config-sanity.sh <java-src-dir>",
		Select: lint.Selection{Names: []string{"*.java"}},
	}
}

type rule struct{}

func (rule) ID() string              { return id }
func (rule) Severity() lint.Severity { return lint.Fail }

// rx compiles a gawk regular expression: leftmost-longest, "." matching a newline.
func rx(pattern string) *regexp.Regexp {
	re := regexp.MustCompile("(?s)" + pattern)
	re.Longest()
	return re
}

var (
	nameAttr    = rx(`name[[:space:]]*=[[:space:]]*"([^"]+)"`)
	defaultAttr = rx(`defaultValue[[:space:]]*=[[:space:]]*"([^"]*)"`)
	spaceRun    = rx(`[[:space:]]`)

	// CS1: the BRelTime factories and constants of javax/baja/sys/BRelTime.java:34-48, :56-83
	factory      = rx(`BRelTime\.(make[A-Za-z]*\([^()]*\)|DEFAULT|SECOND|MINUTE|HOUR|DAY)`)
	makeMs       = rx(`^BRelTime\.make\(([0-9]+)[Ll]?\)$`)
	makeDHMS     = rx(`^BRelTime\.make\(([0-9]+),([0-9]+),([0-9]+),([0-9]+)\)$`)
	makeUnit     = rx(`^BRelTime\.make(Seconds|Minutes|Hours|Days)\(([0-9]+)\)$`)
	unitMs       = map[string]float64{"Seconds": 1000, "Minutes": 60000, "Hours": 3600000, "Days": 86400000}
	constMs      = map[string]float64{"BRelTime.DEFAULT": 0, "BRelTime.SECOND": 1000, "BRelTime.MINUTE": 60000, "BRelTime.HOUR": 3600000, "BRelTime.DAY": 86400000}
	setpointName = rx(`(Setpoint|Set)$`)

	// CS2
	zeroDouble = rx(`BDouble\.make\([[:space:]]*0[.0]*[[:space:]]*\)`)
	minFacet   = rx(`BFacets\.MIN[^)]*BDouble\.make\([[:space:]]*([0-9]+\.?[0-9]*)[[:space:]]*\)`)
	minArg     = rx(`BDouble\.make\([[:space:]]*[0-9]`)
	closeRest  = rx(`\).*`)

	// CS3
	leadSpace  = rx(`^[[:space:]]+`)
	onlyPhrase = rx(`[Oo]nly (when|valid|if)`)

	// CS4
	numPrefix   = rx(`^[[:space:]]*(BInteger|BDouble|BFloat|BLong)\.make\([[:space:]]*`)
	numSuffix   = rx(`[[:space:]]*\)[[:space:]]*$`)
	numType     = rx(`[dDfFlL]$`)
	numLiteral  = rx(`^-?[0-9]+(\.[0-9]*)?$`)
	floorName   = rx(`(^min|Min)(Stages)?On([A-Z0-9]|$)`)
	cutoutName  = rx(`(^lowLimit|LowLimit|^cutout|Cutout)([A-Z0-9]|$)`)
	timerSuffix = rx(`(Time|Delay|Secs?|Seconds|Ms|Millis|Mins?|Minutes)[0-9]*$`)
	// a cutout status / counter slot is not a cutout floor (cutoutCount, cutoutActive, lowLimitReached);
	// Trip / Alarm / Fault are NOT here: they also name cutout setpoints (lpCutoutTrip, lowLimitAlarm)
	statusSuffix = rx(`(Count|Counter|Cnt|Total|Active|Reached|Tripped|State|Status|Flag|Event|Events|Log)[0-9]*$`)
)

// Check never fails: a file it cannot read is the runner's "cannot scan" error.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	var out []lint.Finding
	out = append(out, intervalPass(f.Raw)...)
	out = append(out, setpointPass(f.Raw)...)
	out = append(out, commentPass(f.Raw)...)
	out = append(out, floorPass(f.Raw)...)
	return out, nil
}

// attr is the first group of re's leftmost match in s, or "" when there is none.
func attr(re *regexp.Regexp, s string) (string, bool) {
	m := re.FindStringSubmatch(s)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// slot is one *Interval or *Duration default, the last one declared under its name.
type slot struct {
	ms   float64
	line int
}

// intervalPass is CS1: an *Interval default <= a *Duration default of the same file FAILs (unless both
// are 0), and a BRelTime default it cannot read on such a slot WARNs.
func intervalPass(raw []string) []lint.Finding {
	var out []lint.Finding
	var inames, dnames []string // in the order gawk first inserted them
	ivals, dvals := map[string]slot{}, map[string]slot{}
	put := func(keys *[]string, m map[string]slot, name string, s slot) {
		if _, ok := m[name]; !ok {
			*keys = append(*keys, name)
		}
		m[name] = s
	}
	javasrc.NiagaraProperties(raw, func(text string, line int) {
		pname, _ := attr(nameAttr, text)
		dexpr, _ := attr(defaultAttr, text)
		dexpr = spaceRun.ReplaceAllLiteralString(dexpr, "")
		ms := -1.0
		if strings.Contains(dexpr, "BRelTime") {
			ms = reltimeMs(oneFactory(dexpr))
		}
		switch {
		case pname != "" && ms >= 0:
			if strings.Contains(pname, "Interval") {
				put(&inames, ivals, pname, slot{ms, line})
			}
			if strings.Contains(pname, "Duration") {
				put(&dnames, dvals, pname, slot{ms, line})
			}
		case pname != "" && strings.Contains(dexpr, "BRelTime") &&
			(strings.Contains(pname, "Interval") || strings.Contains(pname, "Duration")):
			out = append(out, lint.Finding{Severity: lint.Warn, Line: line, Msg: "CS1: unreadable BRelTime default for " +
				pname + " (" + dexpr + ") -- interval/duration not compared; use a literal factory"})
		}
	})
	dorder := awkorder.Order(dnames)
	for _, in := range awkorder.Order(inames) {
		iv := ivals[in]
		for _, dn := range dorder {
			dv := dvals[dn]
			if iv.ms == 0 && dv.ms == 0 {
				continue // a DEFAULT pair on a display mirror is unset, not a contradiction
			}
			if iv.ms <= dv.ms {
				out = append(out, lint.Finding{Line: iv.line, Msg: fmt.Sprintf("CS1: interval<=duration: %s(%s) <= %s(%s)",
					in, human(iv.ms), dn, human(dv.ms))})
			}
		}
	}
	return out
}

// oneFactory is the single BRelTime factory call or constant inside e (a wrapper such as
// make(BRelTime.class, BRelTime.makeSeconds(300)) is allowed), or "" when there is none or more than one.
func oneFactory(e string) string {
	all := factory.FindAllString(e, 2)
	if len(all) != 1 {
		return ""
	}
	return all[0]
}

// reltimeMs is e in milliseconds, or -1 when e is not one literal BRelTime factory or constant.
func reltimeMs(e string) float64 {
	if m := makeMs.FindStringSubmatch(e); m != nil {
		return awkNum(m[1])
	}
	if m := makeUnit.FindStringSubmatch(e); m != nil {
		return awkNum(m[2]) * unitMs[m[1]]
	}
	if m := makeDHMS.FindStringSubmatch(e); m != nil {
		d, h, mi, s := awkNum(m[1]), awkNum(m[2]), awkNum(m[3]), awkNum(m[4])
		return ((d*24+h)*60+mi)*60000 + s*1000
	}
	if ms, ok := constMs[e]; ok {
		return ms
	}
	return -1
}

// human is the awk's duration text: whole hours, minutes or seconds, else milliseconds.
func human(ms float64) string {
	switch {
	case math.Mod(ms, 3600000) == 0 && ms > 0:
		return awkStr(ms/3600000) + "h"
	case math.Mod(ms, 60000) == 0 && ms > 0:
		return awkStr(ms/60000) + "m"
	case math.Mod(ms, 1000) == 0:
		return awkStr(ms/1000) + "s"
	}
	return awkStr(ms) + "ms"
}

// awkStr is gawk's number-to-string conversion: an integral value prints as an integer, any other one
// with CONVFMT (%.6g).
func awkStr(v float64) string {
	if v == math.Trunc(v) && !math.IsInf(v, 0) {
		return strconv.FormatFloat(v, 'f', 0, 64)
	}
	return fmt.Sprintf("%.6g", v)
}

// awkNum is gawk's string-to-number conversion of s: the longest decimal prefix strtod reads (digits, an
// optional fraction and exponent), 0 when there is none. Hex is not read (gawk without
// --non-decimal-data), which the prefix rule gives too: "0x5" reads as 0.
func awkNum(s string) float64 {
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	digits := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i, digits = i+1, digits+1
	}
	if i < len(s) && s[i] == '.' {
		i++
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i, digits = i+1, digits+1
		}
	}
	if digits == 0 {
		return 0
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		j := i + 1
		if j < len(s) && (s[j] == '+' || s[j] == '-') {
			j++
		}
		if j < len(s) && s[j] >= '0' && s[j] <= '9' {
			for j < len(s) && s[j] >= '0' && s[j] <= '9' {
				j++
			}
			i = j
		}
	}
	v, _ := strconv.ParseFloat(s[:i], 64) // an out-of-range value is ±Inf, as strtod gives
	return v
}

// setpointPass is CS2: an OPERATOR *Setpoint / *Set slot whose default is BDouble.make(0) with no
// BFacets.MIN of a positive BDouble FAILs.
func setpointPass(raw []string) []lint.Finding {
	var out []lint.Finding
	javasrc.NiagaraProperties(raw, func(text string, line int) {
		pname, _ := attr(nameAttr, text)
		if !setpointName.MatchString(pname) || !strings.Contains(text, "OPERATOR") {
			return
		}
		if zeroDouble.MatchString(text) && !positiveMin(text) {
			out = append(out, lint.Finding{Line: line, Msg: "CS2: zero-default setpoint \"" + pname + "\" with OPERATOR flag and no min>0 facet"})
		}
	})
	return out
}

// positiveMin reads the min facet as the awk did: from the start of the BFacets.MIN ... BDouble.make(n)
// match, the argument of the first BDouble.make( followed by a digit, up to its ")", spaces removed.
func positiveMin(text string) bool {
	loc := minFacet.FindStringIndex(text)
	if loc == nil {
		return false
	}
	seg := text[loc[0]:]
	a := minArg.FindStringIndex(seg)
	arg := strings.TrimPrefix(seg[a[0]:], "BDouble.make(")
	arg = spaceRun.ReplaceAllLiteralString(closeRest.ReplaceAllLiteralString(arg, ""), "")
	return awkNum(arg) > 0
}

// commentPass is CS3: a "//" comment line saying "only when / valid / if" right before a line holding
// @NiagaraProperty WARNs (a heuristic: the rule lives only in the comment).
func commentPass(raw []string) []lint.Finding {
	var out []lint.Finding
	prev := ""
	for i, ln := range raw {
		stripped := ln
		if loc := leadSpace.FindStringIndex(ln); loc != nil {
			stripped = ln[loc[1]:]
		}
		isComment := strings.HasPrefix(stripped, "//") && onlyPhrase.MatchString(stripped)
		switch {
		case strings.Contains(stripped, "@NiagaraProperty") && prev != "":
			out = append(out, lint.Finding{Severity: lint.Warn, Line: i + 1, Msg: "CS3: flag combo enforced only in comment: " + prev})
			prev = ""
		case isComment:
			prev = stripped
		default:
			prev = ""
		}
	}
	return out
}

// floor is one nonzero permanent-minimum default.
type floor struct {
	name, val string
	line      int
}

// floorPass is CS4: every nonzero permanent-minimum floor default beside every LowLimit / Cutout default
// of 0 in the same file WARNs, floor-major. Its accumulator starts with the opening line, so a one-line
// annotation closes on that line.
func floorPass(raw []string) []lint.Finding {
	var floors []floor
	var cutouts []string
	open, buf, line := false, "", 0
	for i, ln := range raw {
		if !open && strings.Contains(ln, "@NiagaraProperty") {
			open, buf, line = true, "", i+1
		}
		if !open {
			continue
		}
		buf += " " + ln
		if javasrc.ParenDepth(buf) > 0 {
			continue
		}
		open = false
		pname, _ := attr(nameAttr, buf)
		dv := ""
		if d, ok := attr(defaultAttr, buf); ok {
			dv = num(d)
		}
		if pname == "" || dv == "" {
			continue
		}
		// camelCase word boundaries, not substrings: adminOnline, minOnTime and cutoutDelay are not
		// matched; lpCutoutPsi and lowLimitBar are; a non-numeric default never reaches here
		if floorName.MatchString(pname) && !timerSuffix.MatchString(pname) &&
			!strings.Contains(buf, "BRelTime") && awkNum(dv) != 0 {
			floors = append(floors, floor{pname, dv, line})
		}
		if cutoutName.MatchString(pname) && !timerSuffix.MatchString(pname) &&
			!statusSuffix.MatchString(pname) && awkNum(dv) == 0 {
			cutouts = append(cutouts, pname)
		}
	}
	var out []lint.Finding
	for _, fl := range floors {
		for _, c := range cutouts {
			out = append(out, lint.Finding{Severity: lint.Warn, Line: fl.line, Msg: "CS4: floor \"" + fl.name + "\"=" + fl.val +
				" with LP cutout \"" + c + "\"=0 (disabled) -- a permanent minimum needs an enabled cutout that overrides it"})
		}
	}
	return out
}

// num is the numeric literal inside a defaultValue string (BInteger/BDouble/BFloat/BLong.make(...) or a
// bare literal, a type suffix dropped), or "" when it is not numeric.
func num(s string) string {
	s = numPrefix.ReplaceAllLiteralString(s, "")
	if loc := numSuffix.FindStringIndex(s); loc != nil {
		s = s[:loc[0]] + s[loc[1]:]
	}
	if loc := numType.FindStringIndex(s); loc != nil {
		s = s[:loc[0]]
	}
	if numLiteral.MatchString(s) {
		return s
	}
	return ""
}
