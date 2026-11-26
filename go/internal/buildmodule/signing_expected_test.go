package buildmodule

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/buildstatefields"
)

// envelope is a build-state.v1 envelope of module with the given extra field lines.
func envelope(module string, fields ...string) string {
	return "<!-- build-state.v1 -->\nmodule: " + module + "\nretro_required: false\nretro_pending: false\n" +
		strings.Join(append(fields, ""), "\n") + "<!-- /build-state.v1 -->\n"
}

// state writes the kit's BUILD-STATE.src.md (beside the toolbelt directory, where build.sh looks) and returns its path.
func (f *fixture) state(text string) string {
	f.t.Helper()
	path := f.dir + "/BUILD-STATE.src.md"
	f.write(path, text)
	return path
}

// alias is the -PniagaraSigningAlias gradle was handed, "" when none; count is how many it got.
func (f *fixture) alias() (value string, count int) {
	for _, field := range strings.Fields(f.call("gradlew")) {
		if v, ok := strings.CutPrefix(field, "<-PniagaraSigningAlias="); ok {
			value, count = strings.TrimSuffix(v, ">"), count+1
		}
	}
	return value, count
}

// buildSigned runs the chain of the fixture module "Foo" with the options a signing test needs.
func (f *fixture) buildSigned(flags ...string) int {
	f.t.Helper()
	return f.build(append([]string{"--no-drift-check", "--no-report"}, flags...)...)
}

// wantsNoRun fails the test when gradle or the gate ran.
func (f *fixture) wantsNoRun() {
	f.t.Helper()
	if f.call("gradlew") != "" || f.call(gateScript) != "" {
		f.t.Errorf("gradle or the gate ran:\n%s", f.calls())
	}
}

const todayNoSigner = "build.sh: WARN — no niagaraSigningAlias is configured"

// Rule: the field is optional. No BUILD-STATE file, a file with no envelope of the module, an envelope without
// the field or with `unknown` all build as before: no alias handed to gradle, the no-signer WARN, no gate option.
func TestWithoutAnExpectedSignerTheBuildIsUnchanged(t *testing.T) {
	for _, c := range []struct{ name, state string }{
		{"no file", ""},
		{"another module's field only", envelope("Bar", "expected_signer: zzz")},
		{"an envelope without the field", envelope("Foo")},
		{"unknown", envelope("Foo", "expected_signer: unknown")},
	} {
		f := newFixture(t)
		if c.state != "" {
			f.state(c.state)
		}
		code := f.buildSigned()
		f.wants(c.name, code, ExitOK, todayNoSigner)
		if v, n := f.alias(); n != 0 || strings.Contains(f.call(gateScript), "--expected-signer") ||
			strings.Contains(f.out, "expected signer") {
			t.Errorf("%s: alias %q (%d), gate %q, stdout %q", c.name, v, n, f.call(gateScript), f.out)
		}
	}
}

// Rule: set and no alias configured: gradle is handed it, the gate expects it, one stdout line says where it came from.
func TestTheRecordedSignerIsUsedWhenNoAliasIsConfigured(t *testing.T) {
	f := newFixture(t)
	path := f.state(envelope("Bar", "expected_signer: zzz") + envelope("Foo", "expected_signer: acme   # release signer"))
	f.wants("recorded signer", f.buildSigned(), ExitOK,
		"==> expected signer: acme (expected_signer of Foo in "+path+")")
	if v, n := f.alias(); v != "acme" || n != 1 {
		t.Errorf("gradle got alias %q %d times, want acme once:\n%s", v, n, f.calls())
	}
	if !strings.Contains(f.call(gateScript), " <--expected-signer> <acme> ") || strings.Contains(f.err, todayNoSigner) {
		t.Errorf("gate %q, stderr %q", f.call(gateScript), f.err)
	}
}

// Rule: the record is read from a file saved with a byte-order mark and CRLF lines too.
func TestTheRecordIsReadFromAWindowsEditedFile(t *testing.T) {
	f := newFixture(t)
	f.state("\xef\xbb\xbf" + strings.ReplaceAll(envelope("Foo", "expected_signer: acme"), "\n", "\r\n"))
	f.wants("crlf", f.buildSigned(), ExitOK, "expected signer: acme")
	if v, _ := f.alias(); v != "acme" {
		t.Errorf("alias %q, want acme", v)
	}
}

// Rule: a configured alias (any of the three sources) equal to the record builds as before: handed once, no extra line.
func TestAnAliasEqualToTheRecordPasses(t *testing.T) {
	for _, c := range []struct {
		name  string
		setup func(f *fixture) []string
	}{
		{"--gradle-prop", func(*fixture) []string { return []string{"--gradle-prop", "niagaraSigningAlias=acme"} }},
		{"environment", func(f *fixture) []string { f.env[gradleEnvPrefix+aliasProperty] = "acme"; return nil }},
		{"gradle.properties", func(f *fixture) []string {
			f.write(f.root+"/gradle.properties", "niagaraSigningAlias=acme\n")
			return nil
		}},
	} {
		f := newFixture(t)
		f.state(envelope("Foo", "expected_signer: acme"))
		f.wants(c.name, f.buildSigned(c.setup(f)...), ExitOK)
		if _, n := f.alias(); n > 1 || strings.Contains(f.out, "expected signer") || strings.Contains(f.err, "FAIL") {
			t.Errorf("%s: alias handed %d times, stdout %q, stderr %q", c.name, n, f.out, f.err)
		}
	}
}

// Rule: a configured alias that differs from the record ends the chain before gradle (exit 10), naming both.
func TestAnAliasDifferentFromTheRecordFailsBeforeGradle(t *testing.T) {
	for _, c := range []struct {
		name  string
		setup func(f *fixture) []string
	}{
		{"--gradle-prop", func(*fixture) []string { return []string{"--gradle-prop", "niagaraSigningAlias=other"} }},
		{"environment", func(f *fixture) []string { f.env[gradleEnvPrefix+aliasProperty] = "other"; return nil }},
		{"gradle.properties", func(f *fixture) []string {
			f.write(f.root+"/gradle.properties", "niagaraSigningAlias=other\n")
			return nil
		}},
	} {
		f := newFixture(t)
		path := f.state(envelope("Foo", "expected_signer: acme"))
		code := f.buildSigned(c.setup(f)...)
		f.wants(c.name, code, ExitEnv,
			"build.sh: FAIL — niagaraSigningAlias=other is configured but the build-state envelope of Foo records "+
				"expected_signer: acme ("+path+").\n"+
				"  Build with that alias (--gradle-prop niagaraSigningAlias=acme) or, if the signer changed, "+
				"update expected_signer there.\n")
		f.wantsNoRun()
	}
}

// Rule: a value that is not an alias fails before gradle (exit 10); the same text the sweep prints, with the file.
func TestAnInvalidRecordedSignerFailsBeforeGradle(t *testing.T) {
	f := newFixture(t)
	path := f.state(envelope("Foo", "expected_signer: ac me"))
	want := "build.sh: FAIL — envelope (Foo) expected_signer invalid: 'ac me' (want " +
		buildstatefields.SignerPattern + ") in " + path + "\n"
	if code := f.buildSigned(); code != ExitEnv || f.err != want {
		t.Errorf("exit %d, stderr %q; want exit 10 and %q", code, f.err, want)
	}
	f.wantsNoRun()
}

// Rule: --build-state <file> names the file to read instead of the kit's; one that cannot be read is exit 10.
func TestBuildStateOptionNamesTheFile(t *testing.T) {
	f := newFixture(t)
	f.state(envelope("Foo", "expected_signer: kitsigner"))
	own := f.dir + "/own-state.md"
	f.write(own, envelope("Foo", "expected_signer: acme"))
	f.wants("--build-state", f.buildSigned("--build-state", own), ExitOK, "expected_signer of Foo in "+own)
	if v, _ := f.alias(); v != "acme" {
		t.Errorf("alias %q, want acme (the kit's file must not be read)", v)
	}
	g := newFixture(t)
	missing := g.dir + "/nothing.md"
	code := g.buildSigned("--build-state", missing)
	g.wants("missing file", code, ExitEnv, "build.sh: FAIL — --build-state "+missing+" cannot be read (")
	g.wantsNoRun()
}

// Rule: the recorded alias is the configured one for every later step: a root that applies the signing plugin
// without the niagaraSigning block is refused as for a passed alias (exit 10, before gradle).
func TestTheRecordedSignerReachesTheSigningBlockCheck(t *testing.T) {
	f := newFixture(t)
	f.state(envelope("Foo", "expected_signer: acme"))
	f.write(f.root+"/build.gradle.kts", wizardRoot)
	f.wants("no block", f.buildSigned(), ExitEnv, "niagaraSigningAlias=acme is passed but ")
	f.wantsNoRun()
}
