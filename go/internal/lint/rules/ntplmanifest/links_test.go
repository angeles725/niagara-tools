package ntplmanifest

import (
	"os"
	"path/filepath"
	"testing"
)

// outside writes files under a directory that is not part of any module root.
func outside(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, body := range files {
		write(t, dir, rel, body)
	}
	return dir
}

func symlink(t *testing.T, target, at string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, at); err != nil {
		t.Fatal(err)
	}
}

// TestSymlinkedTemplateIsChecked: a .ntpl that is a symlink is read through the link and judged under the
// link's own path; it is never skipped into "nothing to check".
func TestSymlinkedTemplateIsChecked(t *testing.T) {
	t.Parallel()
	ext := outside(t, map[string]string{"broken.ntpl": ntpl(t, entry{bogEntry, bog}), "good.ntpl": template(t, "", "baja")})
	root := moduleRoot(t, nil)
	symlink(t, ext+"/broken.ntpl", root+"/ModA-rt/l.ntpl")
	expect(t, "a link to a broken template", root, 1, "FAIL  ntpl-entries  @ROOT@/ModA-rt/l.ntpl: lacks template-manifest.xml\n")
	root = moduleRoot(t, nil)
	symlink(t, ext+"/good.ntpl", root+"/ModA-rt/l.ntpl")
	expect(t, "a link to a clean template", root, 0, "")
}

// TestTemplateUnderASymlinkedDirectoryIsChecked: a symlinked directory inside the module is entered, so a
// template under it is judged under the link's path; a link back to the root ends instead of looping, and
// a directory reached twice is judged once.
func TestTemplateUnderASymlinkedDirectoryIsChecked(t *testing.T) {
	t.Parallel()
	ext := outside(t, map[string]string{"templates/broken.ntpl": ntpl(t, entry{manifestEntry, manifestXML("")})})
	root := moduleRoot(t, nil)
	symlink(t, ext+"/templates", root+"/ModA-rt/src")
	symlink(t, ext+"/templates", root+"/ModA-rt/src2")
	symlink(t, root, root+"/ModA-rt/loop")
	expect(t, "a linked src directory", root, 1, "FAIL  ntpl-entries  @ROOT@/ModA-rt/src/broken.ntpl: lacks template.bog\n")
}

// TestDanglingTemplateLinkIsExit3: a .ntpl link that leads nowhere is a template nobody could read: exit 3
// naming it, with the rows of the templates that were read, and never "nothing to check".
func TestDanglingTemplateLinkIsExit3(t *testing.T) {
	t.Parallel()
	root := moduleRoot(t, nil)
	symlink(t, root+"/gone.ntpl", root+"/ModA-rt/l.ntpl")
	wantErr := "lint-ntpl-manifest-deps: cannot list every file under " + root + ": find: '" + root + "/ModA-rt/l.ntpl': No such file or directory\n"
	if code, out, errText := run(root); code != 3 || out != "" || errText != wantErr {
		t.Errorf("only a dangling link: exit %d stdout %q stderr %q, want exit 3 and %q", code, out, errText, wantErr)
	}
	write(t, root, "ModA-rt/a.ntpl", ntpl(t, entry{bogEntry, bog}))
	wantOut := "FAIL  ntpl-entries  " + root + "/ModA-rt/a.ntpl: lacks template-manifest.xml\n"
	if code, out, errText := run(root); code != 3 || out != wantOut || errText != wantErr {
		t.Errorf("a dangling link next to a template: exit %d stdout %q stderr %q, want exit 3, %q and %q", code, out, errText, wantOut, wantErr)
	}
}

// TestPathWithAControlCharacterIsOneRow: a template whose name holds a newline or another control
// character is printed quoted with C escapes (rowpath.Display), so its row is one line and cannot pass
// for a second row.
func TestPathWithAControlCharacterIsOneRow(t *testing.T) {
	t.Parallel()
	root := moduleRoot(t, map[string]string{
		"ModA-rt/src/a\nPASS  x.ntpl": ntpl(t, entry{bogEntry, bog}),
		"ModA-rt/src/b\x01.ntpl":      template(t, "", "ghost-rt"),
	})
	expect(t, "a newline and a control byte", root, 1,
		`FAIL  ntpl-entries  "@ROOT@/ModA-rt/src/a\nPASS  x.ntpl": lacks template-manifest.xml`+"\n"+
			`FAIL  ntpl-deps  "@ROOT@/ModA-rt/src/b\001.ntpl": manifest dependencies neither declared (api/nre/implementation) in any *.gradle.kts under @ROOT@ nor shipped by the installation @HOME@: ghost-rt`+"\n")
}
