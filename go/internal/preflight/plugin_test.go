package preflight

import (
	"strings"
	"testing"
)

func TestPluginPin(t *testing.T) {
	d := t.TempDir()
	m2 := "nh/etc/m2/repository/com/x/plugin/"
	mk(t, d, m2+"7.6.22/", "gr/settings.gradle.kts=id(\"a\") version \"1.2\"\nval v = \"7.6.17\" + \"7.6.22\"\n", "gr2/", "gr3/settings.gradle.kts=val v = \"7.6\"\n")
	nh, gr := d+"/nh", d+"/gr"
	for _, c := range []struct {
		env  string
		args []string
		want string
	}{
		{"", []string{"--plugin-version", "7.6.22", nh, gr}, "PASS  plugin-pin    plugin 7.6.22 (from --plugin-version) found in niagara_home/etc/m2"},
		{"7.6.22", []string{nh, gr}, "PASS  plugin-pin    plugin 7.6.22 (from --plugin-version) found in niagara_home/etc/m2"},
		{"9.9.9", []string{"--plugin-version", "7.6.22", nh, gr}, "PASS  plugin-pin    plugin 7.6.22 (from --plugin-version) found in niagara_home/etc/m2"},
		{"", []string{"--plugin-version", "7.6.2", nh, gr}, "FAIL  plugin-pin    plugin 7.6.2 (from --plugin-version) missing from " + nh + "/etc/m2 — pick the version this install ships"},
		{"9.9.9", []string{"--plugin-version", "", nh, gr}, "FAIL  plugin-pin    plugin 7.6.17 missing from " + nh + "/etc/m2 — copy from Niagara install"},
		{"", []string{nh, d + "/gr2"}, "FAIL  plugin-pin    settings.gradle.kts not found in " + d + "/gr2"},
		{"", []string{nh, d + "/gr3"}, "FAIL  plugin-pin    cannot determine plugin version from " + d + "/gr3/settings.gradle.kts"},
	} {
		env := map[string]string{"NIAGARA_PLUGIN_VERSION": c.env}
		if _, out, _ := pf((&fakeHost{}).host(), env, append([]string{"--jvm-dir", d}, c.args...)...); !strings.Contains(out, c.want+"\n") {
			t.Errorf("%q NIAGARA_PLUGIN_VERSION=%q:\n%s\nwant %q", c.args, c.env, out, c.want)
		}
	}
}
