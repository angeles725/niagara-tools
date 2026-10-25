package agentonincludesync

import "github.com/angeles725/niagara-tools/go/internal/javasrc"

// AgentType is one @NiagaraType type that declares string-literal @AgentOn types: its binary name (pkg.Outer$Inner),
// the 1-based line of its name and its on types in source order.
type AgentType struct {
	Class string
	Line  int
	Types []string
}

// AgentTypes is the reader this rule runs (agentTypes), for other checks that need a type's @AgentOn targets, such as
// verify-module's driver mode. toks is javasrc.Tokens of the Strip view.
func AgentTypes(toks []javasrc.Token) []AgentType {
	var out []AgentType
	for _, d := range agentTypes(toks) {
		out = append(out, AgentType{Class: d.class, Line: d.line, Types: d.types})
	}
	return out
}
