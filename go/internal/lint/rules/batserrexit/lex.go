package batserrexit

import "strings"

// The lexer splits bash source into words, control operators and newlines, which is all the list parser
// needs. A word keeps its quotes and expansions: '...' and "..." (both may span lines), $(...), ${...},
// $'...', `...`, <(...) and >(...), and redirections (2>&1, &>file). Inside [[ ]] the operators && || ( )
// are words (the conditional's own operators, not a list). (( )) is one word. A comment runs to the end
// of its line, a backslash-newline joins two lines, and a heredoc body is skipped at the newline that
// ends its command line, so neither a run bash -c '...' string nor a <<EOF body is ever parsed as test
// code.

type tkind int

const (
	tWord tkind = iota
	tOp
	tNL
)

// token is one word, operator (&& || ;; ; & | |& ( )) or newline, with its 1-based line and byte range.
type token struct {
	kind       tkind
	text       string
	line       int // 1-based
	start, end int // byte offsets
}

type lexer struct {
	src     string
	pos     int
	line    int
	toks    []token
	pending []heredoc
	dbl     bool // inside [[ ]]
}

type heredoc struct {
	delim string
	strip bool
}

// lex returns the tokens of src in order.
func lex(src string) []token {
	lx := &lexer{src: src, line: 1}
	lx.run()
	return lx.toks
}

func (lx *lexer) emit(k tkind, text string, line, start int) {
	lx.toks = append(lx.toks, token{kind: k, text: text, line: line, start: start, end: lx.pos})
}

func (lx *lexer) peek(off int) byte {
	if lx.pos+off < len(lx.src) {
		return lx.src[lx.pos+off]
	}
	return 0
}

func (lx *lexer) run() {
	for lx.pos < len(lx.src) {
		c := lx.src[lx.pos]
		switch {
		case c == ' ' || c == '\t' || c == '\r':
			lx.pos++
		case c == '\\' && lx.peek(1) == '\n':
			lx.pos += 2
			lx.line++
		case c == '#':
			for lx.pos < len(lx.src) && lx.src[lx.pos] != '\n' {
				lx.pos++
			}
		case c == '\n':
			start := lx.pos
			lx.pos++
			lx.emit(tNL, "\n", lx.line, start)
			lx.line++
			lx.skipHeredocs()
		case !lx.dbl && (c == '&' && lx.peek(1) == '&' || c == '|' && lx.peek(1) == '|' || c == ';' && lx.peek(1) == ';'):
			start := lx.pos
			lx.pos += 2
			lx.emit(tOp, lx.src[start:lx.pos], lx.line, start)
		case !lx.dbl && c == '(' && lx.peek(1) == '(':
			start, line := lx.pos, lx.line
			lx.pos += 2
			lx.balanced('(', ')', 2)
			lx.emit(tWord, lx.src[start:lx.pos], line, start)
		case !lx.dbl && c == '&' && lx.peek(1) == '>':
			lx.word()
		case !lx.dbl && (c == ';' || c == '&' || c == '|' || c == '(' || c == ')'):
			start := lx.pos
			lx.pos++
			if c == '|' && lx.peek(0) == '&' {
				lx.pos++
			}
			lx.emit(tOp, lx.src[start:lx.pos], lx.line, start)
		default:
			lx.word()
		}
	}
}

// word reads one shell word, quotes and expansions included, redirection operators included.
func (lx *lexer) word() {
	start, line := lx.pos, lx.line
	for lx.pos < len(lx.src) {
		c := lx.src[lx.pos]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			break
		}
		if c == '\\' {
			if lx.peek(1) == '\n' {
				break
			}
			lx.pos += 2
			continue
		}
		op := !lx.dbl || lx.src[start:lx.pos] == "]]"
		if op && (c == ';' || c == '|' || c == '(' || c == ')') {
			break
		}
		if op && c == '&' {
			// 2>&1 and >& are redirections; a bare & ends the word
			if lx.pos > start && (lx.src[lx.pos-1] == '>' || lx.src[lx.pos-1] == '<') {
				lx.pos++
				continue
			}
			if lx.pos == start && lx.peek(1) == '>' {
				lx.pos += 2
				continue
			}
			break
		}
		switch c {
		case '\'':
			lx.pos++
			for lx.pos < len(lx.src) && lx.src[lx.pos] != '\'' {
				if lx.src[lx.pos] == '\n' {
					lx.line++
				}
				lx.pos++
			}
			lx.pos++
		case '"':
			lx.pos++
			lx.dquote()
		case '`':
			lx.pos++
			for lx.pos < len(lx.src) && lx.src[lx.pos] != '`' {
				if lx.src[lx.pos] == '\\' {
					lx.pos++
				}
				if lx.pos < len(lx.src) && lx.src[lx.pos] == '\n' {
					lx.line++
				}
				lx.pos++
			}
			lx.pos++
		case '$':
			if lx.peek(1) == '(' {
				lx.pos += 2
				lx.balanced('(', ')', 1)
			} else if lx.peek(1) == '{' {
				lx.pos += 2
				lx.balanced('{', '}', 1)
			} else if lx.peek(1) == '\'' {
				lx.pos += 2
				for lx.pos < len(lx.src) && lx.src[lx.pos] != '\'' {
					if lx.src[lx.pos] == '\\' {
						lx.pos++
					}
					lx.pos++
				}
				lx.pos++
			} else {
				lx.pos++
			}
		case '<', '>':
			if !lx.dbl && c == '<' && lx.peek(1) == '(' || c == '>' && lx.peek(1) == '(' {
				lx.pos += 2
				lx.balanced('(', ')', 1)
				continue
			}
			lx.pos++
		default:
			lx.pos++
		}
	}
	if lx.pos > len(lx.src) {
		lx.pos = len(lx.src)
	}
	w := lx.src[start:lx.pos]
	lx.emit(tWord, w, line, start)
	if w == "[[" {
		lx.dbl = true
	} else if w == "]]" {
		lx.dbl = false
	}
	// heredoc: <<DELIM, <<-DELIM, << 'DELIM'
	if strings.HasPrefix(w, "<<") && !strings.HasPrefix(w, "<<<") {
		d := w[2:]
		strip := false
		if strings.HasPrefix(d, "-") {
			strip, d = true, d[1:]
		}
		if d == "" {
			// delimiter is the next word
			lx.skipBlank()
			s := lx.pos
			for lx.pos < len(lx.src) && !strings.ContainsRune(" \t\n;|&)", rune(lx.src[lx.pos])) {
				lx.pos++
			}
			d = lx.src[s:lx.pos]
		}
		d = strings.NewReplacer("'", "", "\"", "", "\\", "").Replace(d)
		lx.pending = append(lx.pending, heredoc{d, strip})
	}
}

func (lx *lexer) skipBlank() {
	for lx.pos < len(lx.src) && (lx.src[lx.pos] == ' ' || lx.src[lx.pos] == '\t') {
		lx.pos++
	}
}

func (lx *lexer) dquote() {
	for lx.pos < len(lx.src) {
		c := lx.src[lx.pos]
		switch c {
		case '\\':
			if lx.peek(1) == '\n' {
				lx.line++
			}
			lx.pos += 2
			continue
		case '"':
			lx.pos++
			return
		case '\n':
			lx.line++
		case '$':
			if lx.peek(1) == '(' {
				lx.pos += 2
				lx.balanced('(', ')', 1)
				continue
			}
			if lx.peek(1) == '{' {
				lx.pos += 2
				lx.balanced('{', '}', 1)
				continue
			}
		case '`':
			lx.pos++
			for lx.pos < len(lx.src) && lx.src[lx.pos] != '`' {
				if lx.src[lx.pos] == '\n' {
					lx.line++
				}
				lx.pos++
			}
		}
		lx.pos++
	}
}

// balanced skips to the close that brings depth to 0, honoring quotes.
func (lx *lexer) balanced(open, close byte, depth int) {
	for lx.pos < len(lx.src) && depth > 0 {
		c := lx.src[lx.pos]
		switch {
		case c == '\\':
			lx.pos += 2
			continue
		case c == '\'' && open == '(':
			lx.pos++
			for lx.pos < len(lx.src) && lx.src[lx.pos] != '\'' {
				if lx.src[lx.pos] == '\n' {
					lx.line++
				}
				lx.pos++
			}
		case c == '"':
			lx.pos++
			lx.dquote()
			continue
		case c == '\n':
			lx.line++
		case c == open:
			depth++
		case c == close:
			depth--
		}
		lx.pos++
	}
}

func (lx *lexer) skipHeredocs() {
	for _, h := range lx.pending {
		for lx.pos < len(lx.src) {
			e := strings.IndexByte(lx.src[lx.pos:], '\n')
			var ln string
			if e < 0 {
				ln = lx.src[lx.pos:]
				lx.pos = len(lx.src)
			} else {
				ln = lx.src[lx.pos : lx.pos+e]
				lx.pos += e + 1
			}
			lx.line++
			t := strings.TrimRight(ln, "\r")
			if h.strip {
				t = strings.TrimLeft(t, "\t")
			}
			if t == h.delim {
				break
			}
		}
	}
	lx.pending = nil
}
