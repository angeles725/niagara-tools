package orientguard

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// site is a temporary tree with a legal prefix (<dir>/Cliente/) and places inside and outside it.
type site struct {
	t      *testing.T
	dir    string // canonical, so the rows the guard prints can be compared with it
	prefix string
	env    map[string]string
}

func newSite(t *testing.T) *site {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := &site{t: t, dir: dir, prefix: dir + "/Cliente/"}
	s.env = map[string]string{PrefixEnv: s.prefix, "HOME": dir + "/home"}
	s.mkdir("Cliente/DemoPan", "ClienteX/DemoPan", "other")
	return s
}

func (s *site) mkdir(rel ...string) {
	s.t.Helper()
	for _, r := range rel {
		if err := os.MkdirAll(filepath.Join(s.dir, r), 0o755); err != nil {
			s.t.Fatal(err)
		}
	}
}

func (s *site) symlink(target, rel string) {
	s.t.Helper()
	if err := os.Symlink(target, filepath.Join(s.dir, rel)); err != nil {
		s.t.Fatal(err)
	}
}

// guard runs the guard on args and returns its exit code, stdout and stderr.
func (s *site) guard(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := Main(args, cli.Stdio{Out: &out, Err: &errb, Getenv: func(k string) string { return s.env[k] }})
	return code, out.String(), errb.String()
}

// wants runs the guard on one argument and compares exit code and stdout; stderr must be empty.
func (s *site) wants(name, arg string, code int, stdout string) {
	s.t.Helper()
	got, out, errs := s.guard(arg)
	if got != code || out != stdout || errs != "" {
		s.t.Errorf("%s: exit %d stdout %q stderr %q, want %d %q", name, got, out, errs, code, stdout)
	}
}

func (s *site) seamRow() string {
	return "WARN  orient-guard  non-default legal root " + s.prefix + " (test seam)\n"
}

// failRow is the FAIL row for path: it names the prefix that was compared, the seam one when it is set.
func (s *site) failRow(path string) string {
	compared := s.dir + "/home/modulos_niagara_n4/Cliente/"
	if seam := s.env[PrefixEnv]; seam != "" {
		compared = withSlash(seam)
	}
	return "FAIL  orient-guard  " + path + " — must be under " + compared + "; set BUILD_N4_CLIENTE_OVERRIDE=1 to override\n"
}

// TestUsage: exactly one argument, else the usage line on stderr and exit 3.
func TestUsage(t *testing.T) {
	s := newSite(t)
	for _, args := range [][]string{nil, {"a", "b"}} {
		if code, out, errs := s.guard(args...); code != ExitUsage || out != "" || errs != "usage: orient-guard.sh <module-root>\n" {
			t.Errorf("%q: exit %d stdout %q stderr %q", args, code, out, errs)
		}
	}
}

// TestInsidePrefix (OG1): a path under the legal prefix is a PASS row and exit 0; anything else is not.
func TestInsidePrefix(t *testing.T) {
	s := newSite(t)
	s.wants("a module", s.dir+"/Cliente/DemoPan", ExitOK, s.seamRow()+"PASS  orient-guard  "+s.dir+"/Cliente/DemoPan\n")
	s.wants("the prefix itself", s.dir+"/Cliente", ExitOK, s.seamRow()+"PASS  orient-guard  "+s.dir+"/Cliente\n")
	s.wants("outside", s.dir+"/other", ExitFail, s.seamRow()+s.failRow(s.dir+"/other"))
	s.wants("the root", "/", ExitFail, s.seamRow()+s.failRow("/"))
}

// TestSiblingOfPrefix (OG2): the compare is by whole component, so ClienteX/ is not under Cliente/, and a
// seam prefix given without its final slash gets one.
func TestSiblingOfPrefix(t *testing.T) {
	s := newSite(t)
	s.wants("sibling", s.dir+"/ClienteX/DemoPan", ExitFail, s.seamRow()+s.failRow(s.dir+"/ClienteX/DemoPan"))
	s.env[PrefixEnv] = s.dir + "/Cliente"
	s.wants("sibling, seam without slash", s.dir+"/ClienteX/DemoPan", ExitFail, s.seamRow()+s.failRow(s.dir+"/ClienteX/DemoPan"))
	s.wants("inside, seam without slash", s.dir+"/Cliente/DemoPan", ExitOK, s.seamRow()+"PASS  orient-guard  "+s.dir+"/Cliente/DemoPan\n")
}

// TestOverride (OG3): only the exact value 1 turns the FAIL into a WARN that names the real path.
func TestOverride(t *testing.T) {
	s := newSite(t)
	s.env[OverrideEnv] = "1"
	s.wants("1", s.dir+"/other", ExitOK, s.seamRow()+"WARN  orient-guard  non-standard path: "+s.dir+"/other\n")
	s.wants("1 inside the prefix", s.dir+"/Cliente/DemoPan", ExitOK, s.seamRow()+"PASS  orient-guard  "+s.dir+"/Cliente/DemoPan\n")
	for _, v := range []string{"0", "", "yes", "11", " 1", "true"} {
		s.env[OverrideEnv] = v
		s.wants("override "+v, s.dir+"/other", ExitFail, s.seamRow()+s.failRow(s.dir+"/other"))
	}
}

// TestCanonicalizes (OG4): symlinks, relative components, doubled and trailing slashes are resolved before
// the compare, and the row names the resolved path.
func TestCanonicalizes(t *testing.T) {
	s := newSite(t)
	s.symlink(s.dir+"/other", "Cliente/sneaky")
	s.symlink(s.dir+"/Cliente/DemoPan", "other/in")
	in := s.seamRow() + "PASS  orient-guard  " + s.dir + "/Cliente/DemoPan\n"
	s.wants("symlink leaving the prefix", s.dir+"/Cliente/sneaky", ExitFail, s.seamRow()+s.failRow(s.dir+"/other"))
	s.wants("symlink entering the prefix", s.dir+"/other/in", ExitOK, in)
	s.wants("dot components", s.dir+"/Cliente/../other/../Cliente/./DemoPan", ExitOK, in)
	s.wants("slashes", s.dir+"//Cliente//DemoPan//", ExitOK, in)
	s.wants("trailing slash outside", s.dir+"/other/", ExitFail, s.seamRow()+s.failRow(s.dir+"/other"))
	t.Chdir(s.dir)
	s.wants("relative", "Cliente/DemoPan", ExitOK, in)
	t.Chdir(s.dir + "/Cliente/DemoPan")
	s.wants("dot", ".", ExitOK, in)
	s.wants("relative leaving", "../..", ExitFail, s.seamRow()+s.failRow(s.dir))
}

// TestDefaultPrefix (OG10): without the seam the prefix is $HOME/modulos_niagara_n4/Cliente/ and no WARN row
// is printed; no HOME is exit 3; the seam wins over HOME and stays loud.
func TestDefaultPrefix(t *testing.T) {
	s := newSite(t)
	mod := "home/modulos_niagara_n4/Cliente/DemoPan"
	s.mkdir(mod)
	delete(s.env, PrefixEnv)
	s.wants("inside", s.dir+"/"+mod, ExitOK, "PASS  orient-guard  "+s.dir+"/"+mod+"\n")
	s.wants("outside", s.dir+"/other", ExitFail, s.failRow(s.dir+"/other"))
	s.env["HOME"] = s.dir + "/home/"
	s.wants("HOME with a final slash", s.dir+"/"+mod, ExitOK, "PASS  orient-guard  "+s.dir+"/"+mod+"\n")
	s.env["HOME"] = s.dir + "/home///"
	s.wants("HOME with a run of final slashes", s.dir+"/"+mod, ExitOK, "PASS  orient-guard  "+s.dir+"/"+mod+"\n")
	s.wants("a run of final slashes, outside: the FAIL row names the normalised prefix", s.dir+"/other", ExitFail, s.failRow(s.dir+"/other"))
	s.env["HOME"] = s.dir + "/home"
	s.env[PrefixEnv] = ""
	s.wants("an empty seam is no seam", s.dir+"/"+mod, ExitOK, "PASS  orient-guard  "+s.dir+"/"+mod+"\n")
	s.env["HOME"] = ""
	if code, out, errs := s.guard("/tmp"); code != ExitUsage || out != "" || errs != "orient-guard: HOME is not set; cannot derive the legal root\n" {
		t.Errorf("no HOME: exit %d stdout %q stderr %q", code, out, errs)
	}
	s.env[PrefixEnv], s.env["HOME"] = s.prefix, s.dir+"/nowhere"
	s.wants("the seam wins over HOME", s.dir+"/Cliente/DemoPan", ExitOK, s.seamRow()+"PASS  orient-guard  "+s.dir+"/Cliente/DemoPan\n")
	s.env["HOME"] = ""
	s.wants("the FAIL row names the seam prefix it compared, not the default", s.dir+"/other", ExitFail,
		s.seamRow()+"FAIL  orient-guard  "+s.dir+"/other — must be under "+s.prefix+"; set BUILD_N4_CLIENTE_OVERRIDE=1 to override\n")
}

// TestPathsThatDoNotExist: the longest existing ancestor is resolved and the rest is judged by its text, so
// a new directory several levels deep inside the prefix passes, spelled absolute or relative.
func TestPathsThatDoNotExist(t *testing.T) {
	s := newSite(t)
	s.symlink("nowhere", "Cliente/dangling")
	s.wants("missing last", s.dir+"/Cliente/../Cliente/New/", ExitOK, s.seamRow()+"PASS  orient-guard  "+s.dir+"/Cliente/New\n")
	s.wants("dangling link", s.dir+"/Cliente/dangling", ExitOK, s.seamRow()+"PASS  orient-guard  "+s.dir+"/Cliente/nowhere\n")
	s.wants("two missing levels", s.dir+"/Cliente/New/Deeper", ExitOK, s.seamRow()+"PASS  orient-guard  "+s.dir+"/Cliente/New/Deeper\n")
	t.Chdir(s.dir)
	s.wants("two missing levels, relative", "Cliente/New/Deeper", ExitOK, s.seamRow()+"PASS  orient-guard  "+s.dir+"/Cliente/New/Deeper\n")
	s.wants("three missing levels, relative, a dot-dot among them", "Cliente/New/../Other/Deep/", ExitOK, s.seamRow()+"PASS  orient-guard  "+s.dir+"/Cliente/Other/Deep\n")
	s.wants("missing levels, relative, leaving the prefix", "Cliente/New/../../other/x", ExitFail, s.seamRow()+s.failRow(s.dir+"/other/x"))
	s.wants("missing levels under a sibling of the prefix", "ClienteX/New/Deeper", ExitFail, s.seamRow()+s.failRow(s.dir+"/ClienteX/New/Deeper"))
	s.wants("empty", "", ExitFail, s.seamRow()+s.failRow(""))
}

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

// TestUnresolvedPathNeverPassesOnItsText: a path that cannot be resolved passes only when what does
// resolve of it stays under the prefix; its text alone (.. after a missing directory, a link to a missing
// place outside, a link loop) is not proof.
func TestUnresolvedPathNeverPassesOnItsText(t *testing.T) {
	s := newSite(t)
	s.symlink(s.dir+"/other/missing", "Cliente/away")
	s.symlink("loopb", "Cliente/loopa")
	s.symlink("loopa", "Cliente/loopb")
	s.wants("dot-dot after a missing directory", s.dir+"/Cliente/New/../../other", ExitFail, s.seamRow()+s.failRow(s.dir+"/other"))
	s.wants("below a link to a missing place outside", s.dir+"/Cliente/away/x", ExitFail, s.seamRow()+s.failRow(s.dir+"/other/missing/x"))
	s.wants("link loop", s.dir+"/Cliente/loopa", ExitFail, s.seamRow()+s.failRow(s.dir+"/Cliente/loopa"))
	s.env[OverrideEnv] = "1"
	s.wants("the override still names where it leads", s.dir+"/Cliente/New/../../other", ExitOK, s.seamRow()+"WARN  orient-guard  non-standard path: "+s.dir+"/other\n")
}
