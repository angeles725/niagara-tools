// Package license reads what a station's licence manager says of itself: the text of the spy
// page sysManagers/licenseManager as host id, certificates, licences, features and brand, and
// the dates of it as days left from a given day. It is pure: the caller fetches the page (spy)
// and gives the day (the station's own).
//
// The layout was measured on Niagara 4.15: a line HostId=<id>, then sections named by a line of
// their own (sectionTitles), each line of a section in one of the patterns below. It is not a
// contract, so a line is read whole or counted as unread, never guessed at.
package license

import "regexp"

// Page is the spy page read, below the spy root.
const Page = "sysManagers/licenseManager"

// section is where a line of the page belongs, in page order.
type section int

const (
	preamble section = iota
	certificates
	licenses
	features
	brand
)

// sectionTitles are the lines that open a section, in page order: a title opens its section
// only after the sections before it, so a line "Licenses" inside the feature list is no title.
var sectionTitles = []struct {
	title string
	kind  section
}{
	{"Certificates", certificates},
	{"Licenses", licenses},
	{"Features", features},
	{"Brand", brand},
}

// pageTitle is the heading of the page, a preamble line that says nothing.
const pageTitle = "Niagara Licensing"

// The line patterns. titleLine is a section title with, at most, a qualifier in parentheses
// ("Licenses (perpetual)"): the first one is kept. hostIDLine is the host id. entryLine is a
// certificate or licence and featureLine a feature: their name, expiry and state hold no
// bracket or brace, so two entries on one line match nothing. attributeLine is an attribute of
// a feature and brandLine a value of the Brand section.
var (
	titleLine     = regexp.MustCompile(`^([A-Za-z]+)(?: \(([^()]*)\).*)?$`)
	hostIDLine    = regexp.MustCompile(`^HostId=(.+)$`)
	entryLine     = regexp.MustCompile(`^([^\[\]{}]+) \[expires: ([^\[\]{}]+)\] \{([^\[\]{}]*)\}$`)
	featureLine   = regexp.MustCompile(`^([^\s:=\[\]{}]+:[^\s=\[\]{}]+) \[expires: ([^\[\]{}]+)\]$`)
	attributeLine = regexp.MustCompile(`^([^=\s]+)=(.*)$`)
	brandLine     = regexp.MustCompile(`^(\S+) = (.*)$`)
)

// validState is the state of a certificate or licence the station accepts.
const validState = "valid"

// Never is the expiry of what does not expire; dateLayout is the form of a dated one.
const (
	Never      = "never"
	dateLayout = "2006-01-02"
)

// The features and attributes that identify the installation, and the ones that speak of the
// software maintenance agreement: attributes whose key starts with smaPrefix (smaExempt is the
// one measured, smaTrue its value on an exempt feature), and a feature whose name holds
// maintenanceName.
const (
	aboutFeature    = "tridium:about"
	brandFeature    = "tridium:brand"
	ownerKey        = "owner"
	projectKey      = "project"
	brandKey        = "brandId"
	smaPrefix       = "sma."
	smaExempt       = "sma.exempt"
	smaTrue         = "true"
	maintenanceName = "maintenance"
)

// The kinds of a dated thing, in the order a tie between equal dates is named.
const (
	KindLicense     = "license"
	KindCertificate = "certificate"
	KindFeature     = "feature"
)
