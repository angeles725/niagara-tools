package rcscan

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// source is one scanned file: its row path (relative to the artifact dir), extension and awk records.
type source struct {
	rel, ext string
	lines    []string
}

// re compiles an awk ERE with gawk's leftmost-longest match, so a match's span (RSTART, RLENGTH) is gawk's.
func re(expr string) *regexp.Regexp {
	r := regexp.MustCompile(expr)
	r.Longest()
	return r
}

var (
	commentOnly = re(`^[[:space:]]*(//|/\*|\*+|<!--)`)

	ordLiteral = re("(station:|local:|slot:/)|(^|[|\"'`])h:[0-9a-fA-F]+")
	httpURL    = re(`http://[^ ` + "\t" + `"',;)>]+`)
	ipv4       = re(`[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+`)
	bareCatch  = re(`catch[[:space:]]*\([[:space:]]*\(\)[[:space:]]*=>[[:space:]]*\{[[:space:]]*\}[[:space:]]*\)`)
	nullBranch = re(`\?[[:space:]]*null[[:space:]]*:`)

	scriptOpen  = re(`<script([[:space:]][^>]*)?>`)
	styleOpen   = re(`<style([[:space:]][^>]*)?>`)
	srcAttr     = re(`[[:space:]]src[[:space:]]*=`)
	scriptClose = re(`</script>`)
	styleClose  = re(`</style>`)
	blockClose  = re(`</(script|style)>`)

	dataURI   = re(`data:[^"')[:space:]]+`)
	styleAttr = re(`style[[:space:]]*=[[:space:]]*("[^"]*"|'[^']*')`)
	gapDecl   = re(`(^|[;{[:space:]"'])(row-|column-)?gap[[:space:]]*:`)
	gridDecl  = re(`display[[:space:]]*:[[:space:]]*(inline-)?grid`)
	insetDecl = re(`(^|[;{[:space:]"'])inset[[:space:]]*:`)
	aspect    = re(`aspect-ratio[[:space:]]*:`)
	minCall   = re(`(^|[^A-Za-z0-9_-])min\(`)
	maxCall   = re(`(^|[^A-Za-z0-9_-])max\(`)
	clampCall = re(`(^|[^A-Za-z0-9_-])clamp\(`)
	topAwait  = re(`^await[[:space:]]`)

	loginWord  = re(`login|step-?up`)
	authWord   = re(`[Aa][Uu][Tt][Hh]`)
	asyncFunc  = re(`async[[:space:]]+function[[:space:]]*\*?[[:space:]]*[A-Za-z_$][A-Za-z0-9_$]*`)
	asyncFuncP = re(`^async[[:space:]]+function[[:space:]]*\*?[[:space:]]*`)
	asyncConst = re(`(const|let|var)[[:space:]]+[A-Za-z_$][A-Za-z0-9_$]*[[:space:]]*=[[:space:]]*async([[:space:](]|$)`)
	declP      = re(`^(const|let|var)[[:space:]]+`)
	assignS    = re(`[[:space:]]*=.*$`)

	fetchCall     = re(`(^|[^A-Za-z0-9_$])fetch[[:space:]]*\(`)
	intervalAsync = re(`setInterval\([[:space:]]*async([[:space:](]|$)`)
	intervalName  = re(`setInterval\([[:space:]]*[A-Za-z_$][A-Za-z0-9_$]*[[:space:]]*[,)]`)
	intervalP     = re(`^setInterval\([[:space:]]*`)
	intervalS     = re(`[[:space:]]*[,)]$`)
	htmlAssign    = re(`(inner|outer)HTML[[:space:]]*\+?=([^=]|$)|insertAdjacentHTML[[:space:]]*\(`)
	memberConcat  = re(`\+[[:space:]]*[A-Za-z_$][A-Za-z0-9_$]*\.[A-Za-z_$]|\$\{[[:space:]]*[A-Za-z_$][A-Za-z0-9_$]*\.[A-Za-z_$]`)
	escCall       = re(`esc[A-Za-z]*\(`)
	disabledSet   = re(`\.disabled[[:space:]]*=([^=]|$)`)
	disabledRHS   = re(`^=?[[:space:]]*`)
	notDisabled   = re(`^(false|0)([^A-Za-z0-9_$]|$)`)
	disabledAttr  = re(`setAttribute\([[:space:]]*["']disabled["']`)
	disabledTag   = re(`<(button|input|select|textarea)[^>]*[[:space:]]disabled([[:space:]=>/]|$)`)
)

const (
	fetchMsg = "fetch( with no signal/timeout -- route it through the fetchT helper (AbortController + timeout)"
	gateMsg  = "control disabled in a login-gated file -- a disabled control fires no click, so the gate never opens; mark it (class / aria-disabled) and route the click to the gate"
	pollMsg  = " -- self-schedule with setTimeout after the poll settles (+ inFlight guard)"
)

// dataBudget is the largest data: URI (in chars) an rc file may inline.
const dataBudget = 20480

// fileScan is the per-file state of the awk program.
type fileScan struct {
	src  *source
	o    options
	rows []lint.Finding
	nr   int    // FNR of the line being read
	line string // $0

	inScript, inStyle bool   // inside an inline <script> / <style> block of an .html file
	bstart            int    // the line that opened it
	blkLine           string // that opening line (its allow marker)
	ctxCSS, ctxJS     bool   // the line is CSS / JavaScript

	blk   string // the CSS rule being read (flex gap is decided per rule)
	gapNR int    // the line of its first gap declaration, 0 for none

	gated   bool            // the file mentions a login / auth / step-up gate (any line, comments too)
	isAsync map[string]bool // the async function and async const names of the file

	fPend, fSig       bool // a fetch( whose argument list is being read, and whether it carries a signal
	fNR, fDepth, fLns int  // its line, paren depth and lines read
	prev              [3]string
}

// scanFile runs the per-file checks over src and returns its rows in report order.
func scanFile(src *source, o options) []lint.Finding {
	s := &fileScan{src: src, o: o, isAsync: map[string]bool{}}
	for _, line := range src.lines { // pass 1: file-wide facts
		s.gated = s.gated || mentionsGate(line)
		for t := line; ; {
			m := asyncFunc.FindStringIndex(t)
			if m == nil {
				break
			}
			s.isAsync[asyncFuncP.ReplaceAllString(t[m[0]:m[1]], "")] = true
			t = t[m[1]:]
		}
		if m := asyncConst.FindString(line); m != "" {
			s.isAsync[assignS.ReplaceAllString(declP.ReplaceAllString(m, ""), "")] = true
		}
	}
	for i, line := range src.lines {
		s.nr, s.line = i+1, line
		s.blocks() // before the comment skip, so every line counts
		if commentOnly.MatchString(line) {
			continue
		}
		s.checks()
		s.legacy()
	}
	if s.fPend {
		s.fetchResolve()
	}
	if s.gapNR > 0 {
		s.cssClose()
	}
	return s.rows
}

// blocks tracks the inline <script> (no src) and <style> blocks of an .html file, reports one over 300
// lines at its opening line, and sets the line's CSS / JavaScript context.
func (s *fileScan) blocks() {
	line := s.line
	wasScript, wasStyle, opened := s.inScript, s.inStyle, false
	if s.src.ext == "html" {
		free := !s.inScript && !s.inStyle
		if m := scriptOpen.FindStringIndex(line); free && m != nil {
			src := srcAttr.MatchString(line[m[0]:m[1]])
			if !src && !scriptClose.MatchString(line[m[1]:]) {
				s.inScript, s.bstart, opened = true, s.nr, true
			}
			wasScript = wasScript || !src
		} else if m := styleOpen.FindStringIndex(line); free && m != nil {
			if !styleClose.MatchString(line[m[1]:]) {
				s.inStyle, s.bstart, opened = true, s.nr, true
			}
			wasStyle = true
		}
		if !opened && (s.inScript || s.inStyle) && blockClose.MatchString(line) {
			if n := s.nr - s.bstart - 1; n > 300 && !strings.Contains(s.blkLine, "rc-scan: allow inline-block-size") {
				kind := "style"
				if s.inScript {
					kind = "script"
				}
				s.emitAt(s.bstart, s.o.sevW(), "inline-block-size", fmt.Sprintf("inline <%s> block of %d lines (> 300) -- split per DJS1 into rc/js or rc/css files", kind, n))
			}
			s.inScript, s.inStyle = false, false
		}
		if opened {
			s.blkLine = line
		}
	}
	s.ctxCSS = s.src.ext == "css" || wasStyle
	s.ctxJS = s.src.ext == "js" || wasScript
}

// checks is the WU3 checks of a non-comment line, in the order they report.
func (s *fileScan) checks() {
	line := s.line
	for rest := line; ; {
		m := dataURI.FindStringIndex(rest)
		if m == nil {
			break
		}
		if n := m[1] - m[0]; n > dataBudget {
			if !s.allowed("datauri-budget") {
				s.emit(s.o.sevData(), "datauri-budget", fmt.Sprintf("data: URI of %d chars (> 20480) -- ship it as a separate rc/img file with ?v=", n))
			}
			break
		}
		rest = rest[m[1]:]
	}
	if s.ctxCSS {
		s.floorRow(cssTokens(line))
		s.cssFeed(line)
	} else if s.src.ext == "html" && !s.ctxJS { // inline style="" attributes on markup lines
		f := ""
		for _, sv := range styleAttr.FindAllString(line, -1) {
			f += cssTokens(sv)
			if gapDecl.MatchString(sv) && !gridDecl.MatchString(sv) && !strings.Contains(f, " gap") {
				f += " gap"
			}
		}
		s.floorRow(f)
	}
	if s.ctxJS {
		s.floorRow(s.jsTokens(line))
		s.fetch()
		s.interval()
		if htmlAssign.MatchString(line) && memberConcat.MatchString(line) && !escCall.MatchString(line) && !s.allowed("innerhtml-server") {
			s.emit(s.o.sevW(), "innerhtml-server", "member value concatenated into HTML without esc( -- use textContent or esc()")
		}
	}
	if s.gated && !s.ctxCSS && !s.allowed("disabled-gate") && disables(line) {
		s.emit(s.o.sevW(), "disabled-gate", gateMsg)
	}
	s.prev = [3]string{line, s.prev[0], s.prev[1]}
}

// mentionsGate reports a login / auth / step-up gate mention: `auth` at a word or camelCase start (auth,
// Auth, authToken, getAuth) or after the gate prefixes o / re / un (OAuth, reauth, unauthorized), and not
// author / authority (authorize / authorise still count).
func mentionsGate(line string) bool {
	if loginWord.MatchString(asciiLower(line)) {
		return true
	}
	for t := line; ; {
		m := authWord.FindStringIndex(t)
		if m == nil {
			return false
		}
		i := m[0]
		var pre byte
		pre2 := ""
		if i > 0 {
			pre = t[i-1]
		}
		if i > 1 {
			pre2 = asciiLower(t[i-2 : i])
		}
		nxt := asciiLower(t[i+4 : min(i+8, len(t))])
		upper := t[i] == 'A'
		wordStart := !alnum(pre) || upper && (lowerOrDigit(pre))
		prefix := pre == 'o' || pre == 'O' || pre2 == "re" || pre2 == "un"
		if (wordStart || prefix) && (!strings.HasPrefix(nxt, "or") || strings.HasPrefix(nxt, "oris") || strings.HasPrefix(nxt, "oriz")) {
			return true
		}
		t = t[i+4:]
	}
}

func alnum(c byte) bool        { return lowerOrDigit(c) || 'A' <= c && c <= 'Z' }
func lowerOrDigit(c byte) bool { return 'a' <= c && c <= 'z' || '0' <= c && c <= '9' }

// asciiLower is gawk's tolower under LC_ALL=C: ASCII letters only.
func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}

// fetch reads a fetch( call's argument list (up to 15 lines) for a signal; the 3 lines before it count
// too (the fetchT helper idiom).
func (s *fileScan) fetch() {
	if s.fPend {
		s.fetchFeed(s.line)
		return
	}
	if m := fetchCall.FindStringIndex(s.line); m != nil && !s.allowed("fetch-no-signal") {
		s.fPend, s.fNR, s.fDepth, s.fLns = true, s.nr, 0, 0
		s.fSig = strings.Contains(s.prev[0]+"\n"+s.prev[1]+"\n"+s.prev[2], "signal")
		s.fetchFeed(s.line[m[0]:])
	}
}

func (s *fileScan) fetchFeed(t string) {
	s.fSig = s.fSig || strings.Contains(t, "signal")
	s.fDepth += strings.Count(t, "(") - strings.Count(t, ")")
	s.fLns++
	if s.fDepth <= 0 || s.fLns >= 15 {
		s.fetchResolve()
	}
}

func (s *fileScan) fetchResolve() {
	if !s.fSig {
		s.emitAt(s.fNR, s.o.sevW(), "fetch-no-signal", fetchMsg)
	}
	s.fPend = false
}

// interval reports setInterval over an inline async function or a name the file declares async.
func (s *fileScan) interval() {
	if s.allowed("setinterval-async") {
		return
	}
	if intervalAsync.MatchString(s.line) {
		s.emit(s.o.sevW(), "setinterval-async", "setInterval over an async function"+pollMsg)
	} else if m := intervalName.FindString(s.line); m != "" {
		if nm := intervalS.ReplaceAllString(intervalP.ReplaceAllString(m, ""), ""); s.isAsync[nm] {
			s.emit(s.o.sevW(), "setinterval-async", "setInterval("+nm+") where "+nm+" is async"+pollMsg)
		}
	}
}

// disables reports a `.disabled = <not false>` assignment, a setAttribute("disabled" call or a disabled
// attribute on a form control.
func disables(line string) bool {
	if m := disabledSet.FindStringIndex(line); m != nil {
		if !notDisabled.MatchString(disabledRHS.ReplaceAllString(line[m[1]-1:], "")) {
			return true
		}
	}
	return disabledAttr.MatchString(line) || disabledTag.MatchString(line)
}

// floorRow reports the features f (each with a leading space) above the Chromium 83 panel floor.
func (s *fileScan) floorRow(f string) {
	if f != "" && !s.allowed("browser-floor") {
		s.emit(s.o.sevFloor(), "browser-floor", f[1:]+" above the Chromium 83 panel floor")
	}
}

func cssTokens(t string) string {
	f := ""
	for _, c := range []struct {
		hit  bool
		name string
	}{{insetDecl.MatchString(t), "inset"}, {aspect.MatchString(t), "aspect-ratio"}, {minCall.MatchString(t), "min()"},
		{maxCall.MatchString(t), "max()"}, {clampCall.MatchString(t), "clamp()"}, {strings.Contains(t, "backdrop-filter"), "backdrop-filter"}} {
		if c.hit {
			f += " " + c.name
		}
	}
	return f
}

func (s *fileScan) jsTokens(t string) string {
	f := ""
	for _, c := range [][2]string{{"??=", "??="}, {"||=", "||="}, {"&&=", "&&="}, {".replaceAll(", ".replaceAll()"},
		{".at(", ".at()"}, {"structuredClone(", "structuredClone()"}} {
		if strings.Contains(t, c[0]) {
			f += " " + c[1]
		}
	}
	if s.src.ext == "js" && topAwait.MatchString(t) {
		f += " top-level-await"
	}
	return f
}

// cssFeed reads a CSS line rule by rule: a { starts a rule, a } closes it.
func (s *fileScan) cssFeed(text string) {
	for {
		i := strings.IndexAny(text, "{}")
		if i < 0 {
			break
		}
		s.cssSeg(text[:i])
		if text[i] == '{' {
			s.blk, s.gapNR = "", 0
		} else {
			s.cssClose()
		}
		text = text[i+1:]
	}
	s.cssSeg(text)
}

func (s *fileScan) cssSeg(seg string) {
	s.blk += " " + seg
	if s.gapNR == 0 && gapDecl.MatchString(seg) && !s.allowed("browser-floor") {
		s.gapNR = s.nr
	}
}

// cssClose ends a rule: a gap in a rule that declares no display:grid is a flex gap (Chromium 84).
func (s *fileScan) cssClose() {
	if s.gapNR > 0 && !gridDecl.MatchString(s.blk) {
		s.emitAt(s.gapNR, s.o.sevFloor(), "browser-floor", "gap in a non-grid (flex) rule is above the Chromium 83 panel floor (flex gap = C84)")
	}
	s.blk, s.gapNR = "", 0
}

func (s *fileScan) emitAt(nr int, sev lint.Severity, id, msg string) {
	s.rows = append(s.rows, lint.Finding{Severity: sev, Path: s.src.rel, Line: nr, Msg: id + ": " + msg})
}

func (s *fileScan) emit(sev lint.Severity, id, msg string) { s.emitAt(s.nr, sev, id, msg) }

// allowed reports a trailing `rc-scan: allow <id>` marker on the line.
func (s *fileScan) allowed(id string) bool { return strings.Contains(s.line, "rc-scan: allow "+id) }

// legacy is the campaign-8 checks: the first one that matches reports the line (awk `next`).
func (s *fileScan) legacy() {
	line := s.line
	if ordLiteral.MatchString(line) {
		s.emit(lint.Fail, "ord-literal", "hardcoded ORD literal")
		return
	}
	hasHTTP := strings.Contains(line, "http://")
	for _, m := range httpURL.FindAllString(line, -1) {
		if !strings.HasPrefix(m, "http://www.w3.org/") {
			s.emit(lint.Fail, "host-literal", m)
			return
		}
	}
	switch {
	case ipv4.MatchString(line) && !hasHTTP:
		s.emit(lint.Fail, "host-literal", "bare IPv4 literal")
	case bareCatch.MatchString(line):
		s.emit(s.o.sevW(), "bare-catch", "bare .catch(()=>{}) swallows errors")
	case nullBranch.MatchString(line):
		s.emit(s.o.sevW(), "null-branch", "? null : branch may blank UI on null field")
	}
}
