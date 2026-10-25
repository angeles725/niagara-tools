package niagaratest

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// modulesCopied lists the jar names every modules robocopy of the run was handed.
func (f *fake) modulesCopied() []string {
	var jars []string
	for _, c := range f.of("robocopy") {
		if !strings.HasSuffix(c.Args[2], `\modules`) {
			continue
		}
		for _, a := range c.Args[4:] {
			if !strings.HasPrefix(a, "/") {
				jars = append(jars, a)
			}
		}
	}
	return jars
}

func TestRunExcludeJar(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	for _, n := range []string{"moduleA-ux.jar", "moduleA-wb.jar", "moduleB-rt.jar"} {
		write(t, filepath.Join(fx.install, "modules", n), "old")
	}
	code, out := fx.run(t, append([]string{"run"}, fx.args("--exclude-jar", "ModuleA-u*.jar", "--exclude-jar", "moduleA-w?.jar")...)...)
	if code != exitPass {
		t.Fatalf("exit %d\n%s", code, out)
	}
	got := fx.f.modulesCopied()
	for _, n := range []string{"moduleA-ux.jar", "moduleA-wb.jar"} {
		if slices.Contains(got, n) {
			t.Errorf("modules robocopy copied excluded %s: %q", n, got)
		}
	}
	for _, n := range []string{"moduleA-rt.jar", "moduleB-rt.jar"} {
		if !slices.Contains(got, n) {
			t.Errorf("modules robocopy lacks %s: %q", n, got)
		}
	}
	for _, w := range []string{"PASS  run-niagara-test  copy: excluded 1 jar matching ModuleA-u*.jar: moduleA-ux.jar\n",
		"PASS  run-niagara-test  copy: excluded 1 jar matching moduleA-w?.jar: moduleA-wb.jar\n"} {
		if !strings.Contains(out, w) {
			t.Errorf("no %q in\n%s", w, out)
		}
	}
	for _, n := range []string{"moduleA-ux.jar", "moduleA-wb.jar"} {
		if _, err := os.Stat(filepath.Join(fx.install, "modules", n)); err != nil {
			t.Errorf("the install lost %s: %v", n, err)
		}
	}
}

func TestRunExcludeJarListsTenNames(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	var names []string
	for i := range 12 {
		n := fmt.Sprintf("moduleC%02d-rt.jar", i)
		names = append(names, n)
		write(t, filepath.Join(fx.install, "modules", n), "old")
	}
	code, out := fx.run(t, append([]string{"run"}, fx.args("--exclude-jar", "moduleC*.jar")...)...)
	if code != exitPass {
		t.Fatalf("exit %d\n%s", code, out)
	}
	want := "PASS  run-niagara-test  copy: excluded 12 jars matching moduleC*.jar: " + strings.Join(names[:10], " ") + " and 2 more\n"
	if !strings.Contains(out, want) {
		t.Errorf("no %q in\n%s", want, out)
	}
	if got := fx.f.modulesCopied(); slices.ContainsFunc(got, func(n string) bool { return strings.HasPrefix(n, "moduleC") }) {
		t.Errorf("modules robocopy copied an excluded jar: %q", got)
	}
}

// A glob that matches nothing is probably a typo: FAIL, exit 2, before anything is copied or built.
func TestRunExcludeJarMatchingNothingFails(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	code, out := fx.run(t, append([]string{"run"}, fx.args("--exclude-jar", "moduleA-rt.jar", "--exclude-jar", "moduleZ-*.jar")...)...)
	want := `FAIL  run-niagara-test  copy: --exclude-jar moduleZ-*.jar matched no jar in C:\Niagara\N4\modules`
	if code != exitUsage || !strings.Contains(out, want) {
		t.Fatalf("exit %d, want 2 with %q:\n%s", code, want, out)
	}
	if n := len(fx.f.of("robocopy")) + len(fx.f.of("gradle")) + len(fx.f.of("test")); n != 0 {
		t.Errorf("%d copy/build/run calls after an unmatched --exclude-jar", n)
	}
	if l := fx.leftovers(t); len(l) != 0 {
		t.Errorf("left in %%TEMP%%: %q", l)
	}
}

// --exclude-jar takes a basename glob of * and ? only; anything else is exit 2 before any probe.
func TestRunExcludeJarRefusesNonBasenameGlobs(t *testing.T) {
	t.Parallel()
	for _, g := range []string{"", "mods/a.jar", `mods\a.jar`, "a..jar", "..", "a&b.jar", "a|b.jar", "a;b.jar", "$x.jar",
		"%x%.jar", "a b.jar", "[ab].jar", "a`b.jar", "a(b).jar", "a'b.jar", "a\"b.jar", "a^b.jar", "a,b.jar", "a@b.jar"} {
		t.Run(g, func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			code, out := fx.run(t, append([]string{"run"}, fx.args("--exclude-jar", g)...)...)
			if want := fmt.Sprintf("--exclude-jar %q is not a jar basename glob", g); code != exitUsage || !strings.Contains(out, want) {
				t.Errorf("exit %d, want 2 with %q:\n%s", code, want, out)
			}
			if len(fx.f.calls) != 0 {
				t.Errorf("%d calls after a refused glob", len(fx.f.calls))
			}
		})
	}
}
