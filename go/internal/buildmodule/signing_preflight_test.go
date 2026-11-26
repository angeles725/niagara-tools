package buildmodule

import (
	"path/filepath"
	"strings"
	"testing"
)

const (
	wizardRoot = "plugins {\n  id(\"com.tridium.niagara-signing\")\n}\n"
	blockRoot  = wizardRoot + "niagaraSigning {\n  aliases.set(listOf(\"x\"))\n}\n"
)

// rootRun is a build of the fixture with the given root build.gradle.kts and flags; it reports whether
// gradle ran.
func rootRun(t *testing.T, root string, flags ...string) (f *fixture, code int, gradleRan bool) {
	t.Helper()
	f = newFixture(t)
	f.write(f.root+"/build.gradle.kts", root)
	f.signs() // a configured alias, so the no-signer WARN stays out; --gradle-prop still wins
	code = f.build(append([]string{"--no-drift-check", "--no-report"}, flags...)...)
	return f, code, f.call("gradlew") != ""
}

// Alias passed + a root that applies the plugin without the block: exit 10 before gradle, with the command
// that adds the template (#1121).
func TestAnAliasWithoutTheSigningBlockFailsBeforeGradle(t *testing.T) {
	f, code, ran := rootRun(t, wizardRoot, "--gradle-prop", "niagaraSigningAlias=acme")
	f.wants("alias without the block", code, ExitEnv,
		"build.sh: FAIL — niagaraSigningAlias=acme is passed but "+f.root+"/build.gradle.kts has no niagaraSigning {} block",
		"cat '"+filepath.Join(f.kit, "../templates/signing-root.gradle.kts")+"' >> '"+f.root+"/build.gradle.kts'", "NIAGARA4")
	if ran || f.call(gateScript) != "" {
		t.Errorf("gradle or the gate ran:\n%s", f.calls())
	}
}

// The alias reaches gradle by the environment or gradle.properties too, and --no-preflight does not skip the
// check; a block only in a comment or a string is no block.
func TestTheCheckSeesEveryWayTheAliasIsSet(t *testing.T) {
	f, code, ran := rootRun(t, wizardRoot+"// niagaraSigning {\nval s = \"niagaraSigning {\"\n", "--no-preflight")
	f.wants("environment alias", code, ExitEnv, "niagaraSigningAlias=acmesigner is passed")
	if ran {
		t.Error("gradle ran")
	}
	f = newFixture(t)
	f.write(f.root+"/build.gradle.kts", wizardRoot)
	f.write(f.root+"/gradle.properties", "niagaraSigningAlias=fromprops\n")
	f.wants("gradle.properties alias", f.build(), ExitEnv, "niagaraSigningAlias=fromprops is passed")
}

// Unchanged: the block present, no alias, a root that does not apply the plugin, no root file.
func TestTheSigningCheckLeavesOtherBuildsAlone(t *testing.T) {
	for _, c := range []struct{ name, root, alias string }{
		{"block present", blockRoot, "acme"},
		{"no alias", wizardRoot, ""},
		{"plugin not applied", "plugins {\n  id(\"java\")\n}\n", "acme"},
		{"unscannable root", "val s = \"unterminated\n", "acme"},
	} {
		f := newFixture(t)
		f.write(f.root+"/build.gradle.kts", c.root)
		args := []string{"--no-drift-check", "--no-report"}
		if c.alias != "" {
			args = append(args, "--gradle-prop", "niagaraSigningAlias="+c.alias)
		}
		f.wants(c.name, f.build(args...), ExitOK)
		if f.call("gradlew") == "" || strings.Contains(f.err, "niagaraSigning {}") {
			t.Errorf("%s: gradle did not run or the check spoke:\n%s", c.name, f.err)
		}
	}
	f := newFixture(t)
	f.wants("no root build file", f.build("--no-drift-check", "--no-report", "--gradle-prop", "niagaraSigningAlias=acme"), ExitOK)
}
