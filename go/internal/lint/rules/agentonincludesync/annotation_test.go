package agentonincludesync

import (
	"fmt"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// TestAgentTypes: the @NiagaraType reader on the forms the processor accepts and the lookalikes it must skip.
func TestAgentTypes(t *testing.T) {
	for src, want := range map[string]string{
		`@NiagaraType(agent = @AgentOn(types = "m:A")) class BA {}`:                                                                                                   "[{p.BA 1 [m:A]}]",
		"@NiagaraType(\n agent = @AgentOn(types = {\n  \"m:A\",\n  \"m:B\"\n })\n)\nclass BA {}":                                                                      "[{p.BA 7 [m:A m:B]}]",
		`@NiagaraType(agent = {@AgentOn(types = {"m:A"}), @javax.baja.nre.annotations.AgentOn(app = "x", types = "m:B")}) final class BA {}`:                          "[{p.BA 1 [m:A m:B]}]",
		"// @NiagaraType(agent = @AgentOn(types = \"m:A\"))\n/* @AgentOn(types = \"m:B\") */ @NiagaraType class BA { String s = \"@AgentOn(types = \\\"m:C\\\")\"; }": "[]",
		`@NiagaraType(agent = @AgentOn(types = TYPES)) class BA {}`:                                                                                                   "[]",
		`@NiagaraType(agent = @AgentOn(types = "m:A")) class BO { Class<?> c = X.class; @NiagaraType(agent = @AgentOn(types = "m:B")) static class BI {} }`:           "[{p.BO 1 [m:A]} {p.BO$BI 1 [m:B]}]",
		`@NiagaraType(agent = @AgentOn(types = "m:A")) @interface BAnn {} class BLater {}`:                                                                            "[{p.BAnn 1 [m:A]}]",
	} {
		got := fmt.Sprint(agentTypes(javasrc.Tokens(javasrc.NewFile("A.java", strings.Split("package p; "+src, "\n")).Stripped())))
		if got != want {
			t.Errorf("%q: got %s, want %s", src, got, want)
		}
	}
}
