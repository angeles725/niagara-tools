package puretest

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// adaptersNotice is the line a run without NIAGARA_HOME prints (the fixture has pure sources in src/).
const adaptersNotice = prefix + msgNoAdapters + "\n"

// runHome is run with NIAGARA_HOME set to home ("" leaves it unset).
func (f *fixture) runHome(home string, args ...string) int {
	var out, errb bytes.Buffer
	code := Main(args, cli.Stdio{Out: &out, Err: &errb, Getenv: func(k string) string {
		switch k {
		case "HOME":
			return f.dir + "/home"
		case homeEnv:
			return home
		}
		return ""
	}})
	f.out, f.err = out.String(), errb.String()
	return code
}

// javacCalls are the logged javac calls, one per line.
func (f *fixture) javacCalls() []string {
	return strings.Split(strings.TrimSpace(f.call("javac")), "\n")
}

// TestAdaptersNotCompiledWithoutHome: with NIAGARA_HOME unset the adapters are not compiled, one NOTICE says
// so, and the exit code and the single javac call (the test's) are those of a run without the feature.
func TestAdaptersNotCompiledWithoutHome(t *testing.T) {
	f := newFixture(t)
	if code := f.runHome("", f.rt, testClass); code != ExitOK {
		t.Fatalf("exit %d stderr %q", code, f.err)
	}
	const notice = "run-pure-test: NOTICE the Baja adapters under src/ were not compiled (NIAGARA_HOME is not set)"
	if !strings.Contains(f.err, notice) || !strings.Contains(f.err, "export NIAGARA_HOME=<mirror>") {
		t.Errorf("no NOTICE: %q", f.err)
	}
	if n := len(f.javacCalls()); n != 1 {
		t.Errorf("javac ran %d times, want 1: %s", n, f.call("javac"))
	}
}

// TestAdaptersCompiled: with NIAGARA_HOME the module's src/ sources are compiled with -proc:none into a
// directory of their own, the jars of <home>/modules on the class path, and the tests then run.
func TestAdaptersCompiled(t *testing.T) {
	f := newFixture(t)
	f.write("nh/modules/baja.jar", "")
	f.write("rt/src/com/example/demo/BAdapter.java", "")
	if code := f.runHome(f.dir+"/nh", f.rt, testClass); code != ExitOK {
		t.Fatalf("exit %d stderr %q", code, f.err)
	}
	calls := f.javacCalls()
	if len(calls) != 2 {
		t.Fatalf("javac ran %d times, want 2: %v", len(calls), calls)
	}
	adapters := calls[1]
	for _, want := range []string{"<-proc:none>", "<-cp> <" + f.dir + "/nh/modules/*>", f.rt + "/src/com/example/demo/Adder.java>", f.rt + "/src/com/example/demo/BAdapter.java>"} {
		if !strings.Contains(adapters, want) {
			t.Errorf("adapter compile call lacks %s: %s", want, adapters)
		}
	}
	if strings.Contains(adapters, "srcTest") {
		t.Errorf("the adapter compile took test sources: %s", adapters)
	}
	if strings.Contains(f.err, "NOTICE") || f.call("java") == "" {
		t.Errorf("notice or no test run: stderr %q java %q", f.err, f.call("java"))
	}
}

// TestAdaptersCompileError: an adapter that does not compile is exit 3 naming the file; the tests do not run.
func TestAdaptersCompileError(t *testing.T) {
	f := newFixture(t)
	f.write("nh/modules/baja.jar", "")
	f.write("rt/src/com/example/demo/BAdapter.java", "")
	// the fake javac fails the second call (the adapters) and prints a javac-style diagnostic
	f.write("bin/javac", `#!/bin/sh
if [ -e '`+f.dir+`/javac.second' ]; then
  echo "`+f.rt+`/src/com/example/demo/BAdapter.java:3: error: cannot find symbol" >&2; exit 1
fi
: > '`+f.dir+`/javac.second'
{ for a; do printf ' <%s>' "$a"; done; echo; } >> '`+f.dir+`/javac.log'
exit 0
`)
	if code := f.runHome(f.dir+"/nh", f.rt, testClass); code != ExitEnv {
		t.Errorf("exit %d, want %d", code, ExitEnv)
	}
	if want := "run-pure-test: adapter sources do not compile: " + f.rt + "/src/com/example/demo/BAdapter.java"; !strings.Contains(f.err, want) {
		t.Errorf("stderr lacks %q:\n%s", want, f.err)
	}
	if f.call("java") != "" {
		t.Errorf("the tests ran after an adapter compile error")
	}
}

// TestAdaptersNeedJars: a NIAGARA_HOME without jars in modules/ is exit 3, before anything is compiled.
func TestAdaptersNeedJars(t *testing.T) {
	f := newFixture(t)
	os.MkdirAll(f.dir+"/nh/modules", 0o755)
	if code := f.runHome(f.dir+"/nh", f.rt, testClass); code != ExitEnv || f.call("javac") != "" {
		t.Errorf("exit %d javac %q", code, f.call("javac"))
	}
	if want := "run-pure-test: NIAGARA_HOME " + f.dir + "/nh has no modules/*.jar"; !strings.Contains(f.err, want) {
		t.Errorf("stderr lacks %q: %s", want, f.err)
	}
}
