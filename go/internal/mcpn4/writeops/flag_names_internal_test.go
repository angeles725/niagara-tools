package writeops

import (
	"strings"
	"testing"
)

// The table holds every letter of Flags.js once, in the station's encoding order.
func TestTheFlagTableHoldsEveryStationFlagOnceInEncodingOrder(t *testing.T) {
	letters, names := "", map[string]bool{}
	for _, f := range flagTable {
		letters += f.letter
		if f.name == "" || names[f.name] {
			t.Errorf("flag %q has an empty or repeated name %q", f.letter, f.name)
		}
		names[f.name] = true
	}
	if letters != "rthsandcoxfApRmNL1234" {
		t.Errorf("letters = %q", letters)
	}
}

func TestFlagsAreNamedShownAndOrderedFromTheTable(t *testing.T) {
	if got := namedFlags("ho"); got != "h hidden, o operator" {
		t.Errorf("namedFlags(ho) = %q", got)
	}
	if got := namedFlags(""); got != "none" {
		t.Errorf("namedFlags of no flags = %q", got)
	}
	if got := namedFlags("hZ"); got != "h hidden, Z unknown" { // a letter a newer station may send
		t.Errorf("namedFlags(hZ) = %q", got)
	}
	if got := shownFlags("rA"); got != `"rA" (r readonly, A no-audit)` {
		t.Errorf("shownFlags(rA) = %q", got)
	}
	if got := shownFlags(""); got != `"" (none)` {
		t.Errorf("shownFlags of no flags = %q", got)
	}
	if got := canonicalFlags("4Loh"); got != "hoL4" {
		t.Errorf("canonicalFlags(4Loh) = %q", got)
	}
	if !sameFlags("oh", "ho") || !sameFlags("", "") || sameFlags("h", "ho") || sameFlags("h", "") {
		t.Error("sameFlags is not set equality")
	}
}

func TestAFlagsStringWithAnUnknownOrRepeatedLetterIsRefused(t *testing.T) {
	if err := checkFlags("rthsandcoxfApRmNL1234"); err != nil {
		t.Errorf("every letter once: %v", err)
	}
	if err := checkFlags(""); err != nil {
		t.Errorf("no flags: %v", err)
	}
	want := "unknown flag letter 'z' in flags 'hz'; valid letters: r readonly, t transient, h hidden, s summary, " +
		"a async, n no-run, d default-on-clone, c confirm-required, o operator, x execute-on-change, f fan-in, " +
		"A no-audit, p composite, R remove-on-clone, m metadata, N non-critical, L link-target, 1 user-defined-1, " +
		"2 user-defined-2, 3 user-defined-3, 4 user-defined-4"
	if err := checkFlags("hz"); err == nil || err.Error() != want {
		t.Errorf("unknown letter:\n got  %v\n want %s", err, want)
	}
	if err := checkFlags("hoh"); err == nil || err.Error() != "flag letter 'h' is repeated in flags 'hoh'; give each letter once" {
		t.Errorf("repeated letter: %v", err)
	}
	if err := checkFlags("h o"); err == nil || !strings.Contains(err.Error(), "unknown flag letter ' '") {
		t.Errorf("a space: %v", err)
	}
}

// The copy note is built from the same table and keeps its wording.
func TestTheCopyNoteNamesItsFlagsFromTheTable(t *testing.T) {
	if notCopiedNote != "Not copied (flags r readonly, t transient, d default-on-clone, R remove-on-clone): " {
		t.Errorf("notCopiedNote = %q", notCopiedNote)
	}
}
