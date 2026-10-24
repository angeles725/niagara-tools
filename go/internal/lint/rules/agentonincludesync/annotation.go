package agentonincludesync

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// typeDecl is one @NiagaraType type that declares agent on types: its binary name (nested types joined with "$"),
// the 1-based line of its name, and its declared on types in source order.
type typeDecl struct {
	class string
	line  int
	types []string
}

// scope is an enclosing type declaration: its simple name and the brace depth its body opened at.
type scope struct {
	name  string
	depth int
}

// agentTypes reads every @NiagaraType with string-literal @AgentOn types and the type declaration it annotates. One
// pass keeps the package name, the enclosing type names (by brace depth) and the on types of the last @NiagaraType,
// which the next type declaration takes; an annotation is skipped whole, so its array braces are not scopes.
func agentTypes(toks []javasrc.Token) []typeDecl {
	var (
		out          []typeDecl
		outer        []scope
		pkg, pending string
		depth        int
		types        []string
	)
	for k := 0; k < len(toks); k++ {
		t := toks[k]
		switch {
		case t.Text == "@":
			name, end := annotation(toks, k)
			if name == "NiagaraType" {
				types = agentOnTypes(toks[k:end])
			}
			k = end - 1
		case t.Text == "{":
			depth++
			if pending != "" {
				outer, pending = append(outer, scope{pending, depth}), ""
			}
		case t.Text == "}":
			if n := len(outer); n > 0 && outer[n-1].depth == depth {
				outer = outer[:n-1]
			}
			depth--
		case t.Kind == javasrc.TokIdent && t.Text == "package" && pkg == "":
			for k++; k < len(toks) && toks[k].Text != ";"; k++ {
				pkg += toks[k].Text
			}
		case typeKeyword(toks, k):
			pending = toks[k+1].Text
			if len(types) > 0 {
				out = append(out, typeDecl{class: binaryName(pkg, outer, pending), line: toks[k+1].Line + 1, types: types})
			}
			types = nil
		}
	}
	return out
}

// typeKeyword reports a type declaration at k: class, interface or enum followed by its name (never X.class).
func typeKeyword(toks []javasrc.Token, k int) bool {
	switch toks[k].Text {
	case "class", "interface", "enum":
		return toks[k].Kind == javasrc.TokIdent && (k == 0 || toks[k-1].Text != ".") &&
			k+1 < len(toks) && toks[k+1].Kind == javasrc.TokIdent
	}
	return false
}

// binaryName is the class attribute module-include.xml uses: pkg.Outer$Inner.
func binaryName(pkg string, outer []scope, name string) string {
	for i := len(outer) - 1; i >= 0; i-- {
		name = outer[i].name + "$" + name
	}
	if pkg != "" {
		name = pkg + "." + name
	}
	return name
}

// annotation reads the annotation whose "@" is at k: its simple name (the last part of a qualified one) and the
// index just past it (past its balanced argument list when one follows). "@interface" is no annotation: its name is
// empty and only the "@" is consumed.
func annotation(toks []javasrc.Token, k int) (string, int) {
	name, j := "", k+1
	for j < len(toks) && toks[j].Kind == javasrc.TokIdent && toks[j].Text != "interface" {
		name = toks[j].Text
		if j++; j+1 < len(toks) && toks[j].Text == "." && toks[j+1].Kind == javasrc.TokIdent {
			j++
			continue
		}
		break
	}
	if name != "" && j < len(toks) && toks[j].Text == "(" {
		return name, closing(toks, j) + 1
	}
	return name, j
}

// closing is the index of the ")" pairing with the "(" at k, or the last index when it never closes.
func closing(toks []javasrc.Token, k int) int {
	depth := 0
	for j := k; j < len(toks); j++ {
		switch toks[j].Text {
		case "(":
			depth++
		case ")":
			if depth--; depth == 0 {
				return j
			}
		}
	}
	return len(toks) - 1
}

// agentOnTypes is every string-literal types value of each @AgentOn in a @NiagaraType's tokens, in source order.
func agentOnTypes(ann []javasrc.Token) []string {
	var out []string
	for k := 1; k < len(ann); k++ {
		if ann[k].Text != "@" {
			continue
		}
		name, end := annotation(ann, k)
		if name != "AgentOn" {
			continue
		}
		args := ann[k:end]
		take := func(t javasrc.Token) {
			if t.Kind == javasrc.TokLiteral && strings.HasPrefix(t.Text, `"`) && len(t.Text) >= 2 {
				out = append(out, javasrc.Unescape(t.Text[1:len(t.Text)-1]))
			}
		}
		for j := 0; j+2 < len(args); j++ {
			if args[j].Text != "types" || args[j+1].Text != "=" {
				continue
			}
			if v := j + 2; args[v].Text != "{" {
				take(args[v])
				continue
			}
			for v := j + 3; v < len(args) && args[v].Text != "}"; v++ {
				take(args[v])
			}
		}
		k = end - 1
	}
	return out
}
