package bogaudit

import (
	"bufio"
	"fmt"
	"os"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// TestCrossCheckDump writes the graph the audit reads from BOGAUDIT_DUMP (a .bog) to BOGAUDIT_DUMP_OUT, one
// tab-separated row per component (C handle path type count), slot (S handle name %q-value hasV) and link
// (L srcHandle srcPath.slot targetPath.slot), paths $-decoded, for a diff against niagara-go's
// `bog-nav --json <bog> tree|links|slot` (retro go-bog-audit: handles, paths, types and links agree on the
// client station A and client B bogs). Skipped unless BOGAUDIT_DUMP is set.
func TestCrossCheckDump(t *testing.T) {
	p := os.Getenv("BOGAUDIT_DUMP")
	if p == "" {
		t.Skip("BOGAUDIT_DUMP not set")
	}
	doc, err := xmlsrc.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	g := build(doc)
	f, err := os.Create(os.Getenv("BOGAUDIT_DUMP_OUT"))
	if err != nil {
		t.Fatal(err)
	}
	w := bufio.NewWriter(f)
	path := func(c *comp) string {
		if c == nil {
			return "?"
		}
		return xmlsrc.UnescapeName(c.path)
	}
	for _, h := range g.comps.keys {
		c := g.comp(h)
		n, _ := g.counts.get(h)
		fmt.Fprintf(w, "C\t%s\t%s\t%s\t%d\n", h, path(c), c.typ, n)
		for _, sn := range c.slots.keys {
			sl, _ := c.slots.get(sn)
			fmt.Fprintf(w, "S\t%s\t%s\t%q\t%v\n", h, sn, sl.value, sl.hasV)
		}
	}
	for _, l := range g.links {
		fmt.Fprintf(w, "L\t%s\t%s.%s\t%s.%s\n", l.srcH, path(g.comp(l.srcH)), l.srcSlot, path((&audit{g: g}).container(l)), l.tgtSl)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}
