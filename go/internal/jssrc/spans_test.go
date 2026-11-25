package jssrc

import (
	"reflect"
	"strings"
	"testing"
)

// TestLineCommentAt: the index of the // that starts a comment on one line, -1 when there is none. Every case
// is a reading the lint rules' former jsCommentAt copies made, plus the regex literal they did not read.
func TestLineCommentAt(t *testing.T) {
	for _, tc := range []struct {
		name, line string
		want       int
	}{
		{"no comment", `var a = 1;`, -1},
		{"comment only", `// lint: x`, 0},
		{"comment after code", `f(a); // lint: x`, 6},
		{"url in a double quote", `u = "http://h/x";`, -1},
		{"url in a single quote", `u = 'http://h/x'; // c`, 18},
		{"comment marker in a double quote", `s = "a // b"; // c`, 14},
		{"escaped quote keeps the string open", `s = "a\" // b"; // c`, 16},
		{"escaped quote single", `s = 'it\'s // b'; // c`, 18},
		{"template hides the marker", "t = `a // b`;", -1},
		{"apostrophe in a template", "t = `it's`; // c", 12},
		{"template with a substitution", "t = `a${x}b // c`; // d", 19},
		{"template substitution nesting", "t = `a${ {k: `b`}.k }c`; // d", 25},
		{"literal open at the line end is read to the end", `s = "abc // d`, -1},
		{"template open at the line end", "t = `abc // d", -1},
		{"inline block comment is skipped", `a /* x // y */ b // c`, 17},
		{"block comment hides the marker", `a /* // x */ b`, -1},
		{"unclosed block comment ends the search", `a /* x // y`, -1},
		{"quote inside a block comment", `a /* ' */ b // c`, 12},
		{"regex literal with a quote", `r = /"x/; // c`, 10},
		{"regex literal with a slash class", `r = /[/"]/; // c`, 12},
		{"division is no comment", `x = a / b; // c`, 11},
		{"empty", ``, -1},
	} {
		if got := LineCommentAt(tc.line); got != tc.want {
			t.Errorf("%s: LineCommentAt(%q) = %d, want %d", tc.name, tc.line, got, tc.want)
		}
	}
}

func TestSpans(t *testing.T) {
	src := "a = 'x//y'; // c\n/* b\n */ t = `${ \"q\" /* i */ }`; r = /\"/g"
	var got []string
	for _, sp := range Spans(src) {
		kind := map[SpanKind]string{Comment: "comment", Literal: "literal"}[sp.Kind]
		got = append(got, kind+":"+src[sp.From:sp.To])
	}
	want := []string{"literal:x//y", "comment:// c", "comment:/* b\n */", "literal:${ \"q\" /* i */ }",
		"literal:q", "comment:/* i */", "literal:\""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Spans:\n got %q\nwant %q", got, want)
	}
}

func TestCommentAt(t *testing.T) {
	src := "a = 'x//y'; // c\n/* b */ d = \"/* e */\""
	for _, tc := range []struct {
		name string
		off  int
		want bool
	}{
		{"code", 0, false},
		{"inside a string", strings.Index(src, "//y"), false},
		{"line comment start", strings.Index(src, "// c"), true},
		{"line comment text", strings.Index(src, " c") + 1, true},
		{"the newline after a line comment", strings.Index(src, "\n"), false},
		{"block comment", strings.Index(src, "b */"), true},
		{"after the block comment", strings.Index(src, " d"), false},
		{"block marker in a string", strings.Index(src, "e */"), false},
		{"out of range", len(src) + 3, false},
		{"negative", -1, false},
	} {
		if got := CommentAt(src, tc.off); got != tc.want {
			t.Errorf("%s: CommentAt(%d) = %v, want %v", tc.name, tc.off, got, tc.want)
		}
	}
}
