package skillinstall

import (
	"path/filepath"
	"testing"
)

// H3: --harness pi checks the skill file and the prompt file of that harness, one row each.
func TestDriftHarnessPassesOnAnInstalledHarness(t *testing.T) {
	f := newFixture(t)
	agent := f.mkAgent(".pi/agent")
	f.install("--skill", "mcp-n4", "--harness", "pi")
	prompt := filepath.Join(agent, "prompts", "mcp-n4.md")
	code, out, errText := f.drift("--skill", "mcp-n4", "--harness", "pi")
	expect(t, "pi", code, out, errText, 0,
		"PASS  skill-drift  mcp-n4[pi]  installed copy matches mcp-n4-kit/skill/SKILL.md\n"+
			"PASS  skill-drift  mcp-n4[pi]  prompt "+prompt+" matches its template\n", "")
}

// H3: a modified copy in any harness is a FAIL row naming that harness's remedy; exit 1.
func TestDriftHarnessModifiedCopyFails(t *testing.T) {
	f := newFixture(t)
	agent := f.mkAgent(".gentle-shell/agent")
	f.install("--skill", "mcp-n4", "--harness", "gentle-shell")
	skill := write(t, filepath.Join(agent, "skills", "mcp-n4", "SKILL.md"), "stale\n")
	prompt := write(t, filepath.Join(agent, "prompts", "mcp-n4.md"), "stale\n")
	code, out, errText := f.drift("--skill", "mcp-n4", "--harness", "gentle-shell")
	remedy := " — run scripts/install-skill.sh --force --skill mcp-n4 --harness gentle-shell\n"
	expect(t, "modified", code, out, errText, 1,
		"FAIL  skill-drift  mcp-n4[gentle-shell]  installed copy "+skill+" differs from mcp-n4-kit/skill/SKILL.md"+remedy+
			"FAIL  skill-drift  mcp-n4[gentle-shell]  prompt "+prompt+" differs from its template"+remedy, "")
}

// H3: --all checks every harness; an absent harness is a non-failing SKIP row, a missing prompt is a FAIL.
func TestDriftAllHarnesses(t *testing.T) {
	f := newFixture(t)
	agent := f.mkAgent(".pi/agent")
	f.install("--skill", "mcp-n4", "--harness", "all")
	code, out, errText := f.drift("--skill", "mcp-n4", "--all")
	expect(t, "all current", code, out, errText, 0,
		"PASS  skill-drift  mcp-n4[claude]  installed copy matches mcp-n4-kit/skill/SKILL.md\n"+
			"PASS  skill-drift  mcp-n4[pi]  installed copy matches mcp-n4-kit/skill/SKILL.md\n"+
			"PASS  skill-drift  mcp-n4[pi]  prompt "+filepath.Join(agent, "prompts", "mcp-n4.md")+" matches its template\n"+
			"SKIP  skill-drift  mcp-n4[gentle-shell]  absent: "+f.agentDir(".gentle-shell/agent")+" not found\n", "")

	gs := f.mkAgent(".gentle-shell/agent")
	code, out, _ = f.drift("--skill", "mcp-n4", "--harness", "gentle-shell")
	expect(t, "present, not installed", code, out, "", 1,
		"FAIL  skill-drift  mcp-n4[gentle-shell]  not installed at "+filepath.Join(gs, "skills", "mcp-n4", "SKILL.md")+
			" — run scripts/install-skill.sh --skill mcp-n4 --harness gentle-shell\n"+
			"FAIL  skill-drift  mcp-n4[gentle-shell]  prompt not installed at "+filepath.Join(gs, "prompts", "mcp-n4.md")+
			" — run scripts/install-skill.sh --skill mcp-n4 --harness gentle-shell\n", "")
}

// H3: without --harness the check stays the Claude copy only, with the row it always printed.
func TestDriftDefaultStaysClaudeOnly(t *testing.T) {
	f := newFixture(t)
	f.mkAgent(".pi/agent")
	write(t, f.installed("build-n4-module"), buildSkill)
	code, out, errText := f.drift()
	expect(t, "default", code, out, errText, 0,
		"PASS  skill-drift  build-n4-module  installed copy matches build-n4-module-kit/skill/SKILL.md\n", "")
}
