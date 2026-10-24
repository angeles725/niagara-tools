package xmlsrc

import (
	"errors"
	"testing"
)

// A bajaObjectGraph reads U+0000 as Niagara's XParser does (a character reference or a literal byte, in text or an
// attribute value) and carries one warning at the first one; it stays an error in a name, in any other document,
// and beside a literal U+FDD0, the mark that stands in for it while encoding/xml reads the text.
func TestNUL(t *testing.T) {
	const bog = "<bajaObjectGraph>"
	for _, c := range []struct{ name, text, attr, text0, warn, err string }{
		{name: "hex reference in an attribute", text: bog + "\n <p v='1.9&#x0;&#x0;'/></bajaObjectGraph>",
			attr: "1.9\x00\x00", warn: "m.xml:2:11: U+0000 in 2 places (Niagara reads it; XML 1.0 forbids it)"},
		{name: "decimal reference and a literal byte in text", text: bog + "<p>a&#0;b\x00</p></bajaObjectGraph>",
			text0: "a\x00b\x00", warn: "m.xml:1:22: U+0000 in 2 places (Niagara reads it; XML 1.0 forbids it)"},
		{name: "literal byte in a comment", text: bog + "<!--\x00--><p/></bajaObjectGraph>",
			warn: "m.xml:1:22: U+0000 in 1 place (Niagara reads it; XML 1.0 forbids it)"},
		{name: "a reference in CDATA is text", text: bog + "<p><![CDATA[&#0;]]></p></bajaObjectGraph>", text0: "&#0;"},
		{name: "positions after a NUL are the written ones", text: bog + "<p v='&#x0;&#x0;\x00'><q></r></p></bajaObjectGraph>",
			err: "m.xml:1:40: element <q> opened at 1:37 closed by </r>"},
		{name: "a NUL before the root of a bog", text: "<!--\x00--><bajaObjectGraph/>",
			warn: "m.xml:1:5: U+0000 in 1 place (Niagara reads it; XML 1.0 forbids it)"},
		{name: "other documents refuse it", text: "<r>\n <p v='&#x0;'/></r>", err: "m.xml:2:8: illegal character code U+0000"},
		{name: "other documents refuse it before a later error", text: "<r v='&#x0;'><q></r>",
			err: "m.xml:1:7: illegal character code U+0000"},
		{name: "a bog's earlier error comes first", text: bog + "<q></r><p v='&#x0;'/></bajaObjectGraph>",
			err: "m.xml:1:21: element <q> opened at 1:18 closed by </r>"},
		{name: "never in a name", text: bog + "<p\x00q/></bajaObjectGraph>", err: "m.xml:1:20: illegal character code U+0000 in a name"},
		{name: "never beside U+FDD0", text: bog + "<p v='&#xFDD0;&#x0;'/></bajaObjectGraph>",
			err: "m.xml:1:24: U+FDD0 and U+0000 in one document are not supported"},
	} {
		t.Run(c.name, func(t *testing.T) {
			d, err := Parse("m.xml", []byte(c.text))
			if c.err != "" {
				var xe *Error
				if d != nil || !errors.As(err, &xe) || err.Error() != c.err {
					t.Fatalf("Parse = %v, %v\n want the *Error %q", d, err, c.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var warns []string
			for _, w := range d.Warnings() {
				warns = append(warns, w.Error())
			}
			if want := []string{c.warn}; c.warn == "" && len(warns) != 0 || c.warn != "" && (len(warns) != 1 || warns[0] != want[0]) {
				t.Errorf("Warnings = %q, want %q", warns, c.warn)
			}
			if c.attr != "" {
				if v, _ := d.Elements()[1].Attr("v"); v != c.attr {
					t.Errorf("v = %q, want %q", v, c.attr)
				}
			}
			if c.text0 != "" {
				if got := d.Text(1); got != c.text0 {
					t.Errorf("Text = %q, want %q", got, c.text0)
				}
			}
		})
	}
}
