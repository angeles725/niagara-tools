package reportmodule

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBashInt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		n    int64
		isOK bool
	}{
		{"100", 100, true}, {"+100", 100, true}, {"-5", -5, true}, {" 7\t", 7, true}, {"\n9", 9, true},
		{"", 0, false}, {"+", 0, false}, {"0x10", 0, false}, {"9\n", 0, false}, {"99999999999999999999", 0, false},
	}
	for _, tt := range tests {
		if n, ok := bashInt(tt.in); ok != tt.isOK || (ok && n != tt.n) {
			t.Errorf("bashInt(%q) = %d, %v; want %d, %v", tt.in, n, ok, tt.n, tt.isOK)
		}
	}
}

func TestBaseName(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{"": "", "/": "/", "//": "/", "/t/dir/": "dir", "a/b": "b", "b": "b", "/a//": "a"} {
		if got := baseName(in); got != want {
			t.Errorf("baseName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestVersionMajor(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{"v9.12.0": "9", "10.0.1": "10", "v8.57.0": "8", "v9": "", "9": "", "nightly": "",
		"": "", "vv9.1": "", "9.": "9", "v.1": ""} {
		if got := versionMajor(in); got != want {
			t.Errorf("versionMajor(%q) = %q, want %q", in, got, want)
		}
	}
}

// tree writes files (path -> content; a trailing "/" makes a directory) under a temp dir and returns it.
func tree(t *testing.T, files ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, f := range files {
		p := filepath.Join(root, f)
		if f[len(f)-1] == '/' {
			must(t, os.MkdirAll(p, 0o755))
			continue
		}
		must(t, os.MkdirAll(filepath.Dir(p), 0o755))
		must(t, os.WriteFile(p, []byte("x"), 0o644))
	}
	return root
}

func TestFirstJar(t *testing.T) {
	t.Parallel()
	root := tree(t, "libs/notes.txt", "libs/only.jar", "empty/", "dirjar/d.jar/")
	if got := firstJar(root + "/libs"); got != root+"/libs/only.jar" {
		t.Errorf("firstJar = %q", got)
	}
	if got := firstJar(root + "/dirjar"); got != root+"/dirjar/d.jar" {
		t.Errorf("a directory named *.jar is found by find -name too: got %q", got)
	}
	must(t, os.Symlink(root+"/libs", root+"/link"))
	for _, d := range []string{"empty", "missing", "link"} {
		if got := firstJar(root + "/" + d); got != "" {
			t.Errorf("firstJar(%s) = %q, want none (find -P does not enter a symlinked start point)", d, got)
		}
	}
}

func TestHasLexicon(t *testing.T) {
	t.Parallel()
	root := tree(t, "a/m.lexicon", "b/sub/m.lexicon", "c/x/y/m.lexicon", "d/.lexicon", "e.lexicon/", "f/plain")
	for d, want := range map[string]bool{"a": true, "b": true, "c": false, "d": true, "e.lexicon": true, "f": false} {
		if got := hasLexicon(root + "/" + d); got != want {
			t.Errorf("hasLexicon(%s) = %v, want %v", d, got, want)
		}
	}
}

func TestHasOwnJS(t *testing.T) {
	t.Parallel()
	none := tree(t, "rc/vendor/v.js", "rc/ext/e.js", "rc/x.min.js", "rc/.hid/h.js", "rc/.d.js", "rc/a/vendor/z.js", "rc/s.css")
	must(t, os.Symlink("s.css", none+"/rc/link.js"))
	if hasOwnJS(none + "/rc") {
		t.Error("vendor/, ext/, dot entries, *.min.js and a symlink are not own js")
	}
	if !hasOwnJS(tree(t, "rc/js/deep/app.js") + "/rc") {
		t.Error("a nested js file is own js")
	}
}

// TestFirstJarSkipsSourcesAndJavadoc: the built module jar is chosen over its -sources/-javadoc siblings,
// whatever the directory order, and the choice is the same on every run (byte order).
func TestFirstJarSkipsSourcesAndJavadoc(t *testing.T) {
	t.Parallel()
	for _, order := range [][]string{
		{"X-rt-javadoc.jar", "X-rt-sources.jar", "X-rt.jar"},
		{"X-rt.jar", "X-rt-sources.jar", "X-rt-javadoc.jar"},
		{"X-rt-sources.jar", "X-rt.jar", "X-rt-javadoc.jar"},
	} {
		files := make([]string, len(order))
		for i, n := range order {
			files[i] = "libs/" + n
		}
		root := tree(t, files...)
		if got := firstJar(root + "/libs"); got != root+"/libs/X-rt.jar" {
			t.Errorf("created %q: firstJar = %q, want X-rt.jar", order, filepath.Base(got))
		}
	}
	root := tree(t, "libs/b.jar", "libs/a.jar", "only/X-sources.jar")
	if got := firstJar(root + "/libs"); filepath.Base(got) != "a.jar" {
		t.Errorf("two module jars: firstJar = %q, want the first in byte order", got)
	}
	if got := firstJar(root + "/only"); filepath.Base(got) != "X-sources.jar" {
		t.Errorf("only a sources jar: firstJar = %q, want it (verify-module then reports it)", got)
	}
}
