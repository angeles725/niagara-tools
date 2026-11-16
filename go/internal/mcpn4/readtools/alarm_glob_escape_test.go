package readtools_test

// source_glob and the relative time bounds reach the alarm BQL: the station's /ord/ path decodes
// $xx escapes ($27 is a quote, $20 a space, $7c the ORD separator) before it parses the BQL, so a
// glob carrying '$' could close the like literal. Live on 4.15.6: the glob
// nomatch$27$20or$20source$20like$20$27* matched every alarm record.

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
)

func TestReadAlarmsRefusesASourceGlobWithAnEscapeBeforeAnyRequest(t *testing.T) {
	for _, glob := range []string{"nomatch$27$20or$20source$20like$20$27*", "*$27*", "*$7cbql:x*", "*$u0027*", "*$41*"} {
		s := alarmServer(t, 3)
		msg := callFail(t, s, "n4_read_alarms", "source_glob", glob)
		if len(sent(s)) != 0 || !strings.Contains(msg, "source_glob") {
			t.Errorf("source_glob %q: sent %q, message %s", glob, sent(s), msg)
		}
	}
}

func TestReadAlarmsRefusesQuotedOrMalformedBoundsNamingTheField(t *testing.T) {
	for _, v := range []string{"24", "-1h", "1y", "1.5h", "0h", "24H", " 24h", "100000d", "24h' or '1'='1", "1h|x"} {
		for _, arg := range []string{"source_glob", "since", "until"} {
			if arg == "source_glob" && !strings.ContainsAny(v, "' |") {
				continue
			}
			s := alarmServer(t, 3)
			msg := callFail(t, s, "n4_read_alarms", arg, v)
			if len(sent(s)) != 0 || !strings.Contains(msg, arg) {
				t.Errorf("%s %q: sent %q, message %s", arg, v, sent(s), msg)
			}
		}
	}
}

func TestReadAlarmsResolvesARelativeBoundToNowMinusTheSpan(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.FixedZone("CST", -6*3600))
	old := bql.Now
	bql.Now = func() time.Time { return now }
	t.Cleanup(func() { bql.Now = old })
	out := callOK(t, alarmServer(t, 3), "n4_read_alarms", "since", "24h", "until", "30m", "open_only", false)
	want := fmt.Sprintf("timestamp.millis >= %d and timestamp.millis <= %d", now.Add(-24*time.Hour).UnixMilli(), now.Add(-30*time.Minute).UnixMilli())
	if field(out, "where") != want {
		t.Errorf("where %v, want %s", field(out, "where"), want)
	}
}
