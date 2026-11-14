package mirrorhome

import (
	"os"
	"strings"
	"testing"
)

const usage = "usage: mirror-niagara-home.sh <source_niagara_home> <mirror_dir> [exclude-jar ...]\n"

// TestUsage: fewer than two arguments is the usage line and exit 1.
func TestUsage(t *testing.T) {
	f := newFixture(t)
	f.wants("no argument", f.mirror(), ExitUsage, "", usage)
	f.wants("one argument", f.mirror(f.path("nh")), ExitUsage, "", usage)
}

// TestNotANiagaraHome: a source without etc/m2/repository is exit 1 and nothing is made.
func TestNotANiagaraHome(t *testing.T) {
	f := newFixture(t)
	for _, src := range []string{f.path("nh/bin"), f.path("nope"), ""} {
		f.wants(src, f.mirror(src, f.path("mir")), ExitUsage, "", "not a niagara_home (no etc/m2/repository): "+src+"\n")
	}
	if _, err := os.Lstat(f.path("mir")); err == nil {
		t.Error("the mirror dir was made")
	}
}

// TestMirror: every top-level entry but modules/ and the dot files is linked, modules/ is a real directory
// linking every jar but the excluded ones, the marker names the resolved source, and the count is printed.
func TestMirror(t *testing.T) {
	f := newFixture(t)
	f.mkdir("nh/modules/dir.jar")
	f.symlink("gone", "nh/modules/dangling.jar")
	f.symlink("nh", "link")
	src, mir := f.path("link"), f.path("a/b/mir")
	before := f.tree("nh")
	code := f.mirror(src, mir, "Demo-rt.jar", "absent.jar")
	f.wants("mirror", code, ExitOK, "mirror ready: "+mir+"  (3 jars linked, modules/ writable, excluded: Demo-rt.jar absent.jar)\n", "")
	f.unchanged("mirror", "nh", before)
	want := strings.Join([]string{
		"a/b/mir d---------",
		"a/b/mir/.niagara-mirror ----------",
		"a/b/mir/bin L--------- -> " + src + "/bin",
		"a/b/mir/etc L--------- -> " + src + "/etc",
		"a/b/mir/modules d---------",
		"a/b/mir/modules/baja.jar L--------- -> " + src + "/modules/baja.jar",
		"a/b/mir/modules/control-rt.jar L--------- -> " + src + "/modules/control-rt.jar",
		"a/b/mir/modules/dir.jar L--------- -> " + src + "/modules/dir.jar",
		"a/b/mir/top.txt L--------- -> " + src + "/top.txt",
	}, "\n")
	if got := f.tree("a/b/mir"); got != want {
		t.Errorf("mirror tree:\n%s\nwant:\n%s", got, want)
	}
	marker, _ := os.ReadFile(mir + "/" + Marker)
	if want := "source=" + f.path("nh") + "\ncreated=2026-03-04T05:06:07\n"; string(marker) != want {
		t.Errorf("marker %q, want %q", marker, want)
	}
	if err := os.WriteFile(mir+"/modules/Demo-rt.jar", nil, 0o644); err != nil {
		t.Errorf("modules/ is not writable: %v", err)
	}
}

// TestMirrorCounts: no exclusion prints "none" (an empty one too), a source spelled with a final slash keeps
// it in the links, and an empty or absent modules/ links 0 jars.
func TestMirrorCounts(t *testing.T) {
	f := newFixture(t)
	mir := f.path("mir")
	f.wants("no exclusion", f.mirror(f.path("nh")+"/", mir, ""), ExitOK, "mirror ready: "+mir+"  (3 jars linked, modules/ writable, excluded: none)\n", "")
	if target, _ := os.Readlink(mir + "/modules/baja.jar"); target != f.path("nh")+"//modules/baja.jar" {
		t.Errorf("jar link -> %q", target)
	}
	for _, jar := range []string{"baja.jar", "Demo-rt.jar", "control-rt.jar"} {
		os.Remove(f.path("nh/modules/" + jar))
	}
	f.wants("no jar", f.mirror(f.path("nh"), mir+"/"), ExitOK, "mirror ready: "+mir+"/  (0 jars linked, modules/ writable, excluded: none)\n", "")
	os.RemoveAll(f.path("nh/modules"))
	f.wants("no modules/", f.mirror(f.path("nh"), mir), ExitOK, "mirror ready: "+mir+"  (0 jars linked, modules/ writable, excluded: none)\n", "")
}

// TestRefresh: a second run on its own mirror rebuilds it: build leftovers go (a built jar, build.sh's
// .mirror-user-home) and the source is never written.
func TestRefresh(t *testing.T) {
	f := newFixture(t)
	mir := f.path("mir")
	f.wants("first", f.mirror(f.path("nh"), mir, "Demo-rt.jar"), ExitOK, "mirror ready: "+mir+"  (2 jars linked, modules/ writable, excluded: Demo-rt.jar)\n", "")
	f.touch("mir/modules/built.jar", "mir/.mirror-user-home/.tridium/security/niagara.signing.xml")
	before := f.tree("nh")
	f.wants("second", f.mirror(f.path("nh"), mir), ExitOK, "mirror ready: "+mir+"  (3 jars linked, modules/ writable, excluded: none)\n", "")
	f.unchanged("second", "nh", before)
	got := f.tree("mir")
	for _, gone := range []string{"built.jar", ".mirror-user-home"} {
		if strings.Contains(got, gone) {
			t.Errorf("%s survived the refresh:\n%s", gone, got)
		}
	}
	if !strings.Contains(got, "mir/modules/Demo-rt.jar L") {
		t.Errorf("the jar excluded before is not linked now:\n%s", got)
	}
}

// TestSymlinkedMirrorDir: a mirror dir that is a symlink to a mirror is replaced by a real directory; what
// the link pointed at is left alone.
func TestSymlinkedMirrorDir(t *testing.T) {
	f := newFixture(t)
	f.touch("real/"+Marker, "real/old/file")
	f.symlink("real", "mir")
	before := f.tree("real")
	f.wants("link", f.mirror(f.path("nh"), f.path("mir")), ExitOK, "mirror ready: "+f.path("mir")+"  (3 jars linked, modules/ writable, excluded: none)\n", "")
	f.unchanged("link", "real", before)
	if st, err := os.Lstat(f.path("mir")); err != nil || !st.IsDir() {
		t.Errorf("mir is not a real directory: %v", err)
	}
}

// TestSymlinkedMirrorDirWithSlash: the same link spelled with a final slash names its target: the target
// is emptied and rebuilt as the mirror, and the link stays.
func TestSymlinkedMirrorDirWithSlash(t *testing.T) {
	f := newFixture(t)
	f.touch("real/"+Marker, "real/old/file")
	f.symlink("real", "mir")
	f.wants("link/", f.mirror(f.path("nh"), f.path("mir")+"/"), ExitOK, "mirror ready: "+f.path("mir")+"/  (3 jars linked, modules/ writable, excluded: none)\n", "")
	if target, err := os.Readlink(f.path("mir")); err != nil || target != "real" {
		t.Errorf("mir is no longer the link: %q %v", target, err)
	}
	got := f.tree("real")
	if strings.Contains(got, "real/old") || !strings.Contains(got, "real/modules/baja.jar L") || !strings.Contains(got, "real/"+Marker) {
		t.Errorf("the target was not rebuilt as the mirror:\n%s", got)
	}
}
