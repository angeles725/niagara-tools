package buildmodule

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// newBuild is a build with no options run, for the helpers that need only its streams and environment; its
// stderr is the second result.
func (f *fixture) newBuild() (*build, *lockedBuffer) {
	var out, errs lockedBuffer
	return &build{s: cli.Stdio{Out: &out, Err: &errs, Getenv: os.Getenv}, setenv: map[string]string{}}, &errs
}

// TestCopyKeepsTheModeAndTheTimes: the copy is cp -p's: the bytes, the mode (executable and read-only
// included) and the modification time of the source, and an existing destination is overwritten in place.
func TestCopyKeepsTheModeAndTheTimes(t *testing.T) {
	f := newFixture(t)
	src, dst := f.dir+"/src.jar", f.dir+"/dst.jar"
	f.write(src, "payload")
	f.write(dst, "a longer previous content")
	if err := os.Chmod(src, 0o540); err != nil {
		t.Fatal(err)
	}
	when := time.Date(2020, 2, 3, 4, 5, 6, 0, time.UTC)
	if err := os.Chtimes(src, when, when); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if f.read(dst) != "payload" || info.Mode().Perm() != 0o540 || !info.ModTime().Equal(when) {
		t.Errorf("copy holds %q mode %v mtime %v", f.read(dst), info.Mode().Perm(), info.ModTime())
	}
}

// TestCopyRefusesWhatCpRefuses: a missing source, a source that is a directory, a destination that is the
// same file or a directory each fail with a message naming the path, and the source is never truncated.
func TestCopyRefusesWhatCpRefuses(t *testing.T) {
	f := newFixture(t)
	src := f.dir + "/src.jar"
	f.write(src, "payload")
	f.mkdir(f.dir + "/adir")
	if err := os.Link(src, f.dir+"/hard.jar"); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ name, from, to, want string }{
		{"missing source", f.dir + "/none.jar", f.dir + "/x.jar", "cannot read " + f.dir + "/none.jar: No such file or directory"},
		{"source is a directory", f.dir + "/adir", f.dir + "/x.jar", "cannot read " + f.dir + "/adir: Is a directory"},
		{"destination is the source", src, src, "cannot write " + src + ": it is the file being copied"},
		{"destination is a hard link of the source", src, f.dir + "/hard.jar", "cannot write " + f.dir + "/hard.jar: it is the file being copied"},
		{"destination is a directory", src, f.dir + "/adir", "cannot write " + f.dir + "/adir: Is a directory"},
		{"no parent directory", src, f.dir + "/no/such/x.jar", "cannot write " + f.dir + "/no/such/x.jar: No such file or directory"},
	} {
		b, _ := f.newBuild()
		reason, code := b.copyPreserving(c.from, c.to)
		if code != ExitFault || reason != c.want {
			t.Errorf("%s: code %d reason %q, want %d and %q", c.name, code, reason, ExitFault, c.want)
		}
	}
	if f.read(src) != "payload" {
		t.Errorf("the source was changed: %q", f.read(src))
	}
}

// TestMakeDirAndTempNameTheirFault: mkdir -p makes parents and accepts an existing directory, mkdir refuses
// an existing path, and each failure is one stderr line naming the path and exit 1.
func TestMakeDirAndTempNameTheirFault(t *testing.T) {
	f := newFixture(t)
	b, errs := f.newBuild()
	if code := b.makeDir(f.dir+"/a/b/c", true); code != 0 {
		t.Errorf("mkdir -p: %d", code)
	}
	if code := b.makeDir(f.dir+"/a/b/c", true); code != 0 {
		t.Errorf("mkdir -p of an existing directory: %d", code)
	}
	if code := b.makeDir(f.dir+"/a/b/c", false); code != ExitFault || !strings.Contains(errs.String(), "build.sh: cannot create directory "+f.dir+"/a/b/c: File exists\n") {
		t.Errorf("mkdir of an existing path: %d %q", code, errs.String())
	}
	f.write(f.dir+"/plain", "")
	if code := b.makeDir(f.dir+"/plain/x", true); code != ExitFault || !strings.Contains(errs.String(), "cannot create directory "+f.dir+"/plain/x: Not a directory") {
		t.Errorf("mkdir under a file: %d %q", code, errs.String())
	}
	t.Setenv("TMPDIR", f.dir+"/tmp-here")
	f.mkdir(f.dir + "/tmp-here")
	dir, code := b.makeTemp(true)
	file, code2 := b.makeTemp(false)
	if code != 0 || code2 != 0 || !strings.HasPrefix(dir, f.dir+"/tmp-here/tmp.") || !strings.HasPrefix(file, f.dir+"/tmp-here/tmp.") {
		t.Errorf("temporary paths %q (%d) %q (%d)", dir, code, file, code2)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Errorf("%s is no directory", dir)
	}
	if info, err := os.Stat(file); err != nil || !info.Mode().IsRegular() {
		t.Errorf("%s is no file", file)
	}
}
