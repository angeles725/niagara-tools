package driverscaffold

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

var (
	// call is the leftmost identifier( of a header: the method name.
	call    = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\(`)
	keyword = regexp.MustCompile(`^(if|for|while|switch|catch|try|else|do|synchronized|return|new)$`)
	// newType is a header that ends in `new `, `new pkg.` or `new Outer . `: the identifier( after it is
	// the type of an anonymous class, never a method (issue #282 item 3).
	newType = regexp.MustCompile(`(^|[^A-Za-z0-9_])new[[:space:]]+([A-Za-z_][A-Za-z0-9_]*[[:space:]]*\.[[:space:]]*)*$`)
	factory = regexp.MustCompile(`MessageFactory[^/]*\.java$`)
)

// scanner reads a file's blanked code line by line, char by char. A method body is the text between a
// member header's "{" and its matching "}", at any class depth. The header is the code since the last
// ";", "{" or "}" outside a method, with every annotation (its name and its balanced argument list, on
// one line or several) replaced by one blank, so the row line is the line of the header's first "(".
type scanner struct {
	path     string
	out      []lint.Finding
	depth    int                    // braces open
	mdep     int                    // depth of the method being read (0 outside one)
	ann      javasrc.AnnotationSkip // the header annotation being skipped
	negative bool                   // a "}" closed more than was open
	hdr      strings.Builder
	pline    int // line of the header's first "(" (0 = none yet)
	// the method being read
	name          string
	hline         int
	body, rawbody strings.Builder
}

func (s *scanner) row(sev lint.Severity, line int, msg string) {
	s.out = append(s.out, lint.Finding{Severity: sev, Line: line, Msg: msg})
}

func (s *scanner) resetHeader() {
	s.hdr.Reset()
	s.pline = 0
}

// line scans one line: raw is the source line, code the same line blanked.
func (s *scanner) line(n int, raw, code string) {
	if s.mdep > 0 {
		s.rawbody.WriteString("\n" + raw)
	}
	for k := 0; k < len(code); k++ {
		c := code[k]
		if s.mdep == 0 { // an annotation in a header (never in a body) is one blank
			if n, ended := s.ann.Step(code, k); n > 0 {
				if ended {
					s.hdr.WriteByte(' ')
				}
				k += n - 1
				continue
			}
		}
		switch {
		case s.mdep > 0:
			if c == '{' {
				s.depth++
			} else if c == '}' {
				if s.depth--; s.depth < s.mdep {
					s.finish()
					s.mdep = 0
					s.resetHeader()
					continue
				}
			}
			s.body.WriteByte(c)
		case c == '{':
			s.depth++
			if name := head(s.hdr.String()); name != "" && s.depth >= 2 {
				s.mdep, s.name, s.hline = s.depth, name, s.pline
				s.body.Reset()
				s.rawbody.Reset()
				s.rawbody.WriteString(raw)
			}
			s.resetHeader()
		case c == '}' || c == ';':
			if c == '}' {
				if s.depth--; s.depth < 0 {
					s.depth, s.negative = 0, true
				}
			}
			s.resetHeader()
		default:
			s.hdr.WriteByte(c)
			if c == '(' && s.pline == 0 {
				s.pline = n
			}
		}
	}
	if s.mdep > 0 {
		s.body.WriteByte(' ')
	} else {
		s.hdr.WriteByte(' ')
	}
}

// head is the method name of a member header, or "" for a class, an initializer, a control block or an
// anonymous class body (`new X() {`, `new pkg.X() {`).
func head(h string) string {
	loc := call.FindStringIndex(h)
	if loc == nil {
		return ""
	}
	name := strings.TrimRight(h[loc[0]:loc[1]-1], space)
	if keyword.MatchString(name) || newType.MatchString(h[:loc[0]]) {
		return ""
	}
	return name
}

// finish judges the method body just closed.
func (s *scanner) finish() {
	var b strings.Builder
	for _, c := range []byte(s.body.String()) {
		if strings.IndexByte(space, c) < 0 {
			b.WriteByte(c)
		}
	}
	norm := b.String()
	if s.name == "doPing" {
		if norm == "pingOk();" || norm == "this.pingOk();" {
			s.row(lint.Fail, s.hline, "doPing() only calls pingOk(): every device reports UP (wizard stub, silent success)")
		} else if strings.Contains(s.rawbody.String(), "TODO - add ping implementation") {
			s.row(lint.Fail, s.hline, "doPing() still holds the wizard TODO: replace it with a real reachability check")
		}
	}
	if norm == "returnnull;" && factory.MatchString(s.path) {
		s.row(lint.Warn, s.hline, s.name+"() only returns null (wizard message factory)")
	}
	if s.name == "doSubmitDiscoveryJob" && strings.Contains(norm, ".submit(null)") {
		s.row(lint.Warn, s.hline, "doSubmitDiscoveryJob() submits with a null Context: use job.submit(cx)")
	}
}
