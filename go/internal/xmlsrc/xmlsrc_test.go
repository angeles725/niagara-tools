package xmlsrc

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Every malformed document is an *Error carrying path:line:col, never an empty Doc.
func TestParseMalformed(t *testing.T) {
	for _, c := range []struct{ name, text, want string }{
		{"empty", "", "m.xml:1:1: no root element"},
		{"only a declaration", "<?xml version='1.0'?>\n", "m.xml:2:1: no root element"},
		{"unclosed tag", "<r>\n  <a>\n</r>", "m.xml:3:1: element <a> opened at 2:3 closed by </r>"},
		{"unclosed root", "<r>\n<a/>\n", "m.xml:3:1: element <r> opened at 1:1 is never closed"},
		{"mismatched end", "<r><a></b></r>", "m.xml:1:7: element <a> opened at 1:4 closed by </b>"},
		{"two roots", "<r/>\n<s/>", "m.xml:2:1: junk after document element"},
		{"trailing junk", "<r/>x", "m.xml:1:5: junk after document element"},
		{"text before root", "  x<r/>", "m.xml:1:3: text outside the root element"},
		{"CDATA before root", "<![CDATA[x]]><r/>", "m.xml:1:1: text outside the root element"},
		{"duplicate attribute", "<r a='1' a='2'/>", "m.xml:1:1: duplicate attribute a"},
		{"misplaced XML declaration", "<r/>\n<?xml version='1.0'?>", "m.xml:2:1: XML declaration not at the start of the document"},
		{"DOCTYPE", "<!DOCTYPE r>\n<r/>", "m.xml:1:1: <!DOCTYPE ...> is not supported"},
		{"end without start", "<r/></x>", "m.xml:1:5: end tag </x> without a start tag"},
		{"bad entity", "<r>\n a &foo; b</r>", "m.xml:2:9: invalid character entity &foo;"},
		{"bad entity in attribute", "<r a='&foo;'/>", "m.xml:1:12: invalid character entity &foo;"},
		{"bare ampersand", "<r>a & b</r>", "m.xml:1:7: invalid character entity & (no semicolon)"},
		{"invalid UTF-8", "<r>a\xffb</r>", "m.xml:1:7: invalid UTF-8"},
		{"unterminated tag at EOF", "<r><a", "m.xml:1:6: unexpected EOF"},
	} {
		t.Run(c.name, func(t *testing.T) {
			d, err := Parse("m.xml", []byte(c.text))
			if d != nil {
				t.Errorf("Parse returned a Doc with the error")
			}
			var xe *Error
			if !errors.As(err, &xe) {
				t.Fatalf("err = %v (%T), want *Error", err, err)
			}
			if got := err.Error(); got != c.want {
				t.Errorf("err = %q\n want %q", got, c.want)
			}
		})
	}
}

func TestParseWellFormed(t *testing.T) {
	text := "<?xml version='1.0' encoding='UTF-8'?>\n<!-- c -->\n<r>\n  <a x='1'/>\n</r>\n<?pi after?>\n"
	d, err := Parse("ok.xml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	if d.Path != "ok.xml" || len(d.Elements()) != 2 {
		t.Fatalf("Doc = %q with %d elements, want ok.xml with 2", d.Path, len(d.Elements()))
	}
}

func TestOpen(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.xml")
	bad := filepath.Join(dir, "bad.xml")
	if err := os.WriteFile(good, []byte("<r/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte("<r>&x;</r>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if d, err := Open(good); err != nil || d.Path != good {
		t.Fatalf("Open(good) = %v, %v", d, err)
	}
	if _, err := Open(bad); err == nil || err.Error() != bad+":1:7: invalid character entity &x;" {
		t.Errorf("Open(bad) err = %v", err)
	}
	missing := filepath.Join(dir, "missing.xml")
	if d, err := Open(missing); d != nil || !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Open(missing) = %v, %v; want nil and ErrNotExist", d, err)
	}
	if d, err := Open(dir); d != nil || err == nil {
		t.Errorf("Open(dir) = %v, %v; want a read error", d, err)
	}
}
