package triageconsole

import "testing"

// TestEscaperRawByteWarning: gawk's %c prints a byte >= 0x80 as that byte, not as a UTF-8 rune.
func TestEscaperRawByteWarning(t *testing.T) {
	var warns []string
	e := &escaper{warn: func(w string) { warns = append(warns, w) }}
	if got := e.value("a\\\xe9b\\\xe9"); got != "a\xe9b\xe9" {
		t.Fatalf("value = %q", got)
	}
	want := "awk: warning: escape sequence `\\\xe9' treated as plain `\xe9'\n"
	if len(warns) != 1 || warns[0] != want {
		t.Fatalf("warnings = %q, want one %q", warns, want)
	}
}

// TestBasenameDirname pins the coreutils spellings the subject is built from (no path cleaning).
func TestBasenameDirname(t *testing.T) {
	for _, c := range []struct{ in, base, dir string }{
		{"a.txt", "a.txt", "."}, {"/a.txt", "a.txt", "/"}, {"x//a.txt", "a.txt", "x"}, {"x/y/../a.txt", "a.txt", "x/y/.."},
		{"d/", "d", "."}, {"/", "/", "/"}, {"//a", "a", "/"}, {"a//b//", "b", "a"},
	} {
		if b, d := basename(c.in), dirname(c.in); b != c.base || d != c.dir {
			t.Errorf("%q: basename %q dirname %q, want %q %q", c.in, b, d, c.base, c.dir)
		}
	}
}
