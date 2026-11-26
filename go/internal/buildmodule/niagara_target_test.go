package buildmodule

import (
	"strings"
	"testing"
)

// ownState writes the module's own BUILD-STATE.md in its module root and returns its path.
func (f *fixture) ownState(text string) string {
	f.t.Helper()
	path := f.root + "/BUILD-STATE.md"
	f.write(path, text)
	return path
}

// homeIs makes the fixture's niagara_home an installation of that Niagara version (modules/baja.jar).
func (f *fixture) homeIs(version string) { f.jar(f.home+"/modules/baja.jar", version, "") }

// gateTarget is true when the gate was handed --target-version v.
func (f *fixture) gateTarget(v string) bool {
	return strings.Contains(f.call(gateScript), " <--target-version> <"+v+"> ")
}

// Rule: the record sets the build: niagara_home is that version, --target-version and --plugin-version default
// to it (the gate and gradle get them), and one INFO line says which record did it.
func TestTheRecordedNiagaraTargetSetsTheDefaults(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.homeIs("4.15.6.20")
	path := f.ownState(envelope("Foo", "niagara_target: 4.15.6.20", "plugin_version: 7.6.22"))
	f.wants("recorded target", f.build("--no-drift-check", "--no-report"), ExitOK,
		"build.sh: INFO — "+path+" records niagara_target 4.15.6.20, plugin_version 7.6.22 for Foo: "+
			"building with --target-version 4.15, --plugin-version 7.6.22\n")
	if !f.gateTarget("4.15") || !strings.Contains(f.call("gradlew"), "<-PniagaraPluginVersion=7.6.22>") {
		t.Errorf("gate %q, gradle %q: want --target-version 4.15 and the plugin 7.6.22", f.call(gateScript), f.call("gradlew"))
	}
}

// Rule: the module's own BUILD-STATE.md wins over the kit's file; it is consulted alone when it exists (no
// envelope of the module there records nothing, whatever the kit says); without it the kit's file is read as before;
// --build-state wins over both.
func TestTheModuleRootRecordWinsOverTheKitFile(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.homeIs("4.15.6.20")
	f.state(envelope("Foo", "niagara_target: 4.14"))
	f.ownState(envelope("Foo", "niagara_target: 4.15"))
	f.wants("own file", f.build("--no-drift-check", "--no-report"), ExitOK, "building with --target-version 4.15")

	g := newFixture(t)
	g.signs()
	g.homeIs("4.15.6.20")
	g.state(envelope("Foo", "niagara_target: 4.14"))
	g.ownState(envelope("Bar", "niagara_target: 4.14"))
	g.wants("own file without Foo", g.build("--no-drift-check", "--no-report"), ExitOK)
	if strings.Contains(g.out+g.err, "niagara_target") || g.gateTarget("4.14") {
		t.Errorf("the kit's record was used: %q %q", g.out, g.err)
	}

	k := newFixture(t)
	k.signs()
	k.homeIs("4.14.0.162")
	k.state(envelope("Foo", "niagara_target: 4.14"))
	k.wants("kit file", k.build("--no-drift-check", "--no-report"), ExitOK, "building with --target-version 4.14")

	e := newFixture(t)
	e.signs()
	e.homeIs("4.13.0.1")
	e.ownState(envelope("Foo", "niagara_target: 4.15"))
	named := e.dir + "/named.md"
	e.write(named, envelope("Foo", "niagara_target: 4.13"))
	e.wants("--build-state", e.build("--no-drift-check", "--no-report", "--build-state", named), ExitOK, "building with --target-version 4.13")
}

// Rule: expected_signer is found in the module's own file as well (the lookup is shared, the kit file still works).
func TestTheExpectedSignerIsReadFromTheModuleRootFile(t *testing.T) {
	f := newFixture(t)
	f.state(envelope("Foo", "expected_signer: kitsigner"))
	path := f.ownState(envelope("Foo", "expected_signer: acme"))
	f.wants("own file", f.buildSigned(), ExitOK, "==> expected signer: acme (expected_signer of Foo in "+path+")")
	if v, _ := f.alias(); v != "acme" {
		t.Errorf("alias %q, want acme", v)
	}
}

// Rule: a niagara_home of another major.minor than the record is exit 10 before gradle, naming both versions and the file.
func TestAHomeOfAnotherNiagaraVersionFailsBeforeGradle(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.homeIs("4.14.0.162")
	path := f.ownState(envelope("Foo", "niagara_target: 4.15.6.20"))
	want := "build.sh: FAIL — Foo is recorded for Niagara 4.15.6.20 (niagara_target in " + path + ") but niagara_home " + f.home +
		" is Niagara 4.14.0.162.\n  Build against an installation of 4.15 (mirror it with toolbelt/mirror-niagara-home.sh) or, " +
		"if the station moved, update niagara_target there.\n"
	if code := f.build("--no-drift-check", "--no-report"); code != ExitEnv || f.err != want {
		t.Errorf("exit %d, stderr %q; want exit 10 and %q", code, f.err, want)
	}
	f.wantsNoRun()
}

// Rule: a mirror is also compared with the installation it was made from, when that one can be read.
func TestAMirrorMadeFromAnotherNiagaraVersionFailsBeforeGradle(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.homeIs("4.15.6.20")
	src := realInstall(f, "4.14.0.162")
	mirrorMadeFrom(f, src)
	f.ownState(envelope("Foo", "niagara_target: 4.15.6.20"))
	f.wants("mirror", f.run(f.root, "Foo", f.home), ExitEnv,
		"but the installation "+src+" (the source of mirror "+f.home+") is Niagara 4.14.0.162.")
	f.wantsNoRun()

	g := newFixture(t) // a source that cannot be read does not fail the build
	g.signs()
	g.homeIs("4.15.6.20")
	mirrorMadeFrom(g, g.dir+"/gone")
	g.ownState(envelope("Foo", "niagara_target: 4.15.6.20"))
	g.wants("mirror, source gone", g.run("--no-drift-check", "--no-report", g.root, "Foo", g.home), ExitOK)
}

// Rule: a home whose version cannot be read is a WARN, not a refusal and not silence.
func TestAHomeWithoutAReadableVersionWarns(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.ownState(envelope("Foo", "niagara_target: 4.15.6.20"))
	f.wants("no baja.jar", f.build("--no-drift-check", "--no-report"), ExitOK,
		"build.sh: WARN — niagara_target 4.15.6.20 ("+f.root+"/BUILD-STATE.md) cannot be compared with niagara_home "+f.home+": ")
}

// Rule: --target-version defaults to the record; the same major.minor passes, another is exit 10 before gradle.
func TestATargetVersionFlagMustMatchTheRecord(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.homeIs("4.15.6.20")
	path := f.ownState(envelope("Foo", "niagara_target: 4.15.6.20"))
	f.wants("same line", f.build("--no-drift-check", "--no-report", "--target-version", "4.15.6.20"), ExitOK)
	if !f.gateTarget("4.15.6.20") {
		t.Errorf("gate %q: an explicit equal value is passed as given", f.call(gateScript))
	}
	g := newFixture(t)
	g.signs()
	g.homeIs("4.15.6.20")
	g.ownState(envelope("Foo", "niagara_target: 4.15.6.20"))
	want := "build.sh: FAIL — --target-version 4.14 differs from niagara_target 4.15.6.20 recorded for Foo in " + g.root +
		"/BUILD-STATE.md (want 4.15).\n  Drop --target-version (it defaults to the record) or, if the station moved, " +
		"update niagara_target there.\n"
	if code := g.build("--no-drift-check", "--no-report", "--target-version", "4.14"); code != ExitEnv || g.err != want {
		t.Errorf("exit %d, stderr %q; want exit 10 and %q (%s)", code, g.err, want, path)
	}
	g.wantsNoRun()
}

// Rule: --plugin-version (and $NIAGARA_PLUGIN_VERSION) defaults to plugin_version; another value is exit 10 before gradle.
func TestAPluginVersionMustMatchTheRecord(t *testing.T) {
	for _, c := range []struct {
		name string
		set  func(f *fixture) []string
	}{
		{"flag", func(*fixture) []string { return []string{"--plugin-version", "7.6.17"} }},
		{"environment", func(f *fixture) []string { f.env["NIAGARA_PLUGIN_VERSION"] = "7.6.17"; return nil }},
	} {
		f := newFixture(t)
		f.signs()
		f.homeIs("4.15.6.20")
		f.ownState(envelope("Foo", "plugin_version: 7.6.22"))
		code := f.build(append([]string{"--no-drift-check", "--no-report"}, c.set(f)...)...)
		f.wants(c.name, code, ExitEnv, "build.sh: FAIL — --plugin-version (or NIAGARA_PLUGIN_VERSION) 7.6.17 differs from plugin_version 7.6.22 recorded for Foo in ")
		f.wantsNoRun()
	}
	f := newFixture(t)
	f.signs()
	f.ownState(envelope("Foo", "plugin_version: 7.6.22"))
	f.wants("equal flag", f.build("--no-drift-check", "--no-report", "--plugin-version", "7.6.22"), ExitOK)
}

// Rule: a malformed niagara_target is exit 10 before gradle with the sweep's text and the file; no record builds as before.
func TestAMalformedTargetRecordFailsBeforeGradle(t *testing.T) {
	f := newFixture(t)
	f.signs()
	path := f.ownState(envelope("Foo", "niagara_target: 4"))
	want := "build.sh: FAIL — envelope (Foo) niagara_target invalid: '4' (want 2 to 4 dotted numbers such as 4.15 or 4.15.6.20, or unknown) in " + path + "\n"
	if code := f.build("--no-drift-check", "--no-report"); code != ExitEnv || f.err != want {
		t.Errorf("exit %d, stderr %q; want exit 10 and %q", code, f.err, want)
	}
	f.wantsNoRun()

	g := newFixture(t)
	g.signs()
	g.homeIs("4.14.0.162")
	g.ownState(envelope("Foo", "niagara_target: unknown"))
	g.wants("unknown", g.build("--no-drift-check", "--no-report"), ExitOK)
	if strings.Contains(g.out+g.err, "niagara_target") || g.gateTarget("4.14") || strings.Contains(g.call(gateScript), "--target-version") {
		t.Errorf("an unknown target changed the build: %q %q %q", g.out, g.err, g.call(gateScript))
	}
}
