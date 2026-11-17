package bql_test

// The encoded history settings n4_histories_audit reads: BCapacity, BCollectionInterval, a
// relative span and a BAbsTime, each as its encodeToString text.

import (
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
)

func TestParseCapacityReadsEveryRestriction(t *testing.T) {
	for _, c := range []struct {
		text string
		want bql.Capacity
		ok   bool
	}{
		{"1:1000", bql.Capacity{RestrictBy: bql.CapacityRecords, Max: 1000}, true},
		{"2:1048576", bql.Capacity{RestrictBy: bql.CapacityBytes, Max: 1048576}, true},
		{"0:0", bql.Capacity{RestrictBy: bql.CapacityUnlimited}, true},
		{" 1:500 ", bql.Capacity{RestrictBy: bql.CapacityRecords, Max: 500}, true},
		{"1000 records", bql.Capacity{}, false},
		{"3:10", bql.Capacity{}, false},
		{"1:-5", bql.Capacity{}, false},
		{"", bql.Capacity{}, false},
	} {
		got, ok := bql.ParseCapacity(c.text)
		if got != c.want || ok != c.ok {
			t.Errorf("ParseCapacity(%q) = %v, %v; want %v, %v", c.text, got, ok, c.want, c.ok)
		}
	}
}

func TestParseCollectionIntervalTellsIntervalFromIrregular(t *testing.T) {
	for _, c := range []struct {
		text      string
		irregular bool
		every     time.Duration
		ok        bool
	}{
		{"false:300000", false, 5 * time.Minute, true},
		{"true:0", true, 0, true},
		{"false:0", false, 0, false},
		{"300000", false, 0, false},
		{"maybe:1000", false, 0, false},
	} {
		irregular, every, ok := bql.ParseCollectionInterval(c.text)
		if irregular != c.irregular || every != c.every || ok != c.ok {
			t.Errorf("ParseCollectionInterval(%q) = %v, %v, %v", c.text, irregular, every, ok)
		}
	}
}

func TestParseSpanAndAbsTime(t *testing.T) {
	if d, ok := bql.ParseSpan("24h"); !ok || d != 24*time.Hour {
		t.Errorf("ParseSpan(24h) = %v, %v", d, ok)
	}
	if d, ok := bql.ParseSpan("7d"); !ok || d != 7*24*time.Hour {
		t.Errorf("ParseSpan(7d) = %v, %v", d, ok)
	}
	for _, bad := range []string{"0h", "1w", "", "24"} {
		if _, ok := bql.ParseSpan(bad); ok {
			t.Errorf("ParseSpan(%q) accepted", bad)
		}
	}
	at, ok := bql.ParseAbsTime("2026-11-16T10:05:00.000-06:00")
	if !ok || !at.Equal(time.Date(2026, 11, 16, 16, 5, 0, 0, time.UTC)) {
		t.Errorf("ParseAbsTime = %v, %v", at, ok)
	}
	for _, bad := range []string{"null", "", "16-Nov-26 10:05 AM CST"} {
		if _, ok := bql.ParseAbsTime(bad); ok {
			t.Errorf("ParseAbsTime(%q) accepted", bad)
		}
	}
}
