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
		{"", []string{"--plugin-version", "7.6.2", nh, gr}, "FAIL  plugin-pin    plugin 7.6.2 (from --plugin-version) missing from " + nh + "/etc/m2 — this install ships 7.6.22: pick one for --plugin-version"},
		{"9.9.9", []string{"--plugin-version", "", nh, gr}, "FAIL  plugin-pin    plugin 7.6.17 missing from " + nh + "/etc/m2 — this install ships 7.6.22: pass --plugin-version 7.6.22 (build.sh, preflight.sh, ng-deploy.sh) or pin it in settings.gradle.kts"},
		{"", []string{nh, d + "/gr2"}, "FAIL  plugin-pin    settings.gradle.kts not found in " + d + "/gr2"},
		{"", []string{nh, d + "/gr3"}, "FAIL  plugin-pin    cannot determine plugin version from " + d + "/gr3/settings.gradle.kts"},
	} {
		env := map[string]string{"NIAGARA_PLUGIN_VERSION": c.env}
		if _, out, _ := pf((&fakeHost{}).host(), env, append([]string{"--jvm-dir", d}, c.args...)...); !strings.Contains(out, c.want+"\n") {
			t.Errorf("%q NIAGARA_PLUGIN_VERSION=%q:\n%s\nwant %q", c.args, c.env, out, c.want)
		}
	}
}

// Retro build-kit-bench-session Δ7: the FAIL lists every plugin version the installation ships (a directory named
// X.Y.Z whose parent's name holds "plugin") and the option that passes one; with none it keeps its old advice.
func TestPluginPinNamesTheVersionsTheInstallShips(t *testing.T) {
	d := t.TempDir()
	mk(t, d, "nh/etc/m2/repository/com/x/plugin/7.6.22/", "nh/etc/m2/repository/com/x/plugin/7.3.40/", "nh/etc/m2/repository/com/x/lib/9.9.9/",
		"nh/etc/m2/repository/com/x/plugin/1.2/", "gr/settings.gradle.kts=val v = \"7.6.17\"\n", "nh0/etc/m2/", "gr0/settings.gradle.kts=val v = \"7.6.17\"\n")
	want := "FAIL  plugin-pin    plugin 7.6.17 missing from " + d + "/nh/etc/m2 — this install ships 7.3.40, 7.6.22: pass --plugin-version <one of them> " +
		"(build.sh, preflight.sh, ng-deploy.sh) or pin it in settings.gradle.kts\n"
	if _, out, _ := pf((&fakeHost{}).host(), nil, "--jvm-dir", d, d+"/nh", d+"/gr"); !strings.Contains(out, want) {
		t.Errorf("rows\n%s\nwant %q", out, want)
	}
	none := "FAIL  plugin-pin    plugin 7.6.17 missing from " + d + "/nh0/etc/m2 — copy from Niagara install\n"
	if _, out, _ := pf((&fakeHost{}).host(), nil, "--jvm-dir", d, d+"/nh0", d+"/gr0"); !strings.Contains(out, none) {
		t.Errorf("rows\n%s\nwant %q", out, none)
	}
}
