package htmltext

import "testing"

// Values from CPython 3.15: open(path, encoding="utf-8", errors="replace").read() (TextIOWrapper) and
// html.unescape.
func TestDecode(t *testing.T) {
	for in, want := range map[string]string{
		"a\xffb":             "a�b",
		"\xff\xfe":           "��",
		"A\xe2\x82B":         "A�B", // one replacement for the truncated sequence
		"\xe2\x82":           "�",
		"x\xf0\x9f\x98":      "x�",
		"\xed\xa0\x80":       "���", // an encoded surrogate is three bad bytes
		"\xc0\xaf":           "��",
		"\xf4\x90\x80\x80":   "����",
		"a\r\nb\rc\n\r":      "a\nb\nc\n\n", // universal newlines
		"\xef\xbb\xbfk":      "\ufeffk",     // utf-8, not utf-8-sig: the BOM stays
		"Presión de succión": "Presión de succión",
	} {
		if got := Decode([]byte(in)); got != want {
			t.Errorf("Decode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUnescape(t *testing.T) {
	for in, want := range map[string]string{
		"no reference":                     "no reference",
		"&amp; &lt;&gt;":                   "& <>",
		"&ampfoo;":                         "&foo;",
		"&amp-x;":                          "&-x;",
		"&#65;&#x42;&#X43":                 "ABC",
		"&#0;&#xD800;&#x110000;":           "���",
		"&#13;&#128;&#x9f;":                "\r€Ÿ",
		"&#1;&#xb;&#xFFFE;":                "",
		"&notit; &amp":                     "¬it; &",
		"&nGt;&nLt;":                       "≫⃒≪⃒",
		"&unknown;":                        "&unknown;",
		"&#99999999999999999999;":          "�",
		"&aacute&eacute;":                  "áé",
		"&; &#; &#x;":                      "&; &#; &#x;",
		"&ldquo;Diagn&oacute;stico&rdquo;": "“Diagnóstico”",
	} {
		if got := Unescape(in); got != want {
			t.Errorf("Unescape(%q) = %q, want %q", in, got, want)
		}
	}
}
