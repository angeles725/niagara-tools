package buildstatefields

import (
	"strings"
	"testing"
)

// targetEnv is a build-state.v1 envelope of module with the given extra field lines.
func targetEnv(module string, fields ...string) string {
	return "<!-- build-state.v1 -->\nmodule: " + module + "\nretro_required: false\nretro_pending: false\n" +
		strings.Join(append(fields, ""), "\n") + "<!-- /build-state.v1 -->\n"
}

// Rule: both fields are optional; the values of the module's envelope come back, comments and CRLF removed;
// another module's fields, no envelope, no field and `unknown` record nothing.
func TestModuleTargetReadsTheFieldsOfTheModule(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, text string
		want       Target
	}{
		{"both", targetEnv("Foo", "niagara_target: 4.15.6.20", "plugin_version: 7.6.22  # the plugin"), Target{"4.15.6.20", "7.6.22"}},
		{"major.minor only", targetEnv("Foo", "niagara_target: 4.15"), Target{Version: "4.15"}},
		{"another module", targetEnv("Bar", "niagara_target: 4.14") + targetEnv("Foo", "plugin_version: 7.6.17"), Target{Plugin: "7.6.17"}},
		{"unknown", targetEnv("Foo", "niagara_target: unknown", "plugin_version: unknown"), Target{}},
		{"no field", targetEnv("Foo"), Target{}},
		{"no envelope", "", Target{}},
		{"windows file", "\xef\xbb\xbf" + strings.ReplaceAll(targetEnv("Foo", "niagara_target: 4.14.0.162"), "\n", "\r\n"), Target{Version: "4.14.0.162"}},
	} {
		got, err := ModuleTarget(c.text, "Foo")
		if err != nil || got != c.want {
			t.Errorf("%s: got %+v, %v; want %+v", c.name, got, err, c.want)
		}
	}
}

// Rule: a value that is not 2 to 4 dotted numbers (or unknown) is an error naming the module, the field and the value.
func TestAnInvalidTargetOrPluginVersionIsAnError(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ field, value, pattern string }{
		{"niagara_target", "4", TargetPattern}, {"niagara_target", "4.x", TargetPattern},
		{"niagara_target", "v4.15", TargetPattern}, {"niagara_target", "4.15.6.20.1", TargetPattern},
		{"niagara_target", "4.15 6", TargetPattern}, {"niagara_target", "", TargetPattern},
		{"plugin_version", "7", PluginPattern}, {"plugin_version", "7.6.x", PluginPattern},
	} {
		text := targetEnv("Foo", c.field+": "+c.value)
		want := "envelope (Foo) " + c.field + " invalid: '" + c.value + "' (want " + c.pattern + ")"
		if _, err := ModuleTarget(text, "Foo"); err == nil || err.Error() != want {
			t.Errorf("%s %q: error %v, want %q", c.field, c.value, err, want)
		}
		if got := CheckTargets(text); got != want {
			t.Errorf("%s %q: CheckTargets %q, want %q", c.field, c.value, got, want)
		}
	}
}

// Rule: the structural errors are those of expected_signer: the field twice, a mistyped line, two envelopes of
// the module when one sets the field. Another module's envelope is an error of the sweep, not of the module.
func TestTheStructuralErrorsOfTheTargetFields(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ name, text, want string }{
		{"twice", targetEnv("Foo", "niagara_target: 4.14", "niagara_target: 4.15"),
			"envelope (Foo) niagara_target is given twice: '4.14' and '4.15'"},
		{"stray", targetEnv("Foo", "  plugin_version: 7.6.22"),
			"envelope (Foo) has a line that is not the plugin_version field: '  plugin_version: 7.6.22' (write plugin_version: <value> at column 0)"},
		{"colon apart", targetEnv("Foo", "niagara_target : 4.15"),
			"envelope (Foo) has a line that is not the niagara_target field: 'niagara_target : 4.15' (write niagara_target: <value> at column 0)"},
		{"two envelopes", targetEnv("Foo") + targetEnv("Foo", "niagara_target: 4.15"),
			"envelope (Foo) is declared twice and niagara_target is set in one of them: keep one envelope"},
	} {
		if _, err := ModuleTarget(c.text, "Foo"); err == nil || err.Error() != c.want {
			t.Errorf("%s: error %v, want %q", c.name, err, c.want)
		}
		if got := CheckTargets(c.text); got != c.want {
			t.Errorf("%s: CheckTargets %q, want %q", c.name, got, c.want)
		}
	}
	other := targetEnv("Bar", "niagara_target: 4.x") + targetEnv("Foo", "niagara_target: 4.15")
	if got, err := ModuleTarget(other, "Foo"); err != nil || got.Version != "4.15" {
		t.Errorf("another module's bad envelope: %+v, %v", got, err)
	}
	if got := CheckTargets(targetEnv("Foo") + targetEnv("Foo")); got != "" {
		t.Errorf("two envelopes without the field: %q, want none", got)
	}
}

// Rule: the sweep's content check (CheckState) fails a bad target field and leaves expected_signer's messages as they were.
func TestCheckStateCoversTheTargetFieldsAndKeepsTheSignerMessages(t *testing.T) {
	t.Parallel()
	if got := CheckState(targetEnv("Foo", "niagara_target: nope")); !strings.HasPrefix(got, "envelope (Foo) niagara_target invalid: 'nope'") {
		t.Errorf("CheckState: %q", got)
	}
	want := "envelope (Foo) expected_signer invalid: 'a b' (want " + SignerPattern + ")"
	if got := CheckState(targetEnv("Foo", "expected_signer: a b", "niagara_target: nope")); got != want {
		t.Errorf("CheckState: %q, want the signer's message %q first", got, want)
	}
	if got := CheckState(targetEnv("Foo", "expected_signer: acme", "niagara_target: 4.15.6.20", "plugin_version: 7.6.22")); got != "" {
		t.Errorf("a valid envelope: %q", got)
	}
}
