package nullcontext

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// request is a code shape that receives a user request: a file holding one is never station-internal.
var request = regexp.MustCompile(`extends[[:space:]]+BWebServlet|HttpServletRequest|HttpServletResponse|` +
	`[^A-Za-z0-9_]WebOp[^A-Za-z0-9_]|@NiagaraRpc|void[[:space:]]+(doGet|doPost|service)[[:space:]]*\(`)

// receiver is the simple name (or this) a write is called on: the text before ".set(" ends in it, with no
// "." before it (t.point.set is a field access; foo().set ends in no name).
var receiver = regexp.MustCompile(`(^|[^A-Za-z0-9_$.[:space:]])[[:space:]]*([A-Za-z_$][A-Za-z0-9_$]*)[[:space:]]*$`)

// seeds decides whether a null-Context write is a station-internal seed of the component's own child.
type seeds struct {
	serves bool // the file receives a user request somewhere
	toks   []javasrc.Token
	decls  []javasrc.Decl
}

func newSeeds(code []string) *seeds {
	s := &seeds{toks: javasrc.Tokens(code)}
	s.decls = javasrc.Decls(s.toks)
	for _, ln := range code {
		s.serves = s.serves || request.MatchString(ln)
	}
	return s
}

// internal reports whether the null-Context write on 0-based line i, whose code before ".set(" is before
// and whose arguments but the Context are values, seeds a component this code owns, with no value a caller
// supplies: the file serves no request (request); the write is in a method or constructor; it is called on
// this, or on a local name every assignment of which in that method (a declaration of the same name in a
// block that does not hold the write is another variable) is (Cast) get(...) or this.get(...)
// (an own child), new X(...) or X.TYPE.getInstance() (a component the method creates), never a parameter;
// and no value names a parameter of the method. Such a write has no user to attribute (types/security.md
// §1.1 targets a servlet/user write; a station-internal write is the documented null-Context norm,
// types/actions.md fire(..., null)). What it cannot prove stays a row: a receiver that is a parameter, a
// field, a field access or a call result, a value from a parameter, any write in a request-serving file.
func (s *seeds) internal(i int, before string, values []string) bool {
	m := receiver.FindStringSubmatch(before)
	if s.serves || m == nil {
		return false
	}
	name := m[2]
	d, ok := s.enclosing(i)
	if !ok {
		return false
	}
	params := s.params(d)
	for _, v := range values {
		for _, t := range javasrc.Tokens([]string{v}) {
			if t.Kind == javasrc.TokIdent && params[t.Text] {
				return false
			}
		}
	}
	if name == "this" {
		return true
	}
	if params[name] {
		return false
	}
	owned := false
	body := s.toks[d.Body[0]:d.Body[1]]
	w := 0 // the write's "." token in body: the tokens of its line before it, from the line's first token
	for w < len(s.toks) && s.toks[w].Line < i {
		w++
	}
	w += len(javasrc.Tokens([]string{before})) - d.Body[0]
	for k := 0; k+1 < len(body); k++ {
		if body[k].Text != name || body[k+1].Text != "=" || (k+2 < len(body) && body[k+2].Text == "=") ||
			(k > 0 && body[k-1].Text == ".") {
			continue
		}
		if k > 0 && declares(body[k-1]) && !inBlock(body, k, w) {
			continue // a declaration of another variable of that name, in a block that does not hold the write
		}
		if !ownedValue(body[k+2:]) {
			return false
		}
		owned = true
	}
	return owned
}

// declares reports whether t, right before "name =", makes it a declaration: a type name, a generic's ">"
// or an array's "]".
func declares(t javasrc.Token) bool {
	return t.Kind == javasrc.TokIdent && t.Text != "return" || t.Text == ">" || t.Text == "]"
}

// inBlock reports whether the innermost brace block of body holding token k also holds token w (true when
// k is in no block of body: the method's own block holds every token).
func inBlock(body []javasrc.Token, k, w int) bool {
	depth := 0
	for o := k - 1; o >= 0; o-- {
		switch body[o].Text {
		case "}":
			depth++
		case "{":
			if depth > 0 {
				depth--
				continue
			}
			if w <= o {
				return false
			}
			for c, d := o+1, 0; c < len(body); c++ {
				switch body[c].Text {
				case "{":
					d++
				case "}":
					if d == 0 {
						return w < c
					}
					d--
				}
			}
			return true
		}
	}
	return true
}

// ownedValue reports whether the expression starting at e (after "name =") is (Cast) get(, this.get(,
// new X or X.TYPE.getInstance(, a cast before any of them allowed.
func ownedValue(e []javasrc.Token) bool {
	k := 0
	if k < len(e) && e[k].Text == "(" { // a cast: (Type) or (a.b.Type)
		for k++; k < len(e) && e[k].Text != ")"; k++ {
		}
		k++
	}
	text := func(n int) string {
		var b strings.Builder
		for j := k; j < len(e) && j < k+n; j++ {
			b.WriteString(e[j].Text)
		}
		return b.String()
	}
	switch {
	case text(2) == "get(", text(4) == "this.get(", k < len(e) && e[k].Text == "new":
		return true
	}
	for j := k; j+4 < len(e) && e[j].Text != ";"; j++ {
		if e[j].Text == "." && e[j+1].Text == "TYPE" && e[j+2].Text == "." && e[j+3].Text == "getInstance" && e[j+4].Text == "(" {
			return j == k+1
		}
	}
	return false
}

// enclosing is the innermost method or constructor whose body holds 0-based line i.
func (s *seeds) enclosing(i int) (javasrc.Decl, bool) {
	var in javasrc.Decl
	ok := false
	for _, d := range s.decls {
		if d.Body[0] < d.Body[1] && s.toks[d.Body[0]].Line <= i && i <= s.toks[d.Body[1]-1].Line && (!ok || d.Body[0] > in.Body[0]) {
			in, ok = d, true
		}
	}
	return in, ok
}

// params is the set of d's parameter names: the name before each "," or the closing ")" at depth 1 of the
// parameter list that follows the declaration's name.
func (s *seeds) params(d javasrc.Decl) map[string]bool {
	out := map[string]bool{}
	k := d.Body[0] - 1
	for ; k >= 0 && !(s.toks[k].Text == d.Name && s.toks[k].Line == d.Line); k-- {
	}
	if k < 0 || k+1 >= len(s.toks) || s.toks[k+1].Text != "(" {
		return out
	}
	depth := 0
	for j := k + 1; j < d.Body[0]; j++ {
		switch s.toks[j].Text {
		case "(", "<":
			depth++
		case ")", ">":
			if s.toks[j].Text == ")" && depth == 1 && s.toks[j-1].Kind == javasrc.TokIdent {
				out[s.toks[j-1].Text] = true
			}
			if depth--; depth == 0 {
				return out
			}
		case ",":
			if depth == 1 && s.toks[j-1].Kind == javasrc.TokIdent {
				out[s.toks[j-1].Text] = true
			}
		}
	}
	return out
}
