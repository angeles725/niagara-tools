package niagaraversion

import (
	"bytes"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

func run(args ...string) (code int, out, errs string) {
	var o, e bytes.Buffer
	code = Main(args, cli.Stdio{Out: &o, Err: &e})
	return code, o.String(), e.String()
}

func TestMainPassesWhenTheInstallationIsTheVersionTheModuleWasCompiledFor(t *testing.T) {
	t.Parallel()
	jar := moduleJar(t, `<dependency name="baja" vendor="Tridium" vendorVersion="4.14"/>`)
	root := installation(t, "4.14.0.162")
	for _, target := range []string{root, root + "/modules"} {
		code, out, errs := run(jar, target)
		want := "PASS  target-version  " + jar + " was compiled against Niagara 4.14; " + root + " is Niagara 4.14.0.162\n"
		if code != 0 || out != want || errs != "" {
			t.Errorf("%s: exit %d, stdout %q, stderr %q; want exit 0 and %q", target, code, out, errs, want)
		}
	}
}

func TestMainFailsOnAnInstallationOfAnotherNiagaraVersionAndSaysBoth(t *testing.T) {
	t.Parallel()
	jar := moduleJar(t, `<dependency name="baja" vendor="Tridium" vendorVersion="4.14"/>`)
	root := installation(t, "4.15.6.20")
	code, out, errs := run(jar, root)
	want := "FAIL  target-version  " + jar + " was compiled against Niagara 4.14 but " + root + " is Niagara 4.15.6.20: " +
		"build it against an installation of 4.15 (mirror-niagara-home.sh), or install it into one of 4.14\n"
	if code != 1 || out != want || errs != "" {
		t.Errorf("exit %d, stdout %q, stderr %q; want exit 1 and %q", code, out, errs, want)
	}
}

func TestMainRefusesWhatItCannotCompareWithExit2(t *testing.T) {
	t.Parallel()
	jar := moduleJar(t, `<dependency name="baja" vendor="Tridium" vendorVersion="4.14"/>`)
	root := installation(t, "4.14.0.162")
	for name, args := range map[string][]string{
		"no argument":           {},
		"one argument":          {jar},
		"three arguments":       {jar, root, root},
		"an option":             {"--help", root},
		"a jar that is missing": {t.TempDir() + "/none.jar", root},
		"a jar with no version": {moduleJar(t, ""), root},
		"no installation there": {jar, t.TempDir()},
	} {
		code, out, errs := run(args...)
		if code != 2 || out != "" || errs == "" {
			t.Errorf("%s: exit %d, stdout %q, stderr %q; want exit 2, no stdout and a message", name, code, out, errs)
		}
	}
	if _, _, errs := run(); !strings.Contains(errs, "usage: check-target-version") {
		t.Errorf("no argument: stderr %q lacks the usage line", errs)
	}
}
