package license

// LicensesDir is where a Niagara installation keeps its licence files, below its home, and
// fileSuffix their extension. A file over maxFileBytes is not read, nor more than MaxFiles
// files in one call.
const (
	LicensesDir  = "security/licenses"
	fileSuffix   = ".license"
	maxFileBytes = 1 << 20
	MaxFiles     = 100
	utf8BOM      = "\xef\xbb\xbf"
)

// The root element of a licence file and the attributes of it that are read (the vendor
// documentation, "About license files"): when it was generated, the highest Niagara version it
// allows, the end of the maintenance agreement, and the host it was issued for (compared only).
const (
	rootTag         = "license"
	generatedAttr   = "generated"
	versionAttr     = "version"
	maintenanceAttr = "maintenanceExpiration"
	hostIDAttr      = "hostId"
)

// Why a licence file that is there was not read.
const (
	SkipNotRegular = "not a regular file (a link, a directory or a device)"
	SkipCase       = "its name differs in letter case (the extension must be .license)"
	SkipCap        = "past the cap of licence files read in one call"
	SkipLarge      = "larger than 1 MiB"
	SkipUnreadable = "could not be opened"
	SkipNotLicense = "not readable as a licence file"
)
