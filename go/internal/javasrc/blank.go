package javasrc

import "strings"

// Literal is one "..." or '...' literal that closes on its line, as BlankLiterals met it.
type Literal struct {
	Line  int    // 0-based index of its line in the src given to BlankLiterals
	Col   int    // 1-based column of the opening quote in the blanked line
	Quote byte   // '"' or '\''
	Text  string // the source text between the quotes, escapes as written
}

// BlankLiterals returns src (stripped lines) with the contents of "...", '...' and """...""" literals
// removed: quotes kept, \ escapes honoured, text-block state carried across lines, and a "..." or '...'
// literal ending at its line end, so a { } ; ( @ or call inside one is never code. It reports whether a
// "..." or '...' literal was still open at a line end, and calls lit (when non-nil) for every "..." or
// '...' literal that closes on its line, in source order.
func BlankLiterals(src []string, lit func(Literal)) (dst []string, unterm bool) {
	dst = make([]string, len(src))
	inText := false
	var b strings.Builder
	for i, ln := range src {
		var open bool
		dst[i], open = blankLine(&b, ln, i, true, 0, &inText, lit)
		unterm = unterm || open
	}
	return dst, unterm
}

// MaskLiterals is BlankLiterals with every removed byte replaced by a blank instead: each line keeps its
// length, so an index found in the masked line is the same index in src (a rule that matches code there
// can still read a literal argument's text from src).
func MaskLiterals(src []string) []string {
	dst := make([]string, len(src))
	inText := false
	var b strings.Builder
	for i, ln := range src {
		dst[i], _ = blankLine(&b, ln, i, true, ' ', &inText, nil)
	}
	return dst
}

// BlankLine is the toolbelt awk blank_str() of one line: the contents of its "..." and '...' literals
// removed (quotes kept, \ escapes honoured, a literal still open ending at the line end). It keeps no
// text-block state: a """ is three quotes, as awk reads it line by line.
func BlankLine(s string) string {
	var b strings.Builder
	inText := false
	out, _ := blankLine(&b, s, 0, false, 0, &inText, nil)
	return out
}

// blankLine blanks line i of a scan whose text-block state is *inText (text blocks are recognised
// only when textBlocks is set) and reports whether a "..." or '...' literal is open at the line end.
// A removed byte is dropped, or written as fill when fill is not 0.
func blankLine(b *strings.Builder, ln string, i int, textBlocks bool, fill byte, inText *bool, lit func(Literal)) (string, bool) {
	if !*inText && !strings.ContainsAny(ln, `"'`) {
		return ln, false // nothing to blank
	}
	b.Reset()
	var quote byte
	start, at := 0, 0 // the opening quote's index in ln and 1-based column in the blanked line
	for j := 0; j < len(ln); j++ {
		c, j0 := ln[j], j
		switch {
		case *inText:
			if c == '\\' {
				j++
				pad(b, fill, min(j+1, len(ln))-j0)
			} else if strings.HasPrefix(ln[j:], `"""`) {
				b.WriteString(`"""`)
				*inText = false
				j += 2
			} else {
				pad(b, fill, 1)
			}
		case quote != 0:
			if c == '\\' {
				j++
				pad(b, fill, min(j+1, len(ln))-j0)
			} else if c == quote {
				b.WriteByte(c)
				if lit != nil {
					lit(Literal{Line: i, Col: at, Quote: quote, Text: ln[start+1 : j]})
				}
				quote = 0
			} else {
				pad(b, fill, 1)
			}
		case textBlocks && strings.HasPrefix(ln[j:], `"""`):
			b.WriteString(`"""`)
			*inText = true
			j += 2
		default:
			if c == '"' || c == '\'' {
				quote, start, at = c, j, b.Len()+1
			}
			b.WriteByte(c)
		}
	}
	return b.String(), quote != 0
}

// pad writes n fill bytes (nothing when fill is 0).
func pad(b *strings.Builder, fill byte, n int) {
	for ; fill != 0 && n > 0; n-- {
		b.WriteByte(fill)
	}
}

// MaskLine is BlankLine with each removed literal byte replaced by a space, so every byte keeps its index
// (a caller that finds a bracket or comma in the masked line slices the original line at the same index).
func MaskLine(s string) string {
	b := []byte(s)
	var quote byte
	for j := 0; j < len(b); j++ {
		switch c := b[j]; {
		case quote != 0 && c == '\\':
			b[j] = ' '
			if j+1 < len(b) {
				j++
				b[j] = ' '
			}
		case quote != 0 && c == quote:
			quote = 0
		case quote != 0:
			b[j] = ' '
		case c == '"' || c == '\'':
			quote = c
		}
	}
	return string(b)
}
