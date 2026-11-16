package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// followTree is a root with one selected file per place a symlink can hide it.
func followTree(t *testing.T) (root, outside string) {
	t.Helper()
	base := t.TempDir()
	root, outside = filepath.Join(base, "root"), filepath.Join(base, "outside")
	for _, rel := range []string{"root/part/src/a.txt", "root/part/src/skip.md", "outside/dir/d.txt", "outside/file.txt", "outside/build/out.txt"} {
		p := filepath.Join(base, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root, outside
}

func link(t *testing.T, target, at string) {
	t.Helper()
	if err := os.Symlink(target, at); err != nil {
		t.Fatal(err)
	}
}

// TestWalkFollowListsWhatALinkHides: a symlinked file and the files under a symlinked directory are listed
// under the link's path, a link's own name is what is selected and pruned, and Walk on the same tree still
// lists neither.
func TestWalkFollowListsWhatALinkHides(t *testing.T) {
	t.Parallel()
	root, outside := followTree(t)
	link(t, filepath.Join(outside, "file.txt"), filepath.Join(root, "part/l.txt"))
	link(t, filepath.Join(outside, "file.txt"), filepath.Join(root, "part/l.md"))
	link(t, filepath.Join(outside, "dir"), filepath.Join(root, "part/linked"))
	link(t, filepath.Join(outside, "build"), filepath.Join(root, "part/build"))
	sel := Selection{Names: []string{"*.txt"}, Prune: []string{"build"}}
	got, err := WalkFollow(root, sel)
	want := []string{root + "/part/l.txt", root + "/part/linked/d.txt", root + "/part/src/a.txt"}
	if err != nil || strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("WalkFollow = %v, %v; want %v", got, err, want)
	}
	plain, err := Walk(root, sel)
	if err != nil || strings.Join(plain, "\n") != root+"/part/src/a.txt" {
		t.Errorf("Walk = %v, %v; want only the regular file (unchanged)", plain, err)
	}
}

// TestWalkFollowEntersADirectoryOnce: a link back to an ancestor ends, and a directory reached under two
// spellings is listed under the first one in name order only.
func TestWalkFollowEntersADirectoryOnce(t *testing.T) {
	t.Parallel()
	root, outside := followTree(t)
	link(t, root, filepath.Join(root, "part/src/up"))
	link(t, filepath.Join(outside, "dir"), filepath.Join(root, "part/one"))
	link(t, filepath.Join(outside, "dir"), filepath.Join(root, "part/two"))
	got, err := WalkFollow(root, Selection{Names: []string{"*.txt"}})
	want := []string{root + "/part/one/d.txt", root + "/part/src/a.txt"}
	if err != nil || strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("WalkFollow = %v, %v; want %v", got, err, want)
	}
}

// TestWalkFollowDanglingLink: a link that leads nowhere is an error when its name is selected (the file
// could not be read) and nothing at all when it is not; the rest of the tree is still listed.
func TestWalkFollowDanglingLink(t *testing.T) {
	t.Parallel()
	root, outside := followTree(t)
	link(t, filepath.Join(outside, "gone"), filepath.Join(root, "part/other"))
	got, err := WalkFollow(root, Selection{Names: []string{"*.txt"}})
	if err != nil || len(got) != 1 {
		t.Errorf("a dangling link of another name: %v, %v; want one file and no error", got, err)
	}
	link(t, filepath.Join(outside, "gone"), filepath.Join(root, "part/gone.txt"))
	got, err = WalkFollow(root, Selection{Names: []string{"*.txt"}})
	wantErr := "find: '" + root + "/part/gone.txt': No such file or directory"
	if err == nil || err.Error() != wantErr || len(got) != 1 {
		t.Errorf("a dangling selected link: %v, %v; want one file and %q", got, err, wantErr)
	}
}
