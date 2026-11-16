package writeops

import "testing"

// An outgoing link recorded by the remove of a tag slot names that slot as the station stores it
// (escaped): the rollback must accept it to re-create the link (lab 2026-11-15, 4.15).
func TestOutgoingSpecAcceptsAnEscapedSourceSlot(t *testing.T) {
	spec := obj("target", "station:|slot:/F/Sink", "target_slot", "in10", "source_path", "",
		"source_slot", "tst$3acap", "target_h", "6eb4e")
	out, err := outgoingSpec(spec)
	if err != nil {
		t.Fatal(err)
	}
	if get(out, "source_slot") != "tst$3acap" {
		t.Fatalf("spec %v", out)
	}
	for _, bad := range []string{"a b", "x|y", "$zz"} {
		if _, err := outgoingSpec(obj("target", "station:|slot:/F/Sink", "target_slot", "in10",
			"source_path", "", "source_slot", bad)); err == nil {
			t.Errorf("source_slot %q accepted", bad)
		}
	}
}

// A link into a removed tag slot is recorded as an unlinked_input whose target_slot is that
// escaped slot name: the rollback must accept it (and report the input) instead of refusing.
func TestUnlinkedSpecAcceptsAnEscapedTargetSlot(t *testing.T) {
	out, err := unlinkedSpec(obj("source_slot", "out", "target_path", "", "target_slot", "tst$3acap"))
	if err != nil {
		t.Fatal(err)
	}
	if get(out, "target_slot") != "tst$3acap" {
		t.Fatalf("spec %v", out)
	}
	if _, err := unlinkedSpec(obj("source_slot", "out", "target_path", "", "target_slot", "a b")); err == nil {
		t.Error("target_slot 'a b' accepted")
	}
}

// A link the rollback re-creates from a Double tag into a StatusNumeric input is a
// baja:ConversionLink on the station (lab 2026-11-15): the read-back must find it.
func TestOutgoingThereFindsAConversionLink(t *testing.T) {
	link := obj("nm", "p", "n", "Link", "t", "baja:ConversionLink")
	nodes := obj("", obj("nm", "p", "h", "6eb4e", "t", "control:NumericWritable"), "Link", link,
		"Link/sourceOrd", obj("n", "sourceOrd", "v", "h:6e9b6"),
		"Link/sourceSlotName", obj("n", "sourceSlotName", "v", "tst$3acap"),
		"Link/targetSlotName", obj("n", "targetSlotName", "v", "in10"))
	sent := obj("target", "station:|slot:/F/Sink", "target_slot", "in10", "source_path", "",
		"source_slot", "tst$3acap", "source_h", "6e9b6")
	there, err := outgoingThere(scanRoot{nodes}, sent)
	if err != nil || !there {
		t.Fatalf("there=%v err=%v", there, err)
	}
}
