package rcscan

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// source is one scanned file: its row path (relative to the artifact dir), its path as walked (rooted at the
// artifact dir argument), extension and awk records. The module-map checks anchor on rcRel(rel), never on the
// walked path: the directories above the artifact dir are not the module's layout.
type source struct {
	rel, path, ext string
	lines          []string
}

// pat is an awk ERE compiled with gawk's leftmost-longest match, so a match's span (RSTART, RLENGTH) is gawk's,
// plus literals one of which every match contains: a line without it never reaches the regexp engine, whose linear NFA is
// still slow on the 100-KB data-URI and minified lines of a real page.
type pat struct {
	lits  []string       // one of them is in every match; none: no pre-check
	r, lr *regexp.Regexp // leftmost-first for a yes/no match (it stops at the first), leftmost-longest for a span
}

func re(expr string, lits ...string) *pat {
	lr := regexp.MustCompile(expr)
	lr.Longest()
	return &pat{lits, regexp.MustCompile(expr), lr}
}

func (p *pat) on(s string) bool {
	for _, l := range p.lits {
		if strings.Contains(s, l) {
			return true
		}
	}
	return len(p.lits) == 0
}

func (p *pat) MatchString(s string) bool { return p.on(s) && p.r.MatchString(s) }

// The span methods below use lr.

func (p *pat) FindString(s string) string {
	if !p.on(s) {
		return ""
	}
	return p.lr.FindString(s)
}

func (p *pat) FindStringIndex(s string) []int {
	if !p.on(s) {
		return nil
	}
	return p.lr.FindStringIndex(s)
}

func (p *pat) FindAllString(s string) []string {
	if !p.on(s) {
		return nil
	}
	return p.lr.FindAllString(s, -1)
}

func (p *pat) ReplaceAllString(s, with string) string {
	if !p.on(s) {
		return s
	}
	return p.lr.ReplaceAllString(s, with)
}

var (
	commentOnly = re(`^[[:space:]]*(//|/\*|\*+|<!--)`)

	httpURL    = re(`http://[^ `+"\t"+`"',;)>]+`, "http://")
	bareCatch  = re(`catch[[:space:]]*\([[:space:]]*\(\)[[:space:]]*=>[[:space:]]*\{[[:space:]]*\}[[:space:]]*\)`, "catch")
	nullBranch = re(`\?[[:space:]]*null[[:space:]]*:`, "null")

	scriptOpen  = re(`<script([[:space:]][^>]*)?>`, "<script")
	styleOpen   = re(`<style([[:space:]][^>]*)?>`, "<style")
	srcAttr     = re(`[[:space:]]src[[:space:]]*=`, "src")
	scriptClose = re(`</script>`, "</script>")
	styleClose  = re(`</style>`, "</style>")
	blockClose  = re(`</(script|style)>`, "</")

	dataURI   = re(`data:[^"')[:space:]]+`, "data:")
	styleAttr = re(`style[[:space:]]*=[[:space:]]*("[^"]*"|'[^']*')`, "style")
	gapDecl   = re(`(^|[;{[:space:]"'])(row-|column-)?gap[[:space:]]*:`, "gap")
	gridDecl  = re(`display[[:space:]]*:[[:space:]]*(inline-)?grid`, "grid")
	insetDecl = re(`(^|[;{[:space:]"'])inset[[:space:]]*:`, "inset")
	aspect    = re(`aspect-ratio[[:space:]]*:`, "aspect-ratio")
	minCall   = re(`(^|[^A-Za-z0-9_-])min\(`, "min(")
	maxCall   = re(`(^|[^A-Za-z0-9_-])max\(`, "max(")
	clampCall = re(`(^|[^A-Za-z0-9_-])clamp\(`, "clamp(")
	topAwait  = re(`^await[[:space:]]`, "await")

	authWord   = re(`[Aa][Uu][Tt][Hh]`)
	asyncFunc  = re(`async[[:space:]]+function[[:space:]]*\*?[[:space:]]*[A-Za-z_$][A-Za-z0-9_$]*`, "async")
	asyncFuncP = re(`^async[[:space:]]+function[[:space:]]*\*?[[:space:]]*`, "async")
	asyncConst = re(`(const|let|var)[[:space:]]+[A-Za-z_$][A-Za-z0-9_$]*[[:space:]]*=[[:space:]]*async([[:space:](]|$)`, "async")
	declP      = re(`^(const|let|var)[[:space:]]+`)
	assignS    = re(`[[:space:]]*=.*$`, "=")

	fetchCall     = re(`(^|[^A-Za-z0-9_$])fetch[[:space:]]*\(`, "fetch")
	intervalAsync = re(`setInterval\([[:space:]]*async([[:space:](]|$)`, "setInterval(")
	intervalName  = re(`setInterval\([[:space:]]*[A-Za-z_$][A-Za-z0-9_$]*[[:space:]]*[,)]`, "setInterval(")
	intervalP     = re(`^setInterval\([[:space:]]*`, "setInterval(")
	intervalS     = re(`[[:space:]]*[,)]$`)
	htmlAssign    = re(`(inner|outer)HTML[[:space:]]*\+?=([^=]|$)|insertAdjacentHTML[[:space:]]*\(`, "HTML")
	memberConcat  = re(`\+[[:space:]]*[A-Za-z_$][A-Za-z0-9_$]*\.[A-Za-z_$]|\$\{[[:space:]]*[A-Za-z_$][A-Za-z0-9_$]*\.[A-Za-z_$]`, ".")
	escCall       = re(`esc[A-Za-z]*\(`, "esc")
	disabledSet   = re(`\.disabled[[:space:]]*=([^=]|$)`, ".disabled")
	disabledRHS   = re(`^=?[[:space:]]*`)
	notDisabled   = re(`^(false|0)([^A-Za-z0-9_$]|$)`)
	disabledAttr  = re(`setAttribute\([[:space:]]*["']disabled["']`, "setAttribute(")
	disabledTag   = re(`<(button|input|select|textarea)[^>]*[[:space:]]disabled([[:space:]=>/]|$)`, "disabled")
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

	view *jsView // the file's JavaScript tokens, built on first use
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
	s.jsFileSize()
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
		for _, sv := range styleAttr.FindAllString(line) {
			f += cssTokens(sv)
			if gapDecl.MatchString(sv) && !gridDecl.MatchString(sv) && !strings.Contains(f, " gap") {
				f += " gap"
			}
		}
		s.floorRow(f)
	}
	if s.ctxJS {
		s.floorRow(s.jsTokens(line))
		s.fetchOutside()
		s.fetch()
		s.interval()
		if htmlAssign.MatchString(line) && memberConcat.MatchString(line) && !escCall.MatchString(line) && !s.allowed("innerhtml-server") && s.innerHTMLRow() {
			s.emit(s.o.sevW(), "innerhtml-server", "member value concatenated into HTML without esc( -- use textContent or esc()")
		}
	}
	if s.gated && !s.ctxCSS && !s.allowed("disabled-gate") && disables(line) && !s.statedReason() {
		s.emit(s.o.sevW(), "disabled-gate", gateMsg)
	}
	s.prev = [3]string{line, s.prev[0], s.prev[1]}
}

// mentionsGate reports a login / auth / step-up gate mention: `auth` at a word or camelCase start (auth,
// Auth, authToken, getAuth) or after the gate prefixes o / re / un (OAuth, reauth, unauthorized), and not
// author / authority (authorize / authorise still count).
func mentionsGate(line string) bool {
	low := asciiLower(line)
	if strings.Contains(low, "login") || strings.Contains(low, "stepup") || strings.Contains(low, "step-up") {
		return true
	}
	for t := line; strings.Contains(low, "auth"); {
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
	return false
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

// maxJSLines is the js-file-size budget (modularity-gaps Δ3): a .js file over it is split by concern.
const maxJSLines = 400

// jsFileSize (module map modes only) reports a .js file of more than maxJSLines lines at its line 1, unless
// line 1 carries the `rc-scan: allow js-file-size` marker.
func (s *fileScan) jsFileSize() {
	if !s.o.mapped() || thirdParty(rcRel(s.src.rel)) || s.src.ext != "js" || len(s.src.lines) <= maxJSLines ||
		allowedOn(s.src.lines[0], "js-file-size") {
		return
	}
	s.emitAt(1, s.o.sevMap(), "js-file-size", fmt.Sprintf("%d lines (> %d) -- split by concern into services/, store,"+
		" components/ and pages/ files (types/frontend-standard.md § Component layout)", len(s.src.lines), maxJSLines))
}

// fetchOutside (module map modes only) reports a fetch( call in JavaScript (a .js file or an inline <script>)
// outside an rc/js/services/ file: only the services call the network (modularity-gaps Δ2). The exemption is
// anchored at rc/js/services/ below the artifact dir: a services/ segment elsewhere (js/pages/services/,
// rc/services/) or above the module is not it.
func (s *fileScan) fetchOutside() {
	if !s.o.mapped() || thirdParty(rcRel(s.src.rel)) || strings.HasPrefix(rcRel(s.src.rel), "js/services/") ||
		!fetchCall.MatchString(s.line) || s.allowed("fetch-outside-services") {
		return
	}
	s.emit(s.o.sevMap(), "fetch-outside-services", "fetch( outside rc/js/services/ -- call the API through"+
		" js/services/api.js (types/frontend-standard.md § Component layout)")
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
func (s *fileScan) allowed(id string) bool { return allowedOn(s.line, id) }

// allowedOn reports an `rc-scan: allow <id>` marker on line.
func allowedOn(line, id string) bool { return strings.Contains(line, "rc-scan: allow "+id) }

// thirdParty is a file the module does not author and so does not split, given its rc-relative path (rcRel): under
// vendor/ (vendored JS is not even scanned; its HTML is) or ext/ (pruned by the walk), or a *.min.js (excluded by
// the walk). The module-map checks state it themselves so a change to the walk cannot turn them on third-party code.
func thirdParty(rc string) bool {
	return strings.HasPrefix(rc, "vendor/") || strings.HasPrefix(rc, "ext/") || strings.HasSuffix(rc, ".min.js")
}

// rcRel is a row path (relative to the artifact dir) made relative to the module's rc/ directory: the part after
// its first `rc` segment, or the whole path when it has none (the artifact dir argument is the rc dir itself).
// Only the path below the artifact dir is read, so an ancestor directory named rc/vendor or rc/js/services cannot
// exempt or skip the module (review of 64cda653, R4/R3).
func rcRel(rel string) string {
	if strings.HasPrefix(rel, "rc/") {
		return rel[len("rc/"):]
	}
	if i := strings.Index(rel, "/rc/"); i >= 0 {
		return rel[i+len("/rc/"):]
	}
	return rel
}

// ordLiteral is the ERE (station:|local:|slot:/)|(^|[|"'`])h:[0-9a-fA-F]+ (a scheme prefix, or a handle ORD
// at a segment start), matched by hand: the regexp engine is slow on a 100-KB line full of "width:".
func ordLiteral(line string) bool {
	if strings.Contains(line, "station:") || strings.Contains(line, "local:") || strings.Contains(line, "slot:/") {
		return true
	}
	for i := strings.Index(line, "h:"); i >= 0; {
		if (i == 0 || strings.IndexByte("|\"'`", line[i-1]) >= 0) && i+2 < len(line) && hexDigit(line[i+2]) {
			return true
		}
		j := strings.Index(line[i+1:], "h:")
		if j < 0 {
			break
		}
		i += 1 + j
	}
	return false
}

func hexDigit(c byte) bool { return '0' <= c && c <= '9' || 'a' <= c|0x20 && c|0x20 <= 'f' }

// ipv4 is the ERE [0-9]+\.[0-9]+\.[0-9]+\.[0-9]+ matched by hand: some run of digits and dots holds four
// non-empty digit groups joined by single dots.
func ipv4(line string) bool {
	groups := 0
	for i := 0; i <= len(line); i++ {
		switch {
		case i < len(line) && '0' <= line[i] && line[i] <= '9':
			if groups == 0 || i > 0 && line[i-1] == '.' {
				groups++
			}
			if groups == 4 {
				return true
			}
		case i < len(line) && line[i] == '.' && i > 0 && '0' <= line[i-1] && line[i-1] <= '9':
		default:
			groups = 0
		}
	}
	return false
}

// legacy is the campaign-8 checks: the first one that matches reports the line (awk `next`).
func (s *fileScan) legacy() {
	line := s.line
	if ordLiteral(line) {
		s.emit(lint.Fail, "ord-literal", "hardcoded ORD literal")
		return
	}
	hasHTTP := strings.Contains(line, "http://")
	for _, m := range httpURL.FindAllString(line) {
		if !strings.HasPrefix(m, "http://www.w3.org/") {
			s.emit(lint.Fail, "host-literal", m)
			return
		}
	}
	switch {
	case !hasHTTP && ipv4(line):
		s.emit(lint.Fail, "host-literal", "bare IPv4 literal")
	case bareCatch.MatchString(line):
		s.emit(s.o.sevW(), "bare-catch", "bare .catch(()=>{}) swallows errors")
	case nullBranch.MatchString(line):
		s.emit(s.o.sevW(), "null-branch", "? null : branch may blank UI on null field")
	}
}
