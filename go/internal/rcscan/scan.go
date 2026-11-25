package rcscan

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// source is one scanned file: its row path (relative to the artifact dir), its rc-relative path (rcRel of the
// path below the artifact dir), its path as walked (rooted at the artifact dir argument), extension and awk
// records. The module-map checks anchor on rc, never on the walked path: the directories above the artifact dir
// are not the module's layout.
type source struct {
	rel, rc, path, ext string
	lines              []string
}

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
	return append(s.rows, staticRows(src)...)
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
		// advisory (audit A7 D2): WARN under every mode, --strict included, until real trees confirm it
		// Mutation: RC-status-advisory -- the --strict severity turns an unconfirmed advisory check into a FAIL
		if strings.Contains(asciiLower(line), "ok") && (strings.Contains(line, "||") || strings.Contains(line, "??")) &&
			!s.allowed("status-default-ok") && s.statusDefaultOK() {
			s.emit(lint.Warn, "status-default-ok", statusOKMsg)
		}
	}
	if s.gated && !s.ctxCSS && !s.allowed("disabled-gate") && disables(line) && !s.statedReason() {
		s.emit(s.o.sevW(), "disabled-gate", gateMsg)
	}
	s.prev = [3]string{line, s.prev[0], s.prev[1]}
}

func (s *fileScan) emitAt(nr int, sev lint.Severity, id, msg string) {
	s.rows = append(s.rows, lint.Finding{Severity: sev, Path: s.src.rel, Line: nr, Msg: id + ": " + msg})
}

func (s *fileScan) emit(sev lint.Severity, id, msg string) { s.emitAt(s.nr, sev, id, msg) }

// allowed reports a trailing `rc-scan: allow <id>` marker on the line.
func (s *fileScan) allowed(id string) bool { return allowedOn(s.line, id) }

// allowedOn reports an `rc-scan: allow <id>` marker on line.
func allowedOn(line, id string) bool { return strings.Contains(line, "rc-scan: allow "+id) }
