package ngdeploy

import (
	"strings"
	"testing"
)

// moduleXMLOf is the META-INF/module.xml of a built demo jar that declares the given Niagara floor ("" declares none).
func moduleXMLOf(floor string) map[string]string {
	deps := ""
	if floor != "" {
		deps = `<dependency name="baja" vendor="Tridium" vendorVersion="` + floor + `"/>`
	}
	return map[string]string{"META-INF/module.xml": `<module name="demo" vendor="Acme" vendorVersion="1.1"><dependencies>` + deps + `</dependencies></module>`}
}

// installed makes the fixture's station directory the modules directory of a Niagara installation of that
// version (baja.jar, as an installation ships it) and returns the installation's directory.
func installed(f *fixture, version string) string {
	f.zipJar(f.modules+"/baja.jar", map[string]string{"META-INF/module.xml": `<module name="baja" vendor="Tridium" vendorVersion="` + version + `"/>`})
	return f.dir
}

// compiledFor makes the built rt and ux jars declare that Niagara floor.
func compiledFor(f *fixture, floor string) {
	f.zipJar(f.builtJar("rt"), moduleXMLOf(floor))
	f.zipJar(f.builtJar("ux"), moduleXMLOf(floor))
}

// A module compiled against the version of the target installation deploys, and the line says both versions.
func TestADeployIntoTheInstallationItWasCompiledForGoesThrough(t *testing.T) {
	f := newFixture(t)
	root := installed(f, "4.14.0.162")
	compiledFor(f, "4.14")
	code := f.run("--no-backup")
	f.want(code, ExitOK,
		"[ng-deploy] target-version ok (demo-rt.jar: compiled against Niagara 4.14, "+root+" is Niagara 4.14.0.162)\n",
		"[ng-deploy] target-version ok (demo-ux.jar: compiled against Niagara 4.14, "+root+" is Niagara 4.14.0.162)\n",
		"[ng-deploy] copy ok\n")
	if f.read(f.deployedJar("rt")) != f.read(f.builtJar("rt")) {
		t.Errorf("the station does not hold the built rt jar")
	}
}

// A target of another Niagara version is exit 55 with both versions in the line, before the gate and the copy;
// the station keeps the jars it held.
func TestADeployIntoAnInstallationOfAnotherNiagaraVersionIsRefusedBeforeTheCopy(t *testing.T) {
	f := newFixture(t)
	upgrade(t, f) // old jars in the station, a version bump: only the target version can refuse
	root := installed(f, "4.15.6.20")
	compiledFor(f, "4.14")
	code := f.run("--no-backup")
	want := "[ng-deploy] ERROR target-version precheck: demo-rt.jar was compiled against Niagara 4.14 but the target installation " + root +
		" is Niagara 4.15.6.20; nothing was copied: build it against an installation of 4.15 (mirror-niagara-home.sh), or deploy into one of 4.14\n"
	if code != ExitTargetVersion || f.err != warnNoBackup+want {
		t.Errorf("exit %d, stderr %q\nwant exit %d and %q", code, f.err, ExitTargetVersion, want)
	}
	if got := f.read(f.deployedJar("rt")); got != "old-rt" {
		t.Errorf("the station's rt jar is now %q, want it unchanged", got)
	}
	if strings.Contains(f.calls(), "vm.sh") || strings.Contains(f.out, "copy ok") {
		t.Errorf("the gate or the copy ran after the refusal:\n%s", f.calls())
	}
}

// Every jar of the mode is looked at: a -ux jar built for another version is refused although the -rt jar is right.
func TestEveryJarOfTheModeIsCheckedAgainstTheTarget(t *testing.T) {
	f := newFixture(t)
	installed(f, "4.14.0.162")
	f.zipJar(f.builtJar("rt"), moduleXMLOf("4.14"))
	f.zipJar(f.builtJar("ux"), moduleXMLOf("4.13"))
	code := f.run("--no-backup")
	f.want(code, ExitTargetVersion, "err:demo-ux.jar was compiled against Niagara 4.13 but the target installation", "err:is Niagara 4.14.0.162;")
}

// A jar that records no Niagara version, or a target whose baja.jar cannot be read, cannot be compared: exit 55
// and nothing copied, never a pass.
func TestAVersionThatCannotBeReadIsRefusedNotPassed(t *testing.T) {
	f := newFixture(t)
	installed(f, "4.14.0.162")
	compiledFor(f, "")
	code := f.run("--no-backup")
	f.want(code, ExitTargetVersion, "err:target-version precheck cannot verify demo-rt.jar: ", "err:names no Tridium dependency", "err:; nothing was copied\n")
	if f.exists(f.deployedJar("rt")) {
		t.Errorf("a jar was copied after a refusal")
	}

	f = newFixture(t)
	f.write(f.modules+"/baja.jar", "not a jar")
	compiledFor(f, "4.14")
	code = f.run("--no-backup")
	f.want(code, ExitTargetVersion, "err:target-version precheck cannot verify the target installation: ", "err:not a readable jar", "err:; nothing was copied\n")
}

// A station directory that holds no baja.jar is not an installation's modules directory: there is nothing to
// compare, one NOTICE says so and the deploy goes on as before.
func TestAStationDirectoryThatIsNoInstallationIsNotChecked(t *testing.T) {
	f := newFixture(t)
	code := f.run("--no-backup")
	f.want(code, ExitOK, "[ng-deploy] NOTICE: "+f.modules+" holds no baja.jar, so it is not the modules directory of a Niagara installation: the Niagara version of the target is not checked\n", "[ng-deploy] copy ok\n")
}

// --no-deploy copies nothing, so no target is compared.
func TestNoDeployDoesNotCompareTheTarget(t *testing.T) {
	f := newFixture(t)
	installed(f, "4.15.6.20")
	compiledFor(f, "4.14")
	code := f.run("--no-backup", "--no-deploy")
	f.want(code, ExitOK)
	if strings.Contains(f.out, "target-version") || strings.Contains(f.err, "target-version") {
		t.Errorf("--no-deploy looked at the target:\n%s%s", f.out, f.err)
	}
}

// A station directory that is <NIAGARA_HOME>/modules holds the jars the build installed, which were built against
// that very installation: there is nothing to compare, and a mismatch it would otherwise be refused for passes.
func TestAStationThatIsTheHomeModulesDirectoryIsNotComparedWithItself(t *testing.T) {
	f := newFixture(t)
	modules := sameDir(t, f)
	f.zipJar(modules+"/baja.jar", map[string]string{"META-INF/module.xml": `<module name="baja" vendor="Tridium" vendorVersion="4.15.6.20"/>`})
	compiledFor(f, "4.14")
	t.Setenv("FAKE_OLD_VERSION", "1.0")
	t.Setenv("FAKE_NEW_VERSION", "1.1")
	f.want(f.run("--no-backup"), ExitOK, "[ng-deploy] copy ok\n")
	if strings.Contains(f.out, "target-version") || strings.Contains(f.err, "target-version") {
		t.Errorf("the home's own modules directory was compared with itself:\n%s%s", f.out, f.err)
	}
}
