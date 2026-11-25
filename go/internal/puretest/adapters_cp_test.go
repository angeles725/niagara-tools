package puretest

import (
	"strings"
	"testing"
)

// A wb or ux part's adapters use the rt part's types: the class path holds NIAGARA_HOME/modules/* and the
// build/libs jars of every sibling part (the parts beside this one), in name order, never this part's own
// jar (issue #1075).
func TestAdaptersSeeTheSiblingPartsJars(t *testing.T) {
	f := newFixture(t)
	f.write("nh/modules/baja.jar", "")
	f.write("rt/src/com/example/demo/BAdapter.java", "")
	f.write("rt/build/libs/demo-self.jar", "")  // this part's own jar: stale output of the sources compiled now
	f.write("zz-wb/build/libs/demo-wb.jar", "") // a sibling part
	f.write("aa-ux/build/libs/demo-ux.jar", "") // another, listed first by name
	f.write("notapart/readme.txt", "")          // a directory without build/libs adds nothing
	if code := f.runHome(f.dir+"/nh", f.rt, testClass); code != ExitOK {
		t.Fatalf("exit %d stderr %q", code, f.err)
	}
	calls := f.javacCalls()
	if len(calls) != 2 {
		t.Fatalf("javac ran %d times, want 2: %v", len(calls), calls)
	}
	want := "<-cp> <" + f.dir + "/nh/modules/*:" + f.dir + "/aa-ux/build/libs/demo-ux.jar:" +
		f.dir + "/zz-wb/build/libs/demo-wb.jar>"
	if !strings.Contains(calls[1], want) {
		t.Errorf("adapter compile call lacks %s:\n%s", want, calls[1])
	}
}

// A compile error names what the class path held, so a sibling part that was never built is visible.
func TestAnAdapterCompileErrorNamesTheSiblingJars(t *testing.T) {
	f := newFixture(t)
	f.write("nh/modules/baja.jar", "")
	f.write("rt/src/com/example/demo/BAdapter.java", "")
	f.write("ux/build/libs/demo-ux.jar", "")
	f.write("bin/javac", `#!/bin/sh
if [ -e '`+f.dir+`/javac.second' ]; then
  echo "`+f.rt+`/src/com/example/demo/BAdapter.java:3: error: cannot find symbol" >&2; exit 1
fi
: > '`+f.dir+`/javac.second'
exit 0
`)
	if code := f.runHome(f.dir+"/nh", f.rt, testClass); code != ExitEnv {
		t.Errorf("exit %d, want %d", code, ExitEnv)
	}
	want := "class path NIAGARA_HOME/modules and the sibling part jars " + f.dir + "/ux/build/libs/demo-ux.jar"
	if !strings.Contains(f.err, want) {
		t.Errorf("stderr lacks %q:\n%s", want, f.err)
	}
}
