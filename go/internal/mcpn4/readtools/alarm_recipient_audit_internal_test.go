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

func TestCutSettingCutsLongTextByRunes(t *testing.T) {
	if got := cutSetting(strings.Repeat("é", 250)); len([]rune(got)) != settingMaxRunes+3 || !strings.HasSuffix(got, "...") {
		t.Errorf("cut to %d runes", len([]rune(got)))
	}
	if got := cutSetting("ops@plant.example"); got != "ops@plant.example" {
		t.Errorf("short text %q", got)
	}
}
