// Package structuraltests is lint-structural-tests (G34): a srcTest file that reads a .java source as text, in a
// part whose srcTest has no station test. Such a test pins the shape of the source, not the component's behaviour
// (types/moduleTest.md: structural tests are the last resort). The contract is the deferred G34 spec
// (build-n4-module-kit/ledger.d/2026-10-23-r8-p1-tests.md, C41); toolbelt/lint-structural-tests.sh is the wrapper
// and its header is the user-facing contract.
//
// It is a lint.TreeRule over each module root: every *.java under a srcTest/ (build/ not entered), read as javasrc
// tokens, so a comment is never a read. A part is the directory holding srcTest/. A row is each string literal
// ending in ".java" inside the arguments of Files.readAllBytes, Files.readAllLines, Files.readString or
// Files.lines, or of a Paths.get or Path.of call, in a part whose srcTest holds no class extending BTestNg or
// BTestNgStation (its simple or qualified superclass name).
package structuraltests

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name n4lint accepts and every row prints in its check column.
const ID = "lint-structural-tests"

// Spec registers the rule: [--strict] [--] <module-root>... (each root on its own), WARN rows printed FAIL under
// --strict (exit 1), left out of n4lint --all (a module root).
func Spec() lint.Spec {
	return lint.Spec{Tree: rule{}, Mode: lint.Each, ModuleRoot: true, StrictFails: true,
		Usage:      "usage: lint-structural-tests.sh [--strict] <module-root>...",
		Select:     lint.Selection{Names: []string{"*.java"}, Paths: []string{"*/srcTest/*"}, Prune: []string{"build"}},
		EmptyError: lint.EmptyRow{Msg: "no *.java under a srcTest/"}}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

// readers are the java.nio.file.Files methods that read a file's text, and paths the calls that build a path.
var (
	readers = map[string]bool{"readAllBytes": true, "readAllLines": true, "readString": true, "lines": true}
	paths   = map[string]string{"Paths": "get", "Path": "of"}
)

type source struct {
	path string
	toks []javasrc.Token
}

// CheckTree groups the srcTest sources by part, then reports the .java reads of each part with no station test.
func (rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	var order []string
	srcs := map[string][]source{}
	classes := map[string][]javasrc.Class{}
	for _, f := range files {
		i := strings.LastIndex(f.Path, "/srcTest/")
		if i < 0 {
			continue
		}
		part := f.Path[:i]
		if _, ok := srcs[part]; !ok {
			order = append(order, part)
		}
		toks := javasrc.Tokens(f.Stripped())
		srcs[part] = append(srcs[part], source{f.Path, toks})
		classes[part] = append(classes[part], javasrc.Classes(toks)...)
	}
	if len(order) == 0 {
		return nil, lint.ErrEmptyRoot
	}
	var out []lint.Finding
	for _, part := range order {
		if len(javasrc.Descendants(classes[part], "BTestNg", "BTestNgStation")) > 0 {
			continue
		}
		name := part[strings.LastIndexByte(part, '/')+1:] + "/srcTest"
		for _, s := range srcs[part] {
			for _, lit := range javaReads(s.toks) {
				out = append(out, lint.Finding{Path: s.path, Line: lit.Line + 1, Msg: "reads " + lit.Text + " as text and " + name +
					" has no BTestNg or BTestNgStation class: test the behaviour in a station test (types/moduleTest.md)"})
			}
		}
	}
	return out, nil
}

// javaReads lists, in source order and once each, the ".java" literals inside the arguments of a Files reader or
// a path-building call.
func javaReads(toks []javasrc.Token) []javasrc.Token {
	hit := make([]bool, len(toks))
	for k := 2; k+1 < len(toks); k++ {
		if toks[k+1].Text != "(" || toks[k-1].Text != "." {
			continue
		}
		owner, method := toks[k-2].Text, toks[k].Text
		if !(owner == "Files" && readers[method]) && paths[owner] != method {
			continue
		}
		depth := 0
		for j := k + 1; j < len(toks); j++ {
			switch t := toks[j]; {
			case t.Text == "(":
				depth++
			case t.Text == ")":
				depth--
			case t.Kind == javasrc.TokLiteral && strings.HasSuffix(t.Text, `.java"`):
				hit[j] = true
			}
			if depth == 0 {
				break
			}
		}
	}
	var out []javasrc.Token
	for j, h := range hit {
		if h {
			out = append(out, toks[j])
		}
	}
	return out
}
