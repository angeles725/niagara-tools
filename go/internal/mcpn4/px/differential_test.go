package px

import (
	"encoding/json"
	"math/rand"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/golden"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// edgeDocs exercise every branch of parse_px: list caps (max_items 2), repeated values,
// stripped ORDs, layers as elements and attributes, charts nested in charts, images by
// extension, query or attribute name, included Px files, a namespaced root and content.
var edgeDocs = []string{
	`<px version="2" media="m"><import><module name="a"/><module name="a"/><module name=""/><module/><module name="b"/><module name="c"/></import>
<content><L text=" "/><L text="x"/><L text="x"/><L text="y"/><L text="z"/>
<layer name="l1"/><Layer name="l2"/><layer/><W layer="l3"/>
<C viewSize="1,1"/><C viewSize="1,1" name="n"/><C viewSize="2,2"/><D viewSize=""/>
<I a="  module://m/a.PNG?x=1 "/><I a="file:^b.gif"/><I image="slot:/x"/><I image="h:1"/><I a="file:^c.JPEG"/>
<P a="file:^i/H.PX"/><PxInclude ord="station:|slot:/S"/><PxInclude ord="history:/x"/><P a="file:^i/G.px"/><P a="file:^i/F.px"/>
<WebChart name="t"><X o="history:/a"/><X o=" history:/b"/><SubChart o="slot:/Q/out"/></WebChart><WebChart name="t"><X o="history:/a"/><X o=" history:/b"/><SubChart o="slot:/Q/out"/></WebChart>
<V ord="station:|slot:/A/out"/><V ord="station:|slot:/A/out"/><V ord="station:|slot:/A|slot:value"/><V ord="station:|slot:/B/x"/><V ord="local:|slot:/C"/>
<O a="bql:select"/><O a="ip:x"/><O a="service:y"/><O a="module:z"/><O a="local:|foxs:"/><O a="file:x"/><O a="station:|h:1"/>
<O a="module://plain"/><O a="nope"/><FooBinding/><Binding/><layerBinding name="lb"/></content></px>`,
	`<?xml version="1.0" encoding="latin-1"?><px><W text="	tab	"/><W text="&#x85;"/><W text="&#x2028;&#xa0;"/><W text="&#x1c;"/></px>`,
	`<px xmlns="urn:x"><content><W text="t"/></content></px>`,
	`<p:px xmlns:p="urn:p"/>`,
	`<px xmlns:p="urn:p"><p:content><W p:text="t" p:ord="slot:/A"/></p:content><content><Q ord="slot:/Z"/></content><content><R ord="slot:/Y"/></content></px>`,
	`<px><W a="file:^x.İCO"/><W a="file:^x.ico"/><W a="file:^x.pİ"/><W a="slot:/A/İn1"/><W a="h:1/in16"/><W a="h:1/in17"/></px>`,
	`<px media="m"/>`, `<px>text only</px>`, `<html/>`, "", "<px><!-- x --><?pi y?></px>", "<px><!doctype x></px>",
	"<px a='&lt;!ENTITY'/>", "<px><![CDATA[<!ENTıTY]]></px>", "<!DOCTYPE px><px/>", "<px>\r\n<W ord='slot:/a\r\nb/out'/></px>",
}

const pxScript = `import json, sys
from mcp_n4 import files
out = []
for case in json.load(sys.stdin):
    if "bound" in case:
        out.append(files.bound_component(case["bound"]))
        continue
    try:
        out.append(json.dumps(files.parse_px(case["text"], max_items=case["max"])))
    except ValueError as exc:
        out.append("error: %s" % exc)
    except Exception as exc:
        out.append("error: internal error (%s)" % type(exc).__name__)
print(json.dumps(out))`

// TestParseDifferentialPython runs parse_px in files.py on the target CPython and Parse on the
// synthetic view, the edge documents (max_items 2 and 200) and 1,500 random edits of the view,
// and bound_component on ORD shapes, and requires the same JSON text or the same refusal.
func TestParseDifferentialPython(t *testing.T) {
	g := golden.Open(t, "px-parse")
	type pxCase struct {
		Text  string `json:"text"`
		Max   int    `json:"max"`
		Bound string `json:"bound,omitempty"`
	}
	cases := []pxCase{{Text: syntheticPx(t), Max: 200}, {Text: syntheticPx(t), Max: 1}}
	for _, d := range edgeDocs {
		cases = append(cases, pxCase{Text: d, Max: 2}, pxCase{Text: d, Max: 200})
	}
	view, alphabet := []rune(syntheticPx(t)), []rune("<>&;\"'/=!?-:|^ \r\n.so")
	rng := rand.New(rand.NewSource(3))
	for range 1500 {
		doc := append([]rune{}, view...)
		i := rng.Intn(len(doc))
		switch c := alphabet[rng.Intn(len(alphabet))]; rng.Intn(3) {
		case 0:
			doc = append(doc[:i], append([]rune{c}, doc[i:]...)...)
		case 1:
			doc[i] = c
		default:
			doc = append(doc[:i], doc[i+1:]...)
		}
		cases = append(cases, pxCase{Text: string(doc), Max: 3})
	}
	for _, b := range []string{"slot:/A/out", "slot:/A/in0", "slot:/A/in1", "slot:/A/in16", "slot:/A/In1", "a:/out", "/out", "out",
		"slot:/A|slot:x$_9", "slot:/A|slot:x|slot:y", "slot:/A|slot:x\n", "slot:/A|slot:x\n\n", "slot:/A|slot:", "slot:/A|slot:é",
		"slot:/A/out/", "slot:/A/proxyExt", "slot:/A/emergencyOverride", "|slot:x", "slot:/A/facets\n"} {
		cases = append(cases, pxCase{Bound: b})
	}
	in, _ := json.Marshal(cases)
	out := g.Run(pxScript, in)
	var want []string
	if err := json.Unmarshal(out, &want); err != nil {
		t.Fatal(err)
	}
	bad := 0
	for i, c := range cases {
		var got string
		if c.Bound != "" {
			got = BoundComponent(c.Bound)
		} else if s, err := Parse(c.Text, c.Max); err != nil {
			got = "error: " + err.Error()
		} else {
			b, _ := pyjson.Marshal(s, pyjson.Options{})
			got = string(b)
		}
		if got != want[i] {
			if bad++; bad <= 20 {
				t.Errorf("case %d %.200q (max %d):\nGo     %.400s\nPython %.400s", i, c.Text+c.Bound, c.Max, got, want[i])
			}
		}
	}
	if bad > 0 {
		t.Errorf("%d of %d differ", bad, len(cases))
	}
}
