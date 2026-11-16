package bql

import (
	"strings"
	"testing"
)

func TestComposeHistoryOrdBuildsTheHistorySchemeOrd(t *testing.T) {
	for _, c := range []struct{ id, params, want string }{
		{"/Prueba_4_15/NumericWritable", "", "history:/Prueba_4_15/NumericWritable|bql:" + HistoryRecordsQuery + "|" + View},
		{"/S/A$20B", "period=today", "history:/S/A$20B?period=today|bql:" + HistoryRecordsQuery + "|" + View},
		{"/S/N", "start=2026-11-15T00:00:00.000-06:00;end=2026-11-15T23:59:59.999-06:00",
			"history:/S/N?start=2026-11-15T00:00:00.000-06:00;end=2026-11-15T23:59:59.999-06:00|bql:" + HistoryRecordsQuery + "|" + View},
	} {
		got, err := ComposeHistoryOrd(c.id, c.params, HistoryRecordsQuery)
		if err != nil || got != c.want {
			t.Errorf("ComposeHistoryOrd(%q, %q) = %q, %v; want %q", c.id, c.params, got, err, c.want)
		}
	}
	if HistoryRecordsQuery != "select timestamp, value, status order by timestamp desc" {
		t.Errorf("records query %q is not newest first", HistoryRecordsQuery)
	}
}

func TestValidateHistoryIdRefusesWhatIsNotAHistoryId(t *testing.T) {
	for _, id := range []string{"", "Prueba/N", "/Prueba", "/Prueba/", "//N", "/S/N/extra", "/S/N|bql:select x",
		"/S/N?period=today", "/S/N;x", "/S/N x", "/S/N\x00", "/S/1N", "/S/N-1", "/S/$zz", "/S/" + strings.Repeat("a", 201),
		"history:/S/N", "/S/Ñ"} {
		if err := ValidateHistoryId(id); err == nil {
			t.Errorf("ValidateHistoryId(%q) accepted it", id)
		} else if !strings.Contains(err.Error(), "history_id must be") {
			t.Errorf("ValidateHistoryId(%q) = %v", id, err)
		}
		if _, err := ComposeHistoryOrd(id, "", HistoryRecordsQuery); err == nil {
			t.Errorf("ComposeHistoryOrd(%q) accepted it", id)
		}
	}
	for _, id := range []string{"/Prueba_4_15/NumericWritable", "/S/A$20B", "/S/x$u00e9", "/S/" + strings.Repeat("a", 200), "/s2/n_1"} {
		if err := ValidateHistoryId(id); err != nil {
			t.Errorf("ValidateHistoryId(%q) = %v", id, err)
		}
	}
}

func TestHistoryParamsAcceptsOnlyTheDocumentedRanges(t *testing.T) {
	for _, c := range []struct{ period, start, end, want string }{
		{"", "", "", ""},
		{"today", "", "", "period=today"},
		{"last7Days", "", "", "period=last7Days"},
		{"", "2026-11-15T00:00:00.000-06:00", "", "start=2026-11-15T00:00:00.000-06:00"},
		{"", "", "2026-11-15T10:00:00Z", "end=2026-11-15T10:00:00Z"},
		{"", "2026-11-15T00:00:00+01:00", "2026-11-16T00:00:00.5Z", "start=2026-11-15T00:00:00+01:00;end=2026-11-16T00:00:00.5Z"},
	} {
		got, err := HistoryParams(c.period, c.start, c.end)
		if err != nil || got != c.want {
			t.Errorf("HistoryParams(%q, %q, %q) = %q, %v; want %q", c.period, c.start, c.end, got, err, c.want)
		}
	}
	for _, c := range [][3]string{{"Today", "", ""}, {"timeRange", "", ""}, {"lastHour", "", ""},
		{"today", "2026-11-15T00:00:00Z", ""}, {"", "2026-11-15", ""}, {"", "2026-11-15T00:00:00", ""},
		{"", "", "2026-11-15T00:00:00Z;period=today"}, {"", "2026-11-15T00:00:00Z|x", ""}} {
		if got, err := HistoryParams(c[0], c[1], c[2]); err == nil {
			t.Errorf("HistoryParams(%q) accepted it: %q", c, got)
		}
	}
	if len(HistoryPeriods) != 10 || HistoryPeriods[0] != "today" || HistoryPeriods[9] != "lastYear" {
		t.Errorf("HistoryPeriods = %v", HistoryPeriods)
	}
}

// ComposeOrd still refuses a history base with its unchanged text.
func TestComposeOrdStillRefusesAHistoryBase(t *testing.T) {
	_, err := ComposeOrd("history:/Prueba_4_15/NumericWritable", "select timestamp from x:Y")
	want := "base must be a plain slot path such as station:|slot:/Drivers, got 'history:/Prueba_4_15/NumericWritable'"
	if err == nil || err.Error() != want {
		t.Errorf("ComposeOrd(history base) = %v; want %q", err, want)
	}
}
