package skillinstall

import (
	"os"
	"path/filepath"
	"testing"
)

// R3-diverged-template-names-missing-file: a skill whose kit has no prompt.md renders the built-in default;
// its diverged prompt names that default, never a template file that does not exist.
func TestDivergedDefaultPromptNamesTheBuiltInTemplate(t *testing.T) {
	f := newFixture(t)
	agent := f.mkAgent(".pi/agent")
	if err := os.Remove(promptSource(f.tracked("mcp-n4"))); err != nil {
		t.Fatal(err)
	}
	f.install("--skill", "mcp-n4", "--harness", "pi")
	prompt := write(t, filepath.Join(agent, "prompts", "mcp-n4.md"), "edited by hand\n")
	rendered, err := renderPrompt("mcp-n4", f.tracked("mcp-n4"), filepath.Join(agent, "skills", "mcp-n4", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	code, out, errText := f.install("--skill", "mcp-n4", "--harness", "pi")
	expect(t, "diverged default prompt", code, out, errText, 1, "",
		"install-skill: installed copy diverges from tracked source\n"+
			"  template: the built-in default (the kit has no "+promptSource(f.tracked("mcp-n4"))+") (rendered for this harness)\n"+
			"  installed: "+prompt+"\n"+
			"  rendered sha256:  "+digest(rendered)+"\n"+
			"  installed sha256: "+sha(t, prompt)+"\n"+
			"Re-run with --force to overwrite.\n")
}
