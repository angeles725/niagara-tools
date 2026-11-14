package findwalk

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// tree creates each path under a fresh directory; a trailing "/" makes a directory, else an empty file.
func tree(t *testing.T, paths ...string) string {
	t.Helper()
	d := t.TempDir()
	for _, p := range paths {
		full := filepath.Join(d, p)
		if p[len(p)-1] == '/' {
			if err := os.MkdirAll(full, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return d
}

func TestWalkVisitsInNameOrderWithinMaxDepth(t *testing.T) {
	t.Parallel()
	d := tree(t, "b/x/deep/f", "a/f", "c")
	var got []string
	Walk(d, false, 2, func(e Entry) bool {
		got = append(got, e.Path[len(d):])
		return true
	}, nil)
	want := []string{"", "/a", "/a/f", "/b", "/b/x", "/c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("visited %q, want %q", got, want)
	}
}

func TestWalkDoesNotEnterAPrunedDirectory(t *testing.T) {
	t.Parallel()
	d := tree(t, "skip/f", "keep/f")
	var got []string
	Walk(d, false, NoLimit, func(e Entry) bool {
		got = append(got, e.Name)
		return e.Name != "skip"
	}, nil)
	want := []string{filepath.Base(d), "keep", "f", "skip"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("visited %q, want %q", got, want)
	}
}

func TestWalkFollowsTheRootSymlinkOnlyWhenAsked(t *testing.T) {
	t.Parallel()
	d := tree(t, "real/f")
	link := filepath.Join(d, "link")
	if err := os.Symlink(filepath.Join(d, "real"), link); err != nil {
		t.Fatal(err)
	}
	for follow, want := range map[bool]int{true: 2, false: 1} {
		n := 0
		Walk(link, follow, NoLimit, func(Entry) bool { n++; return true }, nil)
		if n != want {
			t.Errorf("followRoot %v: %d entries, want %d", follow, n, want)
		}
	}
}

func TestWalkReportsAMissingRoot(t *testing.T) {
	t.Parallel()
	var failed string
	Walk("/nonexistent-findwalk", false, NoLimit, func(Entry) bool { return true }, func(p string, _ error) { failed = p })
	if failed != "/nonexistent-findwalk" {
		t.Errorf("onErr path %q", failed)
	}
}

func TestHasDir(t *testing.T) {
	t.Parallel()
	d := tree(t, "a/b/7.6.17/", "a/7.6.2", "a/b/c/d/9.9/")
	cases := []struct {
		name  string
		depth int
		want  bool
	}{
		{"7.6.17", NoLimit, true},
		{"7.6.1", NoLimit, false}, // an exact name, never a prefix
		{"7.6.2", NoLimit, false}, // a file is not a directory
		{"7.6.*", NoLimit, true},  // find -name takes a glob
		{"9.9", 4, false},         // deeper than -maxdepth 4
		{"9.9", 5, true},
	}
	for _, c := range cases {
		if got := HasDirWithin(d, c.name, c.depth); got != c.want {
			t.Errorf("HasDirWithin(%q, %d) = %v, want %v", c.name, c.depth, got, c.want)
		}
	}
	if !HasDir(d, filepath.Base(d)) {
		t.Error("HasDir does not match the root itself")
	}
}
