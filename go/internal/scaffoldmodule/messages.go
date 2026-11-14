package scaffoldmodule

// Every text scaffold-module prints. The tool name in the usage lines is the launcher's.
const (
	usageText = "usage: scaffold-module.sh <ModuleName> <out-dir> [--vendor <v>] [--target-version <x.y>] [--plugin-version <v>] [--type <logic|dashboard|driver>]\n" +
		"       scaffold-module.sh add-concern <Name> --root <module-root> [--part rt|ux|wb|se] [--adapter]\n"

	unknownOption = "scaffold-module: unknown option: %s\n"
	unknownType   = "scaffold-module: unknown --type: %s (must be logic, dashboard or driver)\n"
	nameFirst     = "scaffold-module: ModuleName must start with an uppercase letter: %s\n"
	nameAlnum     = "scaffold-module: ModuleName must contain only alphanumeric characters: %s\n"
	badValue      = "scaffold-module: %s value cannot be substituted (unescaped | or newline, trailing \\ or an unsupported \\ escape): %s\n"

	toolbeltUnset      = "scaffold-module: " + ToolbeltEnv + " is unset (run through toolbelt/scaffold-module.sh)\n"
	fixtureMissing     = "scaffold-module: fixture skeleton not found: %s\n"
	outDirFault        = "scaffold-module: cannot create out-dir: %s\n"
	destExists         = "scaffold-module: destination already exists: %s\n"
	stationTestMissing = "scaffold-module: station-test template not found: %s\n"
	ledgerMissing      = "scaffold-module: version-ledger template not found: %s\n"
	writeFault         = "scaffold-module: %v\n"
	cannotRun          = "scaffold-module: cannot run %s: %s\n"

	// emittedLines takes the ModuleName and the destination.
	emittedLines = "scaffold-module: emitted %s -> %s\n" +
		"scaffold-module: signing: set niagaraSigningAlias + niagaraSigningProfile (gradle properties) before the first build\n"
)
