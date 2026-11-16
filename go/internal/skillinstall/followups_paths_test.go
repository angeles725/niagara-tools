package skillinstall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// R3-materialize-when-nothing-installed: when every selected harness is absent nothing is installed, so
// the kit state is not materialized either (and --dry-run does not announce it).
func TestNoKitStateWhenEveryHarnessIsAbsent(t *testing.T) {
	f := newFixture(t)
	f.kitState(f.kit, "0")
	pi := f.agentDir(".pi/agent")
	for _, args := range [][]string{{"--harness", "pi"}, {"--harness", "pi", "--dry-run"}} {
		code, out, errText := f.install(args...)
		expect(t, strings.Join(args, " "), code, out, errText, 0,
			"install-skill: absent: pi ("+pi+" not found), nothing installed\n", "")
	}
	absent(t, filepath.Join(f.kit, kitStateLog))
}

// R3-path-join-output: printed paths are the --home value as given plus the harness path, as the bash
// installer printed them (no cleaning of a trailing slash).
func TestPrintedPathKeepsTheHomeAsGiven(t *testing.T) {
	f := newFixture(t)
	code, out, errText := f.installNoHome("--home", f.home+"/", "--skill", "mcp-n4", "--dry-run")
	expect(t, "trailing slash", code, out, errText, 0,
		"install-skill: [dry-run] would install to: "+f.home+"//.claude/skills/mcp-n4/SKILL.md\n", "")
}

// R3-relative-home-prompt: a relative --home still renders an absolute SKILL.md path into the prompt, so the
// harness can read it from any working directory.
func TestRelativeHomeRendersAnAbsolutePromptPath(t *testing.T) {
	f := newFixture(t)
	agent := f.mkAgent(".gentle-shell/agent")
	t.Chdir(filepath.Dir(f.home))
	code, _, errText := f.installNoHome("--home", filepath.Base(f.home), "--skill", "mcp-n4", "--harness", "gentle-shell")
	if code != 0 || errText != "" {
		t.Fatalf("exit %d, stderr %q", code, errText)
	}
	want := mcpPrompt(filepath.Join(agent, "skills", "mcp-n4", "SKILL.md"))
	if got := read(t, filepath.Join(agent, "prompts", "mcp-n4.md")); got != want {
		t.Errorf("prompt =\n%s\nwant\n%s", got, want)
	}
	code, out, _ := f.driftNoHome("--home", filepath.Base(f.home), "--skill", "mcp-n4", "--harness", "gentle-shell")
	if code != 0 {
		t.Errorf("drift on the same relative home: exit %d, stdout %q", code, out)
	}
}

// T4(4): `--home ""` prints the usage line in install-skill as in check-skill-drift.
func TestEmptyHomePrintsUsageInBothCommands(t *testing.T) {
	f := newFixture(t)
	code, out, errText := f.installNoHome("--home", "")
	expect(t, "install", code, out, errText, 2, "", "install-skill: --home requires a non-empty directory\n"+wantUsage)
	code, out, errText = f.driftNoHome("--home", "")
	expect(t, "drift", code, out, errText, 2, "", "check-skill-drift: --home requires a non-empty directory\n"+driftUsage)
	if _, err := os.Lstat(f.env["HOME"]); err == nil {
		t.Errorf("$HOME was written")
	}
}
