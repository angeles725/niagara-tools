package lint

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDirnameBasename: expected values from GNU coreutils; the walks up to a .git sentinel (lint-structure,
// lint-write-path) step with Dirname and keep its spelling (a relative root stops at ".").
func TestDirnameBasename(t *testing.T) {
	for in, want := range map[string]string{
		"a/b": "a", "a/b/": "a", "a//b": "a", "a": ".", "a/": ".", ".": ".", "./": ".", "..": ".",
		"/": "/", "//": "/", "/a": "/", "/a/b/c": "/a/b", "a/b//": "a",
	} {
		if got := Dirname(in); got != want {
			t.Errorf("Dirname(%q) = %q, want %q", in, got, want)
		}
	}
	for _, tc := range []struct{ in, dir, base string }{
		{"a/b", "a", "b"}, {"a/b/", "a", "b"}, {"a", ".", "a"}, {".", ".", "."}, {"/", "/", "/"},
		{"/a", "/", "a"}, {"//a//b//", "//a", "b"}, {"a//b", "a", "b"},
	} {
		if d, b := Dirname(tc.in), Basename(tc.in); d != tc.dir || b != tc.base {
			t.Errorf("%q: dirname %q basename %q, want %q and %q", tc.in, d, b, tc.dir, tc.base)
		}
	}
}

func TestIsFileIsDir(t *testing.T) {
	dir := t.TempDir()
	f, link := filepath.Join(dir, "f"), filepath.Join(dir, "l")
	if err := os.WriteFile(f, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	if !IsFile(f) || IsFile(dir) || IsFile(dir+"/none") || !IsDir(dir) || !IsDir(link) || IsDir(f) {
		t.Error("test -f / test -d mismatch")
	}
}
