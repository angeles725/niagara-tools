package verifymodule

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestHeaderListsEveryCheck: the "Checks" block of toolbelt/verify-module.sh's header names every registered
// check, in the order the gate runs them (an entry is a "#   <name> " line; its continuation lines are indented).
func TestHeaderListsEveryCheck(t *testing.T) {
	script, err := os.ReadFile("../../../build-n4-module-kit/toolbelt/verify-module.sh")
	if err != nil {
		t.Fatal(err)
	}
	_, block, _ := strings.Cut(string(script), "\n# Checks ")
	block, _, found := strings.Cut(block, "\n# Usage:")
	if !found {
		t.Fatal("verify-module.sh: no \"# Checks\" block before \"# Usage:\"")
	}
	var doc, code []string
	for _, m := range regexp.MustCompile(`(?m)^#   ([a-z][a-z0-9-]*) `).FindAllStringSubmatch(block, -1) {
		doc = append(doc, m[1])
	}
	for _, c := range checks {
		code = append(code, c.name)
	}
	if got, want := strings.Join(doc, " "), strings.Join(code, " "); got != want {
		t.Errorf("verify-module.sh header checks:\n %s\nregistered checks (checks.go):\n %s", got, want)
	}
}

// TestUsageStartsAtItsOwnFirstLine (retro build-kit-bench-session Δ15): the usage a bare `verify-module.sh` prints
// opens with the tool's own title, not with a shellcheck directive and the rationale for it.
func TestUsageStartsAtItsOwnFirstLine(t *testing.T) {
	if !strings.HasPrefix(usage, "verify-module.sh — ") {
		t.Errorf("usage opens with %q, want the tool's title line", strings.SplitN(usage, "\n", 2)[0])
	}
	if strings.Contains(usage, "shellcheck disable") {
		t.Error("usage carries a shellcheck directive")
	}
}
