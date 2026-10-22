package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// TestCommand runs the real registry end to end the way toolbelt/lint-null-context-write.sh does.
func TestCommand(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	if err := os.Mkdir(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "A.java"), []byte("class A { void f(){ c.set(p, v, null); } }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	code := command([]string{"lint-null-context-write", root}, cli.Stdio{Out: &o, Err: &e})
	want := "WARN  lint-null-context-write  " + src + "/A.java:1  null-Context write: class A { void f(){ c.set(p, v, null); } }\n"
	if code != 0 || o.String() != want || e.Len() != 0 {
		t.Errorf("exit %d out %q err %q\nwant 0 %q", code, o.String(), e.String(), want)
	}
}
