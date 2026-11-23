package buildmodule

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

// realInstall makes the installation a mirror is made from: modules/baja.jar of that Niagara version.
func realInstall(f *fixture, version string) string {
	src := f.dir + "/real/Niagara-" + version
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("META-INF/module.xml")
	if err == nil {
		_, err = w.Write([]byte(`<module name="baja" vendor="Tridium" vendorVersion="` + version + `"/>`))
	}
	if err == nil {
		err = zw.Close()
	}
	if err != nil {
		f.t.Fatal(err)
	}
	f.write(src+"/modules/baja.jar", buf.String())
	return src
}

// mirrorMadeFrom makes the fixture's home a mirror whose marker names src, as mirror-niagara-home.sh writes it.
func mirrorMadeFrom(f *fixture, src string) {
	f.write(f.home+"/"+mirrorMarker, "source="+src+"\ncreated=2026-11-23T10:00:00\n")
}

// sourceLine is the last line of a passing mirror build: the installation the mirror was made from, that the
// jars were NOT installed there, and the command that installs them.
func sourceLine(f *fixture, src, version string) string {
	check := f.kit + "/check-target-version.sh " + f.built("rt") + " " + src
	return "==> mirror: " + f.home + " was made from " + src + " (Niagara " + version + "); the jars built this run were NOT installed in " + src +
		"/modules — to install them, stop the station and close Workbench of that installation, then run: " + check +
		" && cp " + f.built("rt") + " " + f.built("ux") + " " + src + "/modules/\n"
}

// A passing mirror build ends with one line naming the real installation and that the jars are not in it.
func TestAMirrorBuildEndsWithTheInstallationItWasMadeFrom(t *testing.T) {
	f := newFixture(t)
	f.signs()
	src := realInstall(f, "4.14.0.162")
	mirrorMadeFrom(f, src)
	f.wants("mirror build", f.run(f.root, "Foo", f.home), ExitOK)
	if want := sourceLine(f, src, "4.14.0.162"); !strings.HasSuffix(f.out, want) || strings.Count(f.out, "was made from") != 1 {
		t.Errorf("stdout does not end with the line\n got: %q\nwant: %q", f.out, want)
	}
	if exists(src + "/modules/Foo-rt.jar") {
		t.Errorf("the build wrote into the real installation")
	}
}

// The line comes after the replaced-jar line, which stays one line of its own, and a source whose version
// cannot be read is named "unknown", never left out.
func TestTheInstallationLineFollowsTheReplacedJarLineAndSaysUnknownForAnUnreadableVersion(t *testing.T) {
	f, _ := mirrorBuild(t)
	src := f.dir + "/real/odd"
	f.mkdir(src + "/modules")
	mirrorMadeFrom(f, src)
	f.wants("mirror build", f.run(f.root, "Foo", f.home), ExitOK, "==> mirror: Foo-rt.jar in "+f.home+"/modules replaced")
	if want := sourceLine(f, src, "unknown"); !strings.HasSuffix(f.out, want) {
		t.Errorf("stdout does not end with the line\n got: %q\nwant: %q", f.out, want)
	}
	if replaced := strings.Index(f.out, "replaced ("); replaced < 0 || replaced > strings.Index(f.out, "was made from") {
		t.Errorf("the installation line is not after the replaced-jar line:\n%s", f.out)
	}
}

// No line for a run that is not a mirror build that passed: a real home, a marker that names no source, a
// build that failed.
func TestNoInstallationLineOutsideAPassingMirrorBuild(t *testing.T) {
	f := newFixture(t)
	f.wants("a real home", f.build(), ExitOK)
	if strings.Contains(f.out, "was made from") {
		t.Errorf("a real home: %q", f.out)
	}

	f = newFixture(t)
	f.write(f.home+"/"+mirrorMarker, "")
	f.wants("a marker without a source", f.run(f.root, "Foo", f.home), ExitOK)
	if strings.Contains(f.out, "was made from") {
		t.Errorf("a marker without a source: %q", f.out)
	}

	f = newFixture(t)
	mirrorMadeFrom(f, realInstall(f, "4.14.0.162"))
	f.does("gradlew", `echo "BUILD FAILED"; exit 1`)
	f.wants("a failed build", f.run(f.root, "Foo", f.home), ExitGradle)
	if strings.Contains(f.out+f.err, "was made from") {
		t.Errorf("a failed build: %q %q", f.out, f.err)
	}
}

// A post-jar copy that is locked tells the operator to stop the station and close Workbench of the installation.
func TestACopyLockSaysTheStationAndWorkbenchOfThatInstallationMustBeStopped(t *testing.T) {
	f := newFixture(t)
	f.does("gradlew", `echo "FileAlreadyExistsException: nh/modules/Foo-rt.jar"; exit 1`)
	f.wants("copy lock", f.build(), ExitCopyLocked,
		"  Stop the station and close Workbench of the installation "+f.home+" (on Windows the station is a service or station.exe), then re-run")
}
