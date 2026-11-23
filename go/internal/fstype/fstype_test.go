package fstype

import (
	"os"
	"path/filepath"
	"testing"
)

const mounts = "rootfs / ext4 rw 0 0\nC: /mnt/c 9p rw 0 0\nD:\t/mnt/d\tdrvfs rw 0 0\nnone /mnt/c/deep/er ext4 rw 0 0\n"

// mountCases are the mounts-file lookups, with the answers the retired bash lib (toolbelt/lib/fs-type.sh
// fs_type_for) gave for each.
var mountCases = []struct{ path, want string }{
	{"/home/u/proj", "ext4"},
	{"/mnt/c/Niagara", "9p"},
	{"/mnt/d/x", "drvfs"},
	{"/mnt/c/deep/er/x", "ext4"}, // the longest mountpoint prefix wins
	{"/mnt/cc", "9p"},            // a string prefix, as the bash lib read it
	{"relative", ""},
}

func mountsFile(t *testing.T) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "mounts")
	if err := os.WriteFile(f, []byte(mounts), 0o644); err != nil {
		t.Fatal(err)
	}
	return f
}

func noDf() []byte { return nil }

func TestForReadsTheMountsFile(t *testing.T) {
	t.Parallel()
	f := mountsFile(t)
	for _, c := range mountCases {
		if got := For(c.path, f, noDf); got != c.want {
			t.Errorf("For(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}

func TestForReadsDfWithoutAMountsFile(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, mounts, df, want string }{
		{"the TYPE column of line 2", "", "Filesystem Type 1K-blocks\nC:\\ 9p 100\n", "9p"},
		{"a mounts path that is not a file falls back to df", "/nonexistent-mounts", "h\n/dev/sda ext4 1\n", "ext4"},
		{"df printed nothing", "", "", ""},
		{"df printed the header only", "", "Filesystem Type\n", ""},
	}
	for _, c := range cases {
		if got := For("/p", c.mounts, func() []byte { return []byte(c.df) }); got != c.want {
			t.Errorf("%s: For = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestIsWindowsMount(t *testing.T) {
	t.Parallel()
	for typ, want := range map[string]bool{"9p": true, "drvfs": true, "ext4": false, "": false, "9pfs": false} {
		if got := IsWindowsMount(typ); got != want {
			t.Errorf("IsWindowsMount(%q) = %v, want %v", typ, got, want)
		}
	}
}

// TestOfPathReadsTheMountTableByWholeComponents: no df; the longest mountpoint that holds the path by whole
// components wins, a blank in a mountpoint is its \040 escape, a link is followed, a missing table is "".
func TestOfPathReadsTheMountTableByWholeComponents(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	table := filepath.Join(dir, "mounts")
	body := mounts + "C: /mnt/my\\040drive 9p rw 0 0\n"
	if err := os.WriteFile(table, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ path, want string }{
		{"/home/u/proj", "ext4"},
		{"/mnt/c", "9p"},
		{"/mnt/c/Niagara", "9p"},
		{"/mnt/c/deep/er/x", "ext4"},
		{"/mnt/cc/x", "ext4"}, // not under /mnt/c: a whole-component prefix
		{"/mnt/my drive/x", "9p"},
	} {
		if got := OfPath(c.path, table); got != c.want {
			t.Errorf("OfPath(%q) = %q, want %q", c.path, got, c.want)
		}
	}
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(real, "mnt"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(real, "mnt"), filepath.Join(real, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(table, []byte("rootfs / ext4 rw 0 0\nC: "+real+"/mnt 9p rw 0 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := OfPath(filepath.Join(real, "link"), table); got != "9p" {
		t.Errorf("a link into a mount: %q, want 9p", got)
	}
	if got := OfPath("/x", filepath.Join(dir, "absent")); got != OfPath("/x", ProcMounts) {
		t.Errorf("a table that is no file falls back to %s: %q", ProcMounts, got)
	}
}
