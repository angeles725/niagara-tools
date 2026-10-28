package structure

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-structure.bats run and every edge tree, recorded from the bash lint.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 50) }

func run(args ...string) (int, string, string) {
	var o, e bytes.Buffer
	code := lint.Command(Spec())(append([]string{"lint-structure"}, args...), cli.Stdio{Out: &o, Err: &e})
	return code, o.String(), e.String()
}

func write(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for p, text := range files {
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestSymlinkedRoot (LS-symroot, issue #329): a symlinked module root is walked like find -H and prints
// the directory's rows spelled from the link.
func TestSymlinkedRoot(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, map[string]string{".git/HEAD": "x\n", "M/gradle.properties": "niagara_home=C:\\N\n", "M/Foo-rt/module-include.xml": "<types/>\n"})
	link := filepath.Join(dir, "L")
	if err := os.Symlink(filepath.Join(dir, "M"), link); err != nil {
		t.Fatal(err)
	}
	code, out, _ := run(filepath.Join(dir, "M"))
	lcode, lout, _ := run(link)
	if want := strings.ReplaceAll(out, filepath.Join(dir, "M"), link); lcode != code || lout != want || !strings.Contains(lout, "L10") {
		t.Errorf("link: exit %d out\n%s\nwant %d and\n%s", lcode, lout, code, want)
	}
}

// TestUnreadable: an unreadable root build.gradle.kts is a visible L15 WARN (LS15-unreadable, exit 0);
// any other file or directory the lint cannot read fails closed (exit 3), where the bash grep and find
// went on as if it held no match.
func TestUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := t.TempDir()
	write(t, dir, map[string]string{".git/HEAD": "x\n", "settings.gradle.kts": "// root\n",
		"build.gradle.kts": `plugins { id("com.tridium.niagara-signing") }` + "\n", "M/A-rt/module-include.xml": "<types/>\n",
		"M/A-rt/module.palette": "<p n=\"A\"/>\n", "M/A-rt/src/A.java": "class A {}\n"})
	build, java, src := filepath.Join(dir, "build.gradle.kts"), filepath.Join(dir, "M/A-rt/src/A.java"), filepath.Join(dir, "M/A-rt/src")
	for _, p := range []string{build, java} {
		if err := os.Chmod(p, 0); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { _ = os.Chmod(src, 0o755) })
	l15 := "WARN  lint-structure  " + build + "  L15: could not scan the root for the signing block (awk exit 2: awk: fatal: cannot open file `" +
		build + "' for reading: Permission denied) — signing block unchecked\n"
	code, out, errs := run(filepath.Join(dir, "M"))
	if want := "lint-structure: cannot scan " + java + ": Permission denied\n"; code != 3 || out != l15 || errs != want {
		t.Errorf("unreadable file: exit %d out %q err %q, want 3, %q and %q", code, out, errs, l15, want)
	}
	if err := os.Chmod(src, 0); err != nil {
		t.Fatal(err)
	}
	code, out, errs = run(filepath.Join(dir, "M"))
	// the first walk to meet it is L10's over the whole module root
	if want := "lint-structure: cannot list every file under " + filepath.Join(dir, "M") + ": find: '" + src + "': Permission denied\n"; code != 3 || out != l15 || errs != want {
		t.Errorf("unlistable dir: exit %d out %q err %q, want 3, %q and %q", code, out, errs, l15, want)
	}
}

// TestHostPathShapes (truth audit 2026-10-22 T7): L10 reads gradle.properties as java.util.Properties
// does: a key, then "=", ":" or whitespace with blanks around it, possibly indented. An absolute value is a
// drive path (an escaped "C\:" too), a POSIX path or a UNC share; a comment or a relative value is not.
func TestHostPathShapes(t *testing.T) {
	for text, want := range map[string]bool{
		`niagara_home=C:\N`: true, `niagara_home = C:\N`: true, `niagara_home:C:\N`: true, `  niagara_home=C:\N`: true,
		`niagara_home=/opt/Niagara-4.14`: true, `niagara_home=\\server\share`: true, `nodeHome D:/node`: true,
		`niagara_user_home=C\:\\N`: true, "user_home\t=\t/home/u": true, "niagara_home=C:/N\r": true,
		`#niagara_home=C:\N`: false, `!niagara_home=/opt`: false, `niagara_home=build/n`: false, `niagara_home=`: false,
		`niagara_homeX=/opt`: false, `other=/opt/x`: false, `niagara_home=~/n`: false,
	} {
		dir := t.TempDir()
		write(t, dir, map[string]string{".git/HEAD": "x\n", "M/gradle.properties": "x=1\n" + text + "\n", "M/A-rt/module-include.xml": "<types/>\n"})
		_, out, _ := run(filepath.Join(dir, "M"))
		if got := strings.Contains(out, "L10:"); got != want {
			t.Errorf("%q: L10 row %v, want %v (out %q)", text, got, want, out)
		}
	}
}

// TestRowLines (truth audit 2026-10-22 T7): an L10 (FAIL or R10-D8 WARN) or L14 row names the offending line
// as path:line, one row per offending line, like every other lint's line rows; file and directory rows keep the bare path.
// The L10 text claims no git tracking: the lint is VCS-free and flags the file on disk either way.
func TestRowLines(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, map[string]string{".git/HEAD": "x\n", "M/gradle.properties": "x=1\nuser_home=C:\\N\nnodeHome=/n\n",
		"M/settings.gradle.kts": "// root\nval gradlePluginVersion = \"7.6\"\n", "M/A-rt/module-include.xml": "<types/>\n"})
	m := filepath.Join(dir, "M")
	want := "FAIL  lint-structure  " + m + "/gradle.properties:2  L10: absolute host path in gradle.properties (tracked or not: pass host paths with -P, as build.sh does)\n" +
		"WARN  lint-structure  " + m + "/gradle.properties:3  " + l10Warn("nodeHome") + "\n" +
		"FAIL  lint-structure  A-rt  L5: module.palette missing for -rt profile\n" +
		"FAIL  lint-structure  gradle.properties  " + l12Msg + "\n" +
		"FAIL  lint-structure  " + m + "/settings.gradle.kts:2  L14: hardcoded gradlePluginVersion literal (use providers.gradleProperty(\"niagaraPluginVersion\").getOrElse(...) so -PniagaraPluginVersion applies)\n"
	if code, out, errs := run(m); code != 1 || out != want || errs != "" {
		t.Errorf("exit %d err %q out\n%s\nwant\n%s", code, errs, out, want)
	}
}

const (
	l10Fail = "L10: absolute host path in gradle.properties (tracked or not: pass host paths with -P, as build.sh does)"
	l12Msg  = "L12: absent/commented org.gradle.java.installations.auto-detect=false (pin it in the repo; the JDK paths go in the user-level gradle.properties)"
)

func l10Warn(key string) string {
	return "L10: " + key + " is a host build-location key: migrate it to the user-level gradle.properties (BUILD-LOOP.md bring-up step 2)"
}

// l10Rows is the L10 rows of a run over root holding rel/gradle.properties with text (R10-D8).
func l10Rows(t *testing.T, rel, text string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join(dir, rel)
	write(t, dir, map[string]string{".git/HEAD": "x\n", rel + "/gradle.properties": text, rel + "/A-rt/module-include.xml": "<types/>\n"})
	_, out, _ := run(root)
	var rows []string
	for _, ln := range strings.SplitAfter(out, "\n") {
		if strings.Contains(ln, "  L10: ") {
			rows = append(rows, ln)
		}
	}
	return strings.Join(rows, ""), root + "/gradle.properties"
}

// TestHostBuildKeys (R10-D8): L10 owns the four host build-location keys. In a client root each is a WARN
// to migrate to the user-level gradle.properties; user_home and every key inside the kit's fixtures or
// templates stay the FAIL.
func TestHostBuildKeys(t *testing.T) {
	for _, key := range []string{"niagara_home", "niagara_user_home", "nodeHome", "org.gradle.java.installations.paths"} {
		got, p := l10Rows(t, "M", "x=1\n"+key+"=C:\\\\N\n")
		if want := "WARN  lint-structure  " + p + ":2  " + l10Warn(key) + "\n"; got != want {
			t.Errorf("%s in a client root: got %q, want %q", key, got, want)
		}
	}
	if got, p := l10Rows(t, "M", "user_home=/home/u\n"); got != "FAIL  lint-structure  "+p+":1  "+l10Fail+"\n" {
		t.Errorf("user_home: got %q, want the FAIL row", got)
	}
	for _, rel := range []string{"kit/build-n4-module-kit/fixtures/MinimalPan", "kit/build-n4-module-kit/templates/T"} {
		if got, p := l10Rows(t, rel, "niagara_home=/opt/N\norg.gradle.java.installations.paths=/j\n"); got !=
			"FAIL  lint-structure  "+p+":1  "+l10Fail+"\n"+"FAIL  lint-structure  "+p+":2  "+l10Fail+"\n" {
			t.Errorf("%s: got %q, want two FAIL rows", rel, got)
		}
	}
	// WARN-only exits 0: a clean -doc profile, the keys in the module root.
	dir := t.TempDir()
	write(t, dir, map[string]string{".git/HEAD": "x\n", "M/A-doc/build.gradle.kts": "\n", "M/A-doc/module-include.xml": "<types/>\n",
		"M/gradle.properties": "niagara_home=C:\\\\N\norg.gradle.java.installations.auto-detect=false\n"})
	if code, out, _ := run(filepath.Join(dir, "M")); code != 0 || !strings.HasPrefix(out, "WARN") {
		t.Errorf("WARN-only root: exit %d out %q, want 0 and the WARN row", code, out)
	}
}

// TestJdkPin (R10-D8): L12 asks the repo only for the machine-independent auto-detect=false; the JDK
// paths may live in the user-level gradle.properties.
func TestJdkPin(t *testing.T) {
	for text, want := range map[string]bool{
		"org.gradle.java.installations.auto-detect=false\n":                                         false,
		"org.gradle.java.installations.auto-detect = false\n":                                       false,
		"org.gradle.java.installations.paths=/j\norg.gradle.java.installations.auto-detect=false\n": false,
		"org.gradle.java.installations.paths=/j\n":                                                  true,
		"org.gradle.java.installations.auto-detect=true\n":                                          true,
		"#org.gradle.java.installations.auto-detect=false\n":                                        true,
	} {
		dir := t.TempDir()
		write(t, dir, map[string]string{"M/gradle.properties": text, "M/A-doc/build.gradle.kts": "\n"})
		_, out, _ := run(filepath.Join(dir, "M"))
		if got := strings.Contains(out, "FAIL  lint-structure  gradle.properties  "+l12Msg+"\n"); got != want {
			t.Errorf("%q: L12 row %v, want %v (out %q)", text, got, want, out)
		}
	}
}
