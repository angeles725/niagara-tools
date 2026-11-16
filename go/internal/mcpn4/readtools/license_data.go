package readtools

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/license"
)

// What n4_license_info and the licence section of n4_station_health read and how much of it a
// reply may hold (data only; what they say is in license_text.go).

// licenseSource is where the licence facts come from; licenseMaxText is the text cap of its
// read: the byte cap of a spy read, so only that cap can cut the page.
const (
	licenseSource  = "spy:/" + license.Page
	licenseMaxText = box.MaxSpyBytes
)

// licenseNotRecognized is the status of a page that answered but is no licence manager page.
const licenseNotRecognized = "not_recognized"

// systemTimeSlot is the SystemService slot days_left counts from: the station's own clock.
const systemTimeSlot = "systemTime"

// licenseWarnDays is the window of the health warning: a nearest expiry this many days away or
// fewer is warned about.
const licenseWarnDays = 60

// licenseMaxFeatures is how many features are listed by default.
const licenseMaxFeatures = 50

// listCap bounds one list of a licence reply: at most items entries and at most bytes of their
// JSON, whichever comes first.
type listCap struct{ items, bytes int }

// The caps of the lists of a licence reply. With the string limits of the license package they
// bound a whole reply, whatever the page holds: LicenseReplyBound is that bound, in bytes of
// the JSON-RPC result, for any page up to the byte cap of a spy read. The feature list also
// stops at max_features.
var (
	capEntries  = listCap{license.MaxEntries, 32 << 10} // certificates, and licences
	capExpired  = listCap{50, 24 << 10}
	capSMA      = listCap{50, 80 << 10} // sma_attributes groups
	capExempt   = listCap{50, 24 << 10} // sma_exempt_features names
	capFeatures = listCap{2000, 80 << 10}
)

// LicenseReplyBound is the stated size bound of a whole n4_license_info reply.
const LicenseReplyBound = 2 << 20

// The bounds of a note and of the names one of its reasons lists.
const (
	licenseNoteRunes = 1000
	licenseNoteNames = 5
)
