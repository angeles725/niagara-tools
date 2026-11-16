package bql

import (
	"strings"
	"testing"
)

// n4_bql_query bases besides a station subtree (retro 2026-11-16 M1): the alarm database and
// one history, through the same ITableToCsv view.

func TestComposeBaseOrdReadsTheAlarmDatabase(t *testing.T) {
	for _, base := range []string{"alarm:", "station:|alarm:"} {
		got, err := ComposeBaseOrd(base, " select timestamp, source where sourceState = 'offnormal' order by timestamp desc ")
		want := "alarm:|bql:select timestamp, source where sourceState = 'offnormal' order by timestamp desc|" + View
		if err != nil || got != want {
			t.Errorf("ComposeBaseOrd(%q) = %q, %v; want %q", base, got, err, want)
		}
	}
}

func TestComposeBaseOrdReadsOneHistory(t *testing.T) {
	for _, base := range []string{"history:/Prueba_4_15/NumericInterval", "station:|history:/Prueba_4_15/NumericInterval"} {
		got, err := ComposeBaseOrd(base, "select timestamp, value where timestamp.millis >= 1794808800000")
		want := "history:/Prueba_4_15/NumericInterval|bql:select timestamp, value where timestamp.millis >= 1794808800000|" + View
		if err != nil || got != want {
			t.Errorf("ComposeBaseOrd(%q) = %q, %v; want %q", base, got, err, want)
		}
	}
}

func TestComposeBaseOrdKeepsTheStationBaseOrdUnchanged(t *testing.T) {
	for _, c := range [][2]string{{"station:|slot:/Drivers", "select slotPath from driver:Device"}, {"/", "select name from baja:Folder"},
		{"slot:/A/B/", "select name from control:ControlPoint where out.value > 1"}} {
		want, wantErr := ComposeOrd(c[0], c[1])
		got, err := ComposeBaseOrd(c[0], c[1])
		if got != want || (err == nil) != (wantErr == nil) {
			t.Errorf("ComposeBaseOrd(%q) = %q, %v; ComposeOrd = %q, %v", c[0], got, err, want, wantErr)
		}
	}
}

func TestComposeBaseOrdRefusesANonSelectOverATableBase(t *testing.T) {
	for _, base := range []string{"alarm:", "history:/S/N"} {
		for _, q := range []string{"delete *", "update x set y = 1", "selectx *", "select", "select *|view:web:PropertySheet",
			"select * where a = '\x00'", "select " + strings.Repeat("a", MaxQueryLength)} {
			if _, err := ComposeBaseOrd(base, q); err == nil || !strings.Contains(err.Error(), "query") {
				t.Errorf("ComposeBaseOrd(%q, %q) = %v", base, q, err)
			}
		}
	}
}

func TestComposeBaseOrdRefusesAnUnknownBaseNamingIt(t *testing.T) {
	for _, base := range []string{"file:^px/a.px", "alarm:x", "alarm:|bql:select *", "history:", "history:/S",
		"history:/S/N|bql:select *", "history:/S/N?period=today", "station:|file:^a", "station:|h:1", "ip:host|fox:"} {
		_, err := ComposeBaseOrd(base, "select *")
		if err == nil || !strings.HasPrefix(err.Error(), "base ") {
			t.Errorf("ComposeBaseOrd(%q) = %v", base, err)
		}
	}
}

func TestComposeBaseOrdRefusesAFromClauseOverATableBase(t *testing.T) {
	for _, base := range []string{"alarm:", "history:/S/N", "station:|alarm:"} {
		for _, q := range []string{"select timestamp from history:HistoryRecord", "select * FROM alarm:AlarmRecord where a = 1",
			"select timestamp  from x"} {
			_, err := ComposeBaseOrd(base, q)
			if err == nil || !strings.HasPrefix(err.Error(), "query ") || !strings.Contains(err.Error(), "select <columns> [where ...] [order by ...]") {
				t.Errorf("ComposeBaseOrd(%q, %q) = %v", base, q, err)
			}
		}
	}
	for _, q := range []string{"select timestamp where msgText = 'from x'", "select fromState, timestamp", "select a where b = 'it''s from'"} {
		if _, err := ComposeBaseOrd("alarm:", q); err != nil {
			t.Errorf("ComposeBaseOrd(alarm:, %q) = %v", q, err)
		}
	}
}
