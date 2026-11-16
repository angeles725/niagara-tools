package bql

import (
	"strings"
	"testing"
)

// Verifier probes for PR #713 (retro 2026-11-16 M1): guard bypasses over the alarm: and
// history: bases, and the exact station-base refusal texts kept.

func TestComposeBaseOrdRefusesSmuggledBases(t *testing.T) {
	for _, base := range []string{"history:/S/../x", "history:/S/..", "history:/S/a%7Cb", "history:/S/NW/x",
		"history:S/NW", "ALARM:", "alarm: ", "alarm:|view:file:ITableToCsv", "alarm:|slot:/"} {
		if ord, err := ComposeBaseOrd(base, "select timestamp"); err == nil || !strings.HasPrefix(err.Error(), "base ") {
			t.Errorf("ComposeBaseOrd(%q) = %q, %v", base, ord, err)
		}
	}
}

func TestComposeBaseOrdRefusesPipeAndControlsInTableQueries(t *testing.T) {
	for _, base := range []string{"alarm:", "history:/S/NW", "station:|alarm:"} {
		for _, q := range []string{"select timestamp where msgText = 'a|b'", "select a\nwhere b = 1",
			"select a\rwhere b = 1", "select a\ufeff"} {
			if ord, err := ComposeBaseOrd(base, q); err == nil || !strings.HasPrefix(err.Error(), "query must not contain") {
				t.Errorf("ComposeBaseOrd(%q, %q) = %q, %v", base, q, ord, err)
			}
		}
	}
}

func TestComposeBaseOrdKeepsStationRefusalTexts(t *testing.T) {
	for _, base := range []string{"", "Drivers", "station:", "station:|foo", "station:|station:|alarm:",
		"station:|alarm:|slot:/", "station:|slot:/Drivers//x", "station:|slot:/Drivers|bql:select"} {
		for _, q := range []string{"select name from baja:Component", "select name", "select a|b from x"} {
			wantOrd, wantErr := ComposeOrd(base, q)
			gotOrd, gotErr := ComposeBaseOrd(base, q)
			if gotOrd != wantOrd || (gotErr == nil) != (wantErr == nil) ||
				(gotErr != nil && gotErr.Error() != wantErr.Error()) {
				t.Errorf("ComposeBaseOrd(%q, %q) = %q, %v; ComposeOrd = %q, %v", base, q, gotOrd, gotErr, wantOrd, wantErr)
			}
		}
	}
}
