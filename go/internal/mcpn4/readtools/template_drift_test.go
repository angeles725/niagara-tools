package readtools

import (
	"strings"
	"testing"
)

// drift is true or false only when both signatures are known: the station knows the template's
// signature when it is longer than one character (BTemplateInfo.getModifiedState), and the
// deployed copy's when updateSignatures has set it.
func TestLtDriftIsTrueFalseOrNull(t *testing.T) {
	for _, c := range []struct {
		name, ntpl, local string
		drift             any
		note              string
	}{
		{"same", "abc123", "abc123", false, ""},
		{"edited", "abc123", "ffff00", true, ""},
		{"local not computed", "abc123", "", nil, "localSignature is empty"},
		{"local one character", "abc123", "0", true, ""},
		{"template unknown", "", "abc123", nil, "ntplSignature is empty"},
		{"template one character", "0", "abc123", nil, "ntplSignature is empty"},
		{"both empty", "", "", nil, "ntplSignature is empty"},
	} {
		drift, note := ltDrift(c.ntpl, c.local)
		if drift != c.drift {
			t.Errorf("%s: drift %v, want %v", c.name, drift, c.drift)
		}
		if (c.note == "") != (note == "") || !strings.Contains(note, c.note) {
			t.Errorf("%s: note %q, want it to hold %q", c.name, note, c.note)
		}
	}
}

// The refresh note names the action, what runs it and the stale case (the registry description and
// the README say it too).
func TestLtSignatureNoteNamesUpdateSignatures(t *testing.T) {
	for _, s := range []string{"updateSignatures", "localSignature", "stale", "not invoked by this tool"} {
		if !strings.Contains(ltSignatureNote, s) {
			t.Errorf("note lacks %q: %s", s, ltSignatureNote)
		}
	}
}
