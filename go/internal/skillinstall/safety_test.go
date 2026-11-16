package skillinstall

import (
	"os"
	"path/filepath"
	"testing"
)

// S4: --force never writes through a symlinked SKILL.md or skill directory; the pointee is unchanged.
func TestForceRefusesASymlinkedTarget(t *testing.T) {
	f := newFixture(t)
	pointee := write(t, filepath.Join(filepath.Dir(f.home), "pointee", "SKILL.md"), "someone else's file\n")
	target := f.installed("mcp-n4")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(pointee, target); err != nil {
		t.Fatal(err)
	}
	code, out, errText := f.install("--skill", "mcp-n4", "--force")
	expect(t, "file link", code, out, errText, 3, "",
		"install-skill: refusing to write through a symbolic link: "+target+" (remove the link, then re-run)\n")
	if got := read(t, pointee); got != "someone else's file\n" {
		t.Errorf("the pointee was written: %q", got)
	}

	g := newFixture(t)
	dir := filepath.Dir(pointee)
	if err := os.MkdirAll(filepath.Join(g.home, ".claude", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(dir, filepath.Dir(g.installed("mcp-n4"))); err != nil {
		t.Fatal(err)
	}
	code, _, errText = g.install("--skill", "mcp-n4", "--force")
	if code != 3 || read(t, pointee) != "someone else's file\n" {
		t.Errorf("directory link: exit %d, stderr %q, pointee %q", code, errText, read(t, pointee))
	}
}
