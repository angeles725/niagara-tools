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
