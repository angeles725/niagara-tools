// Package exitdoc reads the exit codes a tool declares in Go, so a test can hold its documentation to them
// (retro build-kit-bench-session Δ20).
package exitdoc

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

// Code is one integer constant named Exit* of a Go source file.
type Code struct {
	Name  string
	Value int
}

// Codes lists the Exit* constants of a Go source file that are set to an integer literal, in file order.
func Codes(file string) ([]Code, error) {
	f, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	if err != nil {
		return nil, err
	}
	var out []Code
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			vs := spec.(*ast.ValueSpec)
			for i, name := range vs.Names {
				if i >= len(vs.Values) || !strings.HasPrefix(name.Name, "Exit") {
					continue
				}
				lit, ok := vs.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.INT {
					continue
				}
				if n, err := strconv.Atoi(lit.Value); err == nil {
					out = append(out, Code{name.Name, n})
				}
			}
		}
	}
	return out, nil
}
