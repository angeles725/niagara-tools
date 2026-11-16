package blackbox

// What a remove plan says about links and depth (port of test_tools_destructive
// TestOutgoingLinks and TestSnapshotTruncation).

import (
	"fmt"
	"strings"
	"testing"
)

// noteUnseen marks the note saying outgoing links cannot be seen without a scan root.
const noteUnseen = "cannot be detected"

// outside links the group's Src.out to a new Other.in10 under the Folder; it returns Other's name.
func (d *dRig) outside(nn string) string {
	d.T.Helper()
	other := d.AddWithOut("Other", 1.0)
	d.Link(Component{"Src", d.node(nn, "Src").Handle}, "out", other, "in10")
	return other.Name
}

func TestASubtreeWithOutputsDisclosesThatOutgoingLinksCannotBeSeen(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, _ := d.group("Grp")
	notes := planOf(d.Srv.Dry("n4_remove_component", removeArgs(nn)))["notes"]
	note := noteWith(notes, noteUnseen)
	if len(note) != 1 || !strings.Contains(note[0], "outside the removed subtree") || !strings.Contains(note[0], "link_scan_ord") {
		t.Errorf("notes = %v", notes)
	}
}

func TestASubtreeWithoutOutputsHasNoSuchNote(t *testing.T) {
	d := destructiveRig(t, nil)
	plain := d.Add("3", "Plain", "baja:Folder")
	plan := planOf(d.Srv.Dry("n4_remove_component", removeArgs(plain.Name)))
	if len(objects(plan["ops"])) == 0 { // a real plan, so the absences below are not vacuous
		t.Fatalf("plan = %v", plan)
	}
	if _, ok := plan["outgoing_links_broken"]; ok || len(noteWith(plan["notes"], noteUnseen)) != 0 {
		t.Errorf("plan = %v", plan)
	}
}

func TestAScanRootListsTheOutgoingLinksThatTheRemoveWouldBreak(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, _ := d.group("Grp")
	other := d.outside(nn)
	args := acknowledged(removeArgs(nn)) // delta 3: links found refuse until acknowledged
	args["link_scan_ord"] = Folder
	plan := planOf(d.Srv.Dry("n4_remove_component", args))
	if !same(plan["outgoing_links_broken"], []any{map[string]any{"target": Folder + "/" + other,
		"target_slot": "in10", "source_path": "Src", "source_slot": "out"}}) {
		t.Errorf("outgoing_links_broken = %v", plan["outgoing_links_broken"])
	}
	if len(noteWith(plan["notes"], noteUnseen)) != 0 || len(noteWith(plan["notes"], "scanned", Folder)) == 0 {
		t.Errorf("notes = %v", plan["notes"])
	}
}

func TestLinksInsideTheSubtreeAreNotReportedAsOutgoing(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, _ := d.group("Grp")
	args := removeArgs(nn)
	args["link_scan_ord"] = Folder
	plan := planOf(d.Srv.Dry("n4_remove_component", args))
	if _, ok := plan["outgoing_links_broken"]; ok {
		t.Errorf("outgoing_links_broken = %v", plan["outgoing_links_broken"])
	}
	note := noteWith(plan["notes"], "no outgoing links")
	// an empty scan must say what it could not see
	if len(note) == 0 || !strings.Contains(note[0], fmt.Sprintf("depth %d", linkScanDepth)) || !strings.Contains(note[0], "deeper") {
		t.Errorf("notes = %v", plan["notes"])
	}
}

func TestTheScanRootMustBeInsideTheWriteScope(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, _ := d.group("Grp")
	args := removeArgs(nn)
	args["link_scan_ord"] = destructiveStationRootOr
	if text := d.Srv.Err("n4_remove_component", args); !strings.Contains(text, "outside the write scope") {
		t.Errorf("text = %s", text)
	}
}

func TestTheScanResultIsCoveredByThePlanHash(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, _ := d.group("Grp")
	plain := d.Srv.Dry("n4_remove_component", removeArgs(nn))
	d.outside(nn)
	args := acknowledged(removeArgs(nn)) // delta 3: links found refuse until acknowledged
	args["link_scan_ord"] = Folder
	scanned := d.Srv.Dry("n4_remove_component", args)
	if plain["plan_hash"] == nil {
		t.Fatalf("the plain dry run has no plan_hash: %v", plain)
	}
	if scanned["plan_hash"] == nil {
		t.Fatalf("the scanned dry run has no plan_hash: %v", scanned)
	}
	if plain["plan_hash"] == scanned["plan_hash"] {
		t.Errorf("the scan result does not change the plan hash: both %v", plain["plan_hash"])
	}
}

// deepGroup is a Folder Deep/L1/L2 holding C (a NumericConst with limit 42.0): one level deeper
// than the snapshot.
func (d *dRig) deepGroup() string {
	d.T.Helper()
	deep := d.Add("3", "Deep", "baja:Folder")
	l1 := d.Add(deep.Handle, "L1", "baja:Folder")
	l2 := d.Add(l1.Handle, "L2", "baja:Folder")
	c := d.Add(l2.Handle, "C", "")
	d.Set(c.Handle, "limit", map[string]any{"nm": "p", "t": "baja:Double", "v": "42.0"})
	return deep.Name
}

func TestASubtreeDeeperThanTheSnapshotIsReportedInThePlan(t *testing.T) {
	d := destructiveRig(t, nil)
	nn := d.deepGroup()
	reply := d.Srv.Dry("n4_remove_component", removeArgs(nn))
	plan := planOf(reply)
	if !same(plan["snapshot_truncated"], map[string]any{"depth": snapshotDepth, "paths": []any{"L1/L2/C/limit"}}) {
		t.Errorf("snapshot_truncated = %v", plan["snapshot_truncated"])
	}
	// only one level below the cut is loaded, so the count is a floor (H6 review R3)
	if note := noteWith(plan["notes"], "deeper than the snapshot"); len(note) == 0 || !strings.Contains(note[0], "at least 1") {
		t.Errorf("notes = %v", plan["notes"])
	}
	inverse := objects(plan["inverse"])
	if len(inverse) == 0 {
		t.Fatalf("inverse = %v", plan["inverse"])
	}
	l1 := objects(Dig(inverse[0], "b", "s"))
	if len(l1) == 0 || len(objects(l1[0]["s"])) == 0 {
		t.Fatalf("body = %v", inverse[0]["b"])
	}
	l2 := objects(l1[0]["s"])[0]
	kids := objects(l2["s"])
	if !same(field(l2["s"], "n"), []any{"C"}) {
		t.Errorf("L2 children = %v", field(l2["s"], "n"))
	} else if _, ok := kids[0]["s"]; ok { // level 4 never enters the re-create body
		t.Errorf("C carries level 4: %v", kids[0])
	}
	out := d.Srv.OK("n4_remove_component", Execute(removeArgs(nn), reply["confirmation_token"]))
	if !same(Dig(out, "snapshot_truncated", "paths"), []any{"L1/L2/C/limit"}) {
		t.Errorf("out = %v", out)
	}
}

func TestAShallowSubtreeHasNoTruncationEntry(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, _ := d.group("Grp")
	plan := planOf(d.Srv.Dry("n4_remove_component", removeArgs(nn)))
	if len(objects(plan["inverse"])) == 0 { // a real plan, so the absence is not vacuous
		t.Fatalf("plan = %v", plan)
	}
	if _, ok := plan["snapshot_truncated"]; ok {
		t.Errorf("snapshot_truncated = %v", plan["snapshot_truncated"])
	}
}
