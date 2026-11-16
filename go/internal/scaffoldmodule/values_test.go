package scaffoldmodule

import (
	"path/filepath"
	"testing"
)

// Every option whose value is written into the emitted tree inserts it as typed: no character of a value
// has a second meaning (`&` is `&`, `\1` is `\1`, `|` is `|`). --vendor is narrowed first by its own rule
// (one Java package segment), so its literal case is a value that rule accepts.
func TestOptionValueIsLiteral(t *testing.T) {
	const odd = `A&B\1|C\n\&\`
	cases := []struct{ option, value, file, want string }{
		{"--vendor", "A_b9", "Foo/Foo-rt/src/com/a_b9/Foo/BFoo.java", "package com.a_b9.Foo;\n// Copyright A_b9. Foo-rt fooTimer f=a f:b\n"},
		{"--target-version", odd, "settings.gradle.kts", "plugin 7.6.17 target " + odd + " root Foo\nno final newline 7x6y17 4x14"},
		{"--plugin-version", odd, "settings.gradle.kts", "plugin " + odd + " target 4.14 root Foo\nno final newline 7x6y17 4x14"},
	}
	tb := newKit(t)
	for _, c := range cases {
		out := t.TempDir()
		r := run(t, tb, "Foo", out, c.option, c.value)
		if r.code != 0 || r.err != "" {
			t.Fatalf("%s %q: code=%d err=%q", c.option, c.value, r.code, r.err)
		}
		wantFile(t, tree(t, filepath.Join(out, "Foo")), c.file, c.want)
	}
}

// A newline in a substituted value is refused by the option's name before anything is written.
func TestOptionValueNewlineRefused(t *testing.T) {
	tb := newKit(t)
	for _, c := range []struct{ option, value string }{
		{"--target-version", "4.14\n4.15"}, {"--plugin-version", "7.6.17\n"},
	} {
		out := filepath.Join(t.TempDir(), "out")
		r := run(t, tb, "Foo", out, c.option, c.value)
		want := "scaffold-module: " + c.option + " value must be a single line (it holds a newline): " + c.value + "\n"
		if r.code != 2 || r.out != "" || r.err != want {
			t.Errorf("%s %q: code=%d out=%q err=%q", c.option, c.value, r.code, r.out, r.err)
		}
		if got := tree(t, out); len(got) != 0 {
			t.Errorf("%s %q wrote %v", c.option, c.value, keys(got))
		}
	}
}

// A value is written once and never read again: a value (or a ModuleName) that holds a token a later rule
// replaces stays as typed.
func TestInsertedTextIsNotRewritten(t *testing.T) {
	tb := newKit(t)
	out := t.TempDir()
	r := run(t, tb, "AngelesBox", out, "--vendor", "Acme", "--plugin-version", "4.14", "--target-version", "4.15")
	if r.code != 0 || r.err != "" {
		t.Fatalf("code=%d err=%q", r.code, r.err)
	}
	got := tree(t, filepath.Join(out, "AngelesBox"))
	wantFile(t, got, "settings.gradle.kts", "plugin 4.14 target 4.15 root AngelesBox\nno final newline 7x6y17 4x14")
	wantFile(t, got, "AngelesBox/AngelesBox-rt/src/com/acme/AngelesBox/BAngelesBox.java",
		"package com.acme.AngelesBox;\n// Copyright Acme. AngelesBox-rt angelesBoxTimer ab=a ab:b\n")
}
