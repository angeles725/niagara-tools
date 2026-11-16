package bql

// The source filter of n4_read_alarms: its clause holds no '|' (the ORD separator cuts the bql
// part there, HTTP 400 live), and its like patterns match both stored source forms of the
// component and everything below it, never a sibling that only shares the name's prefix.

import (
	"regexp"
	"strings"
	"testing"
)

var likeRe = regexp.MustCompile(`source like '([^']*)'`)

// likeMatches is whether any like pattern of clause matches s, '*' standing for any run of
// characters and the whole value having to match (BQL like).
func likeMatches(clause, s string) bool {
	for _, m := range likeRe.FindAllStringSubmatch(clause, -1) {
		parts := strings.Split(m[1], "*")
		for i, p := range parts {
			parts[i] = regexp.QuoteMeta(p)
		}
		if regexp.MustCompile("^" + strings.Join(parts, ".*") + "$").MatchString(s) {
			return true
		}
	}
	return false
}

func TestTheSourceFilterHasNoOrdSeparatorAndMatchesBothStoredForms(t *testing.T) {
	where, err := AlarmWhere(AlarmFilter{Source: "station:|slot:/Drivers"})
	if err != nil {
		t.Fatal(err)
	}
	ord := ComposeAlarmOrd(AlarmRecordsQuery, where, AlarmOrder)
	bqlPart := strings.TrimSuffix(strings.TrimPrefix(ord, AlarmScheme+"|bql:"), "|"+View)
	if strings.Contains(bqlPart, "|") || bqlPart == ord {
		t.Fatalf("the bql part of %q holds the ORD separator", ord)
	}
	for _, c := range []struct {
		source string
		want   bool
	}{
		{"slot:/Drivers", true}, {"slot:/Drivers/Net/D066", true},
		{"local:|station:|slot:/Drivers", true}, {"local:|station:|slot:/Drivers/Net/D066", true},
		{"slot:/DriversX", false}, {"local:|station:|slot:/DriversX/Net", false},
		{"slot:/Other/Drivers", false}, {"local:|station:|slot:/Services/Drivers/A", false},
	} {
		if got := likeMatches(where, c.source); got != c.want {
			t.Errorf("%s: matched %v, want %v (clause %s)", c.source, got, c.want, where)
		}
	}
}
