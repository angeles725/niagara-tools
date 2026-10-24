package rcscan

import (
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
)

// fileScan is the per-file state of the awk program.
type fileScan struct {
	src  *source
	o    options
	rows []lint.Finding
	nr   int    // FNR of the line being read
	line string // $0
}

// scanFile runs the per-file checks over src and returns its rows in report order.
func scanFile(src *source, o options) []lint.Finding {
	s := &fileScan{src: src, o: o}
	for i, line := range src.lines {
		s.nr, s.line = i+1, line
		if commentOnly.MatchString(line) {
			continue
		}
		s.legacy()
	}
	return s.rows
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
