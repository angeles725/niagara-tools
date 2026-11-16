package readtools

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/license"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// healthLicense is the license section: the licences of the licence manager page with their
// days left from the day of system (the system section just read), the nearest expiry, what
// expired, and a warning (licenseWarnings). A page that did not answer, or is no licence page,
// is {error} saying which: never an empty licence. err is only an error that fails the whole
// call (an authentication failure).
func (t *Tools) healthLicense(timeout int, system *pyjson.Object) (*pyjson.Object, error) {
	reply, info, notes, err := t.licenseRead(timeout)
	if err != nil {
		return t.sectionError(err)
	}
	if info == nil {
		status, _ := lookup(reply, "status").(string)
		why, known := licenseNoPageWhy[status]
		if !known {
			why = licenseNoPageOther
		}
		return obj("error", fmt.Sprintf(licenseNoPage, status, lookup(reply, "http_status"), why)), nil
	}
	clock := stationClock(system)
	list := licenseEntries(info, clock, licenseFiles{}, false)
	out := obj("licenses", list.items)
	if info.LicenseCount > len(list.items) {
		out.Set("licenses_count", info.LicenseCount)
	}
	setExpiries(out, info, clock)
	if warnings := licenseWarnings(info, clock); len(warnings) > 0 {
		out.Set("warning", joinNotes(warnings))
	}
	if !clock.known {
		notes = append(notes, clock.why)
	}
	if len(notes) > 0 {
		out.Set("note", joinNotes(notes))
	}
	out.Set("source", licenseSource)
	return out, nil
}

// licenseWarnings are what the operator should hear of unasked: a nearest expiry within
// licenseWarnDays of the station's day or already past (not when nothing is dated or that day
// is unknown), licences the station does not report as valid, and expiries that are not dates.
func licenseWarnings(info *license.Info, clock licenseClock) (warnings []string) {
	if nearest, _, ok := license.Nearest(info.Dated()); ok && clock.known {
		switch days, _ := license.DaysLeft(nearest.Expires, clock.today); {
		case days < 0:
			warnings = append(warnings, fmt.Sprintf(licenseExpiredWarning, nearest.Kind, nearest.Name, nearest.Expires, -days))
		case days <= licenseWarnDays:
			warnings = append(warnings, fmt.Sprintf(licenseNearWarning, nearest.Kind, nearest.Name, nearest.Expires, days, licenseWarnDays))
		}
	}
	var invalid []string
	for _, e := range info.Licenses {
		if !e.Valid() {
			invalid = append(invalid, fmt.Sprintf("%s (%s)", e.Name, e.State))
		}
	}
	if len(invalid) > 0 {
		warnings = append(warnings, fmt.Sprintf(licenseInvalidWarning, strings.TrimSpace(someNames(invalid))))
	}
	if n := info.UnreadExpiries(); n > 0 {
		warnings = append(warnings, fmt.Sprintf(licenseUnreadWarning, n))
	}
	return warnings
}
