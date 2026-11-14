package buildmodule

import (
	"os"
	"strings"
	"testing"
)

// TestPluginVersion: the override wins; else the module's own pin is detected and announced; the version
// gradle will use is checked against etc/m2 (a warning only).
func TestPluginVersion(t *testing.T) {
	const notInM2 = "not found under"
	cases := []struct {
		name                 string
		properties, settings string // gradle.properties and settings.gradle.kts of the module root ("" none)
		flags                []string
		plugin               string // the -PniagaraPluginVersion gradle gets ("" none)
		detected, warned     bool
	}{
		{"nothing pinned", "", "", nil, "", false, false},
		{"gradle.properties", "# c\n  niagaraPluginVersion = 7.6.22  \r\nniagaraPluginVersion=1.0\n", "", nil, "7.6.22", true, false},
		{"what follows the last = of the line", "niagaraPluginVersion=7.6.17 # was a=7.3.40\n", "", nil, "7.3.40", true, false},
		{"the settings default", "", `val niagaraPluginVersion: String = props.getOrElse("7.6.17")` + "\n", nil, "7.6.17", true, false},
		{"the gradleProperty form", "", `val v = gradleProperty("niagaraPluginVersion").getOrElse("7.6.17")` + "\n", nil, "7.6.17", true, false},
		{"the providers.gradleProperty form", "", `  val gradlePluginVersion: String = providers.gradleProperty("niagaraPluginVersion").getOrElse("7.6.17")` + "\n", nil, "7.6.17", true, false},
		{"the getOrElse argument, not a number in the name", "", `val niagaraPluginVersion10x = props.getOrElse("7.6.17")` + "\n", nil, "7.6.17", true, false},
		{"an empty property falls back to settings", "niagaraPluginVersion=\n", `val niagaraPluginVersion = props.getOrElse("7.3.40")` + "\n", nil, "7.3.40", true, false},
		{"a one-digit default is not a version", "", `val niagaraPluginVersion = props.getOrElse("7")` + "\n", nil, "", false, false},
		{"the override wins over the pin", "niagaraPluginVersion=7.6.22\n", "", []string{"--plugin-version", "7.6.17"}, "7.6.17", false, false},
		{"gradlePluginVersion absent from m2", "", `val gradlePluginVersion: String = "7.3.40"` + "\n", nil, "", false, true},
		{"gradlePluginVersion present in m2", "", `val gradlePluginVersion : String="7.6.17"` + "\n", nil, "", false, false},
		{"an override absent from m2", "", `val gradlePluginVersion: String = "7.6.17"` + "\n", []string{"--plugin-version", "9.9.9"}, "9.9.9", false, true},
		{"no settings file: nothing to check", "", "", []string{"--plugin-version", "9.9.9"}, "9.9.9", false, false},
	}
	for _, c := range cases {
		f := newFixture(t)
		if c.properties != "" {
			f.write(f.root+"/gradle.properties", c.properties)
		}
		if c.settings != "" {
			f.write(f.root+"/settings.gradle.kts", c.settings)
		}
		f.wants(c.name, f.build(c.flags...), ExitOK)
		if got := argValue(f.call("gradlew"), "-PniagaraPluginVersion="); got != c.plugin {
			t.Errorf("%s: gradle got plugin %q, want %q", c.name, got, c.plugin)
		}
		announce := "==> detected niagaraPluginVersion=" + c.plugin + " from the module's gradle config\n"
		if got := strings.Contains(f.out, "==> detected"); got != c.detected || (c.detected && !strings.Contains(f.out, announce)) {
			t.Errorf("%s: detection announced = %v, want %v (%q)", c.name, got, c.detected, f.out)
		}
		if got := strings.Contains(f.err, notInM2); got != c.warned {
			t.Errorf("%s: m2 warning = %v, want %v (%q)", c.name, got, c.warned, f.err)
		}
	}
}

// TestPluginVersionScaffoldSettings: the settings.gradle.kts the scaffold emits (the kit fixtures) pins its
// plugin through providers.gradleProperty("niagaraPluginVersion").getOrElse(default); that default is detected.
func TestPluginVersionScaffoldSettings(t *testing.T) {
	for _, module := range []string{"MinimalPan", "MinimalDriver", "MinimalDash"} {
		settings, err := os.ReadFile("../../../build-n4-module-kit/fixtures/" + module + "/settings.gradle.kts")
		if err != nil {
			t.Fatal(err)
		}
		f := newFixture(t)
		f.write(f.root+"/settings.gradle.kts", string(settings))
		f.wants(module, f.build(), ExitOK)
		if got := argValue(f.call("gradlew"), "-PniagaraPluginVersion="); got != "7.6.17" {
			t.Errorf("%s: gradle got plugin %q, want the scaffold default 7.6.17", module, got)
		}
	}
}

// argValue is the text after prefix in the first <prefix...> argument of a logged call, "" when absent.
func argValue(call, prefix string) string {
	_, rest, ok := strings.Cut(call, "<"+prefix)
	if !ok {
		return ""
	}
	value, _, _ := strings.Cut(rest, ">")
	return value
}

// TestPluginFilesGrepCannotRead: the rows GNU grep printed for the module's gradle files are kept — a file
// that cannot be read, and a match in a file holding a NUL byte — and neither yields a version.
func TestPluginFilesGrepCannotRead(t *testing.T) {
	const pin = "niagaraPluginVersion=7.6.22\n"
	f := newFixture(t)
	f.write(f.root+"/gradle.properties", pin+"\x00\n")
	f.write(f.root+"/settings.gradle.kts", `val niagaraPluginVersion = props.getOrElse("7.6.17")`+"\n\x00\n")
	f.wants("binary files", f.build("--no-preflight"), ExitOK,
		"grep: "+f.root+"/gradle.properties: binary file matches\ngrep: "+f.root+"/settings.gradle.kts: binary file matches\n")
	if strings.Contains(f.call("gradlew"), "-PniagaraPluginVersion") {
		t.Errorf("a version was read from a binary file: %s", f.call("gradlew"))
	}

	if os.Geteuid() == 0 {
		t.Skip("running as root: every file is readable")
	}
	f = newFixture(t)
	f.write(f.root+"/gradle.properties", pin)
	if err := os.Chmod(f.root+"/gradle.properties", 0); err != nil {
		t.Fatal(err)
	}
	f.wants("unreadable file", f.build("--no-preflight"), ExitOK, "grep: "+f.root+"/gradle.properties: Permission denied\n")
}
