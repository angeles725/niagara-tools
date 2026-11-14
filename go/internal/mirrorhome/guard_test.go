package mirrorhome

import (
	"os"
	"testing"
)

// TestGuardA: a mirror dir that is the source, inside it or around it is refused with exit 20 and nothing
// is removed, whichever way either is spelled.
func TestGuardA(t *testing.T) {
	f := newFixture(t)
	f.symlink("nh", "link")
	nh := f.path("nh")
	before := f.tree("")
	inside, contains := "refusing: mirror dir is the source install or inside it: ", "refusing: mirror dir contains the source install: "
	cases := []struct{ name, src, mir, refusal string }{
		{"the source", nh, nh, inside},
		{"the source, final slash", nh, nh + "/", inside},
		{"the source through a link", f.path("link"), nh, inside},
		{"a link to the source", nh, f.path("link"), inside},
		{"inside", nh, nh + "/modules", inside},
		{"inside, not there yet", nh, nh + "/mirror", inside},
		{"around", nh, f.dir, contains},
		{"around, from a deeper source", nh + "/etc/..", f.dir, contains},
	}
	for _, c := range cases {
		f.wants(c.name, f.mirror(c.src, c.mir), ExitRefused, "", c.refusal+c.mir+"\n")
		f.unchanged(c.name, "", before)
	}
	f.wants("a name the source's starts", f.mirror(nh, nh+"2"), ExitOK, "mirror ready: "+nh+"2  (3 jars linked, modules/ writable, excluded: none)\n", "")
}

// TestGuardB: an existing mirror dir is wiped only when it is empty or holds the marker file.
func TestGuardB(t *testing.T) {
	f := newFixture(t)
	nh, ready := f.path("nh"), "  (3 jars linked, modules/ writable, excluded: none)\n"
	f.touch("precious/keep.txt", "precious/sub/deep.txt", "file")
	f.mkdir("markdir/"+Marker, "empty")
	for _, dir := range []string{"precious", "markdir"} {
		before := f.tree(dir)
		f.wants(dir, f.mirror(nh, f.path(dir)), ExitRefused, "",
			"refusing to wipe "+f.path(dir)+": non-empty and not a mirror made by this script (no .niagara-mirror)\n")
		f.unchanged(dir, dir, before)
	}
	f.wants("a file", f.mirror(nh, f.path("file")), ExitRefused, "", "refusing: "+f.path("file")+" exists and is not a directory\n")
	if st, err := os.Lstat(f.path("file")); err != nil || !st.Mode().IsRegular() {
		t.Errorf("the file was touched: %v", err)
	}
	f.wants("empty", f.mirror(nh, f.path("empty")), ExitOK, "mirror ready: "+f.path("empty")+ready, "")
	f.touch("marked/"+Marker, "marked/old/file")
	f.wants("marked", f.mirror(nh, f.path("marked")), ExitOK, "mirror ready: "+f.path("marked")+ready, "")
	if _, err := os.Lstat(f.path("marked/old")); err == nil {
		t.Error("a marked mirror kept its old content")
	}
}

// TestRelativeSourceRefused: the links hold the source as it was given, so a relative one makes a mirror
// of dangling links; it is refused before anything is wiped or made, after the guards.
func TestRelativeSourceRefused(t *testing.T) {
	f := newFixture(t)
	f.touch("marked/"+Marker, "marked/old/file", "precious/keep.txt")
	t.Chdir(f.dir)
	before := f.tree("")
	relative := "refusing: source_niagara_home must be an absolute path (the mirror's links would dangle): nh\n"
	f.wants("new mirror", f.mirror("nh", "mir"), ExitUsage, "", relative)
	f.wants("a mirror to refresh", f.mirror("nh", "marked"), ExitUsage, "", relative)
	f.wants("guard A first", f.mirror("nh", "nh"), ExitRefused, "", "refusing: mirror dir is the source install or inside it: nh\n")
	f.wants("guard B first", f.mirror("nh", "precious"), ExitRefused, "", "refusing to wipe precious: non-empty and not a mirror made by this script (no .niagara-mirror)\n")
	f.unchanged("relative source", "", before)
	f.wants("a relative mirror dir is fine", f.mirror(f.path("nh"), "mir"), ExitOK, "mirror ready: mir  (3 jars linked, modules/ writable, excluded: none)\n", "")
}

// TestUnreadableSourceRefused: a source (or its modules/) that cannot be listed is exit 1, never a
// "mirror ready" over nothing.
func TestUnreadableSourceRefused(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root reads a directory of any mode")
	}
	for _, dir := range []string{"nh", "nh/modules"} {
		f := newFixture(t)
		if err := os.Chmod(f.path(dir), 0o311); err != nil {
			t.Fatal(err)
		}
		code := f.mirror(f.path("nh"), f.path("mir"))
		os.Chmod(f.path(dir), 0o755)
		if want := "mirror-niagara-home: open " + f.path(dir) + ": Permission denied\n"; code != ExitUsage || f.out != "" || f.err != want {
			t.Errorf("%s unreadable: exit %d stdout %q stderr %q, want 1 and %q", dir, code, f.out, f.err, want)
		}
	}
}
