package vacuoustests

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// class is one class declared in a srcTest/ source: its tokens, its declaration, the class whose body holds it (nil
// at top level) and the methods declared directly in it.
type class struct {
	path  string
	toks  []javasrc.Token
	c     javasrc.Class
	outer *class
	decls []javasrc.Decl
	// statics are the file's `import static` declarations, shared by every class of the file
	statics []staticImport
}

// staticImport is one `import static <pkg>.<Class>.<member>;` (member "" for `.*`), the class by simple name.
type staticImport struct{ class, member string }

// helpers is where an unqualified helper call of a test may resolve: the classes of scope, and the members the
// test's file imports statically from a srcTest/ class.
type helpers struct {
	idx     *index
	classes []*class
	statics []staticImport
}

// method is one declared method and the class that owns it, in source order.
type method struct {
	owner *class
	d     javasrc.Decl
}

// index is every srcTest/ class by simple name and every method in file and source order.
type index struct {
	bySimple map[string][]*class
	methods  []method
}

// newIndex scans every file; one that ends inside a comment or a text block, or whose class body is not closed,
// is not indexed and is named in bad.
func newIndex(files []*javasrc.File) (*index, []string) {
	idx := &index{bySimple: map[string][]*class{}}
	var bad []string
	for _, f := range files {
		switch f.End() {
		case javasrc.OpenComment:
			bad = append(bad, "cannot scan "+f.Path+": it ends inside a /* comment")
			continue
		case javasrc.OpenTextBlock:
			bad = append(bad, "cannot scan "+f.Path+`: it ends inside a """ text block`)
			continue
		}
		if err := idx.add(f.Path, javasrc.Tokens(f.Stripped())); err != nil {
			bad = append(bad, err.Error())
		}
	}
	return idx, bad
}

// add indexes one file's classes and methods; a class body the scanner did not close refuses the whole file.
func (idx *index) add(path string, toks []javasrc.Token) error {
	var owned []*class
	statics := staticImports(toks)
	for _, c := range javasrc.Classes(toks) {
		if c.Body[1] >= len(toks) {
			return fmt.Errorf("cannot scan %s: the body of %s is not closed", path, c.Name)
		}
		k := &class{path: path, toks: toks, c: c, statics: statics}
		for i := len(owned) - 1; i >= 0; i-- { // the innermost earlier class whose body holds it
			if o := owned[i].c; o.Body[0] <= c.At && c.At < o.Body[1] {
				k.outer = owned[i]
				break
			}
		}
		owned = append(owned, k)
	}
	var methods []method
	for _, d := range javasrc.Decls(toks) {
		if d.Anonymous || d.Body[1] >= len(toks) {
			continue
		}
		if k := innermost(owned, d.Body[0]); k != nil {
			k.decls = append(k.decls, d)
			methods = append(methods, method{k, d})
		}
	}
	for _, k := range owned {
		idx.bySimple[k.c.Name] = append(idx.bySimple[k.c.Name], k)
	}
	idx.methods = append(idx.methods, methods...)
	return nil
}

// innermost is the innermost class whose body holds token off, nil when none does.
func innermost(classes []*class, off int) *class {
	var in *class
	for _, k := range classes {
		if k.c.Body[0] <= off && off < k.c.Body[1] && (in == nil || k.c.Body[0] > in.c.Body[0]) {
			in = k
		}
	}
	return in
}

// staticImports reads every `import static a.b.C.m;` and `import static a.b.C.*;` of a file's tokens.
func staticImports(toks []javasrc.Token) []staticImport {
	var out []staticImport
	for k := 0; k+2 < len(toks); k++ {
		if toks[k].Text != "import" || toks[k+1].Text != "static" || (k > 0 && toks[k-1].Text != ";" && toks[k-1].Text != "}") {
			continue
		}
		name, j := qualified(toks, k+2)
		switch {
		case j+1 < len(toks) && toks[j].Text == "." && toks[j+1].Text == "*":
			out = append(out, staticImport{class: last(name)})
		case strings.Contains(name, "."):
			cls := name[:strings.LastIndex(name, ".")]
			out = append(out, staticImport{class: last(cls), member: last(name)})
		}
	}
	return out
}

// scope is where a test of k resolves an unqualified helper call: k, its enclosing classes, the srcTest/
// superclasses of each (every class of that simple name; a cycle stops), and the file's static imports.
func (idx *index) scope(k *class) *helpers {
	return &helpers{idx: idx, classes: idx.classes(k), statics: k.statics}
}

// classes is k, its enclosing classes and the srcTest/ superclasses of each.
func (idx *index) classes(k *class) []*class {
	var out []*class
	seen := map[*class]bool{}
	var visit func(c *class)
	visit = func(c *class) {
		if c == nil || seen[c] {
			return
		}
		seen[c] = true
		out = append(out, c)
		if c.c.Extends != "" {
			for _, s := range idx.bySimple[c.c.Extends] {
				visit(s)
			}
		}
	}
	for c := k; c != nil; c = c.outer {
		visit(c)
	}
	return out
}

// asserts says whether a method named name that h resolves holds a real assertion in its own body (one level
// deep): one of h's classes, or a srcTest/ class whose member name the file imports statically.
func (h *helpers) asserts(name string) bool {
	if declAsserts(h.classes, name) {
		return true
	}
	for _, s := range h.statics {
		if (s.member == "" || s.member == name) && declAsserts(h.idx.bySimple[s.class], name) {
			return true
		}
	}
	return false
}

// declAsserts says whether a method named name declared in one of classes holds a real assertion in its body.
func declAsserts(classes []*class, name string) bool {
	for _, c := range classes {
		for _, d := range c.decls {
			if d.Name == name {
				if real, _ := assertions(c.toks[d.Body[0]:d.Body[1]], nil); real > 0 {
					return true
				}
			}
		}
	}
	return false
}
