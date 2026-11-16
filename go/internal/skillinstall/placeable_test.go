package skillinstall

import (
	"os"
	"path/filepath"
	"testing"
)

// T4(3) residual: the plan also checks that every file can be placed. A non-directory where a directory
// must be, or a directory where a file must be, refuses the run before the first write (exit 3), so
// SKILL.md is never left without its prompt.
func TestUnplaceableTargetRefusesBeforeAnyWrite(t *testing.T) {
	for name, c := range map[string]struct {
		arrange func(t *testing.T, agent string) string // returns the path the refusal names
		want    string
	}{
		"prompts/ is a file": {
			func(t *testing.T, agent string) string { return write(t, filepath.Join(agent, "prompts"), "a file\n") },
			"not a directory"},
		"the skill directory is a file": {
			func(t *testing.T, agent string) string {
				return write(t, filepath.Join(agent, "skills", "mcp-n4"), "a file\n")
			},
			"not a directory"},
		"the prompt is a directory": {
			func(t *testing.T, agent string) string {
				path := filepath.Join(agent, "prompts", "mcp-n4.md")
				if err := os.MkdirAll(path, 0o755); err != nil {
					t.Fatal(err)
				}
				return path
			},
			"not a regular file"},
	} {
		for _, force := range []bool{false, true} {
			f := newFixture(t)
			agent := f.mkAgent(".pi/agent")
			blocker := c.arrange(t, agent)
			args := []string{"--skill", "mcp-n4", "--harness", "all"}
			if force {
				args = append(args, "--force")
			}
			before := snapshot(t, filepath.Dir(f.root))
			code, out, errText := f.install(args...)
			expect(t, name, code, out, errText, 3, "",
				"install-skill: refusing to install: "+c.want+": "+blocker+" (move it away, then re-run)\n")
			if after := snapshot(t, filepath.Dir(f.root)); after != before {
				t.Errorf("%s (force %v): the tree changed before the refusal", name, force)
			}
		}
	}
}

// Previous behaviour kept: --dry-run only reports, so an unplaceable target still prints its plan, exit 0.
func TestDryRunStillReportsAnUnplaceableTarget(t *testing.T) {
	f := newFixture(t)
	agent := f.mkAgent(".pi/agent")
	write(t, filepath.Join(agent, "prompts"), "a file\n")
	code, out, errText := f.install("--skill", "mcp-n4", "--harness", "pi", "--dry-run")
	expect(t, "dry-run", code, out, errText, 0,
		"install-skill: [dry-run] would install to: "+filepath.Join(agent, "skills", "mcp-n4", "SKILL.md")+"\n"+
			"install-skill: [dry-run] would install to: "+filepath.Join(agent, "prompts", "mcp-n4.md")+"\n", "")
}
