package mirrorhome

import (
	"os"
	"strings"
	"testing"
)

// TestLinkAfterDotDotIntoSource (verifier of #785, B2): a mirror dir spelled through a missing directory
// and then a link to the source is judged inside the source, and nothing is created there first.
func TestLinkAfterDotDotIntoSource(t *testing.T) {
	f := newFixture(t)
	f.symlink(f.path("nh"), "srclink")
	before := f.tree("")
	mir := f.path("missing/../srclink/a/b/c/x")
	f.wants("into the source", f.mirror(f.path("nh"), mir), ExitRefused, "", "refusing: mirror dir is the source install or inside it: "+mir+"\n")
	f.unchanged("into the source", "", before)
}

// TestMirrorThroughLinkDotDot (verifier of #785, A1): "ml/../o" with ml a link to <out>/o names <out>/o; the
// mirror there is wiped and rebuilt at that same path, and the link stays.
func TestMirrorThroughLinkDotDot(t *testing.T) {
	f := newFixture(t)
	f.touch("out/o/"+Marker, "out/o/old")
	f.symlink(f.path("out/o"), "ml")
	mir := f.path("ml/../o")
	f.wants("ml/../o", f.mirror(f.path("nh"), mir), ExitOK, "mirror ready: "+mir+"  (3 jars linked, modules/ writable, excluded: none)\n", "")
	got := f.tree("out")
	if strings.Contains(got, "out/o/old") || !strings.Contains(got, "out/o/modules/baja.jar L") || !strings.Contains(got, "out/o/"+Marker) {
		t.Errorf("the mirror was not rebuilt at out/o:\n%s", got)
	}
	if target, err := os.Readlink(f.path("ml")); err != nil || target != f.path("out/o") {
		t.Errorf("ml is no longer the link: %q %v", target, err)
	}
}

// TestNothingMadeBeforeEveryCheck (verifier of #785, A3): a refused run (here a relative source) creates no
// directory, even for a mirror dir whose parents are missing.
func TestNothingMadeBeforeEveryCheck(t *testing.T) {
	f := newFixture(t)
	t.Chdir(f.dir)
	before := f.tree("")
	f.wants("relative source", f.mirror("nh", "work/new/mir"), ExitUsage, "",
		"refusing: source_niagara_home must be an absolute path (the mirror's links would dangle): nh\n")
	f.unchanged("relative source", "", before)
}

// TestOptionLikeArguments (verifier of #785, A4): a source or mirror dir starting with a dash ("-" too) is a
// usage error (exit 1) and nothing is made.
func TestOptionLikeArguments(t *testing.T) {
	f := newFixture(t)
	t.Chdir(f.dir)
	before := f.tree("")
	for _, args := range [][]string{{f.path("nh"), "-"}, {f.path("nh"), "--"}, {f.path("nh"), "-x"}, {"-", "mir"}, {"--help", "mir"}} {
		bad := args[1]
		if strings.HasPrefix(args[0], "-") {
			bad = args[0]
		}
		want := "mirror-niagara-home: " + bad + " looks like an option, not a path; spell a path that starts with - as ./" + bad + "\n"
		f.wants(strings.Join(args, " "), f.mirror(args...), ExitUsage, "", want)
		f.unchanged(strings.Join(args, " "), "", before)
	}
}
