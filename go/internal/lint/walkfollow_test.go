package lint

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// followTree is a root with one selected file per place a symlink can hide it: store/ holds the link
// targets inside the root, outside/ (beside the root, like a sibling module) the ones outside it.
func followTree(t *testing.T) (root, outside string) {
	t.Helper()
	base := t.TempDir()
	root, outside = filepath.Join(base, "root"), filepath.Join(base, "outside")
	for _, rel := range []string{"root/part/src/a.txt", "root/part/src/skip.md", "root/store/dir/d.txt", "root/store/file.txt",
		"root/Mod-rt/build/out.txt", "outside/dir/o.txt", "outside/file.txt"} {
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

func joined(s []string) string { return strings.Join(s, "\n") }

// TestWalkFollowListsWhatALinkHides: a symlinked file and the files under a symlinked directory, both
// inside the root, are listed under the link's path, each file once (the first spelling in name order), a
// link's own name is what is selected, and Walk on the same tree still lists no link.
func TestWalkFollowListsWhatALinkHides(t *testing.T) {
	t.Parallel()
	root, _ := followTree(t)
	link(t, filepath.Join(root, "store/file.txt"), filepath.Join(root, "part/l.txt"))
	link(t, filepath.Join(root, "store/file.txt"), filepath.Join(root, "part/l.md"))
	link(t, filepath.Join(root, "store/dir"), filepath.Join(root, "part/linked"))
	got, out, err := WalkFollow(root, Selection{Names: []string{"*.txt"}, PrunePaths: PartOutputs()})
	want := []string{root + "/part/l.txt", root + "/part/linked/d.txt", root + "/part/src/a.txt"}
	if err != nil || joined(got) != joined(want) || len(out) != 0 {
		t.Errorf("WalkFollow = %v, %v, %v; want %v and no link left out", got, out, err, want)
	}
	plain, err := Walk(root, Selection{Names: []string{"*.txt"}, PrunePaths: PartOutputs()})
	if err != nil || joined(plain) != joined([]string{root + "/part/src/a.txt", root + "/store/dir/d.txt", root + "/store/file.txt"}) {
		t.Errorf("Walk = %v, %v; want only the regular files (unchanged)", plain, err)
	}
}

// TestWalkFollowStaysInsideTheRoot (K6): a link whose real target is outside the root is not followed and
// is returned once, in path order: a link to the root's parent (a sibling module), to "/" and to a file
// outside. A relative link and a link through another link are judged by the real target too.
func TestWalkFollowStaysInsideTheRoot(t *testing.T) {
	t.Parallel()
	root, outside := followTree(t)
	link(t, filepath.Dir(root), filepath.Join(root, "part/up"))
	link(t, "/", filepath.Join(root, "part/slash"))
	link(t, filepath.Join(outside, "file.txt"), filepath.Join(root, "part/f.txt"))
	link(t, "../../outside/dir", filepath.Join(root, "part/rel"))
	link(t, filepath.Join(root, "part/rel"), filepath.Join(root, "part/via"))
	link(t, filepath.Join(outside, "file.txt"), filepath.Join(root, "part/f.md")) // not selected: no row
	got, out, err := WalkFollow(root, Selection{Names: []string{"*.txt"}, PrunePaths: PartOutputs()})
	wantOut := []string{root + "/part/f.txt", root + "/part/rel", root + "/part/slash", root + "/part/up", root + "/part/via"}
	if err != nil || joined(got) != joined([]string{root + "/part/src/a.txt", root + "/store/dir/d.txt", root + "/store/file.txt"}) || joined(out) != joined(wantOut) {
		t.Errorf("WalkFollow = %v, %v, %v; want the root's own files and the links left out %v", got, out, err, wantOut)
	}
}

// TestWalkFollowPrunesByTheTarget (K6): a link inside the root is pruned by what it leads to, so a link
// named out to a part's build output does not re-expose it; a pruned link name still prunes.
func TestWalkFollowPrunesByTheTarget(t *testing.T) {
	t.Parallel()
	root, _ := followTree(t)
	link(t, "build", filepath.Join(root, "Mod-rt/out"))
	link(t, filepath.Join(root, "store/dir"), filepath.Join(root, "Mod-rt/.hidden"))
	got, out, err := WalkFollow(root, Selection{Names: []string{"*.txt"}, PrunePaths: PartOutputs()})
	want := []string{root + "/part/src/a.txt", root + "/store/dir/d.txt", root + "/store/file.txt"}
	if err != nil || joined(got) != joined(want) || len(out) != 0 {
		t.Errorf("WalkFollow = %v, %v, %v; want %v", got, out, err, want)
	}
}

// TestWalkFollowEntersADirectoryOnce: a link back to the root ends, and a directory reached under two
// spellings is listed under the first one in name order only.
func TestWalkFollowEntersADirectoryOnce(t *testing.T) {
	t.Parallel()
	root, _ := followTree(t)
	link(t, root, filepath.Join(root, "part/src/up"))
	link(t, filepath.Join(root, "store/dir"), filepath.Join(root, "part/one"))
	link(t, filepath.Join(root, "store/dir"), filepath.Join(root, "part/two"))
	got, out, err := WalkFollow(root, Selection{Names: []string{"*.txt"}, PrunePaths: PartOutputs()})
	want := []string{root + "/part/one/d.txt", root + "/part/src/a.txt", root + "/store/file.txt"}
	if err != nil || joined(got) != joined(want) || len(out) != 0 {
		t.Errorf("WalkFollow = %v, %v, %v; want %v", got, out, err, want)
	}
}

// TestWalkFollowDanglingLink: a link that leads nowhere is an error when its name is selected (the file
// could not be read) and nothing at all when it is not; the rest of the tree is still listed.
func TestWalkFollowDanglingLink(t *testing.T) {
	t.Parallel()
	root, _ := followTree(t)
	link(t, filepath.Join(root, "gone"), filepath.Join(root, "part/other"))
	got, _, err := WalkFollow(root, Selection{Names: []string{"*.txt"}})
	if err != nil || len(got) != 4 {
		t.Errorf("a dangling link of another name: %v, %v; want four files and no error", got, err)
	}
	link(t, filepath.Join(root, "gone"), filepath.Join(root, "part/gone.txt"))
	got, _, err = WalkFollow(root, Selection{Names: []string{"*.txt"}})
	wantErr := "find: '" + root + "/part/gone.txt': No such file or directory"
	if err == nil || err.Error() != wantErr || len(got) != 4 {
		t.Errorf("a dangling selected link: %v, %v; want four files and %q", got, err, wantErr)
	}
}

// TestWalkFollowSelectedEntryThatIsNoFile (K7): a selected name that is neither a regular file nor a
// directory (a FIFO, or a link to one) is a file nobody can read as one: the walk error, never a skip.
func TestWalkFollowSelectedEntryThatIsNoFile(t *testing.T) {
	t.Parallel()
	root, _ := followTree(t)
	if err := syscall.Mkfifo(filepath.Join(root, "store/pipe"), 0o644); err != nil {
		t.Skip("no FIFO on this file system:", err)
	}
	link(t, filepath.Join(root, "store/pipe"), filepath.Join(root, "part/p.txt"))
	got, _, err := WalkFollow(root, Selection{Names: []string{"*.txt"}})
	wantErr := "find: '" + root + "/part/p.txt': not a regular file"
	if err == nil || err.Error() != wantErr || len(got) != 4 {
		t.Errorf("a link to a FIFO: %v, %v; want four files and %q", got, err, wantErr)
	}
	if _, _, err := WalkFollow(root, Selection{Names: []string{"*.md"}}); err != nil {
		t.Errorf("a FIFO of another name: %v, want no error", err)
	}
}
