package structure

import "testing"

// TestDirname: the walks up to the .git sentinel step with dirname(1), whose spelling the rows keep
// (a relative root stops at ".", a trailing slash is dropped). Expected values from GNU coreutils.
func TestDirname(t *testing.T) {
	for in, want := range map[string]string{
		"a/b": "a", "a/b/": "a", "a//b": "a", "a": ".", "a/": ".", ".": ".", "./": ".", "..": ".",
		"/": "/", "//": "/", "/a": "/", "/a/b/c": "/a/b", "a/b//": "a",
	} {
		if got := dirname(in); got != want {
			t.Errorf("dirname(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestDotName: find applies -name '.*' -prune to its start point too, so "." and "./" list nothing.
func TestDotName(t *testing.T) {
	for in, want := range map[string]bool{
		".": true, "./": true, "a/..": true, "a/.x": true, "a/.x//": true, "a": false, "a/": false, "/": false, "a/b.c": false,
	} {
		if got := dotName(in); got != want {
			t.Errorf("dotName(%q) = %v, want %v", in, got, want)
		}
	}
}
