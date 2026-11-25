package rcscan

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	mediaWidth = regexp.MustCompile(`(^|[^a-z-])((min|max)-)?width([^a-z-]|$)`)
	fixedWidth = regexp.MustCompile(`^((min-)?width)\s*:\s*([0-9]*\.?[0-9]+)px(\s*!important)?$`)
)

// wideDecl is a width / min-width declaration of more than 360px found outside every @media block.
type wideDecl struct {
	line int
	decl string
}

// scanCSS reads the CSS of src (a .css file, or the <style> blocks of an .html file) and reports whether it holds an
// @media rule with a width feature and the fixed wide width declarations outside @media. Comments are not CSS; a
// declaration on a line carrying `rc-scan: allow responsive` is waived.
func scanCSS(src *source) (wide []wideDecl, hasMedia bool) {
	var stack []bool // per open block: inside an @media block
	var buf strings.Builder
	var start int
	var comment, inStyle bool
	var quote byte
	flush := func(end byte) {
		t := strings.TrimSpace(buf.String())
		buf.Reset()
		defer func() { start = 0 }()
		inMedia := len(stack) > 0 && stack[len(stack)-1]
		switch {
		case end == '{':
			lt := strings.ToLower(t)
			isMedia := strings.HasPrefix(lt, "@media")
			if isMedia && mediaWidth.MatchString(lt[len("@media"):]) {
				hasMedia = true
			}
			stack = append(stack, inMedia || isMedia)
		case t != "" && start > 0 && !inMedia && !allowedOn(src.lines[start-1], respAllow):
			if m := fixedWidth.FindStringSubmatch(strings.ToLower(t)); m != nil {
				if px, err := strconv.ParseFloat(m[3], 64); err == nil && px > phoneWidth {
					wide = append(wide, wideDecl{start, strings.TrimSpace(t)})
				}
			}
		}
		if end == '}' && len(stack) > 0 {
			stack = stack[:len(stack)-1]
		}
	}
	for i, line := range src.lines {
		if src.ext == "html" {
			line, inStyle = styleText(line, inStyle)
		}
		for j := 0; j < len(line); j++ {
			c := line[j]
			switch {
			case comment:
				if c == '*' && j+1 < len(line) && line[j+1] == '/' {
					comment = false
					j++
				}
			case quote != 0:
				buf.WriteByte(c)
				if c == quote {
					quote = 0
				}
			case c == '/' && j+1 < len(line) && line[j+1] == '*':
				comment = true
				j++
			case c == '"' || c == '\'':
				quote = c
				buf.WriteByte(c)
			case c == '{' || c == '}' || c == ';':
				flush(c)
			default:
				if start == 0 {
					if c == ' ' || c == '\t' || c == '\r' {
						continue
					}
					start = i + 1
				}
				buf.WriteByte(c)
			}
		}
		if start != 0 {
			buf.WriteByte(' ')
		}
	}
	if strings.TrimSpace(buf.String()) != "" {
		flush(';')
	}
	return wide, hasMedia
}

// styleText returns the part of an HTML line that is inside a <style> block, and whether the next line starts inside
// one. A line with no block gives "".
func styleText(line string, in bool) (string, bool) {
	var out strings.Builder
	for line != "" {
		if !in {
			m := styleOpen.FindStringIndex(line)
			if m == nil {
				break
			}
			line, in = line[m[1]:], true
			continue
		}
		if m := styleClose.FindStringIndex(line); m != nil {
			out.WriteString(line[:m[0]] + " ")
			line, in = line[m[1]:], false
			continue
		}
		out.WriteString(line)
		line = ""
	}
	return out.String(), in
}
