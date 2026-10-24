// Package jssrc is a minimal, literal-aware JavaScript tokenizer for the lint rules: comments are dropped,
// and "..." / '...' strings, `...` template literals (${...} nesting followed) and /.../ regex literals are
// single tokens, so a // or a quote inside one is never read as code. It is not a parser: a / is a regex
// when the token before it cannot end an expression (an operator, an opening bracket, a keyword such as
// return, or the start of input), else a division. Scanning is byte-wise; non-ASCII bytes are identifier
// characters.
package jssrc

import "strings"

// Kind is a token class.
type Kind int

// The token kinds.
const (
	Ident    Kind = iota // a name or a keyword
	Number               // a numeric literal
	String               // "..." or '...'; Text is the source between the quotes, escapes as written
	Template             // `...`; Text is the source between the backticks, ${...} included
	Regex                // /.../flags; Text is the whole literal
	Punct                // an operator or punctuator
)

func (k Kind) String() string {
	return [...]string{"ident", "number", "string", "template", "regex", "punct"}[k]
}

// Token is one token; Line is the 0-based line it starts on. A Template's Subs are the tokens of each of its
// ${...} substitutions, in order (nested templates carry their own).
type Token struct {
	Kind Kind
	Line int
	Text string
	Subs [][]Token
}

// operators are the multi-byte punctuators, longest first; every other punctuator is one byte.
var operators = []string{">>>=", "...", "===", "!==", "**=", "<<=", ">>=", ">>>", "&&=", "||=", "??=",
	"==", "!=", "=>", "<=", ">=", "&&", "||", "??", "?.", "++", "--", "+=", "-=", "*=", "/=", "%=", "&=",
	"|=", "^=", "**", "<<", ">>"}

// regexAfter are the keywords after which a / starts a regex.
var regexAfter = map[string]bool{"return": true, "typeof": true, "case": true, "do": true, "else": true,
	"in": true, "of": true, "new": true, "delete": true, "void": true, "throw": true, "instanceof": true,
	"yield": true, "await": true}

type scanner struct {
	src   string
	i     int
	line  int
	out   []Token
	blank [][2]int // the comment and literal-content byte spans Code blanks
}

// Tokens tokenizes lines (as javasrc.Lines splits them; a trailing "\r" is whitespace).
func Tokens(lines []string) []Token {
	s := &scanner{src: strings.Join(lines, "\n")}
	s.run(false)
	return s.out
}

// Code is lines with every comment and the contents of every string, template and regex literal replaced by
// blanks (quotes, backticks and slashes kept, line breaks kept), so each line keeps its length and a pattern
// matched on it sees code only: a "//" in a string, a call quoted in a template or a brace in a regex is gone.
func Code(lines []string) []string {
	s := &scanner{src: strings.Join(lines, "\n")}
	s.run(false)
	b := []byte(s.src)
	for _, sp := range s.blank {
		for k := sp[0]; k < sp[1] && k < len(b); k++ {
			if b[k] != '\n' {
				b[k] = ' '
			}
		}
	}
	return strings.Split(string(b), "\n")
}

// run scans tokens until the input ends or, inside a template substitution (sub), its closing }.
func (s *scanner) run(sub bool) {
	depth := 0
	for s.i < len(s.src) {
		c := s.src[s.i]
		start, line := s.i, s.line
		switch {
		case c == '\n':
			s.line++
			s.i++
		case c == ' ' || c == '\t' || c == '\r' || c == '\f' || c == '\v':
			s.i++
		case strings.HasPrefix(s.src[s.i:], "//"):
			for s.i < len(s.src) && s.src[s.i] != '\n' {
				s.i++
			}
			s.blank = append(s.blank, [2]int{start, s.i})
		case strings.HasPrefix(s.src[s.i:], "/*"):
			end := strings.Index(s.src[s.i+2:], "*/")
			if end < 0 {
				end = len(s.src) - s.i - 4
			}
			s.skip(s.i + 2 + end + 2)
			s.blank = append(s.blank, [2]int{start, s.i})
		case c == '"' || c == '\'':
			end := s.str(c)
			s.blank = append(s.blank, [2]int{start + 1, end})
			s.emit(String, line, s.src[start+1:end])
		case c == '`':
			s.i++
			end, subs := s.template()
			s.blank = append(s.blank, [2]int{start + 1, end})
			s.out = append(s.out, Token{Template, line, s.src[start+1 : end], subs})
		case c == '/' && s.regexOK():
			s.regex()
			text := s.src[start:s.i]
			s.blank = append(s.blank, [2]int{start + 1, start + max(strings.LastIndexByte(text, '/'), 1)})
			s.emit(Regex, line, text)
		case ident(c):
			for s.i < len(s.src) && (ident(s.src[s.i]) || digit(s.src[s.i])) {
				s.i++
			}
			s.emit(Ident, line, s.src[start:s.i])
		case digit(c):
			for s.i < len(s.src) && (ident(s.src[s.i]) || digit(s.src[s.i]) || s.src[s.i] == '.') {
				s.i++
			}
			s.emit(Number, line, s.src[start:s.i])
		default:
			if sub && c == '}' && depth == 0 {
				s.i++
				return
			}
			if c == '{' {
				depth++
			} else if c == '}' {
				depth--
			}
			op := string(c)
			for _, o := range operators {
				if strings.HasPrefix(s.src[s.i:], o) {
					op = o
					break
				}
			}
			s.i += len(op)
			s.emit(Punct, line, op)
		}
	}
}

func (s *scanner) emit(k Kind, line int, text string) {
	s.out = append(s.out, Token{Kind: k, Line: line, Text: text})
}

// skip moves to end (clamped), counting the newlines passed.
func (s *scanner) skip(end int) {
	end = min(end, len(s.src))
	s.line += strings.Count(s.src[s.i:end], "\n")
	s.i = end
}

// str reads a quoted string from its opening quote and returns where its content ends. An unescaped line
// end ends it unclosed (the newline is left for run), as does the end of input.
func (s *scanner) str(q byte) int {
	s.i++
	for s.i < len(s.src) {
		switch c := s.src[s.i]; {
		case c == '\\' && s.i+1 < len(s.src):
			s.skip(s.i + 2) // an escaped line end continues the string
		case c == q:
			s.i++
			return s.i - 1
		case c == '\n':
			return s.i
		default:
			s.i++
		}
	}
	return len(s.src)
}

// template reads a template literal body after its opening backtick, through its closing backtick, and
// returns where its content ends and the tokens of each substitution.
func (s *scanner) template() (int, [][]Token) {
	var subs [][]Token
	for s.i < len(s.src) {
		switch c := s.src[s.i]; {
		case c == '\\':
			s.skip(s.i + 2)
		case c == '`':
			s.i++
			return s.i - 1, subs
		case strings.HasPrefix(s.src[s.i:], "${"):
			s.i += 2
			n := len(s.out)
			s.run(true) // the substitution's tokens belong to the template, not the stream
			subs = append(subs, append([]Token(nil), s.out[n:]...))
			s.out = s.out[:n]
		default:
			s.skip(s.i + 1)
		}
	}
	return len(s.src), subs
}

// regex reads a regex literal from its opening slash: classes may hold a /, a line end stops it.
func (s *scanner) regex() {
	s.i++
	class := false
	for s.i < len(s.src) && s.src[s.i] != '\n' {
		c := s.src[s.i]
		s.i++
		switch {
		case c == '\\':
			s.i++
		case c == '[':
			class = true
		case c == ']':
			class = false
		case c == '/' && !class:
			for s.i < len(s.src) && ident(s.src[s.i]) {
				s.i++
			}
			return
		}
	}
	s.i = min(s.i, len(s.src))
}

// regexOK reports that the previous token cannot end an expression, so a / here starts a regex.
func (s *scanner) regexOK() bool {
	if len(s.out) == 0 {
		return true
	}
	t := s.out[len(s.out)-1]
	switch t.Kind {
	case Ident:
		return regexAfter[t.Text]
	case Number, String, Template, Regex:
		return false
	}
	return t.Text != ")" && t.Text != "]" && t.Text != "}" && t.Text != "++" && t.Text != "--"
}

func ident(c byte) bool { return c == '_' || c == '$' || c >= 0x80 || ('a' <= c|0x20 && c|0x20 <= 'z') }
func digit(c byte) bool { return '0' <= c && c <= '9' }
