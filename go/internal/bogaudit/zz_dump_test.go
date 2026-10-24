package bogaudit

import (
	"fmt"
	"os"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

func TestZZDump(t *testing.T) {
	p := os.Getenv("BOGAUDIT_DUMP")
	if p == "" {
		t.Skip()
	}
	doc, err := xmlsrc.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	g := build(doc)
	f, _ := os.Create(os.Getenv("BOGAUDIT_OUT"))
	defer f.Close()
	for _, h := range g.comps.keys {
		c := g.comp(h)
		n, _ := g.counts.get(h)
		fmt.Fprintf(f, "C\t%s\t%s\t%s\t%d\n", h, xmlsrc.UnescapeName(c.path), c.typ, n)
	}
	for _, l := range g.links {
		sp, tp := "?", "?"
		if c := g.comp(l.srcH); c != nil {
			sp = xmlsrc.UnescapeName(c.path)
		}
		if l.hasContainer {
			if c := g.comp(l.container); c != nil {
				tp = xmlsrc.UnescapeName(c.path)
			}
		}
		fmt.Fprintf(f, "L\t%s\t%s.%s\t%s.%s\n", l.srcH, sp, l.srcSlot, tp, l.tgtSl)
	}
}
