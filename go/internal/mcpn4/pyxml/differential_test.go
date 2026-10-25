package pyxml

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/golden"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// probes are hand-picked documents: every error class, positions after line breaks and wide
// characters, namespaces, references and attribute normalization.
var probes = []string{
	"", "   ", "<", "<b", "<b>", "<b>abc", "not xml", "x<r/>", "<r/>x", "<r/><r/>", "<r/>\n<?xml version='1.0'?>",
	"<r>&foo;</r>", "<r a='&foo;'/>", "<r a='1' b='2' a='3'/>", "<r>a & b</r>", "<r a='a & b'/>",
	"<r>a\x01b</r>", "<r a='a\x1fb'/>", "<r>&#0;</r>", "<r a='&#x110000;'/>", "<r><!-- a -- b --></r>",
	"<r>a]]>b</r>", "<r a='\ufffe'/>", "<r>&#xZZ;</r>", "<?xml encoding='UTF-8'?><b/>", "<?xml version='1.0' foo='x'?><b/>",
	"<?xml version='1.0' standalone='maybe'?><b/>", "<?x<ml v?><b/>", "<b><?Xml x?></b>", "<b><?x y\x01?></b>",
	"<b><!-- \x01 --></b>", "<b a='1'b='2'/>", "<b>&#xD800;</b>", "<b a='&#55296;'/>",
	"\ufeff<?xml version='1.0' standalone='yes'?><b/>", "\ufeff<b>\n<c/></x>",
	"<b><![CDATA[&#xD800;]]><!-- &#xD800; --></b>", "<b><![CDATA[x]]", "<b><![CDATA[x\r", "<b><![CDA",
	"<?xml version='1.0' encoding='UTF-8'?><r a='&#x9;&#65;&lt;&apos;'>&#x10FFFF;&amp;&gt;&quot;</r>",
	"<?xml version='1.0' encoding='latin-1'?><r a='é'/>", "<?xml version='1.0' encoding='UTF-16'?><r/>",
	"<?xml version='1.1'?><r/>", "<?xml version='1.0'  ?><r/>", "<?xml version=\"1.0\"?>\r\n<r>\r\n<s></t></r>",
	"<p:r xmlns:p='urn:p' p:a='1' a='2'><p:s/><t xmlns='urn:d'><u/></t></p:r>", "<p:r/>", "<r p:a='1'/>",
	"<r xmlns:p=''/>", "<r xmlns:p='urn:p' xmlns:q='urn:p' p:a='1' q:a='2'/>", "<r xmlns=''><s/></r>",
	"<xml:r xml:lang='en'/>", "<r:/>", "<r:s:t/>", "<:r/>", "<r a:='1'/>", "<r xmlns:a='u'><a:s></a:t></r>",
	"<r a='x\ty\nz\r\nw\rv'/>", "<r a='&#10;&#13;&#9;'/>", "<é ñ='ü'>日本</é>", "<r>日本<s>\n日本</t></r>",
	"<r>\r", "<r>\r\n", "<r></r >", "<r></ r>", "<r></r", "<r/ >", "<r / >", "<r a=1/>", "<r a = '1' />", "<r a/>",
	"<r>&#x;</r>", "<r>&#;</r>", "<r>&;</r>", "<r>&a</r>", "<r>&#12a;</r>", "<r>&", "<r a='&", "<r a='&#1",
	"<r><!-x--></r>", "<r><!--x-", "<r><!", "<r><!x", "<r><?", "<r><?x", "<r><?x?", "<r><?x y", "<r><?x?></r>",
	"<!-- c --><r/><!-- d --><?pi x?>", "<!x y><r/>", "<![CDATA[x]]><r/>", "<r/><!x>", "<r/>]", "<r/>'a",
	"<r/>&", "<r/>%a;", "%a;<r/>", "#a<r/>", "(<r/>", "<r/><", "<r/></r>", "<r>\x00</r>", "<r>\u0085\u2028</r>",
	"<\u00b7/>", "<a\u00b7\u0300/>", "<\U00010000/>", "<a\U00010000/>", "<r a='\U0010FFFF'/>", "\ufeff",
	"\ufeff\ufeff<r/>", " <?xml version='1.0'?><r/>", "<?xml version='1.0'?><?xml version='1.0'?><r/>",
	"<?xml version='1.0' encoding='9x'?><r/>", "<?xml version='1.0' standalone='yes' x?><r/>",
	"<?xml version='2.0'?><r/>", "<r><?xml version='1.0'?></r>", "<r/><?xml version='1.0'?>", "<r/><?xml?>",
	"<r/><?xml x", "<r/>\r\n<?xml version='bad'?>", "<r/><?xml version='1.0'?><!-- c -->",
	"<!é- x --><r/>", "<!a_b x><r/>", "<!a:b x><r/>", ":?<r/>", "a:?<r/>", "a:b?<r/>", "a:b:c?<r/>", ".?<r/>",
	"a?<r/>", "a:", "<r/>a:", "<r/>a:b:", "#a:b<r/>", "%a:b;<r/>", "%:a;<r/>", "#:a<r/>", "·?<r/>", "é:è+<r/>", "<?xml version='1.'?><r/>", "<?xmlx?><r/>", "<?XML version='1.0'?><r/>",
}

// nsDoc is a second mutation base: the prolog, namespaces, references, CDATA and a PI.
const nsDoc = "\ufeff<?xml version='1.0' standalone='no'?>\r\n<!-- c --><?pi x?>\n<p:px xmlns:p='urn:p' xmlns='urn:d' " +
	"p:a='&#65;&amp;' b=\"x\ty\"><c d='1'><![CDATA[<x>]]>t&lt;</c><?q r?><e:f xmlns:e='urn:e'/></p:px>\n<!-- e -->"

// mutants edits the synthetic Px view and nsDoc at random (fixed seed): characters that open,
// close or break markup inserted, replaced or deleted, one to three edits per document.
func mutants(t *testing.T, n int) []string {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "mcp-n4-kit", "tests", "fixtures", "synthetic_view.px"))
	if err != nil {
		t.Fatal(err)
	}
	bases := [][]rune{[]rune(string(raw)), []rune(nsDoc)}
	alphabet := []rune("<>&;\"'/=!?-[]:#x \r\n\té\x01\ufffd\ufeffp")
	rng := rand.New(rand.NewSource(7))
	out := make([]string, 0, n)
	for len(out) < n {
		doc := append([]rune{}, bases[len(out)%2]...)
		for k := rng.Intn(3); k >= 0; k-- {
			i := rng.Intn(len(doc))
			switch c := alphabet[rng.Intn(len(alphabet))]; rng.Intn(3) {
			case 0:
				doc = append(doc[:i], append([]rune{c}, doc[i:]...)...)
			case 1:
				doc[i] = c
			default:
				doc = append(doc[:i], doc[i+1:]...)
			}
		}
		out = append(out, string(doc))
	}
	return out
}

// suite is niagara-go's internal/pyxml/testdata/xmlerrors.json when PYXML_EXPAT_SUITE names
// it (750 KB, not copied): each input decoded as parse_px decodes a body.
func suite(t *testing.T) []string {
	path := os.Getenv("PYXML_EXPAT_SUITE")
	if path == "" {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		In  *string `json:"in"`
		Hex string  `json:"hex"`
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, c := range cases {
		data := []byte(c.Hex)
		if c.In != nil {
			data = []byte(*c.In)
		} else if data, err = hex.DecodeString(c.Hex); err != nil {
			t.Fatal(err)
		}
		text, _ := pyjson.DecodeUTF8(data, true)
		out = append(out, text)
	}
	return out
}

const treeScript = `import json, sys
import xml.etree.ElementTree as ET
def tree(el):
    return [el.tag, [[k, v] for k, v in el.attrib.items()], [tree(c) for c in el]]
out = []
for text in json.load(sys.stdin):
    try:
        out.append(tree(ET.fromstring(text)))
    except ET.ParseError as exc:
        out.append(str(exc))
    except Exception as exc:
        out.append("!" + type(exc).__name__)
print(json.dumps(out))`

// treeJSON renders el like treeScript's tree().
func treeJSON(el *Element) any {
	attrs := []any{}
	for _, a := range el.Attrs {
		attrs = append(attrs, []any{a.Name, a.Value})
	}
	kids := []any{}
	for _, c := range el.Children {
		kids = append(kids, treeJSON(c))
	}
	return []any{el.Tag, attrs, kids}
}

// TestDifferentialPython parses the probes, 3,000 mutants of the synthetic Px view and, when
// PYXML_EXPAT_SUITE is set, niagara-go's expat suite with ElementTree.fromstring(str) on the
// target CPython and with Parse, and requires the same tree or the same ParseError text. A
// document holding a DOCTYPE is only checked to be one (parse_px never parses it). Python's
// trees are the goldens testdata/tree.golden.jsonl and, for the suite, tree-suite.golden.jsonl.
func TestDifferentialPython(t *testing.T) {
	diffTrees(t, "tree", append(append([]string{}, probes...), mutants(t, 3000)...))
	if docs := suite(t); docs != nil {
		diffTrees(t, "tree-suite", docs)
	}
}

// diffTrees compares Parse with the trees golden name recorded for inputs.
func diffTrees(t *testing.T, name string, inputs []string) {
	in, _ := json.Marshal(inputs)
	out := golden.Open(t, name).Run(treeScript, in)
	var want []json.RawMessage
	if err := json.Unmarshal(out, &want); err != nil {
		t.Fatal(err)
	}
	bad, doctypes := 0, 0
	for i, text := range inputs {
		var got any
		switch root, err := Parse(text); {
		case errors.Is(err, ErrDoctype):
			if doctypes++; !strings.Contains(text, "<!DOCTYPE") {
				t.Errorf("%q: ErrDoctype without a DOCTYPE", text)
			}
			continue
		case err != nil:
			got = err.Error()
		default:
			got = treeJSON(root)
		}
		g, _ := json.Marshal(got)
		var w any
		json.Unmarshal(want[i], &w)
		ws, _ := json.Marshal(w)
		if string(g) != string(ws) {
			if bad++; bad <= 40 {
				t.Errorf("%q:\nGo     %.300s\nPython %.300s", text, g, ws)
			}
		}
	}
	t.Logf("%d documents compared, %d with a DOCTYPE skipped", len(inputs)-doctypes, doctypes)
	if bad > 0 {
		t.Errorf("%d of %d differ", bad, len(inputs)-doctypes)
	}
}
