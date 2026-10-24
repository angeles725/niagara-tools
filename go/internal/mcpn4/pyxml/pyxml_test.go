package pyxml

import "testing"

// TestPinnedExpatAnswers pins ElementTree.fromstring answers on CPython 3.15 (expat 2.9.0)
// for the cases where this port departs from niagara-go's, so they hold without a Python.
func TestPinnedExpatAnswers(t *testing.T) {
	for text, want := range map[string]string{
		"":                          "no element found: line 1, column 0",
		"<r/><?xml version='1.0'?>": "junk after document element: line 1, column 4",
		"<!é- x --><r/>":            "not well-formed (invalid token): line 1, column 2",
		":?<r/>":                    "not well-formed (invalid token): line 1, column 1",
		"a:":                        "unclosed token: line 1, column 0",
		"<r/><\ufeff":               "junk after document element: line 1, column 4",
		"<·/>":                      "not well-formed (invalid token): line 1, column 1",
		"<p:r/>":                    "unbound prefix: line 1, column 0",
		"<r xmlns:p=''/>":           "must not undeclare prefix: line 1, column 0",
		"<r>\r":                     "no element found: line 1, column 3",
		"<r><![CDATA[x]]":           "unclosed CDATA section: line 1, column 13",
	} {
		if _, err := Parse(text); err == nil || err.Error() != want {
			t.Errorf("%q: %v\nwant %s", text, err, want)
		}
	}
	root, err := Parse("<p:r xmlns:p='urn:p' xmlns='urn:d' a='x\ty\r\nz' p:b='&#65;&lt;'><s/><t xmlns=''/></p:r>")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := root.Get("a")
	b, _ := root.Get("{urn:p}b")
	if root.Tag != "{urn:p}r" || len(root.Attrs) != 2 || a != "x y z" || b != "A<" {
		t.Errorf("root %+v", root)
	}
	if tags := []string{root.Iter()[1].Tag, root.Iter()[2].Tag}; tags[0] != "{urn:d}s" || tags[1] != "t" {
		t.Errorf("children %v", tags)
	}
	if _, err := Parse("<!DOCTYPE r><r/>"); err != ErrDoctype {
		t.Errorf("DOCTYPE: %v", err)
	}
}
