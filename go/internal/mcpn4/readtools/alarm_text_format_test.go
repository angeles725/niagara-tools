package readtools

import (
	"reflect"
	"testing"
)

// formatTokenCases is the shared table of BFormat tokenizing cases (retro row P2b). niagara-go
// internal/bognav/alarmtext_format_test.go and niagara-tools
// go/internal/mcpn4/readtools/alarm_text_format_test.go hold the same rows: bog-nav alarm-text and
// n4_alarm_text_check must judge every text the same way. ok is false when BFormat cannot parse the
// text and renders it as "ERROR <text>", so no token is ever evaluated.
var formatTokenCases = []struct {
	name, text string
	ok         bool
	want       []string
}{
	{"lone percent before a token", "Load at 90% of %parent.out.value%", false, nil},
	{"escaped percent before a token", "Load at 90%% of %parent.out.value%", true, []string{"%parent.out.value%"}},
	{"double percent literal", "100%% done", true, nil},
	{"lone percent at end", "Over %parent.amps% at 90%", false, nil},
	{"lone percent at start", "% load %parent.amps%", false, nil},
	{"two real tokens", "%parent.a% and %out.value%", true, []string{"%parent.a%", "%out.value%"}},
	{"only literals", "plain text, 50%% and 60%%", true, nil},
	{"percent inside call parentheses", "%lexicon(x:a%b)% %parent.a%", true, []string{"%parent.a%"}},
	{"fallback script in the same pair", "%parent.a?lexicon(x:y)%", true, []string{"%parent.a?lexicon(x:y)%"}},
	{"record fields and calls", "High %alarmData.presentValue% %lexicon(x:hi)% (%sourceState%)", true, nil},
	{"empty text", "", true, nil},
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
