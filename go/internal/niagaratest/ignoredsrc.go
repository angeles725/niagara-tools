package niagaratest

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/agentonincludesync"
	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// The source half of the ignored gate. testng-results.xml counts the ignored test-methods but never names them
// (ignored.go), so run names them from the source: every <type class> of <part>/moduleTest-include.xml is looked up
// among the classes declared under <part>/srcTest/ (the shared javasrc scanner, binary names as the
// test-type-in-prod lint derives them), and each method-level @Test(enabled = false) a registered concrete class
// declares or inherits through srcTest/ superclasses is one ignored method, named <SimpleName>.<method> (the class
// TestNG runs it on). How each case counts:
//   - an abstract registered type adds nothing itself: TestNG runs its methods on the concrete subclasses;
//   - an inherited method counts on each registered concrete subclass, unless a nearer class declares a method of
//     that name (the nearest declaration decides);
//   - a superclass outside srcTest/ (BTestNg, a library) is not read: a disable there makes the source and TestNG
//     counts disagree, which FAILs, and the count fallback (--allow-ignored-count) is the way out;
//   - a class-level @Test(enabled = false) and an enabled value that is no true/false literal are refused, as is a
//     registered source the scanner cannot read to its end: the scan never guesses (run exit 3).
//
// JUnit's @Ignore is no TestNG disable and is not read.

// maxSourceBytes is the largest moduleTest-include.xml or srcTest/*.java the scan reads; a larger one is refused.
const maxSourceBytes = 4 << 20

const testIncludeName = "moduleTest-include.xml"

// srcClass is one class declared under srcTest/: its file, the file's tokens, the declaration and its methods
// (the method and constructor bodies declared directly in it).
type srcClass struct {
	file  string
	toks  []javasrc.Token
	c     javasrc.Class
	decls []javasrc.Decl
}

// srcIndex is every class declared under a part's srcTest/, by binary name and by simple name, and the files the
// scanner could not read to their end (path -> why).
type srcIndex struct {
	byBinary map[string]*srcClass
	bySimple map[string][]*srcClass
	bad      map[string]string
}

// disabledInSource is the sorted list of the ignored test-methods the registered types of part (a part directory)
// declare, one entry per method TestNG ignores. A part with no moduleTest-include.xml registers none.
func disabledInSource(part string) ([]string, error) {
	inc := filepath.Join(part, testIncludeName)
	if !lint.IsFile(inc) {
		return nil, nil
	}
	data, err := readSource(inc)
	if err != nil {
		return nil, err
	}
	doc, err := xmlsrc.Parse(inc, data)
	if err != nil {
		return nil, err
	}
	var registered []string
	for _, i := range doc.Find("type") {
		if class, ok := doc.Elements()[i].Attr("class"); ok && !slices.Contains(registered, class) {
			registered = append(registered, class)
		}
	}
	if len(registered) == 0 {
		return nil, nil
	}
	idx, err := indexSrcTest(filepath.Join(part, "srcTest"))
	if err != nil {
		return nil, err
	}
	simple := map[string]string{}
	var out []string
	for _, class := range registered {
		short := class[strings.LastIndex(class, ".")+1:]
		if other, dup := simple[short]; dup {
			return nil, fmt.Errorf("registered types %s and %s share the simple name %s: their methods cannot be named <Class>.<method>", other, class, short)
		}
		simple[short] = class
		names, err := idx.disabled(class, short, part)
		if err != nil {
			return nil, fmt.Errorf("cannot name the disabled test-methods of %s: %v", class, err)
		}
		out = append(out, names...)
	}
	slices.Sort(out)
	return out, nil
}

// readSource reads a file of at most maxSourceBytes; the read stops at the cap whatever its size says.
func readSource(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxSourceBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	if len(data) > maxSourceBytes {
		return nil, fmt.Errorf("%s: larger than %d MiB, not read", path, maxSourceBytes>>20)
	}
	return data, nil
}

// indexSrcTest scans every *.java under dir (build/ not entered); a file that ends inside a comment or a text
// block is recorded as bad, not indexed.
func indexSrcTest(dir string) (*srcIndex, error) {
	idx := &srcIndex{byBinary: map[string]*srcClass{}, bySimple: map[string][]*srcClass{}, bad: map[string]string{}}
	if !lint.IsDir(dir) {
		return idx, nil
	}
	files, err := lint.Walk(dir, lint.Selection{Names: []string{"*.java"}, Prune: []string{"build"}})
	if err != nil {
		return nil, err
	}
	for _, p := range files {
		data, err := readSource(p)
		if err != nil {
			return nil, err
		}
		lines, _ := javasrc.Lines(bytes.NewReader(data)) // a bytes.Reader never fails
		f := javasrc.NewFile(p, lines)
		switch f.End() {
		case javasrc.OpenComment:
			idx.bad[p] = "ends inside a /* comment"
			continue
		case javasrc.OpenTextBlock:
			idx.bad[p] = `ends inside a """ text block`
			continue
		}
		toks := javasrc.Tokens(f.Stripped())
		classes := javasrc.Classes(toks)
		binary := agentonincludesync.TestClasses(toks)
		owned := make([]*srcClass, len(classes))
		for i, c := range classes {
			owned[i] = &srcClass{file: p, toks: toks, c: c}
			idx.byBinary[binary[i]] = owned[i]
			idx.bySimple[c.Name] = append(idx.bySimple[c.Name], owned[i])
		}
		for _, d := range javasrc.Decls(toks) {
			if d.Anonymous {
				continue
			}
			if i := innermost(classes, d.Body[0]); i >= 0 {
				owned[i].decls = append(owned[i].decls, d)
			}
		}
	}
	return idx, nil
}

// innermost is the index of the innermost class whose body holds token off, -1 when none does.
func innermost(classes []javasrc.Class, off int) int {
	in := -1
	for i, c := range classes {
		if c.Body[0] <= off && off < c.Body[1] && (in < 0 || c.Body[0] > classes[in].Body[0]) {
			in = i
		}
	}
	return in
}

// disabled names the ignored methods of the registered class (binary name) on its simple name short.
func (idx *srcIndex) disabled(class, short, part string) ([]string, error) {
	sc, ok := idx.byBinary[class]
	if !ok {
		conv := filepath.Join(part, "srcTest", filepath.FromSlash(strings.ReplaceAll(strings.SplitN(class, "$", 2)[0], ".", "/"))+".java")
		if why, bad := idx.bad[conv]; bad {
			return nil, fmt.Errorf("%s %s", conv, why)
		}
		if len(idx.bad) > 0 {
			return nil, fmt.Errorf("no declaration under srcTest/, and the scanner cannot read %s", idx.badList())
		}
		return nil, fmt.Errorf("no declaration under %s", filepath.Join(part, "srcTest"))
	}
	if sc.c.Kind != "class" || hasModifier(sc.toks, sc.c.At, "abstract") {
		return nil, nil
	}
	var out []string
	nearer := map[string]bool{} // method names a nearer class declares
	seen := map[*srcClass]bool{}
	for cur := sc; cur != nil; {
		if err := cur.check(); err != nil {
			return nil, err
		}
		seen[cur] = true
		for _, d := range cur.decls {
			if nearer[d.Name] {
				continue
			}
			off, err := cur.disabledAt(d)
			if err != nil {
				return nil, err
			}
			if off {
				out = append(out, short+"."+d.Name)
			}
		}
		for _, d := range cur.decls {
			nearer[d.Name] = true
		}
		next, err := idx.parent(cur)
		if err != nil {
			return nil, err
		}
		if seen[next] {
			break // a cycle: javac refuses it, the scan stops
		}
		cur = next
	}
	return out, nil
}

// check refuses a class whose body the scanner did not close and a class-level @Test(enabled = false).
func (sc *srcClass) check() error {
	if sc.c.Body[1] >= len(sc.toks) {
		return fmt.Errorf("%s: the body of %s is not closed", sc.file, sc.c.Name)
	}
	off, err := testDisabled(sc.toks, sc.c.At)
	switch {
	case err != nil:
		return fmt.Errorf("%s: %s: %v", sc.file, sc.c.Name, err)
	case off:
		return fmt.Errorf("%s: class-level @Test(enabled = false) on %s disables every public method, which the scan does not "+
			"name: waive them with --allow-ignored-count <n> --allow-ignored-reason <why>", sc.file, sc.c.Name)
	}
	return nil
}

// disabledAt says whether method d of sc carries a @Test(enabled = false).
func (sc *srcClass) disabledAt(d javasrc.Decl) (bool, error) {
	at := -1
	for i := d.Body[0] - 1; i > sc.c.Body[0]; i-- {
		if t := sc.toks[i]; t.Kind == javasrc.TokIdent && t.Text == d.Name && t.Line == d.Line && sc.toks[i+1].Text == "(" {
			at = i
			break
		}
	}
	if at < 0 {
		return false, fmt.Errorf("%s:%d: cannot find the declaration of %s.%s", sc.file, d.Line+1, sc.c.Name, d.Name)
	}
	off, err := testDisabled(sc.toks, at)
	if err != nil {
		return false, fmt.Errorf("%s:%d: %s.%s: %v", sc.file, d.Line+1, sc.c.Name, d.Name, err)
	}
	return off, nil
}

// parent is the srcTest/ class sc extends, nil when it extends nothing or a class declared outside srcTest/.
func (idx *srcIndex) parent(sc *srcClass) (*srcClass, error) {
	if sc.c.Extends == "" {
		return nil, nil
	}
	switch cs := idx.bySimple[sc.c.Extends]; {
	case len(cs) == 1:
		return cs[0], nil
	case len(cs) > 1:
		return nil, fmt.Errorf("%s: superclass %s of %s is declared %d times under srcTest/", sc.file, sc.c.Extends, sc.c.Name, len(cs))
	case len(idx.bad) > 0:
		return nil, fmt.Errorf("%s: superclass %s of %s is not found, and the scanner cannot read %s", sc.file, sc.c.Extends, sc.c.Name, idx.badList())
	}
	return nil, nil
}

func (idx *srcIndex) badList() string {
	var out []string
	for p, why := range idx.bad {
		out = append(out, p+" ("+why+")")
	}
	slices.Sort(out)
	return strings.Join(out, ", ")
}

// errEnabledValue is an enabled attribute that is no true/false literal: the scan does not evaluate constants.
var errEnabledValue = errors.New("@Test enabled is not a true or false literal")

// testDisabled reads the annotations and modifiers before the declaration token at (a class keyword or a method
// name) and says whether a @Test (any qualifier) among them sets enabled = false.
func testDisabled(toks []javasrc.Token, at int) (bool, error) {
	var args []javasrc.Token // the arguments of the last ")...(" walked over
	for i := at - 1; i >= 0; {
		t := toks[i]
		switch {
		case t.Text == ")" || t.Text == ">" || t.Text == "]":
			open := openOf(toks, i)
			if open < 0 {
				return false, nil
			}
			if t.Text == ")" {
				args = toks[open+1 : i]
			}
			i = open - 1
		case t.Kind == javasrc.TokIdent:
			j := i // the first identifier of a qualified name a.b.Name
			for j >= 2 && toks[j-1].Text == "." && toks[j-2].Kind == javasrc.TokIdent {
				j -= 2
			}
			if j >= 1 && toks[j-1].Text == "@" && t.Text == "Test" {
				if v, found, err := enabledArg(args); err != nil || (found && v == "false") {
					return err == nil, err
				}
			}
			args = nil
			i = j - 1
			if i >= 0 && toks[i].Text == "@" {
				i--
			}
		default:
			return false, nil // the end of the previous member or statement
		}
	}
	return false, nil
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

// enabledArg is the value of the enabled attribute among an annotation's top-level arguments.
func enabledArg(args []javasrc.Token) (string, bool, error) {
	depth := 0
	for k := 0; k < len(args); k++ {
		switch args[k].Text {
		case "(", "{", "[":
			depth++
		case ")", "}", "]":
			depth--
		case "enabled":
			if depth != 0 || k+1 >= len(args) || args[k+1].Text != "=" {
				continue
			}
			if k+2 < len(args) && (args[k+2].Text == "true" || args[k+2].Text == "false") && (k+3 == len(args) || args[k+3].Text == ",") {
				return args[k+2].Text, true, nil
			}
			return "", true, errEnabledValue
		}
	}
	return "", false, nil
}

// hasModifier says whether the keyword mod stands among the modifiers before the declaration token at.
func hasModifier(toks []javasrc.Token, at int, mod string) bool {
	for i := at - 1; i >= 0; i-- {
		switch t := toks[i]; {
		case t.Text == ")":
			if i = openOf(toks, i); i < 0 {
				return false
			}
		case t.Kind == javasrc.TokIdent:
			if t.Text == mod && (i == 0 || (toks[i-1].Text != "@" && toks[i-1].Text != ".")) {
				return true
			}
		case t.Text != "@" && t.Text != ".":
			return false
		}
	}
	return false
}
