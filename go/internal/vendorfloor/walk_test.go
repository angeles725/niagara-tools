package vendorfloor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// write creates root/rel with body, making its directories.
func write(t *testing.T, root, rel, body string) string {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestListVendored(t *testing.T) {
	t.Parallel()
	d := t.TempDir() + "/vendor"
	for _, rel := range []string{"zz.js", "a/two.cjs", "a-b/one.js", ".hidden/h.js", "x-worktrees/w.mjs", "readme.txt", "lib.json", "deep/er/lib.min.js"} {
		write(t, d, rel, "")
	}
	if err := os.Symlink(d+"/zz.js", d+"/link.js"); err != nil {
		t.Fatal(err)
	}
	files, errs := listVendored(d)
	if len(errs) != 0 {
		t.Fatalf("errs = %v", errs)
	}
	// bytewise order of the whole path ("-" sorts before "/"), dot directories included, the symlink left out
	want := []string{".hidden/h.js", "a-b/one.js", "a/two.cjs", "deep/er/lib.min.js", "x-worktrees/w.mjs", "zz.js"}
	for i := range want {
		want[i] = d + "/" + want[i]
	}
	if strings.Join(files, "\n") != strings.Join(want, "\n") {
		t.Errorf("files:\n%s\nwant:\n%s", strings.Join(files, "\n"), strings.Join(want, "\n"))
	}
}

// A vendor/ that is a symlink is followed like the other lint roots (find -H): the libraries under the
// link are listed with their paths spelled through it, so a library vendored through a link is linted
// (retro go-lint-vendor-floor Δ1). This replaces the pin of the bash walk, which listed nothing here.
func TestListVendoredFollowsASymlinkedRoot(t *testing.T) {
	t.Parallel()
	d := t.TempDir()
	write(t, d, "real/lib.js", "")
	write(t, d, "real/deep/two.mjs", "")
	if err := os.Symlink(d+"/real", d+"/vendor"); err != nil {
		t.Fatal(err)
	}
	files, errs := listVendored(d + "/vendor")
	if want := d + "/vendor/deep/two.mjs\n" + d + "/vendor/lib.js"; strings.Join(files, "\n") != want || len(errs) != 0 {
		t.Errorf("files = %v, errs = %v, want\n%s", files, errs, want)
	}
}

// Only the root link is followed, so a vendor/ that points at an ancestor ends: the link is met again
// below itself as a symlink and is neither followed nor listed. Every file is listed once.
func TestListVendoredVisitsALinkCycleOnce(t *testing.T) {
	t.Parallel()
	d := t.TempDir()
	write(t, d, "top.js", "")
	write(t, d, "sub/a.js", "")
	if err := os.Symlink(d, d+"/vendor"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(d+"/vendor", d+"/sub/back"); err != nil {
		t.Fatal(err)
	}
	files, errs := listVendored(d + "/vendor")
	if want := d + "/vendor/sub/a.js\n" + d + "/vendor/top.js"; strings.Join(files, "\n") != want || len(errs) != 0 {
		t.Errorf("files = %v, errs = %v, want\n%s", files, errs, want)
	}
}

// Through the command: the linked library reaches the tools (node is forced missing, so the proof is
// the SKIP row and exit 4 instead of "nothing to check" and exit 0).
func TestSymlinkedVendorIsLinted(t *testing.T) {
	t.Parallel()
	d := t.TempDir()
	write(t, d, "shared/lib.js", "x ??= 1;\n")
	if err := os.Symlink(d+"/shared", d+"/vendor"); err != nil {
		t.Fatal(err)
	}
	code, out, errs := call(map[string]string{"KIT_NODE": "/nonexistent/node"}, d)
	check(t, "symlinked vendor/", code, out, errs, ExitUnavailable,
		"SKIP  vendor-floor  "+d+"  unavailable: node not found (install Node, or set KIT_NODE)\n", "")
}

// A vendor/ link that leads nowhere cannot be followed: an env fault, never "no vendor/".
func TestDanglingVendorLinkIsAnEnvFault(t *testing.T) {
	t.Parallel()
	d := t.TempDir()
	if err := os.Symlink(d+"/gone", d+"/vendor"); err != nil {
		t.Fatal(err)
	}
	code, out, errs := call(nil, d)
	check(t, "dangling vendor/", code, out, errs, ExitEnv, "",
		"lint-vendor-floor: env fault: "+d+"/vendor is a symlink that leads nowhere\n")
}

// A sub-directory the walk cannot read is an env fault naming it (exit 3, no row), whether or not other
// libraries were listed: the libraries in it were not looked at, so neither a pass nor a tool verdict
// stands (retro go-lint-vendor-floor Δ1). No tool is needed to say so.
// Mutation: VF-unreadable -- printing the walk error and scanning on exits 0 over libraries nobody parsed
func TestUnreadableSubDirectoryIsAnEnvFault(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	for _, c := range []struct {
		name  string
		files map[string]string
	}{
		{"nothing else listed", map[string]string{"locked/in.js": "x ??= 1;\n"}},
		{"another library listed", map[string]string{"locked/in.js": "x ??= 1;\n", "ok.js": "var a = 1;\n"}},
	} {
		d := rc(t, c.files)
		locked := d + "/vendor/locked"
		if err := os.Chmod(locked, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chmod(locked, 0o755) })
		code, out, errs := call(map[string]string{"KIT_NODE": "/nonexistent/node"}, d)
		check(t, c.name, code, out, errs, ExitEnv, "",
			"lint-vendor-floor: env fault: cannot list every file under "+d+"/vendor: find: '"+locked+"': Permission denied\n")
	}
}

func TestListVendoredNamesAnUnreadableDirectory(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	d := t.TempDir() + "/vendor"
	ok := write(t, d, "ok.js", "")
	write(t, d, "locked/in.js", "")
	if err := os.Chmod(d+"/locked", 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(d+"/locked", 0o755) })
	files, errs := listVendored(d)
	if len(files) != 1 || files[0] != ok {
		t.Errorf("files = %v", files)
	}
	if len(errs) != 1 || errs[0].Error() != "find: '"+d+"/locked': Permission denied" {
		t.Errorf("errs = %v", errs)
	}
}

// An rc dir the process may not search cannot say whether it holds a vendor/: an env fault naming it
// (exit 3, no row), never "no vendor/ under rc" and exit 0. No tool is needed to say so.
// Mutation: VF-unsearchable -- reading any stat error as "no vendor/" exits 0 over libraries nobody looked for
func TestUnsearchableRcDirIsAnEnvFault(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block a lookup")
	}
	d := rc(t, map[string]string{"lib.js": "x ??= 1;\n"})
	if err := os.Chmod(d, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(d, 0o755) })
	code, out, errs := call(map[string]string{"KIT_NODE": "/nonexistent/node"}, d)
	check(t, "unsearchable rc dir", code, out, errs, ExitEnv, "",
		"lint-vendor-floor: env fault: cannot look for vendor/ under "+d+": Permission denied\n")
}

// What is simply not a vendor/ directory keeps its note and exit 0: no entry of that name, or a regular
// file of that name.
func TestNoVendorDirectoryStaysClean(t *testing.T) {
	t.Parallel()
	none := t.TempDir()
	file := t.TempDir()
	write(t, file, "vendor", "not a directory\n")
	for _, d := range []string{none, file} {
		code, out, errs := call(map[string]string{"KIT_NODE": "/nonexistent/node"}, d)
		check(t, d, code, out, errs, ExitOK, "", "lint-vendor-floor: no vendor/ under "+d+" — nothing to check\n")
	}
}
