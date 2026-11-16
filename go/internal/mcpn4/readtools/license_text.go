package readtools

import "github.com/angeles725/niagara-tools/go/internal/mcpn4/license"

// What n4_license_info and the licence section of n4_station_health say (data only).

// The notes and hints of a licence reply.
const (
	licenseNotRecognizedHint = "the page at " + licenseSource + " has neither a HostId line nor a Certificates, Licenses, Features or Brand section: this Niagara version lays it out differently (read it with n4_spy_page)"
	licenseMissing           = "section(s) not found on the page: %s"
	licenseUnparsed          = "%d line(s) matched no known pattern and are not in this reply (read the page with n4_spy_page)"
	licenseCut               = "the page was cut (%s): features, or whole sections, may be missing"
	licenseUnreadExpiry      = "%d expiry value(s) are neither never nor a date (YYYY-MM-DD): they are returned as written, with no days_left, and are in neither nearest_expiry nor expired"
	licenseManyEntries       = "the page has %d %s; only the first %d were read"
	licenseListCut           = "the %s list was cut to fit the reply"
	licenseNoClock           = "days_left counts from the station's own " + systemTimeSlot + " (SystemService), which could not be read as a whole timestamp: %s"
	licenseNoClockSlot       = "the slot is absent or not a time"
	andMore                  = " and %d more"
)

// Why generated and version are missing, and what is said of a licence whose file was not used.
const (
	licenseNoHome        = "generated and version are attributes of the licence files of the Niagara installation (security/licenses/*.license), which the licence manager page does not carry: start the server with --niagara-home %s=PATH (the installation directory, reachable from this machine) to add them"
	licenseNoDir         = "generated and version were not read: the directory security/licenses under the --niagara-home of station %s could not be listed"
	licenseFilesNote     = "generated and version are missing where the licence file could not be used: %s"
	licenseFileAbsent    = "no file of that name under security/licenses"
	licenseFileOtherHost = "issued for another host than this station's"
	licenseFileCapHit    = "; at most %d licence files are read in one call"
)

// licenseFileReasons are the reasons a licence has no file facts, in the order the note lists them.
var licenseFileReasons = []string{license.SkipNotLicense, license.SkipLarge, license.SkipNotRegular, license.SkipCase,
	license.SkipUnreadable, license.SkipCap, licenseFileOtherHost, licenseFileAbsent}

// What the maintenance entry says of itself.
const (
	maintenancePerFeature = " sma_attributes lists, per feature, its sma.* attributes; sma_exempt_features names the features whose sma.exempt is true. The exemption is stated per feature (that one feature needs no active agreement) and says nothing of the licence as a whole. feature is a feature named after maintenance, when the licence has one. sma_truncated says a list was cut (sma_feature_count and sma_exempt_count are whole)."
	maintenanceNote       = "The licence manager page states no end date of the software maintenance agreement (SMA): the vendor documentation shows it as the maintenanceExpiration attribute of a licence file, which this server reads only when started with --niagara-home for this station." + maintenancePerFeature
	maintenanceFromFile   = "expires is the maintenanceExpiration attribute of the licence file named in license (the earliest one, when several state it), read from the local installation (--niagara-home); days_left counts from the station's date." + maintenancePerFeature
	maintenanceNoDate     = "The licence files read from the local installation (--niagara-home) state no maintenanceExpiration, and the licence manager page carries no end date of the software maintenance agreement (SMA): expires is null because the licence does not state it." + maintenancePerFeature
)

// What the license section of n4_station_health says: its error per status of the page read
// (licenseNoPageOther for any other), and its warnings.
const (
	licenseNoPage         = "the licence manager page was not read (status %s, HTTP %v): %s"
	licenseNoPageOther    = "read it with n4_spy_page path=" + license.Page
	licenseNearWarning    = "the nearest expiry (%s %s, %s) is %d day(s) away on the station's date: within the %d-day warning window"
	licenseExpiredWarning = "the nearest expiry (%s %s, %s) expired %d day(s) ago on the station's date"
	licenseInvalidWarning = "licence(s) the station reports as not valid: %s"
	licenseUnreadWarning  = "%d expiry value(s) could not be read as a date, so their days left are unknown (see n4_license_info)"
)

var licenseNoPageWhy = map[string]string{
	"forbidden":          "it needs a station user allowed to read spy pages",
	"not_found":          "the station has no page at " + licenseSource + " (another Niagara version, or the licence manager is not there)",
	spyNotHTML:           "the station answered something that is not an HTML page",
	spyParseTimeout:      "the page was fetched but not parsed within the timeout",
	licenseNotRecognized: "the page answered but is not a licence manager page in the known layout (read it with n4_spy_page)",
}
