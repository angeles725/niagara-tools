package buildmodule

import (
	"strings"
	"testing"
)

// TestGradleFailures: what the log of a failed run shows decides the exit code and the advice; the gate
// never runs.
func TestGradleFailures(t *testing.T) {
	cases := []struct {
		name, gradlew string // what the fake gradlew does
		flags         []string
		code          int
		has, hasNot   string
	}{
		{"a plain failure", "exit 1", nil, ExitGradle, "build.sh: gradle failed\n", "slot-o-matic"},
		{"killed by a signal", "kill -9 $$", nil, ExitGradle, "build.sh: gradle failed\n", ""},
		{":clean locked by a station", `echo "Unable to delete file 'nh/modules/Foo-rt.jar'" >&2; exit 1`, nil, ExitCleanLocked, "mirror-niagara-home.sh", "gradle failed"},
		{"the post-jar copy locked", `echo "java.nio.file.FileAlreadyExistsException: /nh/modules/Foo-rt.jar" >&2; exit 1`, nil, ExitCopyLocked, "artifact of record", "gradle failed"},
		{"the copy lock, Windows spelling", `printf 'kotlin.io.FileAlreadyExistsException: C:\\nh\\modules\\Foo-rt.jar: Tried to overwrite\r\n'; exit 1`, nil, ExitCopyLocked, "artifact of record", ""},
		{"the copy lock outranks the clean lock", `echo "Unable to delete x/modules/a.jar"; echo "FileAlreadyExistsException: x/modules/b.jar"; exit 1`, nil, ExitCopyLocked, "", ":clean could not"},
		{"a lock message split over two lines is no lock", `echo FileAlreadyExistsException; echo x/modules/a.jar; exit 1`, nil, ExitGradle, "", "artifact of record"},
		{"missing slots with the skipped file named", `echo "Could not process /x/src/BHmi.java: Found multiple metadata blocks"; echo "error: have you run slot-o-matic?"; exit 1`, nil, ExitGradle,
			"build.sh: compileJava found slots slot-o-matic never generated.\nFAIL slotomatic-skip /x/src/BHmi.java (Found multiple metadata blocks)\n  A file with empty BEGIN/END BAJA AUTO GENERATED markers", "Rerun"},
		{"missing slots with no skipped file", `echo "error: have you run slot-o-matic?"; exit 1`, nil, ExitGradle, "  Rerun :<part>:slotomatic --info", "empty BEGIN/END"},
		{"a --werror failure names the remedy", "exit 1", []string{"--werror"}, ExitGradle, "build.sh: --werror build: javac warnings are errors", ""},
	}
	for _, c := range cases {
		f := newFixture(t)
		f.does("gradlew", c.gradlew)
		f.wants(c.name, f.build(c.flags...), c.code, c.has)
		if c.hasNot != "" && strings.Contains(f.err, c.hasNot) {
			t.Errorf("%s: stderr holds %q: %s", c.name, c.hasNot, f.err)
		}
		if f.call(gateScript) != "" {
			t.Errorf("%s: the gate ran", c.name)
		}
	}
}

// TestGradleOutputGoesToStdoutInOrder: gradle's stdout and stderr are one stream, on the chain's stdout.
func TestGradleOutputGoesToStdoutInOrder(t *testing.T) {
	f := newFixture(t)
	f.does("gradlew", `for i in 1 2 3; do echo "out $i"; echo "err $i" >&2; done`)
	f.wants("order", f.build("--no-preflight", "--profiles", "rt"), ExitOK)
	want := "==> build (Java 8 + slotomatic): :Foo-rt:clean :Foo-rt:slotomatic :Foo-rt:jar\nout 1\nerr 1\nout 2\nerr 2\nout 3\nerr 3\n==> verify gate (verify-module.sh):\n"
	if !strings.HasPrefix(f.out, want) || f.err != "" {
		t.Errorf("stdout %q stderr %q, want stdout prefix %q", f.out, f.err, want)
	}
}

// TestCopyLockNamesTheJarsBuiltThisRun: only a build/libs jar newer than the start of the run is the
// artifact of record.
func TestCopyLockNamesTheJarsBuiltThisRun(t *testing.T) {
	f := newFixture(t)
	f.write(f.built("ux"), "stale")
	f.does("gradlew", `sleep 0.05; mkdir -p "`+f.root+`/Foo/Foo-rt/build/libs"; printf jar > "`+f.built("rt")+`"; echo "FileAlreadyExistsException: nh/modules/Foo-rt.jar"; exit 1`)
	f.wants("copy lock", f.build(), ExitCopyLocked,
		"    "+f.built("rt")+" (built this run)\n    "+f.built("ux")+" — not rebuilt this run (missing or stale): gradle stopped before this profile\n")
}

// TestSlotomaticSkipInAPassingRun: a skipped file fails the chain although gradle passed, the pre-build jar
// is put back, and the gate does not run.
func TestSlotomaticSkipInAPassingRun(t *testing.T) {
	f := newFixture(t)
	f.jar(f.deployed("rt"), "1.0.0", "old")
	before := f.read(f.deployed("rt"))
	f.does("gradlew", `printf new > "`+f.deployed("rt")+`"; echo "WARNING: Could not process /x/src/BHmi.java: Found multiple metadata blocks"; echo "BUILD SUCCESSFUL"`)
	f.wants("skip", f.build(), ExitSlotomatic,
		"FAIL slotomatic-skip /x/src/BHmi.java (Found multiple metadata blocks)\nbuild.sh: Slot-o-Matic skipped the file(s) above",
		"  Restored the previously-deployed Foo-rt.jar into "+f.home+"/modules.\n")
	if f.read(f.deployed("rt")) != before {
		t.Error("the pre-build jar was not restored")
	}
	if f.call(gateScript) != "" {
		t.Error("the gate ran")
	}

	f = newFixture(t)
	f.does("gradlew", `echo "Could not process /x/srcTest/BFooTest.java: Found multiple metadata blocks"`)
	f.wants("a skipped test source is not a skip", f.build(), ExitOK)
}
