package buildmodule

import (
	"os"
	"strings"
	"testing"
)

// mirrorLine is the stdout line of a mirror build that replaced Foo-rt.jar: this build's old and new jar, then
// the first jar replaced this cycle and the command that restores it.
func mirrorLine(f *fixture, oldVer, oldSum, newVer, newSum, firstVer, firstSum string) string {
	return "==> mirror: Foo-rt.jar in " + f.home + "/modules replaced (vendorVersion " + oldVer + " sha256:" + oldSum +
		" -> " + newVer + " sha256:" + newSum + "); first replaced this cycle: vendorVersion " + firstVer +
		" sha256:" + firstSum + "; restore with: " + f.kit + "/build.sh --restore-mirror " + f.home +
		"/.mirror-replaced/Foo-rt.jar." + firstSum + "\n"
}

// TestTwoBuildsRestoreTheOriginal pins B6 itself (a module built twice in one day): the second build replaces
// the FIRST build's jar, yet the restore command still names the jar that was there before the first build,
// and both replaced jars are kept by their hash.
func TestTwoBuildsRestoreTheOriginal(t *testing.T) {
	f, deployedSum := mirrorBuild(t)
	deployed := f.read(f.deployed("rt"))
	f.wants("first build", f.run(f.root, "Foo", f.home), ExitOK)
	firstSum := sha12(t, f.deployed("rt"))

	f.jar(f.built("rt"), "1.0.2", "NEWER payload")
	f.wants("second build", f.run(f.root, "Foo", f.home), ExitOK)
	secondSum := sha12(t, f.deployed("rt"))
	if want := mirrorLine(f, "1.0.1", firstSum, "1.0.2", secondSum, "1.0.0", deployedSum); !strings.HasSuffix(f.out, want) {
		t.Errorf("second build line\n got: %q\nwant suffix: %q", f.out, want)
	}
	for _, sum := range []string{deployedSum, firstSum} {
		if !exists(f.home + "/.mirror-replaced/Foo-rt.jar." + sum) {
			t.Errorf("replaced jar %s not kept", sum)
		}
	}
	f.wants("restore", f.run("--restore-mirror", f.home+"/.mirror-replaced/Foo-rt.jar."+deployedSum), ExitOK)
	if f.read(f.deployed("rt")) != deployed {
		t.Errorf("the restore did not put the pre-first-build jar back")
	}

	// The restore closes the cycle: the next build's first replaced jar is the restored one again.
	f.wants("third build", f.run(f.root, "Foo", f.home), ExitOK)
	if want := mirrorLine(f, "1.0.0", deployedSum, "1.0.2", secondSum, "1.0.0", deployedSum); !strings.HasSuffix(f.out, want) {
		t.Errorf("third build line\n got: %q\nwant suffix: %q", f.out, want)
	}
}

// TestRestoreRefusesAKeptJarWhoseHashDiffers: a kept jar whose sha256 prefix is not the one in its name is
// refused (exit 10) and modules/ is not written.
func TestRestoreRefusesAKeptJarWhoseHashDiffers(t *testing.T) {
	f, oldSum := mirrorBuild(t)
	f.wants("build", f.run(f.root, "Foo", f.home), ExitOK)
	kept := f.home + "/.mirror-replaced/Foo-rt.jar." + oldSum
	f.write(kept, "tampered")
	installed := f.read(f.deployed("rt"))
	f.wants("hash mismatch", f.run("--restore-mirror", kept), ExitEnv,
		"build.sh: --restore-mirror refuses "+kept+": its sha256 starts with "+sha12(t, kept)+", not "+oldSum+" (the jar was changed after it was kept)\n")
	if f.read(f.deployed("rt")) != installed {
		t.Errorf("a refused restore wrote modules/")
	}
}

// TestKeepFailureWarnsAndKeepsTheExitCode: when the replaced jar cannot be kept (an unwritable
// .mirror-replaced/), the run's exit code is what it was, stderr has one WARN saying the old jar was not kept,
// and no restore line is printed.
func TestKeepFailureWarnsAndKeepsTheExitCode(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes an unwritable directory")
	}
	for _, c := range []struct {
		name   string
		report string
		code   int
	}{{"passing run", "exit 0", ExitOK}, {"incomplete report", "exit 4", ExitReportPartial}} {
		f, _ := mirrorBuild(t)
		f.does(reportScript, c.report)
		f.mkdir(f.home + "/.mirror-replaced")
		if err := os.Chmod(f.home+"/.mirror-replaced", 0o555); err != nil {
			t.Fatal(err)
		}
		f.wants(c.name, f.run(f.root, "Foo", f.home), c.code)
		_ = os.Chmod(f.home+"/.mirror-replaced", 0o755)
		warn := "build.sh: WARN — the replaced Foo-rt.jar was NOT kept in " + f.home + "/.mirror-replaced ("
		if n := strings.Count(f.err, warn); n != 1 {
			t.Errorf("%s: %d keep WARN lines, want 1\nstderr: %s", c.name, n, f.err)
		}
		if strings.Contains(f.out, "==> mirror:") {
			t.Errorf("%s: a restore line was printed for a jar that was not kept", c.name)
		}
	}
}

// TestIncompleteCrossRefOnlyForSchemaRisk: the INCOMPLETE line points at the schema-risk SKIP row only when an
// artifact lacks .deploy-baseline (and --first-deploy is off); otherwise the line is what it was.
func TestIncompleteCrossRefOnlyForSchemaRisk(t *testing.T) {
	const crossRef = " (the schema-risk SKIP row above names the command that creates it)"
	for _, c := range []struct {
		name       string
		baselines  bool
		flags      []string
		wantSuffix bool
	}{
		{"no baseline", false, nil, true},
		{"every artifact has a baseline", true, nil, false},
		{"--first-deploy", false, []string{"--first-deploy"}, false},
	} {
		f := newFixture(t)
		f.signs()
		f.does(reportScript, "exit 4")
		if c.baselines {
			for _, p := range []string{"rt", "ux", "wb"} {
				f.mkdir(f.root + "/Foo/Foo-" + p + "/.deploy-baseline")
			}
		}
		f.wants(c.name, f.build(c.flags...), ExitReportPartial, msgReportIncomplete)
		if got := strings.Contains(f.err, msgReportIncomplete+crossRef+"\n"); got != c.wantSuffix {
			t.Errorf("%s: cross-reference = %v, want %v\nstderr: %s", c.name, got, c.wantSuffix, f.err)
		}
		if !c.wantSuffix && !strings.Contains(f.err, msgReportIncomplete+"\n") {
			t.Errorf("%s: the INCOMPLETE line changed: %s", c.name, f.err)
		}
	}
}
