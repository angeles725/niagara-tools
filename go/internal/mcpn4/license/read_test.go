package license

import (
	"strings"
	"testing"
	"time"
)

func TestReadTakesALineWholeOrCountsIt(t *testing.T) {
	t.Parallel()
	info := Read(strings.Join([]string{"HostId=One", "HostId=Two", "Licenses (trial) (other)",
		"A.license [expires: 2020-01-01] {invalid} B.license [expires: never] {valid}", "C.license [expires: never] {valid}",
		"D.license [expires: never]", "Features", "tridium:a [expires: 2020-01-01] {invalid}", "noColon [expires: never]",
		"tridium:b [expires: ]", "tridium:c [expires: never]", "k=v", "Licenses", "Z.license [expires: never] {valid}"}, "\n"))
	if info.HostID != "One" || info.LicenseKind != "trial" || len(info.Licenses) != 1 || info.Licenses[0].Name != "C.license" ||
		len(info.Features) != 1 || info.Features[0].Name != "tridium:c" || info.Unparsed != 8 {
		t.Errorf("host %q, kind %q, licences %v, features %v, unparsed %d", info.HostID, info.LicenseKind, info.Licenses, info.Features, info.Unparsed)
	}
}

func TestReadBoundsWhatItKeeps(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("A", 4096)
	lines := []string{"HostId=" + long, "Licenses (" + long + ")"}
	for n := 0; n < MaxEntries+7; n++ {
		lines = append(lines, long+" [expires: "+long+"] {"+long+"}")
	}
	lines = append(lines, "Features", "v:"+long+" [expires: never]")
	for n := 0; n < MaxFeatureAttrs+5; n++ {
		lines = append(lines, long+"="+long)
	}
	info := Read(strings.Join(lines, "\n"))
	e, f := info.Licenses[0], info.Features[0]
	for what, got := range map[string]int{"host id": len(info.HostID), "kind": len(info.LicenseKind), "name": len(e.Name),
		"expiry": len(e.Expires), "state": len(e.State), "feature": len(f.Name)} {
		if got != MaxNameRunes+len(CutMarker) {
			t.Errorf("%s kept at %d characters", what, got)
		}
	}
	if len(info.Licenses) != MaxEntries || info.LicenseCount != MaxEntries+7 || len(f.Attrs) != MaxFeatureAttrs || f.AttrsCut != 5 ||
		len(f.Attrs[0][0]) != MaxKeyRunes+len(CutMarker) || len(f.Attrs[0][1]) != MaxValueRunes+len(CutMarker) {
		t.Errorf("%d licences of %d, %d attributes (+%d)", len(info.Licenses), info.LicenseCount, len(f.Attrs), f.AttrsCut)
	}
	if got := Clip("ñandú", 3); got != "ñan"+CutMarker {
		t.Errorf("Clip counts bytes, not characters: %q", got)
	}
}

func TestDaysLeftHasNoLongestSpan(t *testing.T) {
	t.Parallel()
	today := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)
	for expires, want := range map[string]int{"9999-12-31": 2912124, "0001-01-01": -739934, "2400-01-01": 136282, "1700-01-01": -119387,
		"2028-03-01": 472} {
		if got, ok := DaysLeft(expires, today); !ok || got != want {
			t.Errorf("%s = %d %v, want %d", expires, got, ok, want)
		}
	}
	for _, text := range []string{"2026-11-15", "2026-11-15T", "2026-11-15garbage", "2026-02-30T00:00:00Z", "null", ""} {
		if _, ok := StationDay(text); ok {
			t.Errorf("%q read as a station time", text)
		}
	}
	if day, ok := StationDay("2026-11-15T23:59:59.999+14:00"); !ok || !day.Equal(today) {
		t.Errorf("station day = %v %v", day, ok)
	}
}
