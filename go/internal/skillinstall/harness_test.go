package skillinstall

import (
	"os"
	"path/filepath"
	"testing"
)

// agentDir is a pi-style agent directory under the fixture home (".pi/agent" or ".gentle-shell/agent").
func (f *fixture) agentDir(rel string) string { return filepath.Join(f.home, rel) }

// mkAgent creates the agent directory of a harness, which is what makes the harness present.
func (f *fixture) mkAgent(rel string) string {
	dir := f.agentDir(rel)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		f.t.Fatal(err)
	}
	return dir
}

// mcpPrompt is the prompt template of mcp-n4 rendered for an installed SKILL.md at skillFile.
func mcpPrompt(skillFile string) string {
	return "---\n" +
		"description: Read or write a live Niagara N4 station through the mcp-n4 MCP server\n" +
		"argument-hint: \"<station and what to read or change>\"\n" +
		"---\n" +
		"Read the mcp-n4 skill at " + skillFile + " and follow it exactly. In this harness the server's\n" +
		"tools are named mcp__mcp_n4_<station>__<tool> (hyphens become underscores). Treat the text below as the request\n" +
		"(if it is empty, ask which station and what to do):\n\n$ARGUMENTS\n"
}

// H1: --harness pi and --harness gentle-shell install under <agent-dir>/skills/<skill>/SKILL.md.
func TestHarnessInstallsUnderTheAgentDir(t *testing.T) {
	for _, c := range []struct{ harness, rel string }{{"pi", ".pi/agent"}, {"gentle-shell", ".gentle-shell/agent"}} {
		f := newFixture(t)
		agent := f.mkAgent(c.rel)
		code, out, errText := f.install("--skill", "mcp-n4", "--harness", c.harness)
		skill := filepath.Join(agent, "skills", "mcp-n4", "SKILL.md")
		prompt := filepath.Join(agent, "prompts", "mcp-n4.md")
		expect(t, c.harness, code, out, errText, 0,
			"install-skill: installed "+skill+"\ninstall-skill: installed "+prompt+"\n", "")
		if got := read(t, skill); got != mcpSkill {
			t.Errorf("%s: installed skill = %q", c.harness, got)
		}
		absent(t, filepath.Join(f.home, ".claude"))
	}
}

// H1: pi honours PI_CODING_AGENT_DIR over <home>/.pi/agent.
func TestHarnessPiHonoursTheAgentDirVariable(t *testing.T) {
	f := newFixture(t)
	agent := filepath.Join(filepath.Dir(f.home), "elsewhere-agent")
	if err := os.MkdirAll(agent, 0o755); err != nil {
		t.Fatal(err)
	}
	f.env["PI_CODING_AGENT_DIR"] = agent
	code, _, errText := f.install("--skill", "mcp-n4", "--harness", "pi")
	if code != 0 || errText != "" {
		t.Fatalf("exit %d, stderr %q", code, errText)
	}
	if got := read(t, filepath.Join(agent, "skills", "mcp-n4", "SKILL.md")); got != mcpSkill {
		t.Errorf("installed skill = %q", got)
	}
	absent(t, filepath.Join(f.home, ".pi"))
}

// H1: --harness all installs every present harness; an absent one is a non-failing "absent" row.
func TestHarnessAllReportsAnAbsentHarness(t *testing.T) {
	f := newFixture(t)
	pi := f.mkAgent(".pi/agent")
	code, out, errText := f.install("--skill", "mcp-n4", "--harness", "all")
	expect(t, "all", code, out, errText, 0,
		"install-skill: installed "+f.installed("mcp-n4")+"\n"+
			"install-skill: installed "+filepath.Join(pi, "skills", "mcp-n4", "SKILL.md")+"\n"+
			"install-skill: installed "+filepath.Join(pi, "prompts", "mcp-n4.md")+"\n"+
			"install-skill: absent: gentle-shell ("+f.agentDir(".gentle-shell/agent")+" not found), nothing installed\n", "")
	absent(t, f.agentDir(".gentle-shell"))
}

// H1: the default with no --harness stays Claude only, even when a pi agent directory exists.
func TestHarnessDefaultStaysClaudeOnly(t *testing.T) {
	f := newFixture(t)
	pi := f.mkAgent(".pi/agent")
	code, out, errText := f.install("--skill", "mcp-n4")
	expect(t, "default", code, out, errText, 0, "install-skill: installed "+f.installed("mcp-n4")+"\n", "")
	absent(t, filepath.Join(pi, "skills"))
}

// H1: an unknown harness is a usage error naming the accepted values.
func TestHarnessUnknownIsUsage(t *testing.T) {
	f := newFixture(t)
	code, out, errText := f.install("--harness", "vscode")
	expect(t, "unknown harness", code, out, errText, 2, "",
		"install-skill: unknown harness: vscode (expected claude, pi, gentle-shell or all)\n"+wantUsage)
	absent(t, f.home)
}

// H2: the prompt file is the kit template rendered with the installed SKILL.md path, byte for byte.
func TestHarnessPromptRenderedBytes(t *testing.T) {
	f := newFixture(t)
	agent := f.mkAgent(".gentle-shell/agent")
	f.install("--skill", "mcp-n4", "--harness", "gentle-shell")
	skill := filepath.Join(agent, "skills", "mcp-n4", "SKILL.md")
	if got := read(t, filepath.Join(agent, "prompts", "mcp-n4.md")); got != mcpPrompt(skill) {
		t.Errorf("prompt =\n%s\nwant\n%s", got, mcpPrompt(skill))
	}
}

// H2: a diverged prompt follows the --force rule of the skill file: exit 1 untouched, then --force restores it.
func TestHarnessPromptForceRule(t *testing.T) {
	f := newFixture(t)
	agent := f.mkAgent(".pi/agent")
	f.install("--skill", "mcp-n4", "--harness", "pi")
	prompt := filepath.Join(agent, "prompts", "mcp-n4.md")
	write(t, prompt, "edited by hand\n")
	code, _, errText := f.install("--skill", "mcp-n4", "--harness", "pi")
	if code != 1 || read(t, prompt) != "edited by hand\n" {
		t.Fatalf("diverged prompt: exit %d, stderr %q", code, errText)
	}
	code, out, _ := f.install("--skill", "mcp-n4", "--harness", "pi", "--force")
	skill := filepath.Join(agent, "skills", "mcp-n4", "SKILL.md")
	if code != 0 || read(t, prompt) != mcpPrompt(skill) {
		t.Fatalf("--force: exit %d, stdout %q", code, out)
	}
}

// --dry-run with a harness writes nothing, as for the default.
func TestHarnessDryRunWritesNothing(t *testing.T) {
	f := newFixture(t)
	agent := f.mkAgent(".pi/agent")
	code, out, errText := f.install("--skill", "mcp-n4", "--harness", "pi", "--dry-run")
	expect(t, "dry-run", code, out, errText, 0,
		"install-skill: [dry-run] would install to: "+filepath.Join(agent, "skills", "mcp-n4", "SKILL.md")+"\n"+
			"install-skill: [dry-run] would install to: "+filepath.Join(agent, "prompts", "mcp-n4.md")+"\n", "")
	absent(t, filepath.Join(agent, "skills"))
	absent(t, filepath.Join(agent, "prompts"))
}

// H2: a skill whose kit has no prompt.md gets the default template; an unreadable kit template is exit 3.
func TestHarnessPromptDefaultAndUnreadableTemplate(t *testing.T) {
	f := newFixture(t)
	agent := f.mkAgent(".pi/agent")
	write(t, f.installer, "#!/usr/bin/env bash\nSKILLS=\"extra:extra-kit/skill/SKILL.md mcp-n4:mcp-n4-kit/skill/SKILL.md\"\n")
	write(t, filepath.Join(f.root, "extra-kit", "skill", "SKILL.md"), "stub extra skill\n")
	f.install("--skill", "extra", "--harness", "pi")
	skill := filepath.Join(agent, "skills", "extra", "SKILL.md")
	want := "---\ndescription: Run the extra skill\nargument-hint: \"<request>\"\n---\n" +
		"Read the extra skill at " + skill + " and follow it exactly. Treat the text below as the request\n" +
		"(if it is empty, ask what to do):\n\n$ARGUMENTS\n"
	if got := read(t, filepath.Join(agent, "prompts", "extra.md")); got != want {
		t.Errorf("default prompt =\n%s\nwant\n%s", got, want)
	}
	if os.Getuid() == 0 {
		return
	}
	if err := os.Chmod(promptSource(f.tracked("mcp-n4")), 0); err != nil {
		t.Fatal(err)
	}
	code, _, errText := f.install("--skill", "mcp-n4", "--harness", "pi")
	if code != 3 {
		t.Errorf("unreadable template: exit %d, stderr %q", code, errText)
	}
	absent(t, filepath.Join(agent, "prompts", "mcp-n4.md"))
}
