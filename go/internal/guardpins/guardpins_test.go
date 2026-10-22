package guardpins

import (
	"reflect"
	"strings"
	"testing"
)

func TestDeclarations(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, text string
		want       []Decl
	}{
		{"T4-match: one declaration", "#!/usr/bin/env bash\n# lint-foo\n# Mutation: FOO1 -- drops the guard\n", []Decl{{3, "FOO1"}}},
		{"T4-nomutation: none", "#!/usr/bin/env bash\n# lint-baz\n", nil},
		{"grammar: ASCII --, id starts with a letter, column 0, text after --", "#!/bin/sh\n# lint-a\n# Mutation: A -- x\n# Mutation: B -- y\n" +
			"#  Mutation: C -- indented\n# Mutation: D --\n# Mutation: E - single\n# Mutation: 9F -- digit start\n" +
			"# Mutation: G_h-1 -- ok\n# Mutation: A -- repeat\n# Mutation: H — em dash\n",
			[]Decl{{3, "A"}, {4, "B"}, {9, "G_h-1"}, {10, "A"}}},
		{"line 60 is read, line 61 is not", "#!/bin/sh\n" + strings.Repeat("# f\n", 58) + "# Mutation: ON60 -- x\n# Mutation: L61 -- y\n", []Decl{{60, "ON60"}}},
		{"CRLF line", "#!/bin/sh\r\n# Mutation: A -- crlf\r\n", []Decl{{2, "A"}}},
		{"an unterminated last line is read", "#!/bin/sh\n# Mutation: A -- no newline", []Decl{{2, "A"}}},
		{"empty file", "", nil},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Declarations(tt.text); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFixtures(t *testing.T) {
	t.Parallel()
	files := []BatsFile{
		{Path: "k/tests/b.bats", Text: "@test \"A:first file\"\n@test \"Z x\"\r\n"},
		{Path: "k/tests/z.bats", Text: "@test \"A-x: no\"\n@test \"AB: no\"\n  @test \"A: indented\"\n@test \"A x\"\n@test \"A: second\"\n@test \"Q\"\n@test \"R:\" {\n"},
	}
	idx := IndexFixtures(files)
	cases := []struct {
		id   string
		want Hit
		ok   bool
	}{
		{"A", Hit{"k/tests/b.bats", 1}, true}, // first file in order wins
		{"A-x", Hit{"k/tests/z.bats", 1}, true},
		{"AB", Hit{"k/tests/z.bats", 2}, true},
		{"Z", Hit{"k/tests/b.bats", 2}, true},
		{"R", Hit{"k/tests/z.bats", 7}, true},
		{"Q", Hit{}, false}, // `@test "Q"` has no ':' or ' ' after the id
		{"A-", Hit{}, false},
		{"X", Hit{}, false},
	}
	for _, tt := range cases {
		got, ok := idx[tt.id]
		if ok != tt.ok || got != tt.want {
			t.Errorf("fixture %q = (%v, %v), want (%v, %v)", tt.id, got, ok, tt.want, tt.ok)
		}
	}
}
