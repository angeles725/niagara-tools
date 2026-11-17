package readtools

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/spy"
)

func TestKilobytesReadsTheSysInfoMemoryText(t *testing.T) {
	cases := []struct {
		text string
		want int64
		ok   bool
	}{
		{"1000960kb", 1000960, true},
		{" 611941kb ", 611941, true},
		{"0kb", 0, true},
		{"1000960", 0, false},
		{"-5kb", 0, false},
		{"abckb", 0, false},
		{"", 0, false},
	}
	for _, c := range cases {
		if got, ok := kilobytes(c.text); got != c.want || ok != c.ok {
			t.Errorf("kilobytes(%q) = %d, %v; want %d, %v", c.text, got, ok, c.want, c.ok)
		}
	}
}

func TestCertificateDayReadsJavaDateText(t *testing.T) {
	cases := []struct{ text, want string }{
		{"Sat Mar 31 17:59:59 CST 2027", "2027-03-31"},
		{"Wed Jan 01 00:00:00 GMT+01:00 2031", "2031-01-01"},
		{"Tue Dec 9 08:00:00 UTC 2025", "2025-12-09"},
		{"2027-03-31", ""},
		{"Sat Foo 31 17:59:59 CST 2027", ""},
		{"Sat Feb 30 17:59:59 CST 2027", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got, ok := certificateDay(c.text); got != c.want || ok != (c.want != "") {
			t.Errorf("certificateDay(%q) = %q, %v; want %q", c.text, got, ok, c.want)
		}
	}
}

func jmxTable(pool, max string) spy.Table {
	return spy.Table{Title: pool + " (KB)", Header: []string{"Type", "init", "used", "committed", "max"},
		Rows: [][]string{{"current", "1", "2", "3", max}, {"peak", "1", "2", "3", max}}}
}

// The heap maximum is the sum of the heap pools' defined maxima: a pool whose maximum the JVM
// leaves undefined (-1, printed 0 after the division by 1024) adds nothing, a non-heap pool is
// never counted, and no heap pool with a maximum is no maximum.
func TestHeapMaxSumsTheDefinedHeapPoolMaxima(t *testing.T) {
	g1 := []spy.Table{jmxTable("G1 Eden Space", "0"), jmxTable("G1 Survivor Space", "0"),
		jmxTable("G1 Old Gen", "1048576"), jmxTable("Metaspace", "0"), jmxTable("Code Cache", "245760")}
	if got, pools, ok := heapMax(g1); !ok || got != 1048576 || pools != 3 {
		t.Errorf("G1: %d, %d pools, %v", got, pools, ok)
	}
	ps := []spy.Table{jmxTable("PS Eden Space", "300000"), jmxTable("PS Survivor Space", "50000"),
		jmxTable("PS Old Gen", "700000"), jmxTable("Compressed Class Space", "1048576")}
	if got, _, ok := heapMax(ps); !ok || got != 1050000 {
		t.Errorf("PS: %d, %v", got, ok)
	}
	if _, _, ok := heapMax([]spy.Table{jmxTable("Metaspace", "0"), jmxTable("G1 Eden Space", "0")}); ok {
		t.Error("no defined heap maximum must not be a maximum")
	}
	if _, _, ok := heapMax([]spy.Table{jmxTable("PS Old Gen", "n/a")}); ok {
		t.Error("an unreadable maximum must not be a maximum")
	}
}

func TestHistoryRecordsSumsTheRecordsColumn(t *testing.T) {
	tables := []spy.Table{{Title: "LocalHistoryDatabase", Rows: [][]string{{"dbDir", "file:^history"}}},
		{Header: []string{"Name", "Records", "Total Appends", "Header Writes", "Page Writes"},
			Rows: [][]string{{"A", "100", "5", "1", "1"}, {"B", "250", "0", "0", "0"}, {"C", "x", "0", "0", "0"}}}}
	got, ok := historyRecords(tables)
	if !ok || got.histories != 3 || got.records != 350 || got.unreadable != 1 {
		t.Errorf("historyRecords = %+v, %v", got, ok)
	}
	if _, ok := historyRecords(tables[:1]); ok {
		t.Error("a page without the Records table must not be a count")
	}
}

func TestHeapFractionRoundsAndNeedsAMaximum(t *testing.T) {
	if got := heapFraction(388, 1000); got != 0.388 {
		t.Errorf("fraction = %v", got)
	}
	if got := heapFraction(2, 3); got != 0.667 {
		t.Errorf("fraction = %v", got)
	}
	if got := heapFraction(5, 0); got != nil {
		t.Errorf("fraction without max = %v", got)
	}
}

// A CertificateExpiryPoint's value is the days to the certificate's notAfter as a numeric
// point's text with its status: "329,0 {ok}" on a Spanish-locale station.
func TestExpiryDaysReadsTheNumericPointText(t *testing.T) {
	cases := []struct {
		text, status string
		want         int
		ok           bool
	}{
		{"329,0 {ok}", "ok", 329, true},
		{"10.0 {ok}", "ok", 10, true},
		{"-3,0 {down}", "down", -3, true},
		{"1.234,0 {ok}", "ok", 1234, true},
		{"1,234.0 {ok}", "ok", 1234, true},
		{"0,0 {disabled}", "disabled", 0, true},
		{"329", "", 329, true},
		{"- {fault}", "", 0, false},
		{"", "", 0, false},
	}
	for _, c := range cases {
		days, status, ok := expiryDays(c.text)
		if days != c.want || ok != c.ok || (ok && status != c.status) {
			t.Errorf("expiryDays(%q) = %d, %q, %v", c.text, days, status, ok)
		}
	}
}
