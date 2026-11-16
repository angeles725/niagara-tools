package rowpath

import "testing"

// TestDisplay: a path with no control byte, double quote or backslash is printed as it is (non-ASCII
// included); any other is one quoted line: the letter escapes git uses, a backslash and a double quote
// escaped, and every other control byte (DEL included) as a backslash and three octal digits.
func TestDisplay(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, path, want string }{
		{"plain", "Group/build.gradle.kts", "Group/build.gradle.kts"},
		{"non-ASCII", "Cámaras/a b.txt", "Cámaras/a b.txt"},
		{"a backslash alone", `a\b`, `"a\\b"`},
		{"a double quote alone", `a"b`, `"a\"b"`},
		{"DEL", "a\x7fb", `"a\177b"`},
		{"a control byte with no letter escape", "a\x01b", `"a\001b"`},
		{"an escape byte, three digits", "a\x1bb", `"a\033b"`},
		{"the letter escapes", "\a\b\t\n\v\f\r", `"\a\b\t\n\v\f\r"`},
		{"NUL", "a\x00", `"a\000"`},
		{"the last printable byte is no control byte", "a~b", "a~b"},
		{"a space is no control byte", "a b", "a b"},
	}
	for _, c := range cases {
		if got := Display(c.path); got != c.want {
			t.Errorf("%s: Display(%q) = %s, want %s", c.name, c.path, got, c.want)
		}
	}
}
