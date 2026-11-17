package scaffoldmodule

import "testing"

// Without --plugin-version, the Niagara gradle plugin follows --target-version: a 4.15 scaffold pinned to the
// 4.14 plugin (7.6.17) failed preflight plugin-pin on a 4.15 niagara_home, which ships 7.6.22. An explicit
// --plugin-version still wins, and a release outside the table keeps the default.
func TestThePluginVersionFollowsTheTargetVersion(t *testing.T) {
	for _, tc := range []struct {
		extra []string
		want  string
	}{
		{nil, "7.6.17"},
		{[]string{"--target-version", "4.14"}, "7.6.17"},
		{[]string{"--target-version", "4.15"}, "7.6.22"},
		{[]string{"--target-version", "4.13"}, "7.3.40"},
		{[]string{"--target-version", "4.15", "--plugin-version", "7.6.20"}, "7.6.20"},
		{[]string{"--plugin-version", "7.6.20", "--target-version", "4.15"}, "7.6.20"},
		{[]string{"--target-version", "4.99"}, "7.6.17"},
	} {
		o, f := parseArgs(append([]string{"Foo", "out", "--vendor", "Acme"}, tc.extra...))
		if f != nil {
			t.Fatalf("%v: %v", tc.extra, f)
		}
		if o.pluginVersion != tc.want {
			t.Errorf("%v: plugin %s, want %s", tc.extra, o.pluginVersion, tc.want)
		}
	}
}
