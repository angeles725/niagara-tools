package orientguard

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCanonical: realpath's answers, one per kind of component it meets.
func TestCanonical(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"a/b", "c"} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(dir+"/a/file", nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for link, target := range map[string]string{"a/abs": dir + "/c", "a/rel": "../c", "a/dangling": "nowhere", "a/loop": "loop", "a/chain": "rel"} {
		if err := os.Symlink(target, filepath.Join(dir, link)); err != nil {
			t.Fatal(err)
		}
	}
	const fails = "\x00"
	cases := []struct{ name, strict, lenient string }{
		{dir + "/a/b", dir + "/a/b", dir + "/a/b"},
		{dir + "//a/./b/..//b/", dir + "/a/b", dir + "/a/b"},
		{"a/b", dir + "/a/b", dir + "/a/b"},
		{"../../../../../../../../../../..", "/", "/"},
		{dir + "/a/abs", dir + "/c", dir + "/c"},
		{dir + "/a/rel/..", dir, dir},
		{dir + "/a/chain", dir + "/c", dir + "/c"},
		{dir + "/a/file", dir + "/a/file", dir + "/a/file"},
		{dir + "/a/file/", fails, dir + "/a/file"},
		{dir + "/a/file/.", fails, dir + "/a/file"},
		{dir + "/a/file/x", fails, dir + "/a/file/x"},
		{dir + "/a/file/..", fails, dir + "/a"},
		{dir + "/a/new", dir + "/a/new", dir + "/a/new"},
		{dir + "/a/new/", dir + "/a/new", dir + "/a/new"},
		{dir + "/a/new/.", fails, dir + "/a/new"},
		{dir + "/a/new/..", fails, dir + "/a"},
		{dir + "/a/new/x", fails, dir + "/a/new/x"},
		{dir + "/a/new/../../c", fails, dir + "/c"},
		{dir + "/a/dangling", dir + "/a/nowhere", dir + "/a/nowhere"},
		{dir + "/a/dangling/", dir + "/a/nowhere", dir + "/a/nowhere"},
		{dir + "/a/dangling/x", fails, dir + "/a/nowhere/x"},
		{dir + "/a/loop", fails, fails},
		{"", fails, fails},
	}
	t.Chdir(dir)
	for _, c := range cases {
		for _, m := range []struct {
			mode    string
			resolve func(name, cwd string) (string, bool)
			want    string
		}{{"canonical", canonical, c.strict}, {"reachable", Reachable, c.lenient}} {
			got, ok := m.resolve(c.name, dir)
			if !ok {
				got = fails
			}
			if got != m.want {
				t.Errorf("%s(%q) = %q, want %q", m.mode, c.name, got, m.want)
			}
		}
	}
}

// TestNeedsDirectory: what follows a component decides whether it must be a directory.
func TestNeedsDirectory(t *testing.T) {
	for rest, want := range map[string]bool{"": false, "/": true, "//": true, "/.": true, "/..": true, "/../x": true, "/./": true,
		"/./x": false, "/x": false, "/x/": false, "/.x": false, "/..x": false, "/.//..": true} {
		if got := needsDirectory(rest); got != want {
			t.Errorf("needsDirectory(%q) = %v, want %v", rest, got, want)
		}
	}
}
