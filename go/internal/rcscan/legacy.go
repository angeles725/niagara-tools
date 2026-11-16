package rcscan

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

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
