package readtools

import (
	"reflect"
	"testing"
)

// The tokens bog-nav alarm-text judges (niagara-go internal/bognav/alarmtext.go unresolvedTokens):
// the same text must give the same tokens here.
func TestUnresolvedAlarmTokensMatchBogNav(t *testing.T) {
	cases := map[string][]string{
		"Over amps %parent.parent.Ctl.amps1% A":         {"%parent.parent.Ctl.amps1%"},
		"%parent% and %out.value%":                      {"%parent%", "%out.value%"},
		"High %alarmData.presentValue% (%sourceState%)": nil,
		"%lexicon(alarm:hi)% 100%%":                     nil,
		"%alarmClass% %timestamp% %uuid%":               nil,
		"unterminated %parent.out":                      nil,
		"":                                              nil,
	}
	for text, want := range cases {
		if got := unresolvedAlarmTokens(text); !reflect.DeepEqual(got, want) {
			t.Errorf("%q: tokens %q, want %q", text, got, want)
		}
	}
}

// empty-tonormal needs toOffnormalText set: a fault text alone does not fire it (bog-nav's rule).
func TestEmptyToNormalNeedsAnOffnormalText(t *testing.T) {
	if rows, _, n := alarmTextFindings("/A", map[string]string{"toFaultText": "Fault"}); len(rows) != 0 || n != 0 {
		t.Errorf("fault text only: %v", rows)
	}
	if _, _, n := alarmTextFindings("/A", map[string]string{"toOffnormalText": "High"}); n != 1 {
		t.Errorf("offnormal without normal: %d empty-tonormal", n)
	}
}

// The select's text columns are alarmTextSlots in their order, the positions alarmTextFields
// names: the query live 4.15.6.20 answered is unchanged.
func TestAlarmTextQueryFollowsTheSlots(t *testing.T) {
	const want = "select slotPath, toOffnormalText, toFaultText, toNormalText from alarm:AlarmSourceExt"
	if alarmTextQuery != want {
		t.Errorf("query %q, want %q", alarmTextQuery, want)
	}
	if !reflect.DeepEqual(alarmTextFields[1:], alarmTextSlots) || alarmTextFields[0] != "path" {
		t.Errorf("fields %q do not follow the slots %q", alarmTextFields, alarmTextSlots)
	}
}
