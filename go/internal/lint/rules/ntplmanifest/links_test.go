package ntplmanifest

import (
	"os"
	"path/filepath"
	"syscall"
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

// TestSymlinkedTemplateIsChecked: a .ntpl that is a symlink to an archive inside the module is read through
// the link and judged once, under the first spelling in path order; it is never skipped into "nothing to
// check".
func TestSymlinkedTemplateIsChecked(t *testing.T) {
	t.Parallel()
	root := moduleRoot(t, map[string]string{"store/broken.bin": ntpl(t, entry{bogEntry, bog}), "store/good.bin": template(t, "", "baja")})
	symlink(t, root+"/store/broken.bin", root+"/ModA-rt/l.ntpl")
	expect(t, "a link to a broken template", root, 1, "FAIL  ntpl-entries  @ROOT@/ModA-rt/l.ntpl: lacks template-manifest.xml\n")
	root = moduleRoot(t, map[string]string{"store/good.bin": template(t, "", "baja")})
	symlink(t, root+"/store/good.bin", root+"/ModA-rt/l.ntpl")
	expect(t, "a link to a clean template", root, 0, "")
}

// TestTemplateUnderASymlinkedDirectoryIsChecked: a symlinked directory inside the module is entered, so a
// template under it is judged under the link's path; a link back to the root ends instead of looping, and
// a directory reached twice is judged once.
func TestTemplateUnderASymlinkedDirectoryIsChecked(t *testing.T) {
	t.Parallel()
	root := moduleRoot(t, map[string]string{"store/templates/broken.ntpl": ntpl(t, entry{manifestEntry, manifestXML("")})})
	symlink(t, root+"/store/templates", root+"/ModA-rt/src")
	symlink(t, root+"/store/templates", root+"/ModA-rt/src2")
	symlink(t, root, root+"/ModA-rt/loop")
	expect(t, "a linked src directory", root, 1, "FAIL  ntpl-entries  @ROOT@/ModA-rt/src/broken.ntpl: lacks template.bog\n")
}

// linkRow is the WARN ntpl-link row of a link (rel under the root) whose target is outside the module.
func linkRow(rel, target string) string {
	return "WARN  ntpl-link  @ROOT@/" + rel + ": a symlink to " + target + ", outside the module root -- not followed, nothing behind it is checked\n"
}

// TestLinkOutOfTheModuleIsNotFollowed (K6): a link whose real target is outside the module root (the
// parent, which holds a sibling module; "/"; a template of another module) is not followed: one WARN
// ntpl-link row per link, in path order, before the rows of what was read, and never "nothing to check".
// A link that a build-file walk and the template walk both meet is one row.
func TestLinkOutOfTheModuleIsNotFollowed(t *testing.T) {
	t.Parallel()
	sibling := outside(t, map[string]string{"ModB/ModB-rt/src/bad.ntpl": ntpl(t, entry{bogEntry, bog}), "other.ntpl": "not a zip"})
	root := moduleRoot(t, map[string]string{tpl: template(t, "", "ghost-rt")})
	symlink(t, sibling, root+"/ModA-rt/up")
	symlink(t, "/", root+"/ModA-rt/slash")
	symlink(t, sibling+"/other.ntpl", root+"/ModA-rt/x.ntpl")
	expect(t, "three links out", root, 1, linkRow("ModA-rt/slash", "/")+linkRow("ModA-rt/up", sibling)+
		linkRow("ModA-rt/x.ntpl", sibling+"/other.ntpl")+undeclaredRow(tpl, "ghost-rt"))
	bare := moduleRoot(t, nil)
	symlink(t, sibling, bare+"/ModA-rt/up")
	expect(t, "only a link out: no PASS row", bare, 0, linkRow("ModA-rt/up", sibling))
}

// TestLinkToABuildOutputStaysPruned (K6): a link inside the module is pruned by its target, so out ->
// build does not re-expose a part's build output.
func TestLinkToABuildOutputStaysPruned(t *testing.T) {
	t.Parallel()
	root := moduleRoot(t, map[string]string{tpl: template(t, ""), "ModA-rt/build/out.ntpl": "not a zip"})
	symlink(t, "build", root+"/ModA-rt/out")
	expect(t, "out -> build", root, 0, "")
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

// TestTemplateThatIsNoFileIsExit3 (K7): a .ntpl that is not a regular file (a FIFO, or a link to one) is
// reported and never read: exit 3 naming it, never "nothing to check".
func TestTemplateThatIsNoFileIsExit3(t *testing.T) {
	t.Parallel()
	for name, at := range map[string]string{"a FIFO": "ModA-rt/p.ntpl", "a link to a FIFO": "ModA-rt/l.ntpl"} {
		root := moduleRoot(t, nil)
		if err := syscall.Mkfifo(root+"/pipe", 0o644); err != nil {
			t.Skip("no FIFO on this file system:", err)
		}
		if at == "ModA-rt/p.ntpl" {
			if err := os.Rename(root+"/pipe", root+"/"+at); err != nil {
				t.Fatal(err)
			}
		} else {
			symlink(t, root+"/pipe", root+"/"+at)
		}
		wantErr := "lint-ntpl-manifest-deps: cannot list every file under " + root + ": find: '" + root + "/" + at + "': not a regular file\n"
		if code, out, errText := run(root); code != 3 || out != "" || errText != wantErr {
			t.Errorf("%s: exit %d stdout %q stderr %q, want exit 3 and %q", name, code, out, errText, wantErr)
		}
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
