package blackbox

// A link held at the snapshot's last level keeps its slot names (bench retro 2026-11-15 row 14:
// the remove note printed "(None <- None)").

import (
	"strings"
	"testing"
)

func TestALinkAtTheSnapshotsLastLevelIsNamedByItsSlots(t *testing.T) {
	d := destructiveRig(t, nil)
	grp := d.Add("3", "Grp", "baja:Folder")
	a := d.Add(grp.Handle, "A", "baja:Folder")
	b := d.Add(a.Handle, "B", "") // its links sit at depth 3, their slots at depth 4
	x := d.AddWithOut("X", 2.0)
	d.Link(x, "out", b, "in10")
	plan := planOf(d.Srv.Dry("n4_remove_component", removeArgs(grp.Name)))
	note := noteWith(plan["notes"], "comes from outside the removed subtree")
	if len(note) != 1 || !strings.Contains(note[0], "'A/B' (in10 <- out)") {
		t.Errorf("notes = %v", plan["notes"])
	}
	if _, cut := plan["snapshot_truncated"]; cut { // the link's slots are read, not dropped by the cut
		t.Errorf("snapshot_truncated = %v", plan["snapshot_truncated"])
	}
	for _, e := range objects(plan["inverse"]) {
		if spec, ok := e["unlinked_input"].(map[string]any); ok && (spec["target_slot"] != "in10" || spec["source_slot"] != "out") {
			t.Errorf("unlinked_input = %v", spec)
		}
	}
}
