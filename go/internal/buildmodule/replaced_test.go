package buildmodule

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
)

// sha12 is the first 12 hex digits of the sha256 of a file.
func sha12(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:12]
}

// mirrorBuild is a mirror fixture whose deployed Foo-rt is version 1.0.0 and whose build installs a 1.0.1 jar
// with other bytes into modules/, as the plugin's last :jar step does.
func mirrorBuild(t *testing.T) (f *fixture, oldSum string) {
	f = newFixture(t)
	f.signs()
	f.write(f.home+"/"+mirrorMarker, "")
	f.jar(f.deployed("rt"), "1.0.0", "old payload")
	f.jar(f.built("rt"), "1.0.1", "NEW payload")
	f.does("gradlew", `cp "`+f.built("rt")+`" "`+f.deployed("rt")+`"`)
	return f, sha12(t, f.deployed("rt"))
}

// TestMirrorKeepsTheReplacedJar pins B6: a passing build that replaces a mirror's modules/ jar keeps the jar it
// replaced under <mirror>/.mirror-replaced/, prints one line naming both versions and hashes and the restore
// command, and that command puts the deployed jar back.
func TestMirrorKeepsTheReplacedJar(t *testing.T) {
	f, oldSum := mirrorBuild(t)
	before := f.read(f.deployed("rt"))
	kept := f.home + "/.mirror-replaced/Foo-rt.jar." + oldSum
	f.wants("mirror build", f.run(f.root, "Foo", f.home), ExitOK)
	newSum := sha12(t, f.deployed("rt"))
	if f.read(kept) != before {
		t.Fatalf("the replaced jar is not kept at %s", kept)
	}
	line := mirrorLine(f, "1.0.0", oldSum, "1.0.1", newSum, "1.0.0", oldSum)
	if !strings.HasSuffix(f.out, line) {
		t.Errorf("stdout does not end with the mirror line\n got: %q\nwant suffix: %q", f.out, line)
	}

	f.wants("restore", f.run("--restore-mirror", kept), ExitOK,
		"==> mirror: restored "+f.home+"/modules/Foo-rt.jar from "+kept+" (vendorVersion 1.0.0 sha256:"+oldSum+")\n")
	if f.read(f.deployed("rt")) != before {
		t.Errorf("--restore-mirror did not put the deployed jar back")
	}
	if f.read(kept) != before {
		t.Errorf("--restore-mirror must leave the kept jar in place")
	}
}

// TestMirrorKeepsNothingWhenModulesIsUnchanged: no line and no .mirror-replaced/ when the run leaves modules/ as
// it was — identical bytes, a drift FAIL that restored it, --no-drift-check (no snapshot), or a real home.
func TestMirrorKeepsNothingWhenModulesIsUnchanged(t *testing.T) {
	cases := []struct {
		name  string
		setup func(f *fixture)
		args  func(f *fixture) []string
		code  int
	}{
		{"identical jar", func(f *fixture) { f.jar(f.built("rt"), "1.0.0", "old payload") },
			func(f *fixture) []string { return []string{f.root, "Foo", f.home} }, ExitOK},
		{"drift FAIL restores modules/", func(f *fixture) { f.jar(f.built("rt"), "1.0.0", "NEW payload") },
			func(f *fixture) []string { return []string{f.root, "Foo", f.home} }, ExitDrift},
		{"--no-drift-check keeps no snapshot", func(*fixture) {},
			func(f *fixture) []string { return []string{"--no-drift-check", f.root, "Foo", f.home} }, ExitOK},
		{"a real home (--install-into-home)", func(f *fixture) { os.Remove(f.home + "/" + mirrorMarker) },
			func(f *fixture) []string { return []string{"--install-into-home", f.root, "Foo", f.home} }, ExitOK},
	}
	for _, c := range cases {
		f, _ := mirrorBuild(t)
		c.setup(f)
		f.wants(c.name, f.run(c.args(f)...), c.code)
		if exists(f.home+"/.mirror-replaced") || strings.Contains(f.out, "==> mirror:") {
			t.Errorf("%s: kept a jar or printed the mirror line: %q", c.name, f.out)
		}
	}
}

// TestMirrorLineAfterIncompleteReport: an exit-53 run still leaves the new jar in modules/, so it keeps the old
// one too; the exit code stays 53.
func TestMirrorLineAfterIncompleteReport(t *testing.T) {
	f, oldSum := mirrorBuild(t)
	f.does(reportScript, "exit 4")
	f.wants("incomplete report", f.run(f.root, "Foo", f.home), ExitReportPartial, "==> mirror: Foo-rt.jar in ")
	if !exists(f.home + "/.mirror-replaced/Foo-rt.jar." + oldSum) {
		t.Errorf("exit 53 kept no jar")
	}
}

// TestRestoreMirrorRefuses: --restore-mirror takes only a kept jar of a mirror and no positional argument, and
// writes nothing when it refuses.
func TestRestoreMirrorRefuses(t *testing.T) {
	f, oldSum := mirrorBuild(t)
	f.wants("build", f.run(f.root, "Foo", f.home), ExitOK)
	kept := f.home + "/.mirror-replaced/Foo-rt.jar." + oldSum
	installed := f.read(f.deployed("rt"))
	f.wants("no value", f.run("--restore-mirror"), ExitUsage)
	f.wants("positional args", f.run("--restore-mirror", kept, f.root), ExitUsage)
	f.wants("not under .mirror-replaced", f.run("--restore-mirror", f.built("rt")), ExitUsage,
		"build.sh: --restore-mirror needs a jar a build kept: <mirror>/.mirror-replaced/<MOD>-<profile>.jar.<sha12> (got: "+f.built("rt")+")\n")
	f.wants("missing kept jar", f.run("--restore-mirror", f.home+"/.mirror-replaced/Foo-rt.jar.000000000000"), ExitEnv,
		"build.sh: --restore-mirror: no such kept jar: "+f.home+"/.mirror-replaced/Foo-rt.jar.000000000000\n")
	if err := os.Remove(f.home + "/" + mirrorMarker); err != nil {
		t.Fatal(err)
	}
	f.wants("not a mirror", f.run("--restore-mirror", kept), ExitEnv,
		"build.sh: --restore-mirror refuses "+f.home+": no .niagara-mirror marker (only a mirror is restored)\n")
	if f.read(f.deployed("rt")) != installed {
		t.Errorf("a refused --restore-mirror wrote modules/")
	}
}
