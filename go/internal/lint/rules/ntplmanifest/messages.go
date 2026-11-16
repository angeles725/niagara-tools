package ntplmanifest

// ID is the lint name n4lint accepts and the check column of a row that names no sub-check.
const ID = "lint-ntpl-manifest-deps"

// What a template archive must hold, and where a station template lives (types/distribution.md § 8).
const (
	bogEntry      = "template.bog"
	manifestEntry = "template-manifest.xml"
	stationDir    = "newStations"
	stationAttr   = "isStation"
)

// The bounds of one archive read: a .ntpl is input nobody vouches for.
const (
	maxArchive  = 32 << 20 // bytes of the .ntpl on disk
	maxEntries  = 10000    // entries in its directory
	maxManifest = 4 << 20  // inflated bytes of template-manifest.xml
)

// The sub-checks, as the check column prints them.
const (
	checkEntries  = "ntpl-entries"
	checkManifest = "ntpl-manifest"
	checkDeps     = "ntpl-deps"
	checkStation  = "ntpl-station"
)

// Every message a row carries (the text after "<path>: ").
const (
	usage       = "usage: lint-ntpl-manifest-deps.sh [--] <module-root>..."
	emptyRoot   = "no *.gradle.kts and no *.ntpl under the root (wrong path: pass a module root)"
	noTemplates = "no *.ntpl under the root (nothing to check)"

	tooLarge      = "%d bytes on disk, over the %d this lint reads -- archive refused"
	notZip        = "not a zip archive (%v) -- archive refused"
	tooManyNames  = "%d entries, over the %d this lint reads -- archive refused"
	unsafeName    = "unsafe entry name %q (an absolute path, a \"..\" element, a backslash, a drive letter or a control character) -- archive refused"
	lacksEntry    = "lacks %s"
	repeatedEntry = "holds %d entries named %s (which one a reader takes is not defined)"

	manifestTooLarge = manifestEntry + " inflates past %d bytes -- not read"
	manifestUnread   = "cannot read " + manifestEntry + ": %v"
	manifestBadXML   = manifestEntry + " is not well-formed XML: %s"
	namelessDep      = manifestEntry + ":%d: a <dependency> with no name"

	undeclared   = "manifest dependencies not declared (api/nre/implementation) in any *.gradle.kts under %s: %s"
	noBuildFile  = "no *.gradle.kts under %s: its %d manifest dependencies cannot be compared"
	notAStation  = "a template under " + stationDir + "/ is a station template, but its manifest root <%s> lacks " + stationAttr + "=\"true\""
	scanFailure  = "cannot scan %s: %v"
	partialWalk  = "cannot list every file under %s: %v"
	settingsFile = "settings.gradle.kts"
)
