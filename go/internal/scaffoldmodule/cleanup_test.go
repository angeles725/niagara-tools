package scaffoldmodule

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// failWrite makes the n-th file write of a run fail with errDiskFull after running before (if any) on the
// file's directory; the seam is restored when the test ends.
func failWrite(t *testing.T, n int, before func(dir string)) {
	t.Helper()
	real, calls := createFile, 0
	createFile = func(path string, data []byte, mode os.FileMode) error {
		if calls++; calls == n {
			if before != nil {
				before(filepath.Dir(path))
			}
			return errDiskFull
		}
		return real(path, data, mode)
	}
	t.Cleanup(func() { createFile = real })
}

var errDiskFull = errors.New("write injected: no space left on device")

// A write fault mid-copy removes the <out-dir>/<ModuleName> this run created, says so in one line and keeps
// the fault's exit code; the out-dir stays, and the next run is not refused with "already exists".
func TestFailedEmitRemovesThePartialTree(t *testing.T) {
	tb := newKit(t)
	out := filepath.Join(t.TempDir(), "out")
	dest := out + "/Foo"
	failWrite(t, 3, nil)
	r := run(t, tb, "Foo", out, "--vendor", "Angeles")
	want := "scaffold-module: " + errDiskFull.Error() + "\nscaffold-module: removed the partial " + dest + "\n"
	if r.code != 3 || r.out != "" || r.err != want {
		t.Errorf("code=%d out=%q\n err %q\nwant %q", r.code, r.out, r.err, want)
	}
	if _, err := os.Lstat(dest); !os.IsNotExist(err) {
		t.Errorf("the partial tree stays: %v (err %v)", keys(tree(t, dest)), err)
	}
	if !isDir(out) {
		t.Errorf("the out-dir was removed")
	}
	if r := run(t, tb, "Foo", out, "--vendor", "Angeles"); r.code != 0 { // the 4th and later writes are real again
		t.Errorf("rerun: code=%d err=%q", r.code, r.err)
	}
}

// A fault after the fixture copy (a template the kit lacks) removes the tree the same way.
func TestMissingTemplateRemovesTheTree(t *testing.T) {
	tb := newKit(t, "templates/VERSION-LEDGER.md")
	out := t.TempDir()
	r := run(t, tb, "Foo", out, "--vendor", "Angeles")
	want := "scaffold-module: version-ledger template not found: " + tb + "/../templates/VERSION-LEDGER.md\n" +
		"scaffold-module: removed the partial " + out + "/Foo\n"
	if r.code != 3 || r.out != "" || r.err != want {
		t.Errorf("code=%d out=%q\n err %q\nwant %q", r.code, r.out, r.err, want)
	}
	if got := tree(t, out); len(got) != 0 {
		t.Errorf("the partial tree stays: %v", keys(got))
	}
}

// When the partial tree cannot be removed the run says its files stay; the exit code is still the fault's.
func TestFailedEmitThatCannotRemoveSaysTheFilesStay(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root removes a read-only directory")
	}
	tb := newKit(t)
	out := t.TempDir()
	var locked string
	failWrite(t, 3, func(dir string) {
		locked = dir
		if err := os.Chmod(dir, 0o555); err != nil {
			t.Fatal(err)
		}
	})
	t.Cleanup(func() { os.Chmod(locked, 0o755) })
	r := run(t, tb, "Foo", out, "--vendor", "Angeles")
	head := "scaffold-module: " + errDiskFull.Error() + "\nscaffold-module: could not remove the partial " + out + "/Foo ("
	tail := "): its files stay\n"
	if r.code != 3 || r.out != "" || len(r.err) < len(head)+len(tail) || r.err[:len(head)] != head || r.err[len(r.err)-len(tail):] != tail {
		t.Errorf("code=%d out=%q err=%q", r.code, r.out, r.err)
	}
	if !isDir(locked) {
		t.Errorf("%s is gone", locked)
	}
}

// A destination that exists before the run is never this run's to remove: a directory with a file in it is
// refused and left whole.
func TestExistingDestinationDirectoryIsLeftWhole(t *testing.T) {
	tb := newKit(t)
	out := t.TempDir()
	mine := filepath.Join(out, "Foo", "mine.txt")
	if err := os.MkdirAll(filepath.Dir(mine), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mine, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	failWrite(t, 1, nil) // no write is reached
	r := run(t, tb, "Foo", out, "--vendor", "Angeles")
	if r.code != 3 || r.out != "" || r.err != "scaffold-module: destination already exists: "+out+"/Foo\n" {
		t.Errorf("code=%d out=%q err=%q", r.code, r.out, r.err)
	}
	if got := tree(t, out); len(got) != 1 || got["Foo/mine.txt"] != "mine" {
		t.Errorf("existing destination changed: %v", got)
	}
}
