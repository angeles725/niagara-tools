package buildmodule

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/gradleenv"
)

// TestMirrorIsolation: a mirror build moves user.home into the mirror for every JVM, keeps gradle's own home
// and the real maven repository, and runs gradle without a daemon; whitespace in a value JAVA_TOOL_OPTIONS
// would carry is refused before gradle.
func TestMirrorIsolation(t *testing.T) {
	cases := []struct {
		name     string
		env      map[string]string
		code     int
		jto, guh string // what gradlew sees; <nh> and <home> stand for the fixture paths
		stderr   string
	}{
		{"a plain mirror", nil, ExitOK, "-Duser.home=<nh>/.mirror-user-home -Dmaven.repo.local=<home>/.m2/repository", "<home>/.gradle", ""},
		{"options already set are kept in front", map[string]string{"JAVA_TOOL_OPTIONS": "-Xmx1g -Duser.home=/real"}, ExitOK,
			"-Xmx1g -Duser.home=/real -Duser.home=<nh>/.mirror-user-home -Dmaven.repo.local=<home>/.m2/repository", "<home>/.gradle", ""},
		{"a maven repository already set is not overridden", map[string]string{"JAVA_TOOL_OPTIONS": "-Dmaven.repo.local=/x/m2"}, ExitOK,
			"-Dmaven.repo.local=/x/m2 -Duser.home=<nh>/.mirror-user-home", "<home>/.gradle", ""},
		{"a GRADLE_USER_HOME already set is kept", map[string]string{"GRADLE_USER_HOME": "/guh"}, ExitOK,
			"-Duser.home=<nh>/.mirror-user-home -Dmaven.repo.local=<home>/.m2/repository", "/guh", ""},
		{"HOME with whitespace", map[string]string{"HOME": "/ho me"}, ExitWhitespace, "", "", "build.sh: HOME contains whitespace: /ho me\n"},
		{"HOME with whitespace and a maven repository set", map[string]string{"HOME": "/ho me", "JAVA_TOOL_OPTIONS": "-Dmaven.repo.local=/x/m2"}, ExitOK,
			"-Dmaven.repo.local=/x/m2 -Duser.home=<nh>/.mirror-user-home", "/ho me/.gradle", ""},
	}
	for _, c := range cases {
		f := newFixture(t)
		f.write(f.home+"/"+mirrorMarker, "")
		for k, v := range c.env {
			f.env[k] = v
		}
		fill := strings.NewReplacer("<nh>", f.home, "<home>", f.env["HOME"]).Replace
		code := f.run(f.root, "Foo", f.home)
		f.wants(c.name, code, c.code, c.stderr)
		if code != ExitOK {
			if f.calls() != "preflight.sh <"+f.home+"> <"+f.root+">\n" {
				t.Errorf("%s: ran %q", c.name, f.calls())
			}
			continue
		}
		wantEnv := "gradlew cwd=" + f.root + " JTO=[" + fill(c.jto) + "] GUH=[" + fill(c.guh) + "] TB=[unset]\n"
		if env := f.read(f.dir + "/env.log"); !strings.Contains(env, wantEnv) {
			t.Errorf("%s: gradlew environment\n%swant line\n%s", c.name, env, wantEnv)
		}
		if !strings.HasSuffix(f.call("gradlew"), " <--no-daemon>") {
			t.Errorf("%s: gradlew call %q lacks --no-daemon", c.name, f.call("gradlew"))
		}
		info := "==> INFO: mirror niagara_home: user.home isolated to " + f.home + "/.mirror-user-home (GRADLE_USER_HOME=" + fill(c.guh) + ", --no-daemon)"
		if !strings.Contains(f.out, info) || !exists(f.home+"/.mirror-user-home") {
			t.Errorf("%s: no isolation row or no user home directory: %q", c.name, f.out)
		}
	}
}

func TestMirrorPathWithWhitespaceIsRefused(t *testing.T) {
	f := newFixture(t)
	home := f.dir + "/n h"
	f.mkdir(home + "/" + m2Repository)
	f.write(home+"/"+mirrorMarker, "")
	f.wants("mirror path with whitespace", f.run(f.root, "Foo", home), ExitWhitespace, "build.sh: mirror niagara_home path contains whitespace: "+home+"\n")
	if f.call("gradlew") != "" || exists(home+"/.mirror-user-home") {
		t.Error("gradle ran or the user home was created")
	}
}

func TestNoIsolationOutsideAMirror(t *testing.T) {
	f := newFixture(t)
	f.env["HOME"] = "/ho me"
	f.wants("a real home", f.build(), ExitOK)
	if strings.Contains(f.call("gradlew"), "--no-daemon") || strings.Contains(f.out, "INFO") {
		t.Errorf("isolation outside a mirror: %q %q", f.call("gradlew"), f.out)
	}
}

// TestDevKeyWarning: a signing profile in the mirror user home means the jar was signed with a throwaway
// key, unless the root's uncommented code turns the default profile off.
func TestDevKeyWarning(t *testing.T) {
	cases := []struct {
		name, root string // root build.gradle.kts ("" none)
		profile    string // "before" the build, created "during" it, or "" never
		warns      bool
	}{
		{"no profile", "", "", false},
		{"created during the build", "", "during", true},
		{"left by an earlier build", "", "before", true},
		{"the root turns the default profile off (.set)", "niagaraSigning {\n  allowDefaultProfile.set( false )\n}\n", "before", false},
		{"the root turns the default profile off (=)", "allowDefaultProfile = false\n", "before", false},
		{"commented out", "// allowDefaultProfile.set(false)\n/*\nallowDefaultProfile = false\n*/\n", "before", true},
		{"set to true", "allowDefaultProfile.set(true)\n", "before", true},
	}
	for _, c := range cases {
		f := newFixture(t)
		f.write(f.home+"/"+mirrorMarker, "")
		profile := f.home + "/" + gradleenv.UserHomeDir + "/" + gradleenv.SigningProfile
		switch c.profile {
		case "before":
			f.write(profile, "")
		case "during":
			f.does("gradlew", `mkdir -p "`+f.home+`/.mirror-user-home/.tridium/security"; : > "`+profile+`"`)
		}
		if c.root != "" {
			f.write(f.root+"/build.gradle.kts", c.root)
		}
		f.wants(c.name, f.run(f.root, "Foo", f.home), ExitOK)
		warning := "==> WARN: the jar is signed with a throwaway dev key created in the mirror user home (" + profile + ")"
		if got := strings.Contains(f.err, warning); got != c.warns {
			t.Errorf("%s: warned = %v, want %v (%q)", c.name, got, c.warns, f.err)
		}
	}
}
