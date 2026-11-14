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

func TestListVendoredSymlinkedRootIsNotFollowed(t *testing.T) {
	t.Parallel()
	d := t.TempDir()
	write(t, d, "real/lib.js", "")
	if err := os.Symlink(d+"/real", d+"/vendor"); err != nil {
		t.Fatal(err)
	}
	if files, errs := listVendored(d + "/vendor"); len(files) != 0 || len(errs) != 0 {
		t.Errorf("files = %v, errs = %v", files, errs)
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
