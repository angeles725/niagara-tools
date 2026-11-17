package buildmodule

import (
	"os"
	"strings"
	"testing"
)

// interruptGradle makes the fake gradle install the new jars (as the plugin's last :jar step), send sig to
// this process, and wait until the handler has run (the fake exit leaves a marker) before it ends, so the
// handler sees modules/ as a real interrupt mid-build would.
func interruptGradle(t *testing.T, f *fixture, sig string, install string) (exitCode *int) {
	t.Helper()
	marker := f.dir + "/exited"
	code := new(int)
	*code = -1
	exitProcess = func(c int) { *code = c; _ = os.WriteFile(marker, nil, 0o644) }
	t.Cleanup(func() { exitProcess = os.Exit })
	f.does("gradlew", install+"\nkill -"+sig+" $PPID\nfor _ in $(seq 100); do [ -f \""+marker+"\" ] && break; sleep 0.05; done\n")
	return code
}

// A SIGINT or SIGTERM while gradle runs (after it installed the new jar into modules/, before the gate passed)
// puts the pre-build jar back, takes out a jar that had no predecessor, then cleans up and ends with 128 +
// the signal: an interrupted run must not leave an unverified jar where the station loads it (build-mod Δ6).
func TestASignalBeforeTheGateRestoresThePreBuildJars(t *testing.T) {
	for _, tc := range []struct {
		sig  string
		code int
	}{{"INT", 130}, {"TERM", 143}} {
		f := newFixture(t)
		f.signs()
		f.jar(f.deployed("rt"), "1.0.0", "old")
		f.jar(f.built("rt"), "1.0.1", "new")
		f.jar(f.built("ux"), "1.0.1", "new ux")
		before := f.read(f.deployed("rt"))
		code := interruptGradle(t, f, tc.sig,
			`cp "`+f.built("rt")+`" "`+f.deployed("rt")+`"; cp "`+f.built("ux")+`" "`+f.deployed("ux")+`"`)
		f.build()
		if *code != tc.code {
			t.Errorf("SIG%s: exit %d, want %d\nstderr: %s", tc.sig, *code, tc.code, f.err)
		}
		if f.read(f.deployed("rt")) != before {
			t.Errorf("SIG%s: modules/Foo-rt.jar is not the pre-build jar\nstderr: %s", tc.sig, f.err)
		}
		if exists(f.deployed("ux")) {
			t.Errorf("SIG%s: modules/Foo-ux.jar (a first deploy) was left in place\nstderr: %s", tc.sig, f.err)
		}
		for _, text := range []string{
			"build.sh: interrupted (SIG" + tc.sig + ") before the gate passed; putting modules/ back as it was before the build",
			"  Restored the previously-deployed Foo-rt.jar into " + f.home + "/modules.",
			"  Removed the newly-installed Foo-ux.jar from " + f.home + "/modules",
		} {
			if !strings.Contains(f.err, text) {
				t.Errorf("SIG%s: stderr lacks %q:\n%s", tc.sig, text, f.err)
			}
		}
	}
}

// --no-drift-check takes no snapshot, so there is nothing to put back: the signal still cleans up and ends
// with 128 + the signal, and says modules/ was left as the build left it.
func TestASignalWithoutASnapshotSaysModulesIsUntouched(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.jar(f.deployed("rt"), "1.0.0", "old")
	f.jar(f.built("rt"), "1.0.1", "new")
	code := interruptGradle(t, f, "INT", `cp "`+f.built("rt")+`" "`+f.deployed("rt")+`"`)
	f.build("--no-drift-check")
	if *code != 130 || !strings.Contains(f.err, "build.sh: interrupted (SIGINT) under --no-drift-check: no pre-build snapshot, "+f.home+"/modules is left as the build left it") {
		t.Errorf("exit %d, stderr: %s", *code, f.err)
	}
}
