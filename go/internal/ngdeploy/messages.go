package ngdeploy

// Every line the deploy prints, apart from the usage text (usage.txt). say and warn add the "[ng-deploy] "
// prefix and the newline; fail adds "[ng-deploy] ERROR " too.
const (
	prefix      = "[ng-deploy] "
	errorPrefix = prefix + "ERROR "

	versionUnknown = "unknown"

	// options and env file
	msgUnknownFlag   = "unknown flag: %s"
	msgNeedsArgument = "%s requires an argument"
	msgInvalidMode   = "invalid --mode '%s'; valid values are A, B, C"
	msgFileMode      = "MODE set by the env file is not A, B or C"
	msgEnvFileSyntax = "env file %s" // a envfile.SyntaxError: "<path>: line N: <reason>"
	msgEnvFileRead   = "env file %s cannot be read: %s"
	msgEnvFileOwned  = "env file %s: line %d: %s is set by ng-deploy and cannot be assigned"
	msgNotSwitch     = "%s must be 0 or 1"
	msgKeep          = "KEEP_N (--keep) must be a whole number of backups, 1 or more"
	msgUnknownStep   = "unknown step: %s"
	msgStepUsage     = "step usage: %s"

	// required settings
	msgRequired        = "%s is required but not set"
	msgRequiredForMode = "%s is required for mode %s"
	msgTypeCount       = "%s must be a whole number"
	msgBackslashes     = "STATION_MODULES_DIR must use WSL /mnt/c/... paths, not Windows-style backslashes"
	msgStationMissing  = "STATION_MODULES_DIR path not found: %s"

	// backup
	warnBackupSkipped = "WARN backup skipped (--no-backup); ensure the module jars are committed to git for rollback."
	msgBackup         = "backup: %s"
	msgBackupNothing  = "backup: no existing %s-{rt,ux,wb}.jar in %s; nothing to archive"
	msgBackupOK       = "backup ok: %s"
	msgBackupFailed   = "backup failed: %s"
	msgBackupCheck    = "backup failed: cannot check %s: %s"

	// annotation detection and slotomatic
	noticeNoGit        = "NOTICE: git not found; skipping annotation change detection"
	noticeNoRepo       = "NOTICE: not a git repository; skipping annotation change detection"
	msgStrict          = "annotation changes detected; --with-slotomatic required (strict mode)"
	msgStrictUnchecked = "annotation change detection could not run; strict mode cannot tell whether slotomatic is needed (SLOTOMATIC_DETECTION=off skips the check)"
	msgSlotRT          = "slotomatic: running :%s-rt:slotomatic"
	msgSlotRTUX        = "slotomatic: running :%s-rt:slotomatic :%s-ux:slotomatic (-ux has @Niagara annotations)"
	msgSlotFailed      = "slotomatic failed (gradlew exited non-zero); deploy aborted"
	msgSlotSkipped     = "Slot-o-Matic skipped the file(s) above (METHODOLOGY K15: the generated region needs exactly one stamp — delete it and re-run slotomatic); deploy aborted"
	msgSlotOK          = "slotomatic ok"
	msgUxScan          = "cannot scan the -ux sources for @Niagara annotations (%s: %s); deploy aborted"

	// build
	msgBuild       = "build: mode %s — %s"
	msgBuildFailed = "build failed (gradlew exited non-zero)"
	msgBuildOK     = "build ok"
	msgNoDeploy    = "no-deploy: jars in build/libs/"

	// version-bump precheck
	msgVersionUnreadable   = "version-bump precheck cannot verify %s: vendorVersion unreadable (built='%s' deployed='%s')"
	msgVersionSame         = "version-bump precheck: %s bytes changed but vendorVersion is still %s — bump defaultModuleVersion and rebuild"
	msgVersionDown         = "version-bump precheck: %s vendorVersion went down (%s -> %s) — a deploy must raise it; bump defaultModuleVersion above %s and rebuild"
	msgVersionIncomparable = "version-bump precheck cannot verify %s: vendorVersions are not comparable (built='%s' deployed='%s': their non-numeric suffixes differ)"
	msgVersionCheck        = "version-bump precheck cannot verify %s: cannot check %s: %s"
	msgVersionOK           = "version-bump ok (%s: %s -> %s)"

	// schema-risk precheck
	msgSchemaFirstDeploy = "schema-risk: N/A (--first-deploy: no deployed baseline)"
	msgSchemaNeeds       = "schema-risk precheck needs SCHEMA_BASELINE_DIR (the deployed -rt source baseline), or --first-deploy for a module never deployed"
	msgSchemaNotDir      = "SCHEMA_BASELINE_DIR is not a directory: %s"
	msgSchemaUnavailable = "schema-risk precheck unavailable (not executable at %s)"
	msgSchemaOK          = "schema-risk ok (SAFE)"
	warnSchemaLossy      = "WARN schema-risk LOSSY accepted (--accept-lossy)"
	msgSchemaLossy       = "schema-risk verdict LOSSY: a saved value would be lost (use --accept-lossy after checking the live bog)"
	msgSchemaOutage      = "schema-risk verdict OUTAGE: the slot change would break saved data; deploy blocked"
	msgSchemaFault       = "schema-risk precheck could not run (exit %d); deploy blocked"

	// verify and gate
	msgVerifyCount   = "verify failed: expected %s <type entries in %s, found %d"
	msgVerifyOK      = "verify ok (%s: %d/%s)"
	msgCacheBuster   = "verify failed: index.html in %s does not contain ?v=%s (BUILD_ID check)"
	msgCacheBusterOK = "verify ok (cache-buster ?v=%s found)"
	msgGateMissing   = "verify-module gate unavailable (not executable at %s); jars not verified (use --no-gate to override)"
	msgGate          = "gate: verify-module on %s"
	msgGateFailed    = "verify-module gate failed; jars not fit to deploy (use --no-gate to override)"
	msgGateOK        = "gate ok"

	// copy and after
	msgCopyFailed = "copy failed for %s"
	msgCopyOK     = "copy ok"
	msgRestart    = "station restart required (Java classes deployed)"
	msgHardReload = "browser hard-reload only (no station restart needed)"

	// msgCannotRun says why a command could not be started, or a file not written: bash printed the same
	// reason after its own "<script>: line N:".
	msgCannotRun = "ng-deploy.sh: %s: %s"
	whyNoCommand = "command not found"
)

// The warnings of several lines, each printed after the tool's prefix.
var (
	// warnAnnotations is the warning for annotation changes deployed without slotomatic.
	warnAnnotations = []string{
		"WARN: annotation changes detected in Java sources.",
		"WARN: Run with --with-slotomatic to regenerate slot code, or",
		"WARN: set SLOTOMATIC_DETECTION=off to suppress this warning.",
		"WARN: Deploying without slotomatic may cause BComponent type errors.",
	}
	// warnSlotModeB is the warning for --with-slotomatic in mode B.
	warnSlotModeB = []string{
		"WARN --with-slotomatic ignored for mode B (ux-only): ng-deploy runs slotomatic for -rt only.",
		"WARN to regenerate -ux slots, deploy with --mode A --with-slotomatic, or build via toolbelt/build.sh.",
	}
)
