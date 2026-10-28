package reportmodule

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestModularityRow (modularity-gaps Δ8): one INFO row per module naming its mode, concern count, unlisted
// source files and largest class (hand-written lines, slotomatic regions excluded); INFO is not counted.
func TestModularityRow(t *testing.T) {
	t.Parallel()
	big := "class BBig {\n" + strings.Repeat("  int f;\n", 4) +
		"//region /*+ ------------ BEGIN BAJA AUTO GENERATED CODE ------------ +*/\n" + strings.Repeat("  int g;\n", 9) +
		"//endregion /*+ ------------ END BAJA AUTO GENERATED CODE -------------- +*/\n}\n"
	strict := "modularity: strict\n\n| Concern | Part | Files | Tests | Change here when |\n|---|---|---|---|---|\n" +
		"| control | rt | M-rt/src/A.java | M-rt/srcTest/ATest.java | the decision |\n" +
		"| ui | ux | M-ux/src/rc/** | M-rt/srcTest/ATest.java | the screen |\n"
	tests := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"no map", map[string]string{"M-rt/src/A.java": "class A {\n  int a;\n}\n", "M-rt/src/BBig.java": big},
			"(module)  INFO  modularity  mode=none concerns=- unlisted=- largest=M-rt/src/BBig.java (6 hand-written lines)\n"},
		{"strict with an unlisted file", map[string]string{"MODULE-MAP.md": strict, "M-rt/src/A.java": "class A {}\n",
			"M-rt/srcTest/ATest.java": "class ATest {}\n", "M-rt/src/B.java": "class B {\n  int b;\n}\n", "M-ux/src/rc/index.html": "x\n"},
			"(module)  INFO  modularity  mode=strict concerns=2 unlisted=1 largest=M-rt/src/B.java (3 hand-written lines)\n"},
		{"no java", map[string]string{"M-ux/src/rc/index.html": "x\n"},
			"(module)  INFO  modularity  mode=none concerns=- unlisted=- largest=-\n"},
		{"strict header, no valid table: the mode stays strict", map[string]string{"MODULE-MAP.md": "modularity: strict\n\nno table\n",
			"M-rt/src/A.java": "class A {}\n"},
			"(module)  INFO  modularity  mode=strict concerns=- unlisted=- largest=M-rt/src/A.java (1 hand-written lines)\n"},
		{"legacy", map[string]string{"MODULE-MAP.md": "modularity: legacy\nlegacy-reason: deployed before the map\n" + strict[len("modularity: strict\n"):],
			"M-rt/src/A.java": "class A {}\n", "M-rt/srcTest/ATest.java": "class ATest {}\n", "M-ux/src/rc/index.html": "x\n"},
			"(module)  INFO  modularity  mode=legacy concerns=2 unlisted=0 largest=M-rt/src/A.java (1 hand-written lines)\n"},
		{"a tie: the first in path order", map[string]string{"M-rt/src/B.java": "class B {\n  int b;\n}\n", "M-rt/src/A.java": "class A {\n  int a;\n}\n"},
			"(module)  INFO  modularity  mode=none concerns=- unlisted=- largest=M-rt/src/A.java (3 hand-written lines)\n"},
		{"a map but no part", map[string]string{"MODULE-MAP.md": strict, "notes/x.txt": "x\n"},
			"(module)  INFO  modularity  mode=strict concerns=? unlisted=? largest=?\n"},
	}
	for _, tt := range tests {
		root := filepath.Join(t.TempDir(), "M")
		for rel, body := range tt.files {
			p := filepath.Join(root, rel)
			must(t, os.MkdirAll(filepath.Dir(p), 0o755))
			must(t, os.WriteFile(p, []byte(body), 0o644))
		}
		var b bytes.Buffer
		r := &report{out: &b, n: map[string]int{}, o: options{root: root}}
		r.modularity(artifact{"(module)", root})
		if got := strings.ReplaceAll(b.String(), root+"/", ""); got != tt.want || len(r.n) != 0 {
			t.Errorf("%s: %q (counted %v), want %q", tt.name, got, r.n, tt.want)
		}
	}
}

// TestModularityRowSymlinkedPart: a symlinked part is in lint-module-map's file set, so it is in the largest
// class too (the two read the same files, modulemap.PartJava).
func TestModularityRowSymlinkedPart(t *testing.T) {
	t.Parallel()
	root, ext := filepath.Join(t.TempDir(), "M"), t.TempDir()
	must(t, os.MkdirAll(filepath.Join(root, "M-rt", "src"), 0o755))
	must(t, os.WriteFile(filepath.Join(root, "M-rt", "src", "A.java"), []byte("class A {}\n"), 0o644))
	must(t, os.MkdirAll(filepath.Join(ext, "src"), 0o755))
	must(t, os.WriteFile(filepath.Join(ext, "src", "W.java"), []byte("class W {\n  int w;\n}\n"), 0o644))
	must(t, os.Symlink(ext, filepath.Join(root, "M-wb")))
	var b bytes.Buffer
	r := &report{out: &b, n: map[string]int{}, o: options{root: root}}
	r.modularity(artifact{"(module)", root})
	want := "(module)  INFO  modularity  mode=none concerns=- unlisted=- largest=M-wb/src/W.java (3 hand-written lines)\n"
	if got := strings.ReplaceAll(b.String(), root+"/", ""); got != want {
		t.Errorf("%q, want %q", got, want)
	}
}

// TestModularityRowUnreadablePart: a part directory the walk cannot read makes the largest class `?` (no map: the
// counts stay `-`; lint-module-map's own row reports the fault, exit 3), never a class of a partial walk.
func TestModularityRowUnreadablePart(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	root := filepath.Join(t.TempDir(), "M")
	locked := filepath.Join(root, "M-rt", "src", "locked")
	must(t, os.MkdirAll(locked, 0o755))
	must(t, os.WriteFile(filepath.Join(root, "M-rt", "src", "A.java"), []byte("class A {}\n"), 0o644))
	must(t, os.Chmod(locked, 0))
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	var b bytes.Buffer
	r := &report{out: &b, n: map[string]int{}, o: options{root: root}}
	r.modularity(artifact{"(module)", root})
	if want := "(module)  INFO  modularity  mode=none concerns=- unlisted=- largest=?\n"; b.String() != want {
		t.Errorf("%q, want %q", b.String(), want)
	}
}
