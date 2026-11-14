package scaffoldmodule

import (
	"path/filepath"
	"testing"
)

// usage and name errors print to stderr only, exit 2 and create nothing under the out-dir.
func TestUsageAndNameErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string // "OUT" is replaced by the out-dir
		err  string
	}{
		{"no arguments", nil, usageWant},
		{"one argument", []string{"Foo"}, usageWant},
		{"--vendor without value", []string{"Foo", "OUT", "--vendor"}, usageWant},
		{"--target-version without value", []string{"Foo", "OUT", "--target-version"}, usageWant},
		{"--plugin-version without value", []string{"Foo", "OUT", "--plugin-version"}, usageWant},
		{"--type without value", []string{"Foo", "OUT", "--type"}, usageWant},
		{"unknown option", []string{"Foo", "OUT", "--bogus"}, "scaffold-module: unknown option: --bogus\n" + usageWant},
		{"third positional", []string{"Foo", "OUT", "extra"}, "scaffold-module: unknown option: extra\n" + usageWant},
		{"unknown type", []string{"Foo", "OUT", "--type", "zzz"},
			"scaffold-module: unknown --type: zzz (must be logic, dashboard or driver)\n" + usageWant},
		{"empty type", []string{"Foo", "OUT", "--type", ""},
			"scaffold-module: unknown --type:  (must be logic, dashboard or driver)\n" + usageWant},
		{"type is checked before the name", []string{"1bad", "OUT", "--type", "zzz"},
			"scaffold-module: unknown --type: zzz (must be logic, dashboard or driver)\n" + usageWant},
		{"digit first", []string{"1bad", "OUT"}, "scaffold-module: ModuleName must start with an uppercase letter: 1bad\n"},
		{"lowercase first", []string{"coldRoom", "OUT"}, "scaffold-module: ModuleName must start with an uppercase letter: coldRoom\n"},
		{"empty name", []string{"", "OUT"}, "scaffold-module: ModuleName must start with an uppercase letter: \n"},
		{"non-ASCII first", []string{"Éclair", "OUT"}, "scaffold-module: ModuleName must start with an uppercase letter: Éclair\n"},
		{"dash", []string{"Cold-Room", "OUT"}, "scaffold-module: ModuleName must contain only alphanumeric characters: Cold-Room\n"},
		{"underscore", []string{"Cold_Room", "OUT"}, "scaffold-module: ModuleName must contain only alphanumeric characters: Cold_Room\n"},
		{"slash", []string{"Cold/Room", "OUT"}, "scaffold-module: ModuleName must contain only alphanumeric characters: Cold/Room\n"},
		{"non-ASCII inside", []string{"Cañon", "OUT"}, "scaffold-module: ModuleName must contain only alphanumeric characters: Cañon\n"},
	}
	tb := newKit(t)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "out")
			args := append([]string(nil), c.args...)
			for i, a := range args {
				if a == "OUT" {
					args[i] = out
				}
			}
			r := run(t, tb, args...)
			if r.code != 2 || r.out != "" || r.err != c.err {
				t.Errorf("code=%d out=%q\n err %q\nwant %q", r.code, r.out, r.err, c.err)
			}
			if got := tree(t, out); len(got) != 0 {
				t.Errorf("a refused run wrote %v", keys(got))
			}
		})
	}
}

// The last value of a repeated option wins, and an option value may look like an option.
func TestOptionValues(t *testing.T) {
	tb := newKit(t)
	out := t.TempDir()
	r := run(t, tb, "Foo", out, "--type", "driver", "--vendor", "--type", "--type", "logic")
	if r.code != 0 {
		t.Fatalf("code=%d err=%q", r.code, r.err)
	}
	got := tree(t, filepath.Join(out, "Foo"))
	wantFile(t, got, "Foo/Foo-rt/src/com/--type/Foo/BFoo.java",
		"package com.--type.Foo;\n// Copyright --type. Foo-rt fooTimer f=a f:b\n")
}

// Without the launcher's variable the fixtures cannot be found: env fault, never a guess.
func TestToolbeltUnset(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out")
	r := run(t, "", "Foo", out)
	want := "scaffold-module: SCAFFOLD_MODULE_TOOLBELT is unset (run through toolbelt/scaffold-module.sh)\n"
	if r.code != 3 || r.out != "" || r.err != want {
		t.Errorf("code=%d out=%q err=%q", r.code, r.out, r.err)
	}
}
