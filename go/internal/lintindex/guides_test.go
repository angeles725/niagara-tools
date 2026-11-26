package lintindex

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// kitWithGuides is kit() plus a types/ directory holding the given guides (file name -> text).
func kitWithGuides(t *testing.T, lints, guides map[string]string) (root, tb string) {
	t.Helper()
	root, tb = kit(t, lints)
	dir := filepath.Join(root, "types")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, text := range guides {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root, tb
}

func TestGuideRefs(t *testing.T) {
	t.Parallel()
	guides := []Guide{
		{"types/a.md", "intro\n\nRun `lint-foo.sh` here.\nplain\nplain\nplain\nplain\nplain\nplain\n[CERT-doc] far away\n"},
		{"types/b.md", "x\nlint-foo --strict (see toolbelt/lint-foo.sh)\ny\nniagara-help/devguide-clean/x.txt:6 shows it\nlint-foo-bar.sh is another\nmy-lint-foo.sh too\n"},
	}
	cases := []struct {
		script     string
		guide, doc []string
	}{
		// a.md:3 names it; its [CERT-doc] is 7 lines below (line 10): out of the ±6 window. b.md:2 names it
		// (once, though twice on the line); the niagara-help path on line 4 is within 6.
		{"lint-foo.sh", []string{"types/a.md:3", "types/b.md:2"}, []string{"types/b.md:4"}},
		// A longer name never counts for its prefix, nor a prefixed word for the name.
		{"lint-foo-bar.sh", []string{"types/b.md:5"}, []string{"types/b.md:4"}},
		{"lint-none.sh", nil, nil},
	}
	for _, tt := range cases {
		g, d := GuideRefs(guides, tt.script)
		if !reflect.DeepEqual(g, tt.guide) || !reflect.DeepEqual(d, tt.doc) {
			t.Errorf("GuideRefs(%s) = %v, %v; want %v, %v", tt.script, g, d, tt.guide, tt.doc)
		}
	}
	// A [CERT-doc] tag exactly 6 lines away counts (±6), 7 does not.
	edge := []Guide{{"types/c.md", "lint-x.sh\n1\n2\n3\n4\n5\n[CERT-doc]\n8\n"}}
	if _, d := GuideRefs(edge, "lint-x.sh"); !reflect.DeepEqual(d, []string{"types/c.md:7"}) {
		t.Errorf("6 lines away: doc = %v", d)
	}
	edge[0].Text = "lint-x.sh\n1\n2\n3\n4\n5\n6\n[CERT-doc]\n"
	if _, d := GuideRefs(edge, "lint-x.sh"); d != nil {
		t.Errorf("7 lines away: doc = %v, want none", d)
	}
}

func TestKitScope(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, text string
		want       bool
	}{
		{"declared", "#!/bin/sh\n# lint-a.sh — d\n# Scope: kit — checks the kit's own files.\n# Usage: x\nexit 0\n", true},
		{"bare", "#!/bin/sh\n# Scope: kit\nexit 0\n", true},
		{"another scope", "#!/bin/sh\n# Scope: *.java, *.bog — where it looks.\nexit 0\n", false},
		{"word boundary", "#!/bin/sh\n# Scope: kitchen sinks\nexit 0\n", false},
		{"below the header", "#!/bin/sh\nset -u\n# Scope: kit\n", false},
		{"none", "#!/bin/sh\n# lint-a.sh — d\nexit 0\n", false},
	}
	for _, tt := range cases {
		if got := KitScope(tt.text); got != tt.want {
			t.Errorf("%s: KitScope = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestScanGuideColumns(t *testing.T) {
	t.Parallel()
	root, tb := kitWithGuides(t, map[string]string{
		"lint-alpha.sh": lint("lint-alpha.sh", "a", "u", "e"),
		"lint-beta.sh":  lint("lint-beta.sh", "b", "u", "e"),
	}, map[string]string{"one.md": "x\n`lint-alpha.sh` flags it. [CERT-doc] `niagara-help/y.txt:1`\n"})
	rows, _, _, err := Scan(tb)
	if err != nil || len(rows) != 2 {
		t.Fatalf("Scan: %v %v", rows, err)
	}
	if rows[0].Guide != "types/one.md:2" || rows[0].Doc != "types/one.md:2" || rows[1].Guide != "" || rows[1].Doc != "" {
		t.Errorf("rows = %+v", rows)
	}
	index, code := Generate(root, &strings.Builder{})
	want := "| `toolbelt/lint-alpha.sh` | a | `u` | e | no | — | types/one.md:2 | types/one.md:2 |\n" +
		"| `toolbelt/lint-beta.sh` | b | `u` | e | no | — | — | — |\n"
	if code != ExitOK || !strings.Contains(index, "| Evidence | Guide | Doc |\n|---|---|---|---|---|---|---|---|\n"+want) {
		t.Errorf("index (exit %d):\n%s", code, index)
	}
}

func TestMainCheckGuides(t *testing.T) {
	t.Parallel()
	kitScope := strings.Replace(lint("lint-kitty.sh", "k", "u", "e"), "#\n", "#\n# Scope: kit — checks the kit itself.\n", 1)
	files := map[string]string{
		"lint-named.sh":   lint("lint-named.sh", "n", "u", "e"),
		"lint-orphan.sh":  lint("lint-orphan.sh", "o", "u", "e"),
		"lint-orphan2.sh": lint("lint-orphan2.sh", "o2", "u", "e"),
		"lint-kitty.sh":   kitScope,
	}
	root, tb := kitWithGuides(t, files, map[string]string{"g.md": "see lint-named.sh\n"})
	code, out, errs := run("", "--check-guides", root)
	want := "gen-lint-index: lint-orphan.sh: named by no types/*.md guide (name it in the guide section of its rule, or declare \"# Scope: kit\" in its header)\n" +
		"gen-lint-index: lint-orphan2.sh: named by no types/*.md guide (name it in the guide section of its rule, or declare \"# Scope: kit\" in its header)\n"
	if code != 1 || out != "" || errs != want {
		t.Errorf("unnamed lints: got (%d, %q, %q)", code, out, errs)
	}
	if _, err := os.Stat(filepath.Join(tb, "INDEX.md")); err == nil {
		t.Error("--check-guides wrote INDEX.md")
	}

	// Naming the orphans in a guide passes; the kit-scope lint never needed it.
	if err := os.WriteFile(filepath.Join(root, "types", "g.md"), []byte("lint-named.sh lint-orphan.sh\nlint-orphan2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errs = run("", "--check-guides", root)
	if code != 0 || errs != "" || out != "gen-lint-index: every lint under "+root+"/toolbelt is named by a types/*.md guide or declares Scope: kit (3 named, 1 kit)\n" {
		t.Errorf("all named: got (%d, %q, %q)", code, out, errs)
	}

	// With --check too, and still read-only; no types/ is an environment error.
	if code, _, _ = run("", "--check", "--check-guides", root); code != 0 {
		t.Errorf("--check --check-guides: exit %d", code)
	}
	bare, _ := kit(t, files)
	code, out, errs = run("", "--check-guides", bare)
	if code != ExitEnv || out != "" || errs != "gen-lint-index: no types/ under "+bare+"\n" {
		t.Errorf("no types/: got (%d, %q, %q)", code, out, errs)
	}
}

// TestRealKitEveryLintHasAGuide is the gate itself on the real kit: a new module lint must be named in a
// types/*.md guide, or declare `# Scope: kit` when it checks the kit.
func TestRealKitEveryLintHasAGuide(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "..", "build-n4-module-kit")
	if code, out, errs := run(root, "--check-guides"); code != ExitOK {
		t.Fatalf("exit %d: %s%s", code, out, errs)
	}
}
