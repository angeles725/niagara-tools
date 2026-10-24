package structure

import "testing"

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
