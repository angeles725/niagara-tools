package fstype

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const mounts = "rootfs / ext4 rw 0 0\nC: /mnt/c 9p rw 0 0\nD:\t/mnt/d\tdrvfs rw 0 0\nnone /mnt/c/deep/er ext4 rw 0 0\n"

// mountCases are the mounts-file lookups; TestMatchesTheShellLib replays them on the bash lib.
var mountCases = []struct{ path, want string }{
	{"/home/u/proj", "ext4"},
	{"/mnt/c/Niagara", "9p"},
	{"/mnt/d/x", "drvfs"},
	{"/mnt/c/deep/er/x", "ext4"}, // the longest mountpoint prefix wins
	{"/mnt/cc", "9p"},            // a string prefix, as the bash lib reads it
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

// TestMatchesTheShellLib pins this package to toolbelt/lib/fs-type.sh, which ng-deploy's tooling still
// ships: fs_type_for and is_9p_or_drvfs must answer every mounts-file case the same way.
func TestMatchesTheShellLib(t *testing.T) {
	t.Parallel()
	lib, err := filepath.Abs("../../../build-n4-module-kit/toolbelt/lib/fs-type.sh")
	if err != nil {
		t.Fatal(err)
	}
	f := mountsFile(t)
	for _, c := range mountCases {
		cmd := exec.Command("bash", "-c", `. "$1"; fs_type_for "$2"; is_9p_or_drvfs "$2" && echo windows; true`, "_", lib, c.path)
		cmd.Env = append(os.Environ(), MountsEnv+"="+f)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("bash fs_type_for %q: %v", c.path, err)
		}
		lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
		if lines[0] != For(c.path, f, noDf) {
			t.Errorf("fs_type_for %q = %q, For = %q", c.path, lines[0], For(c.path, f, noDf))
		}
		if windows := len(lines) == 2; windows != IsWindowsMount(c.want) {
			t.Errorf("is_9p_or_drvfs %q = %v, IsWindowsMount(%q) = %v", c.path, windows, c.want, !windows)
		}
	}
}
