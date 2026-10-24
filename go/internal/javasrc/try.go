package javasrc

import "strings"

// Tries is the try statements of one file, read structurally from its code: Text is the stripped lines
// with literal contents removed (BlankLiterals) joined by "\n", so no brace, keyword or parenthesis inside a
// comment or a literal is code, and every offset below indexes Text.
type Tries struct {
	Text string
	List []Try // in the order their `try` keyword appears (an outer try before the tries nested in it)
}

// Try is one `try [ (resources) ] { ... }` statement and the clauses attached to its block's closing brace.
type Try struct {
	At      int   // offset of the `try` keyword
	Body    Block // the try block; a resource specification lies between At and Body.Open
	Catches []Catch
}

// Catch is one `catch (Type [| Type...] name) { ... }` clause of a try.
type Catch struct {
	At    int      // offset of the `catch` keyword
	Types []string // the caught types as written, one per `|` alternative, blanks trimmed
	Body  Block
}

// Block is a brace block: the offsets of its "{" and of the matching "}" (len(Text) when it never closes).
type Block struct{ Open, Close int }

// Contains reports whether offset off lies inside the block's braces.
func (b Block) Contains(off int) bool { return b.Open < off && off < b.Close }

// Line is the 1-based line of offset off.
func (t Tries) Line(off int) int { return strings.Count(t.Text[:off], "\n") + 1 }

// Offset is the offset of the first byte of 1-based line n.
func (t Tries) Offset(n int) int {
	off := 0
	for ; n > 1; n-- {
		off += strings.IndexByte(t.Text[off:], '\n') + 1
	}
	return off
}

// TryBlocks reads every try statement of the stripped lines (Strip output). A catch belongs to a try only
// when it follows the try block's closing brace (blanks and line breaks between), or the closing brace of
// the catch before it; a `finally` clause ends the chain. A try whose `{` is missing is not one.
func TryBlocks(stripped []string) Tries {
	code, _ := BlankLiterals(stripped, nil)
	t := Tries{Text: strings.Join(code, "\n")}
	s := t.Text
	for at := wordFrom(s, "try", 0); at >= 0; at = wordFrom(s, "try", at+3) {
		p := skipBlank(s, at+3)
		if p < len(s) && s[p] == '(' {
			p = skipBlank(s, closeParen(s, p)+1)
		}
		if p >= len(s) || s[p] != '{' {
			continue
		}
		tr := Try{At: at, Body: Block{p, closeBrace(s, p)}}
		for end := tr.Body.Close; end < len(s); {
			c := skipBlank(s, end+1)
			if !wordAt(s, c, "catch") {
				break
			}
			open := skipBlank(s, c+len("catch"))
			if open >= len(s) || s[open] != '(' {
				break
			}
			shut := closeParen(s, open)
			brace := skipBlank(s, shut+1)
			if shut >= len(s) || brace >= len(s) || s[brace] != '{' {
				break
			}
			var types []string
			if decl := strings.TrimSpace(s[open+1 : shut]); decl != "" {
				if sp := strings.LastIndexAny(decl, space); sp >= 0 {
					decl = decl[:sp] // drop the parameter name
				}
				for _, ty := range strings.Split(decl, "|") {
					types = append(types, strings.Trim(strings.TrimPrefix(strings.TrimSpace(ty), "final "), space))
				}
			}
			cb := Block{brace, closeBrace(s, brace)}
			tr.Catches = append(tr.Catches, Catch{At: c, Types: types, Body: cb})
			end = cb.Close
		}
		t.List = append(t.List, tr)
	}
	return t
}

// wordFrom is the offset of the first whole-word w in s at or after from, -1 when there is none.
func wordFrom(s, w string, from int) int {
	for from < len(s) {
		i := strings.Index(s[from:], w)
		if i < 0 {
			return -1
		}
		if wordAt(s, from+i, w) {
			return from + i
		}
		from += i + 1
	}
	return -1
}

// wordAt reports whether the whole word w starts at offset i of s.
func wordAt(s string, i int, w string) bool {
	if i < 0 || i+len(w) > len(s) || s[i:i+len(w)] != w {
		return false
	}
	return (i == 0 || !identByte(s[i-1])) && (i+len(w) == len(s) || !identByte(s[i+len(w)]))
}

func identByte(c byte) bool {
	return c == '_' || c == '$' || c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z'
}

// skipBlank is the first offset at or after i that is not a blank or a line break.
func skipBlank(s string, i int) int {
	for i < len(s) && strings.IndexByte(space, s[i]) >= 0 {
		i++
	}
	return i
}

// closeParen is the offset of the ")" matching the "(" at open, len(s) when it never closes.
func closeParen(s string, open int) int { return match(s, open, '(', ')') }

// closeBrace is the offset of the "}" matching the "{" at open, len(s) when it never closes.
func closeBrace(s string, open int) int { return match(s, open, '{', '}') }

func match(s string, open int, l, r byte) int {
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case l:
			depth++
		case r:
			if depth--; depth == 0 {
				return i
			}
		}
	}
	return len(s)
}
