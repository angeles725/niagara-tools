package javasrc

import "strings"

// Decl is one method or constructor declaration with a body, found by Decls in a token stream.
type Decl struct {
	Class    string   // the simple name of the enclosing class (an anonymous class: the type it instantiates)
	Name     string   // the method name (a constructor: the class name)
	Params   []string // the parameter types, annotations, final and names dropped ("Map<K,V>", "byte[]", "String...")
	Override bool     // an @Override annotation precedes it
	Line     int      // the 0-based line of the name token
	Body     [2]int   // the body's token range [Body[0], Body[1]): inside its braces
}

// classKeyword opens a class body; it never names one after a "." (Foo.class).
var classKeyword = map[string]bool{"class": true, "interface": true, "enum": true, "record": true}

// notMethod are the keywords that take a parenthesized header before a block.
var notMethod = map[string]bool{"if": true, "for": true, "while": true, "switch": true, "catch": true,
	"synchronized": true, "try": true, "do": true, "else": true, "new": true, "return": true}

type frame struct {
	class  string // set on a class body
	method int    // index into the result of an open method body, or -1
}

// Decls finds the method and constructor bodies declared directly in a class body (a named, nested,
// inner, enum, record or anonymous class), in source order of their closing brace's opening. A block
// inside a method (a control block, a lambda, an initializer) is not a declaration; an abstract or
// interface method without a body is not returned.
func Decls(toks []Token) []Decl {
	var out []Decl
	var stack []frame
	for k, t := range toks {
		if t.Kind != TokPunct {
			continue
		}
		switch t.Text {
		case "{":
			f := frame{method: -1}
			if name, ok := classBody(toks, k); ok {
				f.class = name
			} else if len(stack) > 0 && stack[len(stack)-1].class != "" {
				if d, ok := method(toks, k); ok {
					d.Class = stack[len(stack)-1].class
					f.method = len(out)
					out = append(out, d)
				}
			}
			stack = append(stack, f)
		case "}":
			if len(stack) == 0 {
				continue
			}
			if f := stack[len(stack)-1]; f.method >= 0 {
				out[f.method].Body[1] = k
			}
			stack = stack[:len(stack)-1]
		}
	}
	var closed []Decl // a body still open at the end of input has no range
	for _, d := range out {
		if d.Body[1] > 0 {
			closed = append(closed, d)
		}
	}
	return closed
}

// headerStart is the index after the previous ";", "{" or "}" before k (0 when none).
func headerStart(toks []Token, k int) int {
	for j := k - 1; j >= 0; j-- {
		if toks[j].Kind == TokPunct && (toks[j].Text == ";" || toks[j].Text == "{" || toks[j].Text == "}") {
			return j + 1
		}
	}
	return 0
}

// classBody reports whether the "{" at k opens a class body, and the class name.
func classBody(toks []Token, k int) (string, bool) {
	for j := headerStart(toks, k); j < k-1; j++ {
		if toks[j].Kind == TokIdent && classKeyword[toks[j].Text] && (j == 0 || toks[j-1].Text != ".") &&
			toks[j+1].Kind == TokIdent {
			return toks[j+1].Text, true
		}
	}
	if k > 0 && toks[k-1].Text == ")" { // new Foo<Bar>(...) { : an anonymous class
		j := match(toks, k-1, "(", ")")
		p := j - 1
		if p >= 0 && toks[p].Text == ">" {
			p = match(toks, p, "<", ">") - 1
		}
		if p >= 1 && toks[p].Kind == TokIdent {
			q := p
			for q >= 2 && toks[q-1].Text == "." && toks[q-2].Kind == TokIdent {
				q -= 2
			}
			if q >= 1 && toks[q-1].Text == "new" {
				return toks[p].Text, true
			}
		}
	}
	return "", false
}

// method reads the declaration whose body opens at the "{" at k: "name(params) [throws A, B] {".
func method(toks []Token, k int) (Decl, bool) {
	start := headerStart(toks, k)
	r := k - 1
	for j := k - 1; j > start; j-- { // a throws clause between ")" and "{"
		if toks[j].Kind == TokIdent && toks[j].Text == "throws" {
			r = j - 1
			break
		}
	}
	if r < start || toks[r].Text != ")" {
		return Decl{}, false
	}
	l := match(toks, r, "(", ")")
	n := l - 1
	if n < start || toks[n].Kind != TokIdent || notMethod[toks[n].Text] {
		return Decl{}, false
	}
	d := Decl{Name: toks[n].Text, Line: toks[n].Line, Params: params(toks[l+1 : r]), Body: [2]int{k + 1, 0}}
	for j := start; j+1 < n; j++ {
		if toks[j].Text == "@" && toks[j+1].Text == "Override" {
			d.Override = true
		}
	}
	return d, true
}

// match is the index of the open token pairing with the close token at k (0 when unbalanced).
func match(toks []Token, k int, open, close string) int {
	depth := 0
	for j := k; j >= 0; j-- {
		switch toks[j].Text {
		case close:
			depth++
		case open:
			if depth--; depth == 0 {
				return j
			}
		}
	}
	return 0
}

// params is the type of each parameter in a parameter list's tokens.
func params(list []Token) []string {
	var out []string
	depth, from := 0, 0
	for j := 0; j <= len(list); j++ {
		if j < len(list) {
			switch list[j].Text {
			case "<", "(":
				depth++
			case ">", ")":
				depth--
			}
			if list[j].Text != "," || depth != 0 {
				continue
			}
		}
		if j > from {
			out = append(out, paramType(list[from:j]))
		}
		from = j + 1
	}
	return out
}

// paramType drops a parameter's annotations, final and name; a C-style "int a[]" is "int[]".
func paramType(p []Token) string {
	var kept []Token
	for j := 0; j < len(p); j++ {
		switch {
		case p[j].Text == "@" && j+1 < len(p):
			j++ // the annotation name, then its arguments
			if j+1 < len(p) && p[j+1].Text == "(" {
				depth := 0
				for j++; j < len(p); j++ {
					if p[j].Text == "(" {
						depth++
					} else if p[j].Text == ")" {
						if depth--; depth == 0 {
							break
						}
					}
				}
			}
		case p[j].Text == "final":
		default:
			kept = append(kept, p[j])
		}
	}
	suffix := ""
	for len(kept) >= 2 && kept[len(kept)-1].Text == "]" && kept[len(kept)-2].Text == "[" {
		kept, suffix = kept[:len(kept)-2], suffix+"[]"
	}
	if len(kept) > 1 {
		kept = kept[:len(kept)-1] // the name
	}
	var b strings.Builder
	for _, t := range kept {
		b.WriteString(t.Text)
	}
	return b.String() + suffix
}

// Statements counts the statements of a body: its ";" tokens, a for header's two counted as none.
func Statements(body []Token) int {
	n := 0
	for j := 0; j < len(body); j++ {
		switch {
		case body[j].Text == "for" && j+1 < len(body) && body[j+1].Text == "(":
			depth := 0
			for j++; j < len(body); j++ {
				if body[j].Text == "(" {
					depth++
				} else if body[j].Text == ")" {
					if depth--; depth == 0 {
						break
					}
				}
			}
		case body[j].Kind == TokPunct && body[j].Text == ";":
			n++
		}
	}
	return n
}

// Shape is the normalized token sequence of a body: token texts (identifiers and literals kept) joined by
// a NUL byte, so layout and comments never change it and token boundaries are kept.
func Shape(body []Token) string {
	var b strings.Builder
	for i, t := range body {
		if i > 0 {
			b.WriteByte(0)
		}
		b.WriteString(t.Text)
	}
	return b.String()
}
