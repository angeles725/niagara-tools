package puretest

import (
	"os"
	"strings"
	"testing"
)

const usageText = "usage: run-pure-test.sh <module-rt-dir> <pkg.TestClass> | run-pure-test.sh --all <module-rt-dir>"

// TestWrongPackageClassNameNamesTheCause (#1115): `test.com…` built from the path resolves to the source (the
// extra test/ root segment), but JUnit cannot load it; the package line of the file is named, exit 3, nothing runs.
func TestWrongPackageClassNameNamesTheCause(t *testing.T) {
	f := newFixture(t)
	src := f.rt + "/srcTest/test/com/example/demo/AdderTest.java"
	f.stops("class named from the path", ExitEnv,
		"class test.com.example.demo.AdderTest not found; the package line of "+src+" says com.example.demo",
		f.rt, "test.com.example.demo.AdderTest")
	f.write("rt/srcTest/test/com/example/demo/NoPkgTest.java", "")
	f.stops("a file without a package line", ExitEnv,
		"class com.example.demo.NoPkgTest not found; the package line of "+f.rt+"/srcTest/test/com/example/demo/NoPkgTest.java says (none)",
		f.rt, "com.example.demo.NoPkgTest")
}

// TestRelativePartDirIsMadeAbsolute (#1115): every path javac and java get names the part dir absolutely, and the
// message for a part dir that does not exist still quotes the argument as given.
func TestRelativePartDirIsMadeAbsolute(t *testing.T) {
	f := newFixture(t)
	t.Chdir(f.dir)
	if code := f.run("rt", testClass); code != ExitOK {
		t.Fatalf("exit %d stderr %q", code, f.err)
	}
	if got := f.call("javac"); !strings.Contains(got, "<-sourcepath> <"+f.rt+"/src:"+f.rt+"/srcTest/test> ") ||
		!strings.HasSuffix(got, "<"+f.rt+"/srcTest/test/com/example/demo/AdderTest.java>\n") {
		t.Errorf("javac call %s", got)
	}
	if got := f.call("java"); !strings.HasPrefix(got, f.rt+" <-cp> <") {
		t.Errorf("java call %s", got)
	}
	f = newFixture(t)
	t.Chdir(f.dir)
	f.stops("relative, missing", ExitEnv, "module-rt-dir not found: nope", "nope", testClass)
}

// fakeJUnit installs a fake java that logs its call like f.tool and prints a JUnitCore summary line.
func (f *fixture) fakeJUnit(summary string, code int) {
	f.write("bin/java", `#!/bin/sh
{ printf '%s' "$(/bin/pwd -P)"; for a; do printf ' <%s>' "$a"; done; echo; } >> '`+f.dir+`/java.log'
echo '`+summary+`'
exit `+string(rune('0'+code))+"\n")
}

// TestAllRunsEveryTestInOneJVM (#1115): one javac for every *Test.java, one JUnitCore call naming each class from
// its package line, one summary line, JUnit's exit code.
func TestAllRunsEveryTestInOneJVM(t *testing.T) {
	f := newFixture(t)
	f.write("rt/srcTest/com/example/other/BetaTest.java", "// c\npackage com.example.other;\n")
	f.write("rt/srcTest/test/Plain2Test.java", "")
	f.write("rt/srcTest/test/Helper.java", "package com.example.demo;\n")
	f.fakeJUnit("OK (5 tests)", 0)
	if code := f.run("--all", f.rt); code != ExitOK || f.out != "javac stdout\nOK (5 tests)\n" {
		t.Fatalf("exit %d stdout %q stderr %q", code, f.out, f.err)
	}
	if !strings.HasSuffix(f.err, "run-pure-test: 3 classes, 5 tests, 0 failures\n") {
		t.Errorf("stderr %q", f.err)
	}
	if n := strings.Count(f.call("javac"), "\n"); n != 1 { // NIAGARA_HOME is unset: no adapter compile
		t.Errorf("javac ran %d times: %s", n, f.call("javac"))
	}
	wantJavac := "<-sourcepath> <" + f.rt + "/src:" + f.rt + "/srcTest/test:" + f.rt + "/srcTest> <-d> <" + f.tempDir() + "> <" +
		f.rt + "/srcTest/com/example/other/BetaTest.java> <" + f.rt + "/srcTest/test/Plain2Test.java> <" +
		f.rt + "/srcTest/test/com/example/demo/AdderTest.java>\n"
	if got := f.call("javac"); !strings.Contains(got, wantJavac) {
		t.Errorf("javac call %s\nwant it to contain %s", got, wantJavac)
	}
	if got := f.call("java"); strings.Count(got, "\n") != 1 || !strings.HasSuffix(got, "> <org.junit.runner.JUnitCore> <com.example.other.BetaTest> <Plain2Test> <com.example.demo.AdderTest>\n") {
		t.Errorf("java call %s", got)
	}
}

// TestAllSummaryAndExitFollowJUnit (#1115): a failed run keeps JUnit's exit code and counts its failures.
func TestAllSummaryAndExitFollowJUnit(t *testing.T) {
	f := newFixture(t)
	f.fakeJUnit("Tests run: 7,  Failures: 2", 1)
	if code := f.run("--all", f.rt); code != ExitFailed || !strings.HasSuffix(f.err, "run-pure-test: 1 classes, 7 tests, 2 failures\n") {
		t.Errorf("exit %d stderr %q", code, f.err)
	}
	f = newFixture(t)
	f.fakeJUnit("OK (1 test)", 0)
	if code := f.run("--all", f.rt); code != ExitOK || !strings.HasSuffix(f.err, "run-pure-test: 1 classes, 1 tests, 0 failures\n") {
		t.Errorf("exit %d stderr %q", code, f.err)
	}
}

// TestAllWithoutTests (#1115): no *Test.java under srcTest is exit 3 before javac; usage needs the part dir.
func TestAllWithoutTests(t *testing.T) {
	f := newFixture(t)
	os.RemoveAll(f.rt + "/srcTest")
	f.write("rt/srcTest/test/Helper.java", "")
	f.stops("no tests", ExitEnv, "no *Test.java under "+f.rt+"/srcTest", "--all", f.rt)
	f.stops("no part dir", ExitUsage, usageText, "--all")
	f.stops("extra argument", ExitUsage, usageText, "--all", f.rt, testClass)
	f.stops("missing part dir", ExitEnv, "module-rt-dir not found: "+f.dir+"/nope", "--all", f.dir+"/nope")
}
