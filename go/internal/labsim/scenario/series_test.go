package scenario

import (
	"strings"
	"testing"
	"time"
)

func TestReadSeriesReadsAPlainTimeValueFile(t *testing.T) {
	t.Parallel()
	s, err := ReadSeries(strings.NewReader("time,value\n" +
		"2026-11-15T20:00:00Z,21.5\n" +
		"2026-11-15T14:01:00-06:00,22\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Values) != 2 || s.Values[0] != 21.5 || s.Values[1] != 22 || s.Skipped != 0 {
		t.Fatalf("series = %+v", s)
	}
	// The zones are honoured: 14:01-06:00 is 20:01Z, one minute after the first.
	if got := s.Times[1].Sub(s.Times[0]); got != time.Minute {
		t.Errorf("second record is %v after the first, want 1m", got)
	}
}

func TestReadSeriesReadsAHistoryExportNewestFirstAndSkipsNonOkRecords(t *testing.T) {
	t.Parallel()
	// The columns of a history read as ITableToCsv with encodeToString cells,
	// newest first as n4_read_history asks for them.
	s, err := ReadSeries(strings.NewReader("\ufeffTimestamp,Value,Status\n" +
		"2026-11-15T20:03:00.000-06:00,24.0,{ok}\n" +
		"2026-11-15T20:02:00.000-06:00,-,\"{stale,down}\"\n" +
		"2026-11-15T20:01:00.000-06:00,22.0,{ok}\n" +
		"2026-11-15T20:00:30.000-06:00,99.0,{fault}\n" +
		"2026-11-15T20:00:00.000-06:00,21.0,{ok}\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Values; len(got) != 3 || got[0] != 21 || got[1] != 22 || got[2] != 24 {
		t.Errorf("values = %v, want the ok records oldest first: 21 22 24", got)
	}
	if s.Skipped != 2 {
		t.Errorf("skipped = %d, want the stale and the fault record", s.Skipped)
	}
}

func TestReadSeriesNamesTheRowAndColumnOfABadFile(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct{ text, want string }{
		"a time without a zone": {"time,value\n2026-11-15T20:00:00,1\n",
			`row 2, column 1 (time): "2026-11-15T20:00:00" is not a time with a zone`},
		"a display time": {"Timestamp,Value,Status\n15 nov 2026 20:00:35 CST,1.0,{ok}\n",
			`row 2, column 1 (Timestamp): "15 nov 2026 20:00:35 CST" is not a time with a zone`},
		"a localized value": {"time,value\n2026-11-15T20:00:00Z,1\n2026-11-15T20:01:00Z,\"21,5\"\n",
			`row 3, column 2 (value): "21,5" is not a number`},
		"a value that is not finite": {"time,value\n2026-11-15T20:00:00Z,NaN\n",
			`row 2, column 2 (value): "NaN" is not a number`},
		"a short row": {"time,value\n2026-11-15T20:00:00Z\n",
			"row 2: 1 field(s), the header has 2"},
		"no time column": {"when,value\n2026-11-15T20:00:00Z,1\n",
			"row 1: the header names no time column (time or timestamp)"},
		"no value column": {"time,reading\n2026-11-15T20:00:00Z,1\n",
			"row 1: the header names no value column (value)"},
		"no ok record": {"time,value,status\n2026-11-15T20:00:00Z,1,{stale}\n",
			"no record has status ok (1 skipped)"},
		"empty": {"", "row 1: no header"},
		"a broken quote": {"time,value\n2026-11-15T20:00:00Z,\"1\n",
			"row 2 (line "},
	} {
		_, err := ReadSeries(strings.NewReader(c.text))
		if err == nil || !strings.HasPrefix(err.Error(), c.want) || strings.Contains(err.Error(), "\n") {
			t.Errorf("%s: err = %v, want one line starting %q", name, err, c.want)
		}
	}
}
