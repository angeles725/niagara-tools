package readtools

import (
	"reflect"
	"strings"
	"testing"
)

func TestEmailAddressesReadsEveryListForm(t *testing.T) {
	got := emailAddresses(" a@x.com;b@y.org|Ops Team , Night Shift <c@z.mx>;;")
	if want := []string{"a@x.com", "b@y.org", "c@z.mx"}; !reflect.DeepEqual(got, want) {
		t.Errorf("addresses %q", got)
	}
	if got := emailAddresses(""); len(got) != 0 {
		t.Errorf("empty list %q", got)
	}
}

func TestRedactDropsCredentialsFromURLsAndCutsLongText(t *testing.T) {
	got := redact("https://user:pw@api.telegram.org/bot123456:AA-bb_cc/sendMessage?chat_id=9&token=zz&x=1")
	if got != "https://<redacted>@api.telegram.org/bot<redacted>/sendMessage?chat_id=9&token=<redacted>&x=1" {
		t.Errorf("redacted %q", got)
	}
	if got := redact(strings.Repeat("é", 250)); len([]rune(got)) != settingMaxRunes+3 || !strings.HasSuffix(got, "...") {
		t.Errorf("cut to %d runes", len([]rune(got)))
	}
}
