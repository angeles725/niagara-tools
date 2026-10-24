package safety

import "testing"

// The os.path behaviours the result paths rely on: no cleaning, Python's dirname, expanduser.
func TestPathHelpersFollowOsPath(t *testing.T) {
	for _, c := range [][3]string{{"/s/", "bql", "/s/bql"}, {"s/.", "bql", "s/./bql"}, {"/s", "/abs", "/abs"}, {"", "x", "x"}} {
		if got := Join(c[0], c[1]); got != c[2] {
			t.Errorf("Join(%q, %q) = %q, want %q", c[0], c[1], got, c[2])
		}
	}
	for in, want := range map[string]string{"/s/bql/x": "/s/bql", "/x": "/", "x": "", "//x": "//", "a//b": "a", "/s/bql/": "/s/bql"} {
		if got := Dirname(in); got != want {
			t.Errorf("Dirname(%q) = %q, want %q", in, got, want)
		}
	}
	t.Setenv("HOME", "/h//")
	for in, want := range map[string]string{"~": "/h", "~/a": "/h/a", "a~": "a~", "~nosuchuser-x/a": "~nosuchuser-x/a"} {
		if got := ExpandUser(in); got != want {
			t.Errorf("ExpandUser(%q) = %q, want %q", in, got, want)
		}
	}
	t.Setenv("HOME", "/")
	if got := ExpandUser("~"); got != "/" {
		t.Errorf("ExpandUser(~) with HOME=/ = %q", got)
	}
}
