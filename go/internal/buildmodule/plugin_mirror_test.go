package buildmodule

import (
	"strings"
	"testing"
)

// TestPluginPickedFromTheMirror: a pin the niagara_home's etc/m2 does not hold is replaced by the one plugin
// version it does ship (a NOTICE names both, and preflight and gradle get the version used); with several or
// none the build ends with exit 10 before preflight, naming the pin and the versions found. A pin that is
// there, an override and no pin at all leave the run as it was. [ev: retro client-e-dashboard-live-check Δ29]
func TestPluginPickedFromTheMirror(t *testing.T) {
	const m2 = "/etc/m2/repository/com/tridium/"
	cases := []struct {
		name    string
		pin     string   // niagaraPluginVersion of gradle.properties ("" none)
		ships   []string // plugin versions the home ships (the fixture's 7.6.17 is replaced by these)
		flags   []string
		code    int
		plugin  string // the -PniagaraPluginVersion gradle gets
		stderr  []string
		notWant string // text stderr must not hold
	}{
		{"the pin is shipped", "7.6.17", []string{"7.6.17"}, nil, ExitOK, "7.6.17", nil, "NOTICE"},
		{"one other version is used", "7.6.17", []string{"7.6.22"}, nil, ExitOK, "7.6.22",
			[]string{"build.sh: NOTICE — niagaraPluginVersion=7.6.17 is not under ", "etc/m2/repository; using 7.6.22, the plugin version it ships"}, ""},
		{"several versions: refused", "7.6.17", []string{"7.3.40", "7.6.22"}, nil, ExitEnv, "",
			[]string{"build.sh: niagaraPluginVersion=7.6.17 is not under ", "it ships 7.3.40, 7.6.22", "--plugin-version"}, ""},
		{"none shipped: refused", "7.6.17", nil, nil, ExitEnv, "",
			[]string{"build.sh: niagaraPluginVersion=7.6.17 is not under ", "it ships no plugin version"}, ""},
		{"an override is kept", "7.6.17", []string{"7.6.22"}, []string{"--plugin-version", "7.3.40"}, ExitOK, "7.3.40", nil, "NOTICE"},
		{"no pin", "", []string{"7.6.22"}, nil, ExitOK, "", nil, "NOTICE"},
	}
	for _, c := range cases {
		f := newFixture(t)
		f.signs()
		home := f.dir + "/pickhome"
		f.mkdir(home + "/modules")
		f.mkdir(home + "/etc/m2/repository")
		for _, v := range c.ships {
			f.mkdir(home + m2 + "plugin/" + v)
		}
		if c.pin != "" {
			f.write(f.root+"/gradle.properties", "niagaraPluginVersion="+c.pin+"\n")
		}
		code := f.run(append(append([]string{"--install-into-home"}, c.flags...), f.root, "Foo", home)...)
		f.wants(c.name, code, c.code, c.stderr...)
		if c.notWant != "" && strings.Contains(f.err, c.notWant) {
			t.Errorf("%s: stderr holds %q: %s", c.name, c.notWant, f.err)
		}
		if got := argValue(f.call("gradlew"), "-PniagaraPluginVersion="); got != c.plugin {
			t.Errorf("%s: gradle got plugin %q, want %q", c.name, got, c.plugin)
		}
		if c.code != ExitOK {
			if f.calls() != "" {
				t.Errorf("%s: a tool ran before the refusal: %s", c.name, f.calls())
			}
			continue
		}
		if c.name == "one other version is used" && !strings.Contains(f.call("preflight.sh"), "<--plugin-version> <7.6.22>") {
			t.Errorf("%s: preflight was not given the version used: %s", c.name, f.call("preflight.sh"))
		}
	}
}
