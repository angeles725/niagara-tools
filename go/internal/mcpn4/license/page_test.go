package license

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/spy"
)

// licRecordedText is the text of the licence manager fixture (the layout measured on Niagara
// 4.15, cut to a dozen features with neutral values) as the spy reader gives it.
func licRecordedText(t *testing.T) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "spy", "testdata", "license-manager.html"))
	if err != nil {
		t.Fatal(err)
	}
	page, err := spy.Parse(string(body), spy.Limits{MaxText: len(body)})
	if err != nil || len(page.Cut) > 0 {
		t.Fatalf("parse: %v, cut %v", err, page.Cut)
	}
	return page.Text
}

func TestReadPlacesEveryLineOfTheRecordedPage(t *testing.T) {
	t.Parallel()
	info := Read(licRecordedText(t))
	if info.Unparsed != 0 || len(info.Missing) != 0 || !info.Recognized() {
		t.Errorf("unparsed %d, missing %v", info.Unparsed, info.Missing)
	}
	if len(info.Certificates) != 3 || len(info.Licenses) != 3 || len(info.Features) != 12 || len(info.Brand) != 5 ||
		info.LicenseKind != "perpetual" {
		t.Errorf("%d certificates, %d licences (%q), %d features, %d brand rows", len(info.Certificates),
			len(info.Licenses), info.LicenseKind, len(info.Features), len(info.Brand))
	}
	if dated := info.Dated(); len(dated) != 15 || dated[0].Kind != KindLicense {
		t.Errorf("%d dated, first %v", len(dated), dated[0])
	}
}

func TestDaysLeftCountsWholeDaysFromTheGivenDay(t *testing.T) {
	t.Parallel()
	today, ok := StationDay("2026-11-15T23:59:59.999-06:00")
	if !ok || !today.Equal(time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("station day = %v %v", today, ok)
	}
	for expires, want := range map[string]int{"2026-11-15": 0, "2026-11-14": -1, "2027-03-31": 136, "2028-02-29": 471} {
		if got, ok := DaysLeft(expires, today); !ok || got != want {
			t.Errorf("%s = %d %v, want %d", expires, got, ok, want)
		}
	}
	for _, text := range []string{Never, "", "31/03/2027", "2027-02-30"} {
		if _, ok := DaysLeft(text, today); ok {
			t.Errorf("%q read as a date", text)
		}
	}
	if _, ok := StationDay("not a time"); ok {
		t.Errorf("a text that is no time read as a day")
	}
}
