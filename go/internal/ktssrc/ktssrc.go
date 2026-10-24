// Package ktssrc is the Kotlin source scanner the kit's *.gradle.kts lints share: Tokens turns a Kotlin
// script into identifiers, string literals, char literals, numbers and punctuation, with // and nested
// /* */ comments dropped. A "..." string honours \ escapes, a """...""" raw string none (it closes on the
// last three quotes of a run), and both carry $name and ${...} templates (the code inside a ${...} is read
// to its matching brace, strings and comments in it included, and stays part of the string token). Code renders
// tokens back as code lines, comments dropped and literal contents removed, for a rule that matches line-wise.
// Scanning is byte-wise: every character it tests is ASCII, so UTF-8 text passes through unchanged.
package ktssrc

import (
	"fmt"
	"strings"
)

// Kind is the class of a Token.
type Kind int

const (
	Ident  Kind = iota // a name: letters, digits, _ (or a `backticked` name, quotes dropped)
	String             // a "..." or """...""" literal
	Char               // a '...' literal
	Number             // a numeric literal
	Punct              // any other single byte
)

// Token is one token of a script.
type Token struct {
	Kind     Kind
	Text     string // as written (a String or Char with its quotes)
	Value    string // String: the text between the quotes, escapes and templates as written
	Template bool   // String: it holds a $name or ${...} template
	Line     int    // 1-based line of its first byte
	Off, End int    // byte offsets of its first byte and just past its last (a `name`'s backquotes included)
}

// Error is a script Tokens cannot read: an unterminated string, char, template or block comment.
type Error struct {
	Line int
	Msg  string
}

func (e *Error) Error() string { return fmt.Sprintf("line %d: %s", e.Line, e.Msg) }

// Tokens scans src.
func Tokens(src string) ([]Token, error) {
	s := &scanner{src: src, line: 1}
	return s.tokens(false)
}

type scanner struct {
	src  string
	i    int
	line int
	out  []Token
}

func (s *scanner) fail(line int, msg string) error { return &Error{Line: line, Msg: msg} }

// tokens scans to the end of src or, inTemplate, to the "}" that closes a ${ template.
func (s *scanner) tokens(inTemplate bool) ([]Token, error) {
	depth := 0
	for s.i < len(s.src) {
		c, rest := s.src[s.i], s.src[s.i:]
		start, line, n := s.i, s.line, len(s.out)
		switch {
		case c == '\n':
			s.line++
			s.i++
		case c == ' ' || c == '\t' || c == '\r' || c == '\f':
			s.i++
		case strings.HasPrefix(rest, "//"):
			for s.i < len(s.src) && s.src[s.i] != '\n' {
				s.i++
			}
		case strings.HasPrefix(rest, "/*"):
			if err := s.blockComment(); err != nil {
				return nil, err
			}
		case strings.HasPrefix(rest, `"""`):
			tok, err := s.str(true)
			if err != nil {
				return nil, err
			}
			s.out = append(s.out, tok)
		case c == '"':
			tok, err := s.str(false)
			if err != nil {
				return nil, err
			}
			s.out = append(s.out, tok)
		case c == '\'':
			if err := s.char(); err != nil {
				return nil, err
			}
		case c == '`':
			end := strings.IndexAny(s.src[s.i+1:], "`\n")
			if end < 0 || s.src[s.i+1+end] != '`' {
				return nil, s.fail(line, "unterminated `name`")
			}
			s.out = append(s.out, Token{Kind: Ident, Text: s.src[s.i+1 : s.i+1+end], Line: line})
			s.i += end + 2
		case isIdentStart(c):
			for s.i < len(s.src) && isIdentPart(s.src[s.i]) {
				s.i++
			}
			s.out = append(s.out, Token{Kind: Ident, Text: s.src[start:s.i], Line: line})
		case c >= '0' && c <= '9':
			for s.i < len(s.src) && (isIdentPart(s.src[s.i]) || s.src[s.i] == '.') {
				s.i++
			}
			s.out = append(s.out, Token{Kind: Number, Text: s.src[start:s.i], Line: line})
		default:
			s.i++
			if inTemplate && c == '{' {
				depth++
			} else if inTemplate && c == '}' {
				if depth == 0 {
					return s.out, nil
				}
				depth--
			}
			s.out = append(s.out, Token{Kind: Punct, Text: string(c), Line: line})
		}
		if len(s.out) > n {
			s.out[n].Off, s.out[n].End = start, s.i
		}
	}
	if inTemplate {
		return nil, s.fail(s.line, "unterminated ${ template")
	}
	return s.out, nil
}

// blockComment skips a /* */ comment; Kotlin block comments nest.
func (s *scanner) blockComment() error {
	line, depth := s.line, 0
	for s.i < len(s.src) {
		switch rest := s.src[s.i:]; {
		case strings.HasPrefix(rest, "/*"):
			depth++
			s.i += 2
		case strings.HasPrefix(rest, "*/"):
			s.i += 2
			if depth--; depth == 0 {
				return nil
			}
		default:
			if s.src[s.i] == '\n' {
				s.line++
			}
			s.i++
		}
	}
	return s.fail(line, "unterminated /* comment")
}

// str reads a "..." (or, raw, a """...""") literal at s.i, templates included.
func (s *scanner) str(raw bool) (Token, error) {
	start, line, q := s.i, s.line, 1
	if raw {
		q = 3
	}
	s.i += q
	tok := Token{Kind: String, Line: line}
	from := s.i
	for s.i < len(s.src) {
		c := s.src[s.i]
		switch {
		case raw && strings.HasPrefix(s.src[s.i:], `"""`):
			for strings.HasPrefix(s.src[s.i+1:], `"""`) { // the last three quotes of a run close it
				s.i++
			}
			tok.Value, s.i = s.src[from:s.i], s.i+3
			tok.Text = s.src[start:s.i]
			return tok, nil
		case !raw && c == '"':
			tok.Value, s.i = s.src[from:s.i], s.i+1
			tok.Text = s.src[start:s.i]
			return tok, nil
		case !raw && c == '\\':
			s.i += 2
		case !raw && c == '\n':
			return Token{}, s.fail(line, "unterminated string")
		case c == '$' && strings.HasPrefix(s.src[s.i+1:], "{"):
			tok.Template = true
			s.i += 2
			inner := &scanner{src: s.src, i: s.i, line: s.line}
			if _, err := inner.tokens(true); err != nil {
				return Token{}, err
			}
			s.i, s.line = inner.i, inner.line
		case c == '$' && s.i+1 < len(s.src) && isIdentStart(s.src[s.i+1]):
			tok.Template = true
			s.i++
		default:
			if c == '\n' {
				s.line++
			}
			s.i++
		}
	}
	return Token{}, s.fail(line, "unterminated string")
}

// char reads a '...' literal at s.i (\ escapes honoured).
func (s *scanner) char() error {
	start, line := s.i, s.line
	for s.i++; s.i < len(s.src) && s.src[s.i] != '\n'; s.i++ {
		switch s.src[s.i] {
		case '\\':
			s.i++
		case '\'':
			s.i++
			s.out = append(s.out, Token{Kind: Char, Text: s.src[start:s.i], Line: line})
			return nil
		}
	}
	return s.fail(line, "unterminated char literal")
}

func isIdentStart(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= 0x80
}
func isIdentPart(c byte) bool { return isIdentStart(c) || c >= '0' && c <= '9' }

// Code renders toks as the code lines a line-wise rule matches: each token on its source line, comments gone, a
// string or char literal reduced to its quotes (the lines a multi-line one spans kept, empty), and one blank where
// the source had a gap between two tokens of a line. Line k of the result is line k+1 of the script.
func Code(toks []Token) []string {
	var lines []string
	var b strings.Builder
	line, end := 1, 0
	for _, t := range toks {
		for ; line < t.Line; line++ {
			lines = append(lines, b.String())
			b.Reset()
		}
		if b.Len() > 0 && t.Off > end {
			b.WriteByte(' ')
		}
		text := t.Text
		switch t.Kind {
		case String:
			q := 1
			if strings.HasPrefix(t.Text, `"""`) {
				q = 3
			}
			text = t.Text[:q] + strings.Repeat("\n", strings.Count(t.Value, "\n")) + t.Text[len(t.Text)-q:]
		case Char:
			text = "''"
		}
		for seg, rest, more := strings.Cut(text, "\n"); ; seg, rest, more = strings.Cut(rest, "\n") {
			b.WriteString(seg)
			if !more {
				break
			}
			lines = append(lines, b.String())
			b.Reset()
			line++
		}
		end = t.End
	}
	return append(lines, b.String())
}
