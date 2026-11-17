package readtools

import (
	"fmt"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/license"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/spy"
)

// n4_license_info: what the station's licence manager says (the spy page license.Page) in one
// reply, each expiry beside the days left from the station's own date.

// licenseRead is the licence manager page read once: the spy reply so far with its source and,
// when the page answered and is a licence page, its data and the notes of what the reading
// could not do (a cut page, missing sections, unread lines). A page that did not answer, or is
// no licence page, is a typed status with nil data.
func (t *Tools) licenseRead(timeout int) (*pyjson.Object, *license.Info, []string, error) {
	out, page, err := t.spyFetch(license.Page, spy.Limits{MaxText: licenseMaxText}, box.MaxSpyBytes, timeout)
	if err != nil {
		return nil, nil, nil, err
	}
	out.Set("source", licenseSource)
	if page == nil {
		return out, nil, nil, nil
	}
	info := license.Read(page.Text)
	if !info.Recognized() {
		out.Set("status", licenseNotRecognized)
		out.Set("hint", licenseNotRecognizedHint)
		return out, nil, nil, nil
	}
	var notes []string
	if len(page.Cut) > 0 {
		notes = append(notes, fmt.Sprintf(licenseCut, strings.Join(page.Cut, ", ")))
	}
	if len(info.Missing) > 0 {
		notes = append(notes, fmt.Sprintf(licenseMissing, strings.Join(info.Missing, ", ")))
	}
	if info.Unparsed > 0 {
		notes = append(notes, fmt.Sprintf(licenseUnparsed, info.Unparsed))
	}
	if n := info.UnreadExpiries(); n > 0 {
		notes = append(notes, fmt.Sprintf(licenseUnreadExpiry, n))
	}
	for _, many := range []struct {
		count, kept int
		what        string
	}{{info.CertificateCount, len(info.Certificates), "certificates"}, {info.LicenseCount, len(info.Licenses), "licences"}} {
		if many.count > many.kept {
			notes = append(notes, fmt.Sprintf(licenseManyEntries, many.count, many.what, many.kept))
		}
	}
	out.Set("truncated", len(page.Cut) > 0)
	return out, info, notes, nil
}

// licenseClock is the day days_left counts from: the station's, known only when its
// SystemService answered a systemTime that reads as a date; why says why not.
type licenseClock struct {
	today time.Time
	known bool
	why   string
}

// stationClock is the clock of the system section of n4_station_health (healthSystem).
func stationClock(system *pyjson.Object) licenseClock {
	if text, failed := lookup(system, "error").(string); failed {
		return licenseClock{why: fmt.Sprintf(licenseNoClock, text)}
	}
	text, _ := lookup(system, systemTimeSlot).(string)
	today, ok := license.StationDay(text)
	if !ok {
		return licenseClock{why: fmt.Sprintf(licenseNoClock, licenseNoClockSlot)}
	}
	return licenseClock{today: today, known: true}
}

// daysLeft is the days from the station's day to an expiry, nil when either is unknown.
func (c licenseClock) daysLeft(expires string) any {
	if days, ok := license.DaysLeft(expires, c.today); ok && c.known {
		return days
	}
	return nil
}

// LicenseInfo is n4_license_info.
func (t *Tools) LicenseInfo(args *pyjson.Object) (*pyjson.Object, error) {
	timeout := intArg(args, "timeout_s", 60)
	out, info, notes, err := t.licenseRead(timeout)
	if err != nil {
		return nil, err
	}
	rows := 0
	if info != nil {
		system, err := t.healthSystem()
		if err != nil {
			return nil, err
		}
		clock := stationClock(system)
		var cut []string
		rows, cut = t.licenseReply(out, info, clock, strArg(args, "feature", ""), intArg(args, "max_features", licenseMaxFeatures))
		if notes = append(notes, cut...); len(notes) > 0 {
			out.Set("note", joinNotes(notes))
		}
	}
	t.licenseUsage(out, info, args, timeout)
	t.recordRead("n4_license_info", "feature", args, rows)
	return out, nil
}
