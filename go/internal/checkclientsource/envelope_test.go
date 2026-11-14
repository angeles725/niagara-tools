package checkclientsource

import "testing"

// TestSourceOfTruth pins the envelope read: the first source_of_truth line after the `module:` line
// naming the module, inside an envelope, with its `#` comment removed.
func TestSourceOfTruth(t *testing.T) {
	t.Parallel()
	const open, shut = "<!-- build-state.v1 -->\n", "<!-- /build-state.v1 -->\n"
	cases := []struct{ name, text, module, want string }{
		{"the module's field", open + "module: Mod\nsource_of_truth: r@main@abc\n" + shut, "Mod", "r@main@abc"},
		{"another module's field", open + "module: Other\nsource_of_truth: r@main@abc\n" + shut, "Mod", ""},
		{"a comment and the blanks before it go", open + "module: Mod  # the module\nsource_of_truth:\tr@main@abc \t# DECLARED\n" + shut, "Mod", "r@main@abc"},
		{"blanks with no comment stay", open + "module: Mod\nsource_of_truth: unknown  \n" + shut, "Mod", "unknown  "},
		{"outside an envelope is not read", "module: Mod\nsource_of_truth: r@main@abc\n", "Mod", ""},
		{"after the close is not read", open + "module: Mod\n" + shut + "source_of_truth: r@main@abc\n", "Mod", ""},
		{"the field before its module line is not read", open + "source_of_truth: x@y@z\nmodule: Mod\n" + shut, "Mod", ""},
		{"the first match wins", open + "module: Mod\nsource_of_truth: a@b@c\nsource_of_truth: d@e@f\n" + shut, "Mod", "a@b@c"},
		{"an empty first field ends the read", open + "module: Mod\nsource_of_truth:\nsource_of_truth: d@e@f\n" + shut, "Mod", ""},
		{"an indented key is not a field", open + "module: Mod\n  source_of_truth: a@b@c\n" + shut, "Mod", ""},
		{"no final newline", open + "module: Mod\nsource_of_truth: a@b@c", "Mod", "a@b@c"},
		{"a second envelope", open + "module: A\n" + shut + open + "module: Mod\nsource_of_truth: a@b@c\n" + shut, "Mod", "a@b@c"},
	}
	for _, c := range cases {
		if got := SourceOfTruth(c.text, c.module); got != c.want {
			t.Errorf("%s: SourceOfTruth = %q, want %q", c.name, got, c.want)
		}
	}
}
