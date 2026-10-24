package stationsnapshot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBasenameDirnameArePOSIX(t *testing.T) {
	for _, c := range []struct{ in, base, dir string }{
		{"", "", "."}, {"/", "/", "/"}, {"//", "/", "/"}, {"out", "out", "."}, {"out/", "out", "."},
		{"a/b", "b", "a"}, {"a//b//", "b", "a"}, {"/a", "a", "/"}, {"/a/b/", "b", "/a"}, {"./x", "x", "."},
	} {
		if got := basename(c.in); got != c.base {
			t.Errorf("basename(%q) = %q, want %q", c.in, got, c.base)
		}
		if got := dirname(c.in); got != c.dir {
			t.Errorf("dirname(%q) = %q, want %q", c.in, got, c.dir)
		}
	}
}

func TestOnPathFindsARegularFileOnly(t *testing.T) {
	d := t.TempDir()
	if err := os.Mkdir(filepath.Join(d, "dirtool"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "tool"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if !onPath("tool", "/nonexistent:"+d) {
		t.Error("tool on PATH not found")
	}
	if onPath("dirtool", d) || onPath("tool", "/nonexistent") {
		t.Error("a directory, or a PATH without the tool, counted as found")
	}
}
