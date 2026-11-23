package scaffoldmodule

// Every text scaffold-module prints. The tool name in the usage lines is the launcher's.
const (
	usageText = "usage: scaffold-module.sh <ModuleName> <out-dir> --vendor <v> [--target-version <x.y>] [--plugin-version <x.y.z>] [--type <logic|dashboard|driver|library>]\n" +
		"       scaffold-module.sh add-concern <Name> --root <module-root> [--part rt|ux|wb|se] [--adapter]\n"

	unknownOption  = "scaffold-module: unknown option: %s\n"
	unknownType    = "scaffold-module: unknown --type: %s (must be logic, dashboard, driver or library)\n"
	nameFirst      = "scaffold-module: ModuleName must start with an uppercase letter: %s\n"
	nameAlnum      = "scaffold-module: ModuleName must contain only alphanumeric characters: %s\n"
	vendorSegment  = "scaffold-module: --vendor must be one Java package segment once lowercased (an ASCII letter or _ first, then ASCII letters, digits or _): %s\n"
	vendorRequired = "scaffold-module: --vendor is required: the module's vendor, also its Java package segment once lowercased (e.g. --vendor Acme)\n"
	targetShape    = "scaffold-module: --target-version must be a Niagara release <major>.<minor>, digits only (e.g. 4.15): %s\n"
	pluginShape    = "scaffold-module: --plugin-version must be a plugin version <major>.<minor>.<patch>, digits only (e.g. 7.6.22): %s\n"

	toolbeltUnset      = "scaffold-module: " + ToolbeltEnv + " is unset (run through toolbelt/scaffold-module.sh)\n"
	fixtureMissing     = "scaffold-module: fixture skeleton not found: %s\n"
	outDirFault        = "scaffold-module: cannot create out-dir: %s\n"
	destExists         = "scaffold-module: destination already exists: %s\n"
	stationTestMissing = "scaffold-module: station-test template not found: %s\n"
	ledgerMissing      = "scaffold-module: version-ledger template not found: %s\n"
	writeFault         = "scaffold-module: %v\n"
	interruptedLine    = "scaffold-module: interrupted (%s)\n"
	partialRemoved     = "scaffold-module: removed the partial %s\n"
	partialStays       = "scaffold-module: could not remove the partial %s (%v): its files stay\n"
	cannotRun          = "scaffold-module: cannot run %s: %s\n"

	// emittedLines takes the ModuleName and the destination.
	emittedLines = "scaffold-module: emitted %s -> %s\n" +
		"scaffold-module: signing: set niagaraSigningAlias + niagaraSigningProfile (gradle properties) before the first build\n"
)
