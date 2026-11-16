package gradleenv

import (
	"errors"
	"os"
	"reflect"
	"testing"
)

// mirror is a niagara_home with the marker of a mirror.
func mirror(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	if err := os.WriteFile(home+"/.niagara-mirror", nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return home
}

func lookup(env map[string]string) func(string) string {
	return func(k string) string { return env[k] }
}

func TestAHomeThatIsNotAMirrorIsNotIsolated(t *testing.T) {
	iso, err := Isolate(t.TempDir(), lookup(map[string]string{"HOME": "/h"}))
	if iso != nil || err != nil {
		t.Errorf("got %+v, %v", iso, err)
	}
}

func TestAMirrorMovesUserHomeForEveryJVM(t *testing.T) {
	home := mirror(t)
	for _, tc := range []struct {
		name string
		env  map[string]string
		jto  string
		guh  string
	}{
		{"plain", map[string]string{"HOME": "/h"}, "-Duser.home=" + home + "/.mirror-user-home -Dmaven.repo.local=/h/.m2/repository", "/h/.gradle"},
		{"options kept in front", map[string]string{"HOME": "/h", "JAVA_TOOL_OPTIONS": "-Xmx1g"}, "-Xmx1g -Duser.home=" + home + "/.mirror-user-home -Dmaven.repo.local=/h/.m2/repository", "/h/.gradle"},
		{"maven repository kept", map[string]string{"HOME": "/h", "JAVA_TOOL_OPTIONS": "-Dmaven.repo.local=/m"}, "-Dmaven.repo.local=/m -Duser.home=" + home + "/.mirror-user-home", "/h/.gradle"},
		{"gradle user home kept", map[string]string{"HOME": "/h", "GRADLE_USER_HOME": "/g"}, "-Duser.home=" + home + "/.mirror-user-home -Dmaven.repo.local=/h/.m2/repository", "/g"},
	} {
		iso, err := Isolate(home, lookup(tc.env))
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		want := &Isolation{
			UserHome:   home + "/.mirror-user-home",
			SigningXML: home + "/.mirror-user-home/.tridium/security/niagara.signing.xml",
			Env:        map[string]string{"JAVA_TOOL_OPTIONS": tc.jto, "GRADLE_USER_HOME": tc.guh},
			Args:       []string{"--no-daemon"},
		}
		if !reflect.DeepEqual(iso, want) {
			t.Errorf("%s:\n got %+v\nwant %+v", tc.name, iso, want)
		}
	}
}

// The JVM splits JAVA_TOOL_OPTIONS on whitespace: a value it would carry with whitespace is refused.
func TestWhitespaceInACarriedValueIsRefused(t *testing.T) {
	spaced := mirror(t) + "/a b"
	if err := os.MkdirAll(spaced, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(spaced+"/.niagara-mirror", nil, 0o644)
	var blank *BlankError
	if _, err := Isolate(spaced, lookup(map[string]string{"HOME": "/h"})); !errors.As(err, &blank) || blank.Of != MirrorPath || blank.Path != spaced {
		t.Errorf("mirror path: %v", err)
	}
	if _, err := Isolate(mirror(t), lookup(map[string]string{"HOME": "/h o"})); !errors.As(err, &blank) || blank.Of != HomePath || blank.Path != "/h o" {
		t.Errorf("HOME: %v", err)
	}
	if _, err := Isolate(mirror(t), lookup(map[string]string{"HOME": "/h o", "JAVA_TOOL_OPTIONS": "-Dmaven.repo.local=/m"})); err != nil {
		t.Errorf("HOME is not carried when the maven repository is set: %v", err)
	}
}

func TestPropIsAKeyValuePairAsGradleTakesIt(t *testing.T) {
	for v, want := range map[string]string{"a=b": "-Pa=b", "a=": "-Pa=", "a=b=c": "-Pa=b=c", "=b": "", "": "", "ab": ""} {
		if got, ok := Prop(v); got != want || ok != (want != "") {
			t.Errorf("Prop(%q) = %q, %v", v, got, ok)
		}
	}
}
