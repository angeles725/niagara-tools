package xmlsrc

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"regexp"
	"testing"
)

// The verdict to share with niagara-go (internal/bog CheckWellFormed, which matches pyexpat): only
// "malformed or not"; the messages differ by design. Deliberate differences, all fail-closed and
// none ever written by Niagara: any DOCTYPE (expat reads an internal DTD), a declared encoding
// other than UTF-8 and an XML version other than 1.0 (encoding/xml reads neither).
var parityProbes = []struct {
	class     string
	text      string
	malformed bool // expat's verdict
}{
	{"undefined entity in content", "<r>&foo;</r>", true},
	{"undefined entity in attribute", "<r a='&foo;'/>", true},
	{"duplicate attribute", "<r a='1' b='2' a='3'/>", true},
	{"bare & in content", "<r>a & b</r>", true},
	{"bare & in attribute", "<r a='a & b'/>", true},
	{"control char in content", "<r>a\x01b</r>", true},
	{"control char in attribute", "<r a='a\x1fb'/>", true},
	{"&#0; in content", "<r>&#0;</r>", true},
	{"char ref beyond U+10FFFF in attribute", "<r a='&#x110000;'/>", true},
	{"-- inside a comment", "<r><!-- a -- b --></r>", true},
	{"]]> in content", "<r>a]]>b</r>", true},
	{"undecodable byte", "<r>a\xffb</r>", true},
	{"noncharacter U+FFFE", "<r a='￾'/>", true},
	{"malformed char ref", "<r>&#xZZ;</r>", true},
	{"misplaced XML declaration", "<r/>\n<?xml version='1.0'?>", true},
	{"text before the root", "x<r/>", true},
	{"XML declaration without version", "<?xml encoding='UTF-8'?><b/>", true},
	{"XML declaration with unknown pseudo-attribute", "<?xml version='1.0' foo='x'?><b/>", true},
	{"XML declaration with bad standalone", "<?xml version='1.0' standalone='maybe'?><b/>", true},
	{"PI target not followed by space", "<?x<ml v?><b/>", true},
	{"reserved PI target", "<b><?Xml x?></b>", true},
	{"control char in a PI", "<b><?x y\x01?></b>", true},
	{"control char in a comment", "<b><!-- \x01 --></b>", true},
	{"no space between attributes", "<b a='1'b='2'/>", true},
	{"surrogate char ref", "<b>&#xD800;</b>", true},
	{"surrogate char ref in attribute", "<b a='&#55296;'/>", true},
	{"BOM then declaration", "\ufeff<?xml version='1.0' standalone='yes'?><b/>", false},
	{"char refs in CDATA and comments are text", "<b><![CDATA[&#xD800;]]><!-- &#xD800; --></b>", false},
	{"legal references", "<?xml version='1.0' encoding='UTF-8'?><r a='&#x9;&#65;&lt;&apos;'>&#x10FFFF;&amp;&gt;&quot;</r>", false},
}

// deliberate matches the xmlsrc errors of the deliberate differences.
var deliberate = regexp.MustCompile(`<!DOCTYPE \.\.\.> is not supported$|CharsetReader is nil$|unsupported version`)

func TestDeliberateDifferences(t *testing.T) {
	for _, text := range []string{"<!DOCTYPE b []><b/>", "<?xml version='1.0' encoding='latin1'?><b/>", "<?xml version='1.1'?><b/>"} {
		if _, err := Parse("d.xml", []byte(text)); err == nil || !deliberate.MatchString(err.Error()) {
			t.Errorf("%q: err = %v, want a deliberate refusal", text, err)
		}
	}
}

func TestParityProbes(t *testing.T) {
	for _, p := range parityProbes {
		_, err := Parse("p.xml", []byte(p.text))
		if (err != nil) != p.malformed {
			t.Errorf("%s: xmlsrc err = %v, expat malformed = %v", p.class, err, p.malformed)
		}
	}
}

// XMLSRC_EXPAT_SUITE names niagara-go's internal/pyxml/testdata/xmlerrors.json: expat's own
// verdict on 3,000+ documents (kept there, not copied: 750 KB). Every verdict must agree except
// the deliberate differences above, which must fail with their own message.
func TestParityExpatSuite(t *testing.T) {
	path := os.Getenv("XMLSRC_EXPAT_SUITE")
	if path == "" {
		t.Skip("XMLSRC_EXPAT_SUITE not set")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		In    *string `json:"in"`
		Hex   string  `json:"hex"`
		Expat *string `json:"expat"`
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	checked, skipped, bad := 0, 0, 0
	for _, c := range cases {
		if c.Expat == nil {
			continue
		}
		text := c.Hex
		if c.In != nil {
			text = *c.In
		} else if b, err := hex.DecodeString(c.Hex); err == nil {
			text = string(b)
		} else {
			t.Fatal(err)
		}
		_, err := Parse("s.xml", []byte(text))
		if err != nil && *c.Expat == "" && deliberate.MatchString(err.Error()) {
			skipped++
			continue
		}
		checked++
		if (err != nil) != (*c.Expat != "") {
			if bad++; bad <= 40 {
				t.Errorf("%q: xmlsrc %v, expat %q", text, err, *c.Expat)
			}
		}
	}
	t.Logf("%d expat verdicts checked, %d deliberate differences skipped", checked, skipped)
	if bad > 0 {
		t.Fatalf("%d of %d verdicts differ", bad, checked)
	}
}
