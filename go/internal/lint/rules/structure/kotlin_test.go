package structure

import (
	"path/filepath"
	"strings"
	"testing"
)

const (
	ktsSign  = `plugins { id("com.tridium.niagara-signing") }` + "\n"
	ktsBlock = "signingServices { signingProfileFactory { allowDefaultProfile.set(false) } }\n" +
		"niagaraSigning {\n  aliases.set(listOf(\"CodeSigning\"))\n  signingProfileFile.set(file(\"p.xml\"))\n}\n"
)

// kotlinRoot runs the lint over a gradle root whose settings.gradle.kts and build.gradle.kts hold the texts.
func kotlinRoot(t *testing.T, settings, build string) (int, string, string, string) {
	t.Helper()
	dir := t.TempDir()
	write(t, dir, map[string]string{".git/HEAD": "x\n", "settings.gradle.kts": settings, "build.gradle.kts": build,
		"M/A-doc/build.gradle.kts": "\n", "M/A-doc/module-include.xml": "<types/>\n"})
	code, out, errs := run(filepath.Join(dir, "M"))
	return code, out, errs, dir
}

// TestSigningKotlin (truth fix 2026-10-24): L15 reads the root build script as Kotlin (go/internal/ktssrc), so a
// block hidden in a nested /* */ comment or in a string inside a ${...} template is not code, and a raw
// """C:\""" string ends at its quotes. The Java scanner it used before ended the comment at the first */, read
// the template's inner string as code and took \" for an escape, keeping the rest of the file in the literal.
func TestSigningKotlin(t *testing.T) {
	for build, want := range map[string]bool{
		ktsSign + ktsBlock: false,
		ktsSign + "/* outer /* inner */\n" + ktsBlock + "*/\n":                                                          true,
		ktsSign + `val note = "${"allowDefaultProfile.set(false) niagaraSigning { aliases signingProfileFile"}"` + "\n": true,
		ktsSign + "val p = \"\"\"C:\\\"\"\"\n" + ktsBlock:                                                               false,
		"plugins {\n  id(\n    \"com.tridium.niagara-signing\"\n  )\n}\n":                                               true,
	} {
		_, out, _, _ := kotlinRoot(t, "// root\n", build)
		if got := strings.Contains(out, "L15: applies"); got != want {
			t.Errorf("%q: L15 row %v, want %v (out %q)", build, got, want, out)
		}
	}
	// A script that is not Kotlin leaves the block unchecked, visibly, like an unreadable one (exit 0).
	code, out, errs, dir := kotlinRoot(t, "// root\n", ktsSign+ktsBlock+"val x = \"open\n")
	want := "WARN  lint-structure  " + dir + "/build.gradle.kts  L15: could not scan the root for the signing block (line 7: unterminated string) — signing block unchecked\n"
	if code != 0 || out != want || errs != "" {
		t.Errorf("not Kotlin: exit %d err %q out %q, want 0 and %q", code, errs, out, want)
	}
}
