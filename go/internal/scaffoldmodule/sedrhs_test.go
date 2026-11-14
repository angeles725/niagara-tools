package scaffoldmodule

import (
	"path/filepath"
	"testing"
)

// Option values are sed replacement text, as in the bash tool: & is the replaced token, a backslash escapes.
func TestOptionValueIsSedReplacement(t *testing.T) {
	tb := newKit(t)
	cases := []struct{ vendor, path, text string }{
		{`A&B`, "aangelesb", "AAngelesB"},
		{`A\&B`, "a&b", "A&B"},
		{`A\\B`, `a\b`, `A\B`},
		{`A\|B`, "a|b", "A|B"},
		{"A\\nB", "a\nb", "A\nB"},
		{"A\\\nB", "a\nb", "A\nB"},
	}
	for _, c := range cases {
		out := t.TempDir()
		r := run(t, tb, "Foo", out, "--vendor", c.vendor)
		if r.code != 0 || r.err != "" {
			t.Fatalf("%q: code=%d err=%q", c.vendor, r.code, r.err)
		}
		wantFile(t, tree(t, filepath.Join(out, "Foo")), "Foo/Foo-rt/src/com/"+c.path+"/Foo/BFoo.java",
			"package com."+c.path+".Foo;\n// Copyright "+c.text+". Foo-rt fooTimer f=a f:b\n")
	}
	out := t.TempDir()
	if r := run(t, tb, "Foo", out, "--target-version", "<&>", "--plugin-version", "[&]"); r.code != 0 {
		t.Fatalf("code=%d err=%q", r.code, r.err)
	}
	wantFile(t, tree(t, filepath.Join(out, "Foo")), "settings.gradle.kts",
		"plugin [7.6.17] target <4.14> root Foo\nno final newline 7x6y17 4x14")
}

// A value sed would reject (the bash tool then printed sed errors, left a broken tree and exited 0) is
// refused before anything is written.
func TestOptionValueSedRejects(t *testing.T) {
	tb := newKit(t)
	for _, c := range []struct{ opt, value string }{
		{"--vendor", "A|B"}, {"--vendor", "A\nB"}, {"--vendor", `A\1`}, {"--vendor", `A\`}, {"--vendor", `A\UB`}, {"--vendor", `A\x41`},
		{"--vendor", `A\DB`}, // usable as typed, not once lowercased for the package path
		{"--target-version", "4|15"}, {"--plugin-version", `7\`},
	} {
		out := filepath.Join(t.TempDir(), "out")
		r := run(t, tb, "Foo", out, c.opt, c.value)
		want := "scaffold-module: " + c.opt + " value cannot be substituted (unescaped | or newline, trailing \\ or an unsupported \\ escape): " + c.value + "\n"
		if r.code != 2 || r.out != "" || r.err != want {
			t.Errorf("%s %q: code=%d out=%q err=%q", c.opt, c.value, r.code, r.out, r.err)
		}
		if got := tree(t, out); len(got) != 0 {
			t.Errorf("%s %q wrote %v", c.opt, c.value, keys(got))
		}
	}
}
