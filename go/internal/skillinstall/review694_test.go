package skillinstall

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// R3-dry-run-rendered-label-untested: --dry-run over a diverged prompt labels the digest it would write as
// rendered, and over a diverged launcher as tracked.
func TestDryRunLabelsTheRenderedAndTheTrackedDigest(t *testing.T) {
	f := newFixture(t)
	agent := f.mkAgent(".pi/agent")
	f.install("--skill", "mcp-n4", "--harness", "pi")
	skill := write(t, filepath.Join(agent, "skills", "mcp-n4", "SKILL.md"), "edited launcher\n")
	prompt := write(t, filepath.Join(agent, "prompts", "mcp-n4.md"), "edited prompt\n")
	rendered, err := renderPrompt("mcp-n4", f.tracked("mcp-n4"), skill)
	if err != nil {
		t.Fatal(err)
	}
	code, out, errText := f.install("--skill", "mcp-n4", "--harness", "pi", "--dry-run")
	expect(t, "dry-run", code, out, errText, 0,
		"install-skill: [dry-run] would update (diverged): "+skill+"\n"+
			"  tracked sha256: "+digest([]byte(mcpSkill))+"\n"+
			"  current sha256: "+sha(t, skill)+"\n"+
			"install-skill: [dry-run] would update (diverged): "+prompt+"\n"+
			"  rendered sha256: "+digest(rendered)+"\n"+
			"  current sha256: "+sha(t, prompt)+"\n", "")
}

// R3-cross-harness-atomicity-untested: with --harness all, a diverged copy in one harness or a symbolic link
// in another refuses the run before any harness is written, whichever comes first.
func TestOneHarnessRefusalWritesNoOtherHarness(t *testing.T) {
	for name, arrange := range map[string]func(f *fixture, agent string){
		"a diverged pi prompt": func(f *fixture, agent string) {
			write(t, filepath.Join(agent, "prompts", "mcp-n4.md"), "edited by hand\n")
		},
		"a symbolic link in pi": func(f *fixture, agent string) {
			if err := os.Symlink(t.TempDir(), filepath.Join(agent, "prompts")); err != nil {
				t.Fatal(err)
			}
		},
	} {
		f := newFixture(t)
		agent := f.mkAgent(".pi/agent")
		arrange(f, agent)
		before := snapshot(t, filepath.Dir(f.root))
		code, out, _ := f.install("--skill", "mcp-n4", "--harness", "all")
		if code == 0 || out != "" || snapshot(t, filepath.Dir(f.root)) != before {
			t.Errorf("%s: exit %d, stdout %q, the tree changed", name, code, out)
		}
	}
}

// snapshot lists every path under root with its type, as one string.
func snapshot(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		b.WriteString(path + " " + d.Type().String() + "\n")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}
