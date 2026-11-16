package ntplmanifest

// ID is the lint name n4lint accepts and the check column of a row that names no sub-check.
const ID = "lint-ntpl-manifest-deps"

// What a template archive must hold, and where a station template lives (types/distribution.md § 8).
const (
	bogEntry      = "template.bog"
	manifestEntry = "template-manifest.xml"
	stationDir    = "newStations"
	stationAttr   = "isStation"
	rootElement   = "template" // the manifest's root element
	pxSuffix      = ".px"      // a Px file inside the archive, the extension in any letter case
)

// The bounds of one archive read: a .ntpl is input nobody vouches for.
const (
	maxArchive   = 32 << 20  // bytes of the .ntpl on disk
	maxEntries   = 10000     // entries in its directory
	maxManifest  = 4 << 20   // inflated bytes of template-manifest.xml
	maxUsesRead  = 32 << 20  // inflated bytes of template.bog and of each *.px entry (the ntpl-uses read)
	maxUsesTotal = 128 << 20 // inflated bytes of all those entries of one archive together
)

// The sub-checks, as the check column prints them.
const (
	checkEntries  = "ntpl-entries"
	checkManifest = "ntpl-manifest"
	checkDeps     = "ntpl-deps"
	checkStation  = "ntpl-station"
	checkLink     = "ntpl-link"
	checkUses     = "ntpl-uses"
)

// The command line and the environment.
const (
	usage      = "usage: lint-ntpl-manifest-deps.sh [--niagara-home <dir>] <module-root>"
	homeOption = "--niagara-home"
	homeEnv    = "NIAGARA_HOME"
	modulesDir = "modules" // under the installation: one <module>.jar per module part it ships
	jarSuffix  = ".jar"
)

// templateNames select a template archive by its name: a station or component template (.ntpl) and an
// application template (.napl, the same zip: types/logic-authoring.md § Shipping a template module), the
// extension in any letter case.
var templateNames = []string{"*.[nN][tT][pP][lL]", "*.[nN][aA][pP][lL]"}

// Every message a row carries (the text after "<path>: ").
const (
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
	wrongRoot        = manifestEntry + ": the root element is <%s>, not <" + rootElement + "> -- not a template manifest"
	namelessDep      = manifestEntry + ":%d: a <dependency> with no name"
	strayDep         = manifestEntry + ":%d: a <dependency> outside <dependencies> (no reader takes it for a dependency)"

	// ntpl-deps: FAIL with the installation's module list, WARN without it
	undeclared    = "manifest dependencies neither declared (api/nre/implementation) in any *.gradle.kts under %s nor shipped by the installation %s: %s"
	undeclaredAny = "manifest dependencies not declared (api/nre/implementation) in any *.gradle.kts under %s: %s"
	noHome        = "the installation's module list is not known, so a stock module cannot be told from a missing one: pass " + homeOption +
		" <dir> or set " + homeEnv + " (the ntpl-deps rows are WARN until then)"

	// ntpl-link: WARN, a symlink whose real target is outside the module root (%s is that target)
	linkOutside = "a symlink to %s, outside the module root -- not followed, nothing behind it is checked"

	// ntpl-uses: FAIL with the installation's module list, WARN without it (%s is the entry, then the modules)
	usesMissing    = "%s uses %s: no <dependency> of " + manifestEntry + " names them, and the installation %s does not ship them"
	usesMissingAny = "%s uses %s: no <dependency> of " + manifestEntry + " names them"
	entryBadXML    = "%s is not well-formed XML: %s"
	entryTooLarge  = "%s inflates past %d bytes -- not read"
	entryUnread    = "cannot read %s: %v"
	usesBudget     = "the bog and Px entries inflate past %d bytes in all, the bound of one archive -- %s and every entry after it not read"

	notAStation  = "a template under " + stationDir + "/ is a station template, but its manifest root <%s> lacks " + stationAttr + "=\"true\""
	scanFailure  = "cannot scan %s: %v"
	partialWalk  = "cannot list every file under %s: %v"
	settingsFile = "settings.gradle.kts"

	// an installation path that cannot give a module list (exit 3; %s is homeOption or homeEnv, then the path)
	homeNoModules = "%s %s: no " + modulesDir + "/ directory (not a Niagara installation)"
	homeUnread    = "%s %s: cannot list " + modulesDir + "/: %v"

	// --niagara-home "" (a usage error: the lint's usage exit, 3)
	emptyHome = ID + ": " + homeOption + ": an empty value names no installation\n" + usage + "\n"
)

// manifestAlt is the manifest name the Templates guide spells (niagara-help Templates, "Example
// template.manifest.xml file"); an archive without manifestEntry is read through it.
const manifestAlt = "template.manifest.xml"

// The bounds of the templates of one template module jar: each inflates to at most maxJarTemplate bytes (the
// bound of a .ntpl on disk), all together to at most maxJarTemplates. Variables, so a test can lower them.
var (
	maxJarTemplate  int64 = maxArchive
	maxJarTemplates int64 = maxUsesTotal
)

// The rows of a template module jar.
const (
	noJarTemplates      = "no *.ntpl or *.napl entry in the jar (nothing to check)"
	jarTemplateTooLarge = "inflates past %d bytes -- not read"
	jarTemplateUnread   = "cannot inflate the template: %v -- not read"
	jarBudget           = "the templates of the jar inflate past %d bytes in all -- this one and every one after it not read"
	// ntpl-deps of a template in a jar, which declares no build dependencies: FAIL with the installation's
	// module list, WARN without it
	jarUnshipped    = "manifest dependencies not shipped by the installation %s (a template module jar declares no build dependencies): %s"
	jarUnshippedAny = "manifest dependencies a template module jar does not declare, and the installation's module list is not known: %s"
)
