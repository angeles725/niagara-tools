package niagaratest

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

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
