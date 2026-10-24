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
}

// scanFile runs the per-file checks over src and returns its rows in report order.
func scanFile(src *source, o options) []lint.Finding {
	s := &fileScan{src: src, o: o}
	for i, line := range src.lines {
		s.nr, s.line = i+1, line
		s.blocks() // before the comment skip, so every line counts
		if commentOnly.MatchString(line) {
			continue
		}
		s.checks()
		s.legacy()
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
	}
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
