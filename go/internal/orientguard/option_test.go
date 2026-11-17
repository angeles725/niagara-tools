package orientguard

import "testing"

// TestOptionLikeArgument: an argument that starts with a dash is a usage error (exit 3, no row), so a
// mistyped flag is never judged as a path; "-" alone and "./-x" are paths.
func TestOptionLikeArgument(t *testing.T) {
	s := newSite(t)
	for _, arg := range []string{"-x", "--help", "-" + s.dir + "/Cliente/DemoPan"} {
		want := "orient-guard: " + arg + " looks like an option, not a module root; spell a path that starts with - as ./" + arg + "\n"
		if code, out, errs := s.guard(arg); code != ExitUsage || out != "" || errs != want {
			t.Errorf("%q: exit %d stdout %q stderr %q, want 3 and %q", arg, code, out, errs, want)
		}
	}
	t.Chdir(s.dir + "/Cliente")
	s.wants("./-x", "./-x", ExitOK, s.seamRow()+"PASS  orient-guard  "+s.dir+"/Cliente/-x\n")
	s.wants("-", "-", ExitOK, s.seamRow()+"PASS  orient-guard  "+s.dir+"/Cliente/-\n")
}
