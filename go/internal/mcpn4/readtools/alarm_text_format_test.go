package readtools

import (
	"reflect"
	"strings"
	"testing"
)

// formatTokenCases is the shared table of BFormat tokenizing cases (retro row P2b). niagara-go
// internal/bognav/alarmtext_format_test.go and niagara-tools
// go/internal/mcpn4/readtools/alarm_text_format_test.go hold the same rows: bog-nav alarm-text and
// n4_alarm_text_check must judge every text the same way. ok is false when BFormat cannot parse the
// text and renders it as "ERROR <text>", so no token is ever evaluated; findings are then one
// bad-format, else one parent-path per token ("rule token").
var formatTokenCases = []struct {
	name, text string
	ok         bool
	want       []string
	findings   []string
}{
	{"lone percent before a token", "Load at 90% of %parent.out.value%", false, nil, []string{"bad-format"}},
	{"escaped percent before a token", "Load at 90%% of %parent.out.value%", true, []string{"%parent.out.value%"}, []string{"parent-path %parent.out.value%"}},
	{"double percent literal", "100%% done", true, nil, nil},
	{"lone percent at end", "Over %parent.amps% at 90%", false, nil, []string{"bad-format"}},
	{"lone percent at start", "% load %parent.amps%", false, nil, []string{"bad-format"}},
	{"two real tokens", "%parent.a% and %out.value%", true, []string{"%parent.a%", "%out.value%"}, []string{"parent-path %parent.a%", "parent-path %out.value%"}},
	{"only literals", "plain text, 50%% and 60%%", true, nil, nil},
	{"percent inside call parentheses", "%lexicon(x:a%b)% %parent.a%", true, []string{"%parent.a%"}, []string{"parent-path %parent.a%"}},
	{"fallback script in the same pair", "%parent.a?lexicon(x:y)%", true, []string{"%parent.a?lexicon(x:y)%"}, []string{"parent-path %parent.a?lexicon(x:y)%"}},
	{"record fields and calls", "High %alarmData.presentValue% %lexicon(x:hi)% (%sourceState%)", true, nil, nil},
	{"empty text", "", true, nil, nil},
}

// The tokens n4_alarm_text_check reports follow BFormat's own parse: a lone % misaligns nothing.
func TestUnresolvedAlarmTokensFollowBFormat(t *testing.T) {
	for _, c := range formatTokenCases {
		if got := unresolvedAlarmTokens(c.text); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %q: tokens %q, want %q", c.name, c.text, got, c.want)
		}
	}
}

// formatScripts says whether BFormat can parse the text at all.
func TestFormatScriptsParses(t *testing.T) {
	for _, c := range formatTokenCases {
		if _, ok := formatScripts(c.text); ok != c.ok {
			t.Errorf("%s: %q: ok %v, want %v", c.name, c.text, ok, c.ok)
		}
	}
}

// Each text gives the findings of the shared table: bad-format for a text BFormat cannot parse.
func TestTextFindingsFollowBFormat(t *testing.T) {
	for _, c := range formatTokenCases {
		var got []string
		for _, f := range textFindings(c.text) {
			got = append(got, strings.TrimSpace(f.rule+" "+f.token))
		}
		if !reflect.DeepEqual(got, c.findings) {
			t.Errorf("%s: %q: findings %q, want %q", c.name, c.text, got, c.findings)
		}
	}
}
