package install

import (
	"os"
	"path/filepath"
	"testing"
)

// A stable path that is a symlink into a git checkout is the H5 failure itself: install must leave
// a regular file there, even when the link target already holds the same bytes with mode 0755.
func TestInstallReplacesASymlinkIntoACheckoutWithARegularFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src", "mcp-n4")
	wt := filepath.Join(dir, "wt", "go", "bin", "mcp-n4")
	prefix := filepath.Join(dir, "home", ".local", "bin")
	for _, p := range []string{src, wt} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("same binary"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "wt", ".git"), []byte("gitdir: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(prefix, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(wt, filepath.Join(prefix, Name)); err != nil {
		t.Fatal(err)
	}
	dst, changed, err := Install(src, prefix)
	if err != nil {
		t.Fatal(err)
	}
	st, err := os.Lstat(dst)
	if err != nil || !st.Mode().IsRegular() || !changed {
		t.Fatalf("changed=%v, mode %v: %s must become a regular file, not stay a link into the checkout", changed, st.Mode(), dst)
	}
}
