package skillinstall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// One test per review finding of #692 (retro 2026-11-15 rows T3 and T4); each names its finding.

// R1-001 / T4(1): an explicit --home wins over PI_CODING_AGENT_DIR, in install-skill and in check-skill-drift.
func TestHomeWinsOverThePiAgentDirVariable(t *testing.T) {
	f := newFixture(t)
	elsewhere := filepath.Join(filepath.Dir(f.home), "real-agent")
	if err := os.MkdirAll(elsewhere, 0o755); err != nil {
		t.Fatal(err)
	}
	f.env["PI_CODING_AGENT_DIR"] = elsewhere
	agent := f.mkAgent(".pi/agent")
	code, _, errText := f.install("--skill", "mcp-n4", "--harness", "pi")
	if code != 0 || errText != "" {
		t.Fatalf("exit %d, stderr %q", code, errText)
	}
	if got := read(t, filepath.Join(agent, "skills", "mcp-n4", "SKILL.md")); got != mcpSkill {
		t.Errorf("installed skill = %q", got)
	}
	absent(t, filepath.Join(elsewhere, "skills"))
	code, out, _ := f.drift("--skill", "mcp-n4", "--harness", "pi")
	if code != 0 || strings.Contains(out, elsewhere) {
		t.Errorf("drift read the variable's directory: exit %d, stdout %q", code, out)
	}
}

// R3-symlink-scope / T4(2): a symlinked skills/ or prompts/ parent is refused like a symlinked target.
func TestSymlinkedParentIsRefused(t *testing.T) {
	for _, parent := range []string{"skills", "prompts"} {
		f := newFixture(t)
		agent := f.mkAgent(".pi/agent")
		pointee := filepath.Join(filepath.Dir(f.home), "pointee")
		if err := os.MkdirAll(pointee, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(pointee, filepath.Join(agent, parent)); err != nil {
			t.Fatal(err)
		}
		code, out, errText := f.install("--skill", "mcp-n4", "--harness", "pi")
		expect(t, parent, code, out, errText, 3, "",
			"install-skill: refusing to write through a symbolic link: "+filepath.Join(agent, parent)+" (remove the link, then re-run)\n")
		if entries, _ := os.ReadDir(pointee); len(entries) != 0 {
			t.Errorf("%s: wrote through the link: %v", parent, entries)
		}
		absent(t, filepath.Join(agent, map[string]string{"skills": "prompts", "prompts": "skills"}[parent]))
	}
}

// T4(3): a refusal is printed before any --force line, and nothing of the harness is written.
func TestRefusalComesBeforeTheForceLine(t *testing.T) {
	f := newFixture(t)
	agent := f.mkAgent(".pi/agent")
	skill := write(t, filepath.Join(agent, "skills", "mcp-n4", "SKILL.md"), "edited by hand\n")
	pointee := write(t, filepath.Join(filepath.Dir(f.home), "pointee.md"), "not ours\n")
	prompt := filepath.Join(agent, "prompts", "mcp-n4.md")
	if err := os.MkdirAll(filepath.Dir(prompt), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(pointee, prompt); err != nil {
		t.Fatal(err)
	}
	code, out, errText := f.install("--skill", "mcp-n4", "--harness", "pi", "--force")
	expect(t, "refusal", code, out, errText, 3, "",
		"install-skill: refusing to write through a symbolic link: "+prompt+" (remove the link, then re-run)\n")
	if read(t, skill) != "edited by hand\n" || read(t, pointee) != "not ours\n" {
		t.Errorf("a file was written before the refusal")
	}
}

// T4(3): both files are checked before either is written: a diverged prompt without --force leaves the
// absent SKILL.md unwritten.
func TestDivergedPromptStopsTheWholeHarness(t *testing.T) {
	f := newFixture(t)
	agent := f.mkAgent(".pi/agent")
	write(t, filepath.Join(agent, "prompts", "mcp-n4.md"), "edited by hand\n")
	code, out, _ := f.install("--skill", "mcp-n4", "--harness", "pi")
	if code != 1 || out != "" {
		t.Errorf("exit %d, stdout %q", code, out)
	}
	absent(t, filepath.Join(agent, "skills"))
}

// R1-002 / T4(3): a diverged prompt names its template and the sha of the rendered prompt as such.
func TestDivergedPromptNamesTheRenderedTemplate(t *testing.T) {
	f := newFixture(t)
	agent := f.mkAgent(".pi/agent")
	f.install("--skill", "mcp-n4", "--harness", "pi")
	prompt := write(t, filepath.Join(agent, "prompts", "mcp-n4.md"), "edited by hand\n")
	rendered, err := renderPrompt("mcp-n4", f.tracked("mcp-n4"), filepath.Join(agent, "skills", "mcp-n4", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	code, out, errText := f.install("--skill", "mcp-n4", "--harness", "pi")
	expect(t, "diverged prompt", code, out, errText, 1, "",
		"install-skill: installed copy diverges from tracked source\n"+
			"  template: "+promptSource(f.tracked("mcp-n4"))+" (rendered for this harness)\n"+
			"  installed: "+prompt+"\n"+
			"  rendered sha256:  "+digest(rendered)+"\n"+
			"  installed sha256: "+sha(t, prompt)+"\n"+
			"Re-run with --force to overwrite.\n")
}

// R3-drift-unreadable-template: check-skill-drift reports an unreadable kit template as exit 3, no row.
func TestDriftUnreadableTemplateIsExit3(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root reads a mode-0 file")
	}
	f := newFixture(t)
	f.mkAgent(".pi/agent")
	f.install("--skill", "mcp-n4", "--harness", "pi")
	if err := os.Chmod(promptSource(f.tracked("mcp-n4")), 0); err != nil {
		t.Fatal(err)
	}
	code, _, errText := f.drift("--skill", "mcp-n4", "--harness", "pi")
	if code != 3 || !strings.Contains(errText, "check-skill-drift: cannot render the prompt template: ") {
		t.Errorf("exit %d, stderr %q", code, errText)
	}
}
