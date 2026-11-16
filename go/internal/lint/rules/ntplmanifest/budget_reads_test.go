package ntplmanifest

import (
	"fmt"
	"testing"
)

// T34 (#716 review, resilience): the run budget counted inflated bytes only, so empty or tiny entries drew
// nothing from it. 100 templates of 10,000 four-byte Px entries each (about 4 MiB of the 128 MiB budget) took
// 12.4 s on the base; the 1,000 templates the count bound allows, about two minutes. The run also stops after
// maxReads entries in all. Here every template is opened for its manifest, its bog and 9,998 Px entries,
// 10,000 reads, so ten templates are read and the manifest of the eleventh is the first entry not read.
func TestRunStopsAtTheEntryReadBound(t *testing.T) {
	t.Parallel()
	per := 10000
	entries := []entry{{manifestEntry, manifestXML("")}, {bogEntry, bog}}
	for i := 0; i < per-2; i++ {
		entries = append(entries, entry{fmt.Sprintf("px/P%05d.px", i), "<a/>"})
	}
	one := ntpl(t, entries...)
	files := map[string]string{}
	stop := maxReads / per
	for i := 0; i <= stop+1; i++ {
		files[fmt.Sprintf("ModA-rt/src/t/T%02d.ntpl", i)] = one
	}
	expect(t, "entry reads over the bound", moduleRoot(t, files), 1, fmt.Sprintf(
		"FAIL  ntpl-manifest  @ROOT@/ModA-rt/src/t/T%02d.ntpl: %d entries read in all, the bound of one run -- template-manifest.xml and everything after it not read\n",
		stop, maxReads))
}
