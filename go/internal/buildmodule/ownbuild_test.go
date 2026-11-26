package buildmodule

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
)

// recordPath is where a mirror build records the jars it installed, in the module root.
func (f *fixture) recordPath() string { return f.root + "/.build/mirror-built" }

// digest is the sha256 of the bytes of the file at path, as `sha256sum` prints it.
func (f *fixture) digest(path string) string {
	f.t.Helper()
	sum := sha256.Sum256([]byte(f.read(path)))
	return hex.EncodeToString(sum[:])
}

// mirrorBuild sets up a build into a mirror: modules/ holds the previous jar (version 1.0.0, payload "old"), the
// build produces a jar of the same version with other bytes and gradle's :jar step installs it, as the drift gate
// reads it (exit 51 without a record).
func (f *fixture) mirrorBuild() {
	f.t.Helper()
	f.signs()
	f.write(f.home+"/.niagara-mirror", "source=/real\n")
	f.jar(f.deployed("rt"), "1.0.0", "old")
	f.jar(f.built("rt"), "1.0.0", "NEW payload")
	f.does("gradlew", `cp "`+f.built("rt")+`" "`+f.deployed("rt")+`"`)
}

const drift51 = "build.sh: FAIL — Foo-rt shipped bytes changed but vendorVersion is still 1.0.0."

// TestOwnMirrorBuildSkipsTheDriftCheck: the jar the mirror held before this build is the one this checkout built
// (its sha256 is the one .build/mirror-built records for Foo-rt.jar), so it is no release: no exit 51, one line
// saying so, and the new jar stays in modules/.
func TestOwnMirrorBuildSkipsTheDriftCheck(t *testing.T) {
	f := newFixture(t)
	f.mirrorBuild()
	f.write(f.recordPath(), "Foo-ux.jar abc\nFoo-rt.jar "+f.digest(f.deployed("rt"))+"\n")
	f.wants("own build", f.build(), ExitOK,
		"build.sh: INFO — the mirror's Foo-rt.jar is this checkout's own build of 1.0.0, not a release: drift check skipped\n")
	if f.read(f.deployed("rt")) != f.read(f.built("rt")) {
		t.Errorf("modules/ does not hold the new jar")
	}
	if strings.Contains(f.err, "Restored") || strings.Contains(f.err, "FAIL") {
		t.Errorf("stderr: %s", f.err)
	}
}

// TestDriftCheckIgnoresARecordThatDoesNotMatch: a different digest, no record, a record that cannot be read and
// a record that names other jars leave the drift check as it was: exit 51 and the pre-build jar restored.
func TestDriftCheckIgnoresARecordThatDoesNotMatch(t *testing.T) {
	for name, setup := range map[string]func(f *fixture){
		"digest differs":  func(f *fixture) { f.write(f.recordPath(), "Foo-rt.jar "+strings.Repeat("0", 64)+"\n") },
		"no record":       func(f *fixture) {},
		"unreadable":      func(f *fixture) { f.mkdir(f.recordPath()) }, // a directory
		"other jar names": func(f *fixture) { f.write(f.recordPath(), "Foo-ux.jar "+f.digest(f.deployed("rt"))+"\n") },
		"garbled":         func(f *fixture) { f.write(f.recordPath(), "Foo-rt.jar\n\x00\x01 junk\n") },
	} {
		f := newFixture(t)
		f.mirrorBuild()
		before := f.read(f.deployed("rt"))
		setup(f)
		f.wants(name, f.build(), ExitDrift, drift51)
		if strings.Contains(f.err, "own build") || f.read(f.deployed("rt")) != before {
			t.Errorf("%s: skipped the check or did not restore the old jar:\n%s", name, f.err)
		}
	}
}

// TestMirrorBuiltIsWrittenAfterTheInstall: a passing mirror build records "<jar file name> <sha256>" for each
// jar it installed, creating .build/, and keeps the lines of jars it did not build; nothing is written into
// modules/ but the jars.
func TestMirrorBuiltIsWrittenAfterTheInstall(t *testing.T) {
	f := newFixture(t)
	f.mirrorBuild()
	f.jar(f.built("ux"), "1.0.0", "ux")
	f.does("gradlew", `cp "`+f.built("rt")+`" "`+f.deployed("rt")+`"; cp "`+f.built("ux")+`" "`+f.deployed("ux")+`"`)
	f.write(f.recordPath(), "Other-rt.jar "+strings.Repeat("a", 64)+"\nFoo-rt.jar "+strings.Repeat("b", 64)+"\n")
	f.wants("record", f.build("--no-drift-check"), ExitOK)
	want := "Foo-rt.jar " + f.digest(f.deployed("rt")) + "\nFoo-ux.jar " + f.digest(f.deployed("ux")) + "\n" +
		"Other-rt.jar " + strings.Repeat("a", 64) + "\n"
	if got := f.read(f.recordPath()); got != want {
		t.Errorf("record:\n%s\nwant:\n%s", got, want)
	}
	if entries, _ := os.ReadDir(f.home + "/modules"); len(entries) != 2 {
		t.Errorf("modules/ holds %d entries, want the 2 jars", len(entries))
	}
}

// TestMirrorBuiltIsNotWrittenForWhatDidNotPass: a failed gate rolls the jar back, and a home that is not a
// mirror is no mirror build: neither leaves a record (not even .build/).
func TestMirrorBuiltIsNotWrittenForWhatDidNotPass(t *testing.T) {
	for name, setup := range map[string]func(f *fixture){
		"gate fails":     func(f *fixture) { f.does(gateScript, "exit 1") },
		"not a mirror":   func(f *fixture) { os.Remove(f.home + "/.niagara-mirror") },
		"drift detected": func(f *fixture) {},
	} {
		f := newFixture(t)
		f.mirrorBuild()
		if name == "drift detected" {
			f.jar(f.built("rt"), "1.0.0", "NEW payload") // no record: exit 51
		} else {
			f.jar(f.deployed("rt"), "1.0.0", "old")
			f.jar(f.built("rt"), "1.0.1", "NEW payload") // a bump: the drift gate passes
		}
		setup(f)
		if code := f.build(); code == ExitOK && name != "not a mirror" {
			t.Errorf("%s: exit 0", name)
		}
		if _, err := os.Stat(f.root + "/.build"); err == nil {
			t.Errorf("%s: %s/.build exists", name, f.root)
		}
	}
}
