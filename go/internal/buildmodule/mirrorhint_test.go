package buildmodule

import (
	"strings"
	"testing"
)

// mirror makes a mirror directory under the fixture's $HOME whose marker records source.
func (f *fixture) mirror(name, source string) string {
	dir := f.env["HOME"] + "/" + name
	f.write(dir+"/.niagara-mirror", "source="+source+"\ncreated=2026-11-10T11:23:37\n")
	return dir
}

// A real niagara_home: the mirrors made from it come first, then the others, and the try line names
// the first match; the build still stops (issue #1088).
func TestARealHomeListsTheMirrorsMadeFromItFirst(t *testing.T) {
	f := newFixture(t)
	other := f.mirror("niagara-mirror-b", "/mnt/c/Other")
	same := f.mirror("niagara-mirror-z", f.home)
	if code := f.run(f.root, "Foo", f.home); code != ExitEnv {
		t.Fatalf("exit %d, want %d: %s", code, ExitEnv, f.err)
	}
	want := "  mirrors under " + f.env["HOME"] + " (made from this installation first):\n" +
		"    " + same + "  (from " + f.home + ")\n" +
		"    " + other + "  (from /mnt/c/Other)\n" +
		"  try: build.sh " + f.root + " Foo " + same + "\n"
	if !strings.HasSuffix(f.err, want) {
		t.Errorf("stderr:\n%s\nwant suffix:\n%s", f.err, want)
	}
}

// A missing niagara_home lists every mirror; the try line names the first one in name order.
func TestAMissingHomeListsTheMirrors(t *testing.T) {
	f := newFixture(t)
	a := f.mirror("niagara-mirror-a", "/mnt/c/A")
	f.write(f.env["HOME"]+"/niagara-mirror-notone/readme", "") // no marker: not a mirror
	if code := f.run("--install-into-home", f.root, "Foo"); code != ExitEnv {
		t.Fatalf("exit %d: %s", code, f.err)
	}
	want := "build.sh: pass niagara_home (arg 3) or export niagara_home\n" +
		"  mirrors under " + f.env["HOME"] + ":\n" +
		"    " + a + "  (from /mnt/c/A)\n" +
		"  try: build.sh " + f.root + " Foo " + a + "\n"
	if f.err != want {
		t.Errorf("stderr:\n%s\nwant:\n%s", f.err, want)
	}
}

// No mirror at all: a missing niagara_home says how to make one; a real one keeps its message, which
// already names mirror-niagara-home.sh.
func TestNoMirrorSaysHowToMakeOne(t *testing.T) {
	f := newFixture(t)
	f.run("--install-into-home", f.root, "Foo")
	want := "build.sh: pass niagara_home (arg 3) or export niagara_home\n" +
		"  no mirror under " + f.env["HOME"] + ": make one with toolbelt/mirror-niagara-home.sh <installation> " +
		f.env["HOME"] + "/niagara-mirror-<version>\n"
	if f.err != want {
		t.Errorf("stderr:\n%s\nwant:\n%s", f.err, want)
	}
}
