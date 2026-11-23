package modjar

import (
	"os"
	"testing"
)

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A niagara_home is a mirror only when the marker is a regular file at its root.
func TestIsMirrorNeedsTheMarkerFile(t *testing.T) {
	home := t.TempDir()
	if IsMirror(home) || IsMirror(home+"/absent") {
		t.Error("a home without the marker reads as a mirror")
	}
	if err := os.Mkdir(home+"/"+MirrorMarker, 0o755); err != nil {
		t.Fatal(err)
	}
	if IsMirror(home) {
		t.Error("a directory under the marker's name reads as a mirror")
	}
	os.Remove(home + "/" + MirrorMarker)
	write(t, home+"/"+MirrorMarker, "")
	if !IsMirror(home) {
		t.Error("a home with the marker does not read as a mirror")
	}
}

// The alias is read in gradle's precedence, and the expected signer is that alias, else the one signer of
// the deployed jars, never the development signer.
func TestExpectedSignerPrecedence(t *testing.T) {
	root, userHome := t.TempDir(), t.TempDir()
	vars := map[string]string{"GRADLE_USER_HOME": userHome}
	env := func(name string) string { return vars[name] }
	if got := ConfiguredAlias(nil, env, root); got != "" {
		t.Errorf("no alias anywhere: %q", got)
	}
	write(t, root+"/"+GradleProperties, AliasProperty+"=fromroot\n")
	if got := ConfiguredAlias(nil, env, root); got != "fromroot" {
		t.Errorf("gradle root: %q", got)
	}
	write(t, userHome+"/"+GradleProperties, AliasProperty+"=fromuserhome\n")
	if got := ConfiguredAlias(nil, env, root); got != "fromuserhome" {
		t.Errorf("gradle user home: %q", got)
	}
	vars[GradleEnvPrefix+AliasProperty] = "fromenv"
	if got := ConfiguredAlias(nil, env, root); got != "fromenv" {
		t.Errorf("environment: %q", got)
	}
	if got := ConfiguredAlias([]string{"-P" + AliasProperty + "=a", "-Pother=x", "-P" + AliasProperty + "=last"}, env, root); got != "last" {
		t.Errorf("gradle properties of the command line: %q", got)
	}
	for _, c := range []struct {
		alias    string
		deployed [][]string
		signer   string
		why      Reason
	}{
		{"acme", [][]string{{"OTHER"}}, "acme", Known},
		{"", [][]string{{"ACMESIGN"}, nil, {"ACMESIGN"}}, "ACMESIGN", Known},
		{"", nil, "", NoSigner},
		{"", [][]string{{"ACMESIGN"}, {"OTHER"}}, "", NoSigner},
		{"", [][]string{{"ACMESIGN", "OTHER"}}, "", NoSigner},
		{"", [][]string{{"niagara4"}}, "", DevDeployed},
	} {
		if signer, why := ExpectedSigner(c.alias, c.deployed); signer != c.signer || why != c.why {
			t.Errorf("ExpectedSigner(%q, %v) = %q, %v; want %q, %v", c.alias, c.deployed, signer, why, c.signer, c.why)
		}
	}
}

// A mirror's marker records the installation it was made from (mirror-niagara-home.sh writes "source=<path>");
// a marker without that line, an empty one and a home that is no mirror name none.
func TestMirrorSourceIsTheInstallationTheMarkerNames(t *testing.T) {
	for name, tc := range map[string]struct{ marker, want string }{
		"source and date":       {"source=/mnt/c/Niagara/Niagara-4.14.0.162\ncreated=2026-11-23T10:00:00\n", "/mnt/c/Niagara/Niagara-4.14.0.162"},
		"a path with a space":   {"source=/mnt/c/Program Files/N 4\ncreated=x\n", "/mnt/c/Program Files/N 4"},
		"an empty marker":       {"", ""},
		"no source line":        {"created=2026-11-23T10:00:00\n", ""},
		"an empty source value": {"source=\ncreated=x\n", ""},
	} {
		home := t.TempDir()
		write(t, home+"/"+MirrorMarker, tc.marker)
		if got := MirrorSource(home); got != tc.want {
			t.Errorf("%s: MirrorSource = %q, want %q", name, got, tc.want)
		}
	}
	if got := MirrorSource(t.TempDir()); got != "" {
		t.Errorf("a home with no marker: MirrorSource = %q, want none", got)
	}
}
