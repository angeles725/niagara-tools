package buildmodule

import (
	"testing"
)

func TestUsageAndFlagErrors(t *testing.T) {
	const usageOnStderr = true
	cases := []struct {
		name   string
		args   []string
		stderr string // the first stderr line
		usage  bool   // the usage follows on stderr
	}{
		{"no arguments", nil, "", usageOnStderr},
		{"one positional", []string{"root"}, "", usageOnStderr},
		{"four positionals", []string{"root", "Foo", "nh", "extra"}, "", usageOnStderr},
		{"a value flag without its value", []string{"--profiles"}, "", usageOnStderr},
		{"--target-version without its value", []string{"--legacy", "--target-version"}, "", usageOnStderr},
		{"--plugin-version without its value", []string{"--plugin-version"}, "", usageOnStderr},
		{"--ui-profile without its value", []string{"--ui-profile"}, "", usageOnStderr},
		{"an unknown flag", []string{"--bogus", "root", "Foo"}, "build.sh: unknown flag --bogus", usageOnStderr},
		{"a lone dash is a flag", []string{"-", "root", "Foo"}, "build.sh: unknown flag -", usageOnStderr},
		{"an unknown --ui-profile", []string{"--ui-profile", "kiosk", "root", "Foo"}, "build.sh: --ui-profile must be hmi|lan|both|unknown (got: kiosk)", false},
		{"--gradle-prop without a value", []string{"--gradle-prop"}, "build.sh: --gradle-prop needs k=v (got: nothing)", false},
		{"--gradle-prop with an empty value", []string{"--gradle-prop", "", "root", "Foo"}, "build.sh: --gradle-prop needs k=v (got: nothing)", false},
		{"--gradle-prop without =", []string{"--gradle-prop", "novalue", "root", "Foo"}, "build.sh: --gradle-prop needs k=v (got: novalue)", false},
		{"--gradle-prop with an empty key", []string{"--gradle-prop", "=v", "root", "Foo"}, "build.sh: --gradle-prop needs k=v (got: =v)", false},
		{"the first error wins over --help", []string{"--bogus", "--help"}, "build.sh: unknown flag --bogus", usageOnStderr},
	}
	for _, c := range cases {
		f := newFixture(t)
		want := ""
		if c.stderr != "" {
			want = c.stderr + "\n"
		}
		if c.usage {
			want += fixtureUsage
		}
		if code := f.run(c.args...); code != ExitUsage || f.err != want || f.out != "" {
			t.Errorf("%s: exit %d stdout %q stderr %q, want exit %d and stderr %q", c.name, code, f.out, f.err, ExitUsage, want)
		}
		if f.calls() != "" {
			t.Errorf("%s: ran %q", c.name, f.calls())
		}
	}
}

func TestHelpPrintsTheUsageOnStdout(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"-h"}, {"--legacy", "--profiles", "rt", "--help", "root"}} {
		f := newFixture(t)
		if code := f.run(args...); code != ExitOK || f.out != fixtureUsage || f.err != "" {
			t.Errorf("%v: exit %d stdout %q stderr %q", args, code, f.out, f.err)
		}
	}
}

// TestFlagsReachTheCommands: every flag ends up in the arguments of the command it is for.
func TestFlagsReachTheCommands(t *testing.T) {
	f := newFixture(t)
	code := f.build("--profiles", "rt,ux", "--target-version", "4.13", "--plugin-version", "7.6.17", "--ui-profile", "hmi",
		"--legacy", "--first-deploy", "--gradle-prop", "a=b", "--gradle-prop", "k=", "--werror")
	f.wants("all flags", code, ExitOK)
	want := "preflight.sh <--plugin-version> <7.6.17> <" + f.home + "> <" + f.root + ">\n" +
		"gradlew <:Foo-rt:clean> <:Foo-rt:slotomatic> <:Foo-rt:jar> <:Foo-ux:clean> <:Foo-ux:slotomatic> <:Foo-ux:jar>" +
		" <-Pniagara_home=" + f.home + "> <-Porg.gradle.java.installations.paths=" + f.dir + "/j8> <-PniagaraPluginVersion=7.6.17>" +
		" <-Pa=b> <-Pk=> <--init-script> <" + f.kit + "/lib/werror.init.gradle>\n" +
		"verify-module.sh <--src> <" + f.root + "/Foo> <--target-version> <4.13> <" + f.built("rt") + "> <" + f.built("ux") + ">\n" +
		"report-module.sh <" + f.root + "/Foo> <--target-version> <4.13> <--profile> <hmi> <--legacy> <--first-deploy> <--niagara-home> <" + f.home + ">\n"
	if f.calls() != want {
		t.Errorf("calls:\n%s\nwant:\n%s", f.calls(), want)
	}
}

func TestFlagsAfterThePositionalsArePositionals(t *testing.T) {
	f := newFixture(t)
	f.wants("flag after the positionals", f.run(f.root, "Foo", "--legacy"), ExitEnv, "build.sh: not a niagara_home (no etc/m2/repository): --legacy")
}
