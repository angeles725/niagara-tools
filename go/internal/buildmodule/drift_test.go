package buildmodule

import (
	"os"
	"strings"
	"testing"
)

// TestDriftVerdict: changed shipped bytes need a changed vendorVersion; what cannot be compared fails closed.
func TestDriftVerdict(t *testing.T) {
	const same, changed = "payload", "NEW payload"
	cases := []struct {
		name           string
		oldVer, newVer string // "" writes a jar without a module version
		payload        string // the new jar's payload; the deployed one holds `same`
		code           int
		has            string
		restored       bool
	}{
		{"bytes changed, version unchanged", "1.0.0", "1.0.0", changed, ExitDrift, "build.sh: FAIL — Foo-rt shipped bytes changed but vendorVersion is still 1.0.0.\n", true},
		{"a missing component is 0", "1.3", "1.3.0", changed, ExitDrift, "vendorVersion is still 1.3.0.", true},
		{"bytes changed, version bumped", "1.0.0", "1.0.1", changed, ExitOK, "", false},
		{"components compare as numbers", "1.9", "1.10", changed, ExitOK, "", false},
		{"a differing suffix is a changed version", "1.3", "1.3-rc", changed, ExitOK, "", false},
		{"a lower version warns and passes", "1.10", "1.9", changed, ExitOK, "build.sh: WARN — Foo-rt vendorVersion went down (1.10 -> 1.9) versus the deployed jar", false},
		{"identical payload", "1.0.0", "1.0.0", same, ExitOK, "", false},
		{"identical payload needs no version", "", "", same, ExitOK, "", false},
		{"bytes changed, new version unreadable", "1.0.0", "", changed, ExitDriftUnknown,
			"build.sh: FAIL — drift gate cannot verify Foo-rt: shipped bytes changed but the module's own vendorVersion is unreadable (old='1.0.0' new='').\n", true},
		{"bytes changed, old version unreadable", "", "1.0.0", changed, ExitDriftUnknown, "(old='' new='1.0.0')", true},
	}
	for _, c := range cases {
		f := newFixture(t)
		f.signs()
		f.jar(f.deployed("rt"), c.oldVer, same)
		f.jar(f.built("rt"), c.newVer, c.payload)
		before := f.read(f.deployed("rt"))
		f.does("gradlew", `cp "`+f.built("rt")+`" "`+f.deployed("rt")+`"`) // as the plugin's last :jar step
		f.wants(c.name, f.build(), c.code, c.has)
		if got := f.read(f.deployed("rt")) == before; got != c.restored && c.payload != same {
			t.Errorf("%s: modules/ holds the pre-build jar = %v, want %v", c.name, got, c.restored)
		}
		if got := strings.Contains(f.err, "  Restored the previously-deployed Foo-rt.jar into "+f.home+"/modules"); got != c.restored {
			t.Errorf("%s: restore row = %v, want %v: %s", c.name, got, c.restored, f.err)
		}
		if ran := f.call(gateScript) != ""; ran != (c.code == ExitOK) {
			t.Errorf("%s: the gate ran = %v", c.name, ran)
		}
		if c.code == ExitOK && c.has == "" && f.err != "" {
			t.Errorf("%s: stderr %q", c.name, f.err)
		}
	}
}

// TestDriftGateScope: no deployed jar, no built jar or --no-drift-check leave nothing to compare.
func TestDriftGateScope(t *testing.T) {
	for name, setup := range map[string]func(f *fixture) []string{
		"no deployed jar": func(f *fixture) []string { f.jar(f.built("rt"), "1.0.0", "new"); return nil },
		"no built jar":    func(f *fixture) []string { f.jar(f.deployed("rt"), "1.0.0", "old"); return nil },
		"--no-drift-check": func(f *fixture) []string {
			f.jar(f.deployed("rt"), "1.0.0", "old")
			f.jar(f.built("rt"), "1.0.0", "new")
			return []string{"--no-drift-check"}
		},
	} {
		f := newFixture(t)
		f.signs()
		f.wants(name, f.build(setup(f)...), ExitOK)
		if f.err != "" {
			t.Errorf("%s: stderr %q", name, f.err)
		}
	}
}

// TestDriftOutranksCannotVerify: with one profile drifted and another unverifiable the run is a drift (51),
// and both rows are printed.
func TestDriftOutranksCannotVerify(t *testing.T) {
	f := newFixture(t)
	f.jar(f.deployed("rt"), "", "old")
	f.jar(f.built("rt"), "1.0", "new")
	f.jar(f.deployed("ux"), "2.0", "old")
	f.jar(f.built("ux"), "2.0", "new")
	f.wants("drift and cannot verify", f.build(), ExitDrift, "drift gate cannot verify Foo-rt", "Foo-ux shipped bytes changed",
		"  Restored the previously-deployed Foo-ux.jar into "+f.home+"/modules — a retry will catch this again.\n",
		"build.sh: deployed-baseline drift detected")
	if strings.Contains(f.err, "could not verify the jar(s) above") {
		t.Errorf("the cannot-verify verdict was printed too: %s", f.err)
	}
}

// TestFailedRestoreSaysWhatModulesHolds: a restore that fails names cp's reason and tells a modules/ jar
// that is still the pre-build one from one that is not.
func TestFailedRestoreSaysWhatModulesHolds(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: a read-only jar does not block the copy")
	}
	for name, c := range map[string]struct{ gradlew, want string }{
		"still the pre-build jar": {``, "); <nh>/modules still holds the pre-build jar, so a retry is still safe.\n"},
		"the new jar":             {`chmod 644 "<jar>"; printf new > "<jar>"; `, ") — <nh>/modules no longer holds it (likely the NEW jar); restore it manually before retrying.\n"},
	} {
		f := newFixture(t)
		fill := strings.NewReplacer("<nh>", f.home, "<jar>", f.deployed("rt")).Replace
		f.jar(f.deployed("rt"), "1.0.0", "old")
		f.jar(f.built("rt"), "1.0.0", "new")
		f.does("gradlew", fill(c.gradlew)+`chmod 444 "`+f.deployed("rt")+`"`)
		f.wants(name, f.build(), ExitDrift, "FAILED TO RESTORE", "Permission denied", fill(c.want))
	}
}

// TestAJarTheGateCannotReadIsNeverEqualToItself: a deployed jar that is no readable zip (the host has no
// unzip to stub, so the fault is injected into the file) makes the comparison "cannot verify" (52), never
// two equal empty hashes; the pre-build jar is put back, and a restore that fails is reported, never silent.
// The cases that stubbed unzip on PATH in tests/build-sh.bats (BS-drift-unverifiable-unzip, -restore-fail,
// BS-drift-empty-listing) are these. [ev: retro go-build-module-port Δ3]
func TestAJarTheGateCannotReadIsNeverEqualToItself(t *testing.T) {
	for _, name := range []string{"unreadable", "empty"} {
		f := newFixture(t)
		f.signs()
		if name == "unreadable" {
			f.write(f.deployed("rt"), "this is not a zip")
		} else {
			f.write(f.deployed("rt"), "")
		}
		f.jar(f.built("rt"), "1.0.0", "NEW payload")
		before := f.read(f.deployed("rt"))
		f.does("gradlew", `cp "`+f.built("rt")+`" "`+f.deployed("rt")+`"`)
		f.wants(name, f.build(), ExitDriftUnknown, "drift gate cannot verify Foo-rt", "  Restored the previously-deployed Foo-rt.jar into "+f.home+"/modules")
		if f.read(f.deployed("rt")) != before {
			t.Errorf("%s: modules/ does not hold the pre-build jar", name)
		}
		if f.call(gateScript) != "" {
			t.Errorf("%s: the gate ran after a failed comparison", name)
		}
	}
}

// TestAFailedRestoreAfterACannotVerifyIsReported: the same, with a modules/ jar the copy cannot overwrite.
func TestAFailedRestoreAfterACannotVerifyIsReported(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: a read-only jar does not block the copy")
	}
	f := newFixture(t)
	f.signs()
	f.write(f.deployed("rt"), "this is not a zip")
	f.jar(f.built("rt"), "1.0.0", "NEW payload")
	f.does("gradlew", `chmod 444 "`+f.deployed("rt")+`"`)
	f.wants("cannot verify, restore fails", f.build(), ExitDriftUnknown, "drift gate cannot verify Foo-rt", "FAILED TO RESTORE", "Permission denied")
}
