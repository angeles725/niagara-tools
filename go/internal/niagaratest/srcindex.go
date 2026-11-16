package niagaratest

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/agentonincludesync"
)

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
