package javasrc

import "strings"

// TokenKind is a Java token class.
type TokenKind int

// The token kinds Tokens emits.
const (
	TokIdent   TokenKind = iota // a name or a keyword
	TokNumber                   // a numeric literal (digits, letters, '.' and '_' after a leading digit)
	TokLiteral                  // a "..." string, a '...' char or a """...""" text block, quotes included
	TokPunct                    // one punctuation byte: operators are sequences of these
)

func (k TokenKind) String() string { return [...]string{"ident", "number", "literal", "punct"}[k] }

// Token is one token; Line is the 0-based line it starts on.
type Token struct {
	Kind TokenKind
	Line int
	Text string
}

// Tokens splits stripped lines (Strip output: comments already blanks) into tokens. A literal is one token
// with its quotes and \ escapes as written, so a brace, a semicolon or a whole method inside a string or a
// text block is never code. An unterminated "..." or '...' ends at its line end, as in Strip. Operators
// are single bytes: two token streams compare equal exactly when their code does.
func Tokens(stripped []string) []Token {
	src := strings.Join(stripped, "\n")
	var out []Token
	line := 0
	for i := 0; i < len(src); {
		c, start, at := src[i], i, line
		switch {
		case c == '\n':
			line++
			i++
			continue
		case c == ' ' || c == '\t' || c == '\r' || c == '\f' || c == '\v':
			i++
			continue
		case strings.HasPrefix(src[i:], textBlock):
			i += 3
			for i < len(src) && !strings.HasPrefix(src[i:], textBlock) {
				if src[i] == '\\' {
					i++ // a \""" inside does not close it
				}
				i++
			}
			i = min(i+3, len(src))
			line += strings.Count(src[start:i], "\n")
			out = append(out, Token{TokLiteral, at, src[start:i]})
			continue
		case c == '"' || c == '\'':
			i++
			for i < len(src) && src[i] != c && src[i] != '\n' {
				if src[i] == '\\' && i+1 < len(src) && src[i+1] != '\n' {
					i++
				}
				i++
			}
			if i < len(src) && src[i] == c {
				i++
			}
			out = append(out, Token{TokLiteral, at, src[start:i]})
			continue
		case identByte(c):
			for i < len(src) && (identByte(src[i]) || isDigit(src[i])) {
				i++
			}
			out = append(out, Token{TokIdent, at, src[start:i]})
			continue
		case isDigit(c):
			for i < len(src) && (identByte(src[i]) || isDigit(src[i]) || src[i] == '.') {
				i++
			}
			out = append(out, Token{TokNumber, at, src[start:i]})
			continue
		}
		i++
		out = append(out, Token{TokPunct, at, src[start:i]})
	}
	return out
}

func identByte(c byte) bool {
	return c == '_' || c == '$' || c >= 0x80 || ('a' <= c|0x20 && c|0x20 <= 'z')
}
func isDigit(c byte) bool { return '0' <= c && c <= '9' }
