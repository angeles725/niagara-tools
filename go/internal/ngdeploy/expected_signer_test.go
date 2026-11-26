package ngdeploy

import (
	"strings"
	"testing"
)

// envelope is a build-state.v1 envelope of module with the given extra field lines.
func envelope(module string, fields ...string) string {
	return "<!-- build-state.v1 -->\nmodule: " + module + "\nretro_required: false\n" +
		strings.Join(append(fields, ""), "\n") + "<!-- /build-state.v1 -->\n"
}

// kitState writes the kit's BUILD-STATE.src.md where ng-deploy looks for it by default (beside the scripts
// directory, in build-n4-module-kit/) and returns its path.
func (f *fixture) kitState(text string) string {
	f.t.Helper()
	path := f.dir + "/repo/build-n4-module-kit/BUILD-STATE.src.md"
	f.write(path, text)
	return path
}

// gateSigner is the --expected-signer the gate got, "" for none.
func (f *fixture) gateSigner() string {
	_, after, ok := strings.Cut(f.calls(), "vm.sh <--src> <"+f.src+"/demo> <--expected-signer> <")
	if !ok {
		return ""
	}
	signer, _, _ := strings.Cut(after, ">")
	return signer
}

// gradleAlias is every -PniagaraSigningAlias the gradlew calls got.
func (f *fixture) gradleAlias() (aliases []string) {
	for _, field := range strings.Fields(f.calls()) {
		if v, ok := strings.CutPrefix(field, "<-PniagaraSigningAlias="); ok {
			aliases = append(aliases, strings.TrimSuffix(v, ">"))
		}
	}
	return aliases
}

// wantRefused fails the test unless the run was refused as the rules say: exit 10, one ERROR line, nothing
// run (no gradle, no gate, no backup) and nothing copied to the station.
func (f *fixture) wantRefused(code int, stderr string) {
	f.t.Helper()
	if code != ExitEnv || f.out != "" || f.err != stderr || f.calls() != "" || f.exists(f.work+"/_backups") {
		f.t.Errorf("exit %d, stdout %q, stderr %q, calls %q; want exit %d, no stdout, stderr %q, nothing run",
			code, f.out, f.err, f.calls(), ExitEnv, stderr)
	}
	if f.exists(f.deployedJar("rt")) || f.exists(f.deployedJar("ux")) {
		f.t.Errorf("a jar was copied to the station")
	}
}

// Rule: the field is optional. No file, no envelope of the module, no field or `unknown` leave the signer as
// it was: the signer of the jars that were deployed, no alias handed to gradle, no line about a record.
func TestWithoutAnExpectedSignerTheDeployIsUnchanged(t *testing.T) {
	for _, c := range []struct{ name, state string }{
		{"no file", ""},
		{"another module's field only", envelope("other", "expected_signer: zzz")},
		{"an envelope without the field", envelope("demo")},
		{"unknown", envelope("demo", "expected_signer: unknown")},
	} {
		signer, warning := signerGiven(t, func(f *fixture) {
			f.zipJar(withHome(f.t, f, true)+"/demo-rt.jar", signedBy("ACMESIGN"))
			if c.state != "" {
				f.kitState(c.state)
			}
		})
		if signer != "ACMESIGN" || warning != "" {
			t.Errorf("%s: --expected-signer %q, stderr %q; want ACMESIGN and no warning", c.name, signer, warning)
		}
	}
}

// Rule: with no alias configured the record is the expected signer (before the signer of any deployed jar), and
// gradle builds with it, as build.sh hands it over.
func TestTheRecordedSignerIsExpectedWhenNoAliasIsConfigured(t *testing.T) {
	f := newFixture(t)
	f.remove(f.src + "/gradle.properties")
	f.zipJar(withHome(t, f, true)+"/demo-rt.jar", signedBy("DEPLOYED"))
	path := f.kitState(envelope("demo", "expected_signer: acme.sign-1_x"))
	f.want(f.run("--no-backup"), ExitOK, "[ng-deploy] expected signer: acme.sign-1_x (expected_signer of demo in "+path+")\n")
	if got := f.gateSigner(); got != "acme.sign-1_x" {
		t.Errorf("the gate expects %q, want the recorded acme.sign-1_x", got)
	}
	if got := f.gradleAlias(); len(got) == 0 || got[0] != "acme.sign-1_x" {
		t.Errorf("gradle got aliases %q, want the recorded one", got)
	}
}

// Rule: a configured alias equal to the record changes nothing, and says nothing about the record.
func TestAConfiguredAliasEqualToTheRecordIsTheSigner(t *testing.T) {
	f := newFixture(t) // gradle.properties configures acmesigner
	f.kitState(envelope("demo", "expected_signer: acmesigner"))
	f.want(f.run("--no-backup"), ExitOK, "[ng-deploy] gate ok\n")
	if f.gateSigner() != "acmesigner" || len(f.gradleAlias()) != 0 || strings.Contains(f.out, "expected signer") {
		t.Errorf("signer %q, gradle aliases %q, stdout:\n%s", f.gateSigner(), f.gradleAlias(), f.out)
	}
}

// Rule: a configured alias that differs from the record is refused before anything runs, naming both.
func TestAConfiguredAliasThatDiffersFromTheRecordIsRefusedBeforeTheCopy(t *testing.T) {
	f := newFixture(t) // gradle.properties configures acmesigner
	path := f.kitState(envelope("demo", "expected_signer: recorded"))
	f.deployed()
	code := f.run()
	want := "[ng-deploy] ERROR niagaraSigningAlias=acmesigner is configured but the build-state envelope of demo " +
		"records expected_signer: recorded (" + path + ").\n" +
		"  Deploy with that alias (--gradle-prop niagaraSigningAlias=recorded) or, if the signer changed, " +
		"update expected_signer there.\n"
	if code != ExitEnv || f.out != "" || f.err != want || f.calls() != "" || f.read(f.deployedJar("rt")) != "old-rt" {
		t.Errorf("exit %d, stdout %q, stderr %q, calls %q; want exit %d and stderr %q", code, f.out, f.err, f.calls(), ExitEnv, want)
	}
}

// Rule: a malformed record of the module is refused before anything runs, naming the file, whatever alias is
// configured. The four kinds are the ones build.sh and the sweep refuse.
func TestAMalformedRecordIsRefusedBeforeTheCopy(t *testing.T) {
	for _, c := range []struct{ name, state, reason string }{
		{"two envelopes of the module, one with the field", envelope("demo", "expected_signer: a") + envelope("demo"),
			"envelope (demo) is declared twice and expected_signer is set in one of them: keep one envelope"},
		{"the key twice", envelope("demo", "expected_signer: a", "expected_signer: b"),
			"envelope (demo) expected_signer is given twice: 'a' and 'b'"},
		{"a mistyped key line", envelope("demo", "expected_signer : a"),
			"envelope (demo) has a line that is not the expected_signer field: 'expected_signer : a' " +
				"(write expected_signer: <alias> at column 0)"},
		{"a value that is not an alias", envelope("demo", "expected_signer: a b"),
			"envelope (demo) expected_signer invalid: 'a b' " +
				"(want a signing alias of letters, digits, '.', '_' or '-', or unknown)"},
	} {
		f := newFixture(t) // an alias is configured: the refusal does not depend on it
		path := f.kitState(c.state)
		f.wantRefused(f.run(), "[ng-deploy] ERROR "+c.reason+" in "+path+"\n")
		if t.Failed() {
			t.Logf("case: %s", c.name)
		}
	}
}

// Rule: --build-state <file> names the file instead of the kit's; one that cannot be read is an error, while a
// default file that is missing is silent (the first test).
func TestBuildStateNamesTheFileAndAMissingOneIsRefused(t *testing.T) {
	f := newFixture(t)
	f.remove(f.src + "/gradle.properties")
	f.kitState(envelope("demo", "expected_signer: fromkit"))
	own := f.dir + "/own-state.md"
	f.write(own, envelope("demo", "expected_signer: fromfile"))
	f.want(f.run("--no-backup", "--build-state", own), ExitOK, "(expected_signer of demo in "+own+")\n")
	if f.gateSigner() != "fromfile" {
		t.Errorf("the gate expects %q, want fromfile", f.gateSigner())
	}

	f = newFixture(t)
	f.kitState(envelope("demo", "expected_signer: acmesigner"))
	missing := f.dir + "/absent.md"
	f.wantRefused(f.run("--build-state", missing),
		"[ng-deploy] ERROR --build-state "+missing+" cannot be read (No such file or directory)\n")
}

// Rule: an empty --build-state is a usage error, not the default file; one with no value is refused like any
// flag that lacks its argument.
func TestAnEmptyBuildStateIsAUsageError(t *testing.T) {
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"--build-state", ""}, "[ng-deploy] ERROR --build-state requires a file name, not an empty value\n"},
		{[]string{"--build-state"}, "[ng-deploy] ERROR --build-state requires an argument\n"},
	} {
		f := newFixture(t)
		f.wantRefused(f.run(c.args...), c.want)
	}
}
