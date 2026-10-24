// Package javasrc is a small Java source scanner: it blanks comments and finds method bodies by
// brace depth. It is the Go port of the kit's awk lib build-n4-module-kit/toolbelt/lib/method-boundary.sh
// (mb_strip + mb_parse) and keeps its results exactly; TestDifferentialAwk runs both on the same inputs.
//
// Semantics carried over from the awk lib (its invariants I1-I6):
//   - Strip blanks // and /* */ comments, carries block-comment and text-block state across lines and
//     keeps line numbers. "..." strings and '...' chars (\ escapes honoured) and """...""" text blocks
//     are copied as-is, contents included. An unterminated "..."/'...' ends at its line end.
//   - Methods ignores braces inside literals (I6), opens a method on a line whose peak depth rises to 2
//     or more (I1), names it from a single-line signature or, for a brace alone on its line, from a
//     backward scan that stops at an annotation, a prior statement or a prior block (I2), never names
//     if/for/while/switch/catch/try/else/do/new (I3) and skips one-liner get/set/is accessors (I4).
//   - One declared difference from mb_parse (2026-10-24): the same backward scan also names a signature
//     wrapped over several lines whose last line holds ") {" (`void m(int a,` / `    int b) {`); mb_parse
//     missed that method, a false negative for every method-scoped lint. TestDifferentialAwk's inputs do
//     not hold that shape; the bash lints still on mb_parse keep the old behaviour.
//
// Scanning is byte-wise: every character the rules test is ASCII, so multi-byte UTF-8 text passes
// through unchanged, as it does in awk.
package javasrc

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

// EndState is the lexical state Strip ends in; callers may treat non-Closed as "boundaries not trusted".
type EndState int

const (
	Closed        EndState = iota // every comment and text block is closed
	OpenComment                   // the input ends inside /* */
	OpenTextBlock                 // the input ends inside a """ text block
)

// Method is one method body: Start is the line of its opening brace, End the line where it closes
// (equal for a one-liner). Lines are 1-based.
type Method struct {
	Name       string
	Start, End int
}

// Lines splits r into lines like awk's default records: on "\n", keeping any "\r", and with no
// empty last line after a final "\n".
func Lines(r io.Reader) ([]string, error) {
	var lines []string
	br := bufio.NewReader(r)
	for {
		ln, err := br.ReadString('\n')
		if len(ln) > 0 {
			lines = append(lines, strings.TrimSuffix(ln, "\n"))
		}
		if err == io.EOF {
			return lines, nil
		}
		if err != nil {
			return lines, err
		}
	}
}

const textBlock = `"""`

// Strip returns the lines with comments blanked (literal text kept) and the state the input ends in.
func Strip(lines []string) ([]string, EndState) {
	out := make([]string, len(lines))
	inComment, inText := false, false
	var b strings.Builder
	for i, ln := range lines {
		b.Reset()
		var quote byte
	scan:
		for j := 0; j < len(ln); {
			rest := ln[j:]
			switch {
			case inComment:
				if strings.HasPrefix(rest, "*/") {
					inComment = false
					j += 2
				} else {
					j++
				}
			case (inText || quote != 0) && ln[j] == '\\':
				b.WriteString(rest[:min(2, len(rest))])
				j += 2
			case inText:
				if strings.HasPrefix(rest, textBlock) {
					b.WriteString(textBlock)
					inText = false
					j += 3
				} else {
					b.WriteByte(ln[j])
					j++
				}
			case quote != 0:
				if ln[j] == quote {
					quote = 0
				}
				b.WriteByte(ln[j])
				j++
			case strings.HasPrefix(rest, textBlock):
				b.WriteString(textBlock)
				inText = true
				j += 3
			case strings.HasPrefix(rest, "/*"):
				inComment = true
				j += 2
			case strings.HasPrefix(rest, "//"):
				break scan
			default:
				if ln[j] == '"' || ln[j] == '\'' {
					quote = ln[j]
				}
				b.WriteByte(ln[j])
				j++
			}
		}
		out[i] = b.String()
	}
	state := Closed
	if inComment {
		state = OpenComment
	} else if inText {
		state = OpenTextBlock
	}
	return out, state
}

const ident = `[A-Za-z_][A-Za-z0-9_<>\[\]]*`

var (
	signature = regexp.MustCompile(ident + `[[:space:]]*\([^)]*\)[[:space:]]*(throws[^{]*)?\{`)
	call      = regexp.MustCompile(ident + `[[:space:]]*\(`)
	leadIdent = regexp.MustCompile(`^` + ident)
	keyword   = regexp.MustCompile(`^(if|for|while|switch|catch|try|else|do|new)$`)
	typeOrKw  = regexp.MustCompile(`^(if|for|while|switch|catch|try|else|do|new|class|interface|enum)$`)
	accessor  = regexp.MustCompile(`^(get|set|is)[A-Z_]`)
	// the last line of a wrapped signature: a ")" with no "(" before it, or a throws clause, then the "{"
	wrappedEnd = regexp.MustCompile(`^[^(]*\)[[:space:]]*(throws[^{]*)?\{|^[[:space:]]*throws[^{(]*\{`)
)

// space is awk's [[:space:]] in the C locale.
const space = " \t\n\v\f\r"

// backScanLines is how many lines above a lone "{" the signature search looks.
const backScanLines = 20

// Methods finds the method bodies in already-stripped lines (the output of Strip).
func Methods(stripped []string) []Method { return bodies(stripped, false) }

// Scopes is Methods plus every lambda body opened outside a method (a field initializer such as
// `Runnable r = () -> {`, its "->" on the line of the "{" or the line before), named "lambda": the
// bodies a statement can belong to, in source order and never nested. A lambda inside a method belongs to
// that method's body, as the locals it captures do.
func Scopes(stripped []string) []Method { return bodies(stripped, true) }

// Enclosing is the scope holding 1-based line, ok false when no scope holds it.
func Enclosing(scopes []Method, line int) (m Method, ok bool) {
	for _, m := range scopes {
		if m.Start <= line && line <= m.End {
			return m, true
		}
	}
	return Method{}, false
}

// lambdaOpen reports whether the body opening on line i (0-based) is a lambda's: "->" on that line, or
// a lone "{" right after a line ending in "->".
func lambdaOpen(stripped []string, i int) bool {
	if strings.Contains(stripped[i], "->") {
		return true
	}
	return i > 0 && strings.Trim(stripped[i], space) == "{" && strings.HasSuffix(strings.TrimRight(stripped[i-1], space), "->")
}

// bodies scans the method bodies, and the lambda bodies opened outside a method when lambdas is set.
func bodies(stripped []string, lambdas bool) []Method {
	var (
		methods        []Method
		depth          int
		inText, inBody bool
		bodyDepth      int
		cur            Method
	)
	for i, ln := range stripped {
		oldDepth, peak := depth, depth
		var quote byte
		for j := 0; j < len(ln); j++ {
			c := ln[j]
			switch {
			case inText:
				if c == '\\' {
					j++
				} else if strings.HasPrefix(ln[j:], textBlock) {
					inText = false
					j += 2
				}
			case quote != 0:
				if c == '\\' {
					j++
				} else if c == quote {
					quote = 0
				}
			case strings.HasPrefix(ln[j:], textBlock):
				inText = true
				j += 2
			case c == '"' || c == '\'':
				quote = c
			case c == '{':
				depth++
				peak = max(peak, depth)
			case c == '}':
				depth--
			}
		}
		if !inBody && peak > oldDepth && peak >= 2 {
			name := methodName(stripped, i)
			if name != "" && depth <= oldDepth && accessor.MatchString(name) {
				name = ""
			}
			if name == "" && lambdas && lambdaOpen(stripped, i) {
				name = "lambda"
			}
			if name != "" {
				inBody, bodyDepth, cur = true, peak, Method{Name: name, Start: i + 1}
			}
		}
		if inBody && depth < bodyDepth {
			cur.End = i + 1
			methods = append(methods, cur)
			inBody = false
		}
	}
	return methods
}

// methodName names the method opening on line i (0-based), or returns "".
func methodName(stripped []string, i int) string {
	ln := stripped[i]
	// The '(' checks only skip regexp work on lines that cannot match.
	if strings.Contains(ln, "(") {
		if loc := signature.FindStringIndex(ln); loc != nil {
			if name := leadIdent.FindString(ln[loc[0]:]); !keyword.MatchString(name) {
				return name
			}
			return ""
		}
	}
	// A lone "{" (awk: the line is "{" once all whitespace is removed), or the ") {" end of a signature
	// wrapped over several lines: the name is on the line above that opened the parameter list.
	if strings.Trim(ln, space) != "{" && !wrappedEnd.MatchString(ln) {
		return ""
	}
	for k := i - 1; k >= 0 && k >= i-backScanLines; k-- {
		lk := stripped[k]
		if strings.HasPrefix(strings.TrimLeft(lk, space), "@") {
			break
		}
		if strings.HasSuffix(strings.TrimRight(lk, space), ";") || strings.Contains(lk, "{") {
			break
		}
		if !strings.Contains(lk, "(") {
			continue
		}
		if loc := call.FindStringIndex(lk); loc != nil {
			if name := leadIdent.FindString(lk[loc[0]:]); !typeOrKw.MatchString(name) {
				return name
			}
		}
	}
	return ""
}
