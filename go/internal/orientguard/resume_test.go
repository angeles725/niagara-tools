package orientguard

import "testing"

// TestResolutionResumesAfterDotDot (verifier of #785, B1): once a ".." climbs from a missing directory back
// into existing ones, symlinks are followed again (as `realpath -m` does), so a link that leaves the prefix
// is judged where it leads.
func TestResolutionResumesAfterDotDot(t *testing.T) {
	s := newSite(t)
	s.mkdir("other/o")
	s.symlink(s.dir+"/other/o", "Cliente/linkout")
	s.wants("link after dot-dot", s.dir+"/Cliente/missing/../linkout", ExitFail, s.seamRow()+s.failRow(s.dir+"/other/o"))
	s.wants("below a link after dot-dot", s.dir+"/Cliente/missing/../linkout/x", ExitFail, s.seamRow()+s.failRow(s.dir+"/other/o/x"))
	s.wants("two missing levels, then the link", s.dir+"/Cliente/m1/m2/../../linkout/x", ExitFail, s.seamRow()+s.failRow(s.dir+"/other/o/x"))
	s.wants("still inside", s.dir+"/Cliente/missing/../DemoPan/New", ExitOK, s.seamRow()+"PASS  orient-guard  "+s.dir+"/Cliente/DemoPan/New\n")
}

// TestNewlineIsPartOfTheName (verifier of #785, A2): a final newline is part of a real name, so a sibling
// "Cliente\n" of the prefix is not the prefix.
func TestNewlineIsPartOfTheName(t *testing.T) {
	s := newSite(t)
	s.mkdir("Cliente\n")
	s.wants("sibling with a newline", s.dir+"/Cliente\n", ExitFail, s.seamRow()+s.failRow(s.dir+"/Cliente\n"))
	s.mkdir("Cliente/Mod\n")
	s.wants("inside, a name with a newline", s.dir+"/Cliente/Mod\n", ExitOK, s.seamRow()+"PASS  orient-guard  "+s.dir+"/Cliente/Mod\n\n")
}
