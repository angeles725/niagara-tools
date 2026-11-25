package readtools_test

import (
	"strings"
	"testing"
)

// depthTree: a link on the Folder itself (level 0), one on its child B (level 1) and one on B's child X
// (level 2).
const depthTree = `{"": {"t": "baja:Folder", "h": "3"},
	"Own": {"t": "baja:Link"}, "Own/sourceOrd": {"v": "h:4"}, "Own/sourceSlotName": {"v": "out"}, "Own/targetSlotName": {"v": "in1"},
	"B": {"t": "control:NumericWritable", "h": "4"},
	"B/Near": {"t": "baja:Link"}, "B/Near/sourceOrd": {"v": "h:4"}, "B/Near/sourceSlotName": {"v": "out"}, "B/Near/targetSlotName": {"v": "in1"},
	"B/X": {"t": "baja:Folder", "h": "5"},
	"B/X/Far": {"t": "baja:Link"}, "B/X/Far/sourceOrd": {"v": "h:4"}, "B/X/Far/sourceSlotName": {"v": "out"}, "B/X/Far/targetSlotName": {"v": "in1"}}`

// The depth argument counts levels below ord, ord itself being level 0 and always scanned: depth 1 lists the
// links of ord and of its direct children, no deeper (retro 2026-11-02-dashboard-live-check-4-15 Δ17). The
// schema range starts at 1, so "depth 0" (ord alone) is never asked for.
func TestListLinksDepthOneIsTheComponentAndItsDirectChildren(t *testing.T) {
	s := serve(t, "station:|slot:/Folder", depthTree)
	for _, tt := range []struct {
		depth string
		want  []string
	}{
		{"1", []string{"Own", "Near"}},
		{"2", []string{"Own", "Near", "Far"}},
	} {
		text, isErr, out := s.Call(t, "n4_list_links", "ord", "station:|slot:/Folder", "depth", num(tt.depth))
		if isErr {
			t.Fatal(text)
		}
		var got []string
		for _, l := range lookupOut(out, "links").([]any) {
			n, _ := l.(interface{ Get(string) (any, bool) }).Get("link_name")
			got = append(got, n.(string))
		}
		if strings.Join(got, ",") != strings.Join(tt.want, ",") {
			t.Errorf("depth %s: links %v, want %v", tt.depth, got, tt.want)
		}
	}
}
