package buildmodule

import (
	"strings"
	"testing"
)

// TestSlowMountWarning: a gradle root or niagara_home on a 9p/drvfs mount warns and the build goes on.
func TestSlowMountWarning(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		mounts func(f *fixture) string
		want   func(f *fixture) []string // the labels warned about, in order
	}{
		{"the gradle root", func(f *fixture) string { return "C: " + f.root + " 9p rw\n" }, func(f *fixture) []string { return []string{"gradle-root (" + f.root + ") is on a 9p mount"} }},
		{"niagara_home", func(f *fixture) string { return "C: " + f.home + " drvfs rw\n" }, func(f *fixture) []string { return []string{"niagara_home (" + f.home + ") is on a drvfs mount"} }},
		{"both", func(f *fixture) string { return "C: " + f.dir + " 9p rw\n" }, func(f *fixture) []string {
			return []string{"gradle-root (" + f.root + ") is on a 9p mount", "niagara_home (" + f.home + ") is on a 9p mount"}
		}},
		{"neither", func(f *fixture) string { return "rootfs / ext4 rw\nC: /mnt/c 9p rw\n" }, func(f *fixture) []string { return nil }},
	}
	for _, c := range cases {
		f := newFixture(t)
		f.write(f.dir+"/mounts", c.mounts(f))
		f.wants(c.name, f.build(), ExitOK)
		var got []string
		for _, line := range strings.Split(f.err, "\n") {
			if rest, ok := strings.CutPrefix(line, "build.sh: WARN — "); ok {
				got = append(got, strings.TrimSuffix(rest, " (WSL /mnt/<drive>) — builds run far slower than ext4."))
			}
		}
		if want := c.want(f); strings.Join(got, "|") != strings.Join(want, "|") {
			t.Errorf("%s: warned %q, want %q", c.name, got, want)
		}
	}
}

// TestPreflight: it runs first with the plugin override; any failure of it is exit 10 before gradle.
func TestPreflight(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, exit string
		flags      []string
		code       int
		stderr     string
	}{
		{"passes", "0", nil, ExitOK, ""},
		{"FAILs", "1", nil, ExitEnv, "build.sh: preflight FAILed — fix the environment (or --no-preflight to skip)\n"},
		{"usage error: no FAILed line", "2", nil, ExitEnv, ""},
		{"skipped", "1", []string{"--no-preflight"}, ExitOK, ""},
	}
	for _, c := range cases {
		f := newFixture(t)
		f.does(preflightScript, "exit "+c.exit)
		if code := f.build(c.flags...); code != c.code || f.err != c.stderr {
			t.Errorf("%s: exit %d stderr %q, want %d %q", c.name, code, f.err, c.code, c.stderr)
		}
		if ran := f.call("gradlew") != ""; ran != (c.code == ExitOK) {
			t.Errorf("%s: gradlew ran = %v", c.name, ran)
		}
		if skipped := f.call(preflightScript) == ""; skipped != (len(c.flags) > 0) {
			t.Errorf("%s: preflight skipped = %v", c.name, skipped)
		}
	}
	f := newFixture(t)
	f.env["NIAGARA_PLUGIN_VERSION"] = "7.6.17"
	f.wants("plugin from the environment", f.build(), ExitOK)
	if want := "preflight.sh <--plugin-version> <7.6.17> <" + f.home + "> <" + f.root + ">"; f.call(preflightScript) != want {
		t.Errorf("preflight call %q, want %q", f.call(preflightScript), want)
	}
	if !strings.HasPrefix(f.out, "==> preflight\n") {
		t.Errorf("stdout does not start with the preflight row: %q", f.out)
	}
}
