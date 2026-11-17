package writeops

// Unit tests of the n4_ack_alarm selection: its checks and the source normalization.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func TestParseSelectNormalizesEverySourceSpelling(t *testing.T) {
	want := "(source like 'slot:/Services/A' or source like 'slot:/Services/A/*' or " +
		"source like 'local:*slot:/Services/A' or source like 'local:*slot:/Services/A/*') and ackState = 'unacked'"
	for _, src := range []string{"station:|slot:/Services/A", "slot:/Services/A", "local:|station:|slot:/Services/A"} {
		s, err := parseSelect(obj("source", src))
		if err != nil {
			t.Fatalf("%s: %v", src, err)
		}
		if s.where != want || pyjson.Repr(s.criteria) != "{'source': 'station:|slot:/Services/A'}" || s.max != maxAckIDs {
			t.Errorf("%s: where %q criteria %s max %d", src, s.where, pyjson.Repr(s.criteria), s.max)
		}
	}
}

func TestParseSelectBuildsEveryCriterion(t *testing.T) {
	s, err := parseSelect(obj("alarm_class", "Critical", "before", "2026-11-15T00:00:00Z", "max", pyjson.IntLiteral("7")))
	if err != nil {
		t.Fatal(err)
	}
	if s.where != "alarmClass = 'Critical' and ackState = 'unacked' and timestamp.millis <= 1794700800000" || s.max != 7 {
		t.Errorf("where %q max %d", s.where, s.max)
	}
	if pyjson.Repr(s.criteria) != "{'alarm_class': 'Critical', 'before': '2026-11-15T00:00:00Z'}" {
		t.Errorf("criteria %s", pyjson.Repr(s.criteria))
	}
}

func TestParseSelectRefusals(t *testing.T) {
	for _, c := range []struct {
		v    any
		want string
	}{
		{"slot:/Services", "select must be an object"},
		{obj(), "at least one of source, alarm_class or before"},
		{obj("max", pyjson.IntLiteral("5")), "at least one of source, alarm_class or before"},
		{obj("source", "slot:/A", "x", 1), "select.x is not a selection key"},
		{obj("source", 5), "select.source must be a string"},
		{obj("source", "slot:/"), "select.source must be a slot ORD"},
		{obj("source", "slot:/A; slot:/B"), "select.source must be a slot ORD"},
		{obj("source", "slot:/A/../B"), "select.source must be a slot ORD"},
		{obj("source", "slot:/A$7c"), "must not hold a $ escape"}, // decodes to '|'
		{obj("alarm_class", "a'b"), "alarm_class must be an alarm class name"},
		{obj("alarm_class", "a$27b"), "must not hold a $ escape"}, // decodes to a quote
		{obj("before", "0h"), "select.before must be a BAbsTime"},
		{obj("source", "slot:/A", "max", pyjson.IntLiteral("0")), "select.max must be a whole number from 1 to 100"},
		{obj("source", "slot:/A", "max", pyjson.FloatNumber(2.5)), "select.max must be a whole number"},
		{obj("source", "slot:/A", "max", "3"), "select.max must be a whole number"},
	} {
		if _, err := parseSelect(c.v); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v", pyjson.Repr(c.v), err)
		}
	}
}
