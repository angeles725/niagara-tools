package scaffoldmodule

import (
	"path/filepath"
	"testing"
)

// The scaffold never guesses the vendor: a run without --vendor exits 2 naming the option and writes
// nothing; the other refusals keep their order (--type and the ModuleName are read first).
func TestVendorIsRequired(t *testing.T) {
	tb := newKit(t)
	cases := []struct {
		name string
		args []string
		err  string
	}{
		{"no --vendor", []string{"Foo", "OUT"}, vendorRequiredWant + usageWant},
		{"no --vendor with other options", []string{"Foo", "OUT", "--type", "driver", "--target-version", "4.15"}, vendorRequiredWant + usageWant},
		{"type is checked before the vendor", []string{"Foo", "OUT", "--type", "zzz"},
			"scaffold-module: unknown --type: zzz (must be logic, dashboard or driver)\n" + usageWant},
		{"name is checked before the vendor", []string{"1bad", "OUT"}, "scaffold-module: ModuleName must start with an uppercase letter: 1bad\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "out")
			r := run(t, tb, withOut(c.args, out)...)
			if r.code != 2 || r.out != "" || r.err != c.err {
				t.Errorf("code=%d out=%q\n err %q\nwant %q", r.code, r.out, r.err, c.err)
			}
			if got := tree(t, out); len(got) != 0 {
				t.Errorf("a refused run wrote %v", keys(got))
			}
		})
	}
}

// --target-version is <major>.<minor> and --plugin-version <major>.<minor>.<patch>, digits only; any other
// value is refused by the option's name before anything is written.
func TestVersionShapes(t *testing.T) {
	tb := newKit(t)
	refused := []struct{ option, value, want string }{
		{"--target-version", "4", targetShapeWant},
		{"--target-version", "4.15.1", targetShapeWant},
		{"--target-version", "v4.15", targetShapeWant},
		{"--target-version", "4.14\n4.15", targetShapeWant},
		{"--target-version", "--type", targetShapeWant},
		{"--target-version", "", targetShapeWant},
		{"--plugin-version", "banana", pluginShapeWant},
		{"--plugin-version", "7.6", pluginShapeWant},
		{"--plugin-version", "4.14-p", pluginShapeWant},
		{"--plugin-version", "7.6.17\n", pluginShapeWant},
	}
	for _, c := range refused {
		out := filepath.Join(t.TempDir(), "out")
		r := run(t, tb, "Foo", out, "--vendor", "Acme", c.option, c.value)
		if want := c.want + c.value + "\n"; r.code != 2 || r.out != "" || r.err != want {
			t.Errorf("%s %q: code=%d out=%q err=%q want %q", c.option, c.value, r.code, r.out, r.err, want)
		}
		if got := tree(t, out); len(got) != 0 {
			t.Errorf("%s %q wrote %v", c.option, c.value, keys(got))
		}
	}
	for _, v := range [][2]string{{"4.15", "7.6.22"}, {"10.0", "12.34.567"}} {
		out := t.TempDir()
		if r := run(t, tb, "Foo", out, "--vendor", "Acme", "--target-version", v[0], "--plugin-version", v[1]); r.code != 0 || r.err != "" {
			t.Errorf("%v: code=%d err=%q", v, r.code, r.err)
		}
	}
}

const (
	vendorRequiredWant = "scaffold-module: --vendor is required: the module's vendor, also its Java package segment once lowercased (e.g. --vendor Acme)\n"
	targetShapeWant    = "scaffold-module: --target-version must be a Niagara release <major>.<minor>, digits only (e.g. 4.15): "
	pluginShapeWant    = "scaffold-module: --plugin-version must be a plugin version <major>.<minor>.<patch>, digits only (e.g. 7.6.22): "
)

// withOut is args with every "OUT" replaced by out.
func withOut(args []string, out string) []string {
	args = append([]string(nil), args...)
	for i, a := range args {
		if a == "OUT" {
			args[i] = out
		}
	}
	return args
}
