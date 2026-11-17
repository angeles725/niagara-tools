package readtools

import (
	"testing"
	"time"
)

func TestExportLineQuotesOnlyWhatCsvNeedsAndKeepsTheDecimalPoint(t *testing.T) {
	for _, c := range []struct {
		cells [4]string
		want  string
	}{
		{[4]string{"/S/T", "2026-11-16T02:20:00.000-06:00", "21.0", "{ok}"}, "/S/T,2026-11-16T02:20:00.000-06:00,21.0,{ok}\r\n"},
		{[4]string{"/S/T", "t", "-1.5", "{fault,stale}"}, "/S/T,t,-1.5,\"{fault,stale}\"\r\n"},
		{[4]string{"/S/T", "t", "say \"hi\"", ""}, "/S/T,t,\"say \"\"hi\"\"\",\r\n"},
		{[4]string{"/S/T", "t", "a\nb", "{ok}"}, "/S/T,t,\"a\nb\",{ok}\r\n"},
	} {
		if got := exportLine(c.cells[0], c.cells[1], c.cells[2], c.cells[3]); got != c.want {
			t.Errorf("exportLine(%q) = %q, want %q", c.cells, got, c.want)
		}
	}
}

func TestExportFileNameIsTheStationAndTheUtcTime(t *testing.T) {
	at := time.Date(2026, 11, 17, 9, 5, 7, 123e6, time.FixedZone("CST", -6*3600))
	for station, want := range map[string]string{
		"Prueba_4_15": "Prueba_4_15-20261117T150507.123Z.csv",
		"My$20Site":   "My$20Site-20261117T150507.123Z.csv",
		"../x/y":      "___x_y-20261117T150507.123Z.csv",
		"":            "station-20261117T150507.123Z.csv",
	} {
		if got := exportFileName(station, at); got != want {
			t.Errorf("exportFileName(%q) = %q, want %q", station, got, want)
		}
	}
}
