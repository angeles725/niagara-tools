package vacuoustests

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// vacuous is one flagged test method: Class.method, its file and 1-based line, its check and the row text.
type vacuous struct {
	name, path string
	line       int
	check, why string
}

const (
	whyTrivial = "its only assertions are trivially true, so it cannot fail"
	whyNone    = "no assertion (assert*, verify*, expect*, fail, an asserting helper, expected exceptions), so it cannot fail"
)

// flagged judges every indexed method that carries a method-level @Test.
func (idx *index) flagged() ([]vacuous, error) {
	var out []vacuous
	for _, m := range idx.methods {
		k, d := m.owner, m.d
		at := nameAt(k.toks, k.c.Body[0], d)
		if at < 0 {
			return out, fmt.Errorf("cannot scan %s:%d: cannot find the declaration of %s.%s", k.path, d.Line+1, k.c.Name, d.Name)
		}
		test, expects := testAnnotation(k.toks, at)
		if !test {
			continue
		}
		v := vacuous{name: d.Class + "." + d.Name, path: k.path, line: d.Line + 1}
		body := k.toks[d.Body[0]:d.Body[1]]
		if skip := skipFirst(body); skip != "" {
			v.check, v.why = "always-skipped", "its first statement always skips it ("+skip+"), so it never runs"
			out = append(out, v)
			continue
		}
		if expects {
			continue
		}
		switch real, trivial := assertions(body, idx.scope(k)); {
		case real > 0:
			continue
		case trivial > 0:
			v.check, v.why = "trivially-true", whyTrivial
		default:
			v.check, v.why = "no-assertion", whyNone
		}
		out = append(out, v)
	}
	return out, nil
}

// nameAt is the index of the name token of method d, searched back from its body to the class body's start
// (from), -1 when it is not found.
func nameAt(toks []javasrc.Token, from int, d javasrc.Decl) int {
	for i := d.Body[0] - 1; i > from; i-- {
		if t := toks[i]; t.Kind == javasrc.TokIdent && t.Text == d.Name && t.Line == d.Line && toks[i+1].Text == "(" {
			return i
		}
	}
	return -1
}

// testAnnotation reads the annotations and modifiers before the method name token at: whether one is @Test (any
// qualifier) and whether that @Test sets expectedExceptions (TestNG) or expected (JUnit 4).
func testAnnotation(toks []javasrc.Token, at int) (test, expects bool) {
	var args []javasrc.Token // the arguments of the last ")...(" walked over
	for i := at - 1; i >= 0; {
		t := toks[i]
		switch {
		case t.Text == ")" || t.Text == ">" || t.Text == "]":
			open := openOf(toks, i)
			if open < 0 {
				return test, expects
			}
			args = nil
			if t.Text == ")" {
				args = toks[open+1 : i]
			}
			i = open - 1
		case t.Kind == javasrc.TokIdent:
			j := i // the first identifier of a qualified name a.b.Name
			for j >= 2 && toks[j-1].Text == "." && toks[j-2].Kind == javasrc.TokIdent {
				j -= 2
			}
			if j >= 1 && toks[j-1].Text == "@" {
				if t.Text == "Test" {
					test, expects = true, hasAttr(args, "expectedExceptions") || hasAttr(args, "expected")
				}
				j--
			}
			args = nil
			i = j - 1
		default:
			return test, expects // the end of the previous member
		}
	}
	return test, expects
}

// hasAttr says whether an annotation's top-level arguments set name = ....
func hasAttr(args []javasrc.Token, name string) bool {
	depth := 0
	for k, t := range args {
		switch t.Text {
		case "(", "{", "[":
			depth++
		case ")", "}", "]":
			depth--
		case name:
			if depth == 0 && k+1 < len(args) && args[k+1].Text == "=" {
				return true
			}
		}
	}
	return false
}

// openOf is the index of the token opening the bracket closed at toks[i] (")", ">" or "]"), -1 when unbalanced.
func openOf(toks []javasrc.Token, i int) int {
	closer := toks[i].Text
	opener := map[string]string{")": "(", ">": "<", "]": "["}[closer]
	depth := 0
	for j := i; j >= 0; j-- {
		switch toks[j].Text {
		case closer:
			depth++
		case opener:
			if depth--; depth == 0 {
				return j
			}
		}
	}
	return -1
}

// closeOf is the index of the ")" closing the "(" at toks[i], len(toks) when it is not closed.
func closeOf(toks []javasrc.Token, i int) int {
	depth := 0
	for j := i; j < len(toks); j++ {
		switch toks[j].Text {
		case "(":
			depth++
		case ")":
			if depth--; depth == 0 {
				return j
			}
		}
	}
	return len(toks)
}

// split is the top-level comma-separated arguments of a call's argument tokens.
func split(args []javasrc.Token) [][]javasrc.Token {
	if len(args) == 0 {
		return nil
	}
	var out [][]javasrc.Token
	depth, start := 0, 0
	for k, t := range args {
		switch t.Text {
		case "(", "{", "[":
			depth++
		case ")", "}", "]":
			depth--
		case ",":
			if depth == 0 {
				out = append(out, args[start:k])
				start = k + 1
			}
		}
	}
	return append(out, args[start:])
}

// skipFirst is the text of the body's first statement when it always skips the test: assumeTrue(false) or
// assumeFalse(true) (any qualifier, one more argument allowed for a message), or throw new SkipException(...)
// (any qualifier); "" otherwise.
func skipFirst(body []javasrc.Token) string {
	if len(body) >= 3 && body[0].Text == "throw" && body[1].Text == "new" {
		name, j := qualified(body, 2)
		if last(name) == "SkipException" && j < len(body) && body[j].Text == "(" {
			return "throw new " + last(name)
		}
		return ""
	}
	name, j := qualified(body, 0)
	want := map[string]string{"assumeTrue": "false", "assumeFalse": "true"}[last(name)]
	if want == "" || j >= len(body) || body[j].Text != "(" {
		return ""
	}
	end := closeOf(body, j)
	if end+1 >= len(body) || body[end+1].Text != ";" {
		return ""
	}
	args := split(body[j+1 : end])
	if len(args) > 2 {
		return ""
	}
	for _, a := range args {
		if len(a) == 1 && a[0].Kind == javasrc.TokIdent && a[0].Text == want {
			return name + "(" + want + ")"
		}
	}
	return ""
}

// qualified reads the dotted name a.b.c starting at toks[i]: its text and the index after it ("" when toks[i] is
// no identifier).
func qualified(toks []javasrc.Token, i int) (string, int) {
	if i >= len(toks) || toks[i].Kind != javasrc.TokIdent {
		return "", i
	}
	name, j := toks[i].Text, i+1
	for j+1 < len(toks) && toks[j].Text == "." && toks[j+1].Kind == javasrc.TokIdent {
		name += "." + toks[j+1].Text
		j += 2
	}
	return name, j
}

func last(name string) string { return name[strings.LastIndex(name, ".")+1:] }

// isAssertion says whether a called name is an assertion: assert*, verify*, expect* or fail.
func isAssertion(name string) bool {
	return name == "fail" || strings.HasPrefix(name, "assert") || strings.HasPrefix(name, "verify") || strings.HasPrefix(name, "expect")
}

// assertions counts the calls in body: real assertions (with a new AssertionError and, when scope is set, an
// unqualified or this./super. call of a helper scope resolves whose own body asserts) and trivially true ones.
func assertions(body []javasrc.Token, scope *helpers) (real, trivial int) {
	for k := 0; k+1 < len(body); k++ {
		t := body[k]
		if t.Kind != javasrc.TokIdent || body[k+1].Text != "(" {
			continue
		}
		prev := ""
		if k > 0 {
			prev = body[k-1].Text
		}
		switch {
		case prev == "new":
			if t.Text == "AssertionError" {
				real++
			}
		case isAssertion(t.Text):
			if trivialCall(t.Text, split(body[k+2:closeOf(body, k+1)])) {
				trivial++
			} else {
				real++
			}
		case scope != nil && (prev != "." || (k >= 2 && (body[k-2].Text == "this" || body[k-2].Text == "super"))):
			if scope.asserts(t.Text) {
				real++
			}
		}
	}
	return real, trivial
}

// trivialCall says whether an assertion call cannot fail: assertTrue(true) or assertFalse(false), or
// assertEquals/assertSame of one literal with itself, each with one string-literal message argument allowed
// first (JUnit) or last (TestNG).
func trivialCall(name string, args [][]javasrc.Token) bool {
	switch name {
	case "assertTrue", "assertFalse":
		want := map[string]string{"assertTrue": "true", "assertFalse": "false"}[name]
		switch len(args) {
		case 1:
			return isWord(args[0], want)
		case 2:
			return (isWord(args[0], want) && isMessage(args[1])) || (isMessage(args[0]) && isWord(args[1], want))
		}
	case "assertEquals", "assertSame":
		switch len(args) {
		case 2:
			return sameLiteral(args[0], args[1])
		case 3:
			return (sameLiteral(args[0], args[1]) && isConstant(args[2])) || (isMessage(args[0]) && sameLiteral(args[1], args[2]))
		}
	}
	return false
}

func isWord(a []javasrc.Token, w string) bool {
	return len(a) == 1 && a[0].Kind == javasrc.TokIdent && a[0].Text == w
}

func isMessage(a []javasrc.Token) bool { return len(a) == 1 && a[0].Kind == javasrc.TokLiteral }

// isConstant is one literal token: a string or char, a number, true, false or null.
func isConstant(a []javasrc.Token) bool {
	if len(a) != 1 {
		return false
	}
	switch t := a[0]; t.Kind {
	case javasrc.TokLiteral, javasrc.TokNumber:
		return true
	case javasrc.TokIdent:
		return t.Text == "true" || t.Text == "false" || t.Text == "null"
	}
	return false
}

func sameLiteral(a, b []javasrc.Token) bool {
	return isConstant(a) && isConstant(b) && a[0].Text == b[0].Text
}
