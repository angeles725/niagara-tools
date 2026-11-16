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
	if rows, _, n, _ := alarmTextFindings("/A", map[string]string{"toFaultText": "Fault"}); len(rows) != 0 || n != 0 {
		t.Errorf("fault text only: %v", rows)
	}
	if _, _, n, _ := alarmTextFindings("/A", map[string]string{"toOffnormalText": "High"}); n != 1 {
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

// A text BFormat cannot parse is one bad-format row naming its slot, counted on its own.
func TestBadFormatIsOneRowPerText(t *testing.T) {
	rows, parents, empties, bad := alarmTextFindings("/Load", map[string]string{
		"toOffnormalText": "Load at 90% of %parent.out.value%", "toNormalText": "Normal"})
	want := []any{alarmTextFinding(ruleBadFormat, "/Load", "toOffnormalText", "", "Load at 90% of %parent.out.value%")}
	if !reflect.DeepEqual(rows, want) || parents != 0 || empties != 0 || bad != 1 {
		t.Errorf("rows %v parents %d empties %d bad %d", rows, parents, empties, bad)
	}
	if s := alarmTextCounts(0, 0, 1); s != "0 parent-path, 0 empty-tonormal, 1 bad-format" {
		t.Errorf("counts %q", s)
	}
	if s := alarmTextCounts(2, 1, 0); s != "2 parent-path, 1 empty-tonormal" {
		t.Errorf("counts without bad-format %q", s)
	}
}
