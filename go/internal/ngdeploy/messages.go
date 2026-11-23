package ngdeploy

import "github.com/angeles725/niagara-tools/go/internal/modjar"

// Every line the deploy prints, apart from the usage text (usage.txt). say and warn add the "[ng-deploy] "
// prefix and the newline; fail adds "[ng-deploy] ERROR " too.
const (
	prefix      = "[ng-deploy] "
	errorPrefix = prefix + "ERROR "

	versionUnknown = "unknown"

	// options and env file
	msgUnknownFlag   = "unknown flag: %s"
	msgNeedsArgument = "%s requires an argument"
	msgInvalidMode   = "invalid --mode '%s'; valid values are A, B, C, D"
	msgFileMode      = "MODE set by the env file is not A, B, C or D"
	msgEnvFileSyntax = "env file %s" // a envfile.SyntaxError: "<path>: line N: <reason>"
	msgEnvFileRead   = "env file %s cannot be read: %s"
	msgEnvFileDir    = "env file %s is a directory"
	msgEnvFileOwned  = "env file %s: line %d: %s is set by ng-deploy and cannot be assigned"
	// msgEnvFileGuarded takes the file, the line, the key, the step it would switch off and the flag that does.
	msgEnvFileGuarded = "env file %s: line %d: %s would switch %s off, which an env file may not do: give %s on the command line"
	// msgEnvFileSchemaTool takes the file and the line of a SCHEMA_RISK_BIN assignment.
	msgEnvFileSchemaTool = "env file %s: line %d: SCHEMA_RISK_BIN would put another program in the place of the schema-risk precheck, which an env file may not do: set it in the environment of the command"
	warnEnvFileCR        = "WARN env file %s: a carriage return was stripped from the end of %d line(s), the first at line %d (CRLF line endings; save the file with LF)"
	msgNotSwitch         = "%s must be 0 or 1"
	msgKeep              = "KEEP_N (--keep) must be a whole number of backups, 1 or more"

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
	msgPurgeFailed    = "backup written (%s), but an older backup could not be purged: %s: %s"
	msgBackupCheck    = "backup failed: cannot check %s: %s"

	// annotation detection and slotomatic
	noticeNoGit         = "NOTICE: git not found; skipping annotation change detection"
	noticeNoRepo        = "NOTICE: not a git repository; skipping annotation change detection"
	noticeDiffFailed    = "NOTICE: git diff %s..HEAD failed (exit %d); skipping annotation change detection"
	noticeBadRecord     = "NOTICE: %s does not hold a full commit id; annotation changes are looked for since %s only"
	noticeUnknownRecord = "NOTICE: %s names a commit git does not know; annotation changes are looked for since %s only"
	msgStrict           = "annotation changes detected; --with-slotomatic required (strict mode)"
	msgStrictUnchecked  = "annotation change detection could not run; strict mode cannot tell whether slotomatic is needed (SLOTOMATIC_DETECTION=off skips the check)"
	msgSlotRT           = "slotomatic: running :%s-rt:slotomatic"
	msgSlotRTUX         = "slotomatic: running :%s-rt:slotomatic :%s-ux:slotomatic (-ux has @Niagara annotations)"
	msgSlotFailed       = "slotomatic failed (gradlew exited non-zero); deploy aborted"
	msgSlotSkipped      = "Slot-o-Matic skipped the file(s) above (METHODOLOGY K15: the generated region needs exactly one stamp — delete it and re-run slotomatic); deploy aborted"
	msgSlotOK           = "slotomatic ok"
	msgUxScan           = "cannot scan the -ux sources for @Niagara annotations (%s: %s); deploy aborted"

	// mirror isolation (isolate.go): the lines build.sh prints for the same rule
	msgMirrorIsolated = "mirror niagara_home: user.home isolated to %s (GRADLE_USER_HOME=%s, --no-daemon) so the signing plugin cannot read the real ~/.tridium keystore"
	msgMirrorBlank    = "mirror niagara_home path contains whitespace: %s: JAVA_TOOL_OPTIONS cannot carry -Duser.home=<path> with whitespace (the JVM splits it); move the mirror"
	msgHomeBlank      = "HOME contains whitespace: %s: JAVA_TOOL_OPTIONS cannot carry -Dmaven.repo.local=$HOME/.m2/repository with whitespace (the JVM splits it); " +
		"set -Dmaven.repo.local=<path without whitespace> in JAVA_TOOL_OPTIONS yourself, or use a HOME without whitespace"
	msgGradleProp    = "--gradle-prop needs k=v (got: %s)"
	msgPluginVersion = "--plugin-version needs <major>.<minor>.<patch> (got: %s)"

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
	msgVersionNoJar        = "version-bump precheck cannot verify %s: the build left no jar at %s"
	msgVersionOK           = "version-bump ok (%s: %s -> %s)"
	// msgVersionSameContent takes the jar and the version both jars carry.
	msgVersionSameContent = "version-bump ok (%s: vendorVersion still %s, and the same content as the deployed jar apart from the build time and host, the manifest digests and the signature files)"
	// msgVersionContent takes the jar name and why a jar could not be read (below: the path, then the reason).
	msgVersionContent = "version-bump precheck cannot verify %s: cannot compare its content with the deployed jar: %s"
	whyNotAJarFile    = "%s is not a readable jar (%s)"
	whyNoEntry        = "%s holds no entry"
	whyTwoEntries     = "%s holds two entries named %s"
	whyEntryUnread    = "%s: %s cannot be read (%s)"
	// The limits (patterns.go): the jar and maxJarContent; the jar, the file and maxDescriptor; then a
	// descriptor encoding/xml cannot read (the jar, the file, why).
	whyJarTooLarge  = "%s declares more than %d bytes of content, the most the precheck reads"
	whyFileTooLarge = "%s: %s is over %d bytes"
	whyNoRoot       = "%s: %s cannot be read as a descriptor (%s)"

	// target-version precheck (target.go)
	msgTargetOK       = "target-version ok (%s: compiled against Niagara %s, %s is Niagara %s)"
	msgTargetMismatch = "target-version precheck: %s was compiled against Niagara %s but the target installation %s is Niagara %s; nothing was copied: " +
		"build it against an installation of %s (mirror-niagara-home.sh), or deploy into one of %s"
	msgTargetJar    = "target-version precheck cannot verify %s: %v; nothing was copied"
	msgTargetTarget = "target-version precheck cannot verify the target installation: %v; nothing was copied"
	// noticeTargetNone takes the station directory and baja.jar.
	noticeTargetNone = "NOTICE: %s holds no %s, so it is not the modules directory of a Niagara installation: the Niagara version of the target is not checked"

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
	msgVerifyCount      = "verify failed: expected %s <type entries in %s, found %d"
	msgVerifyUnreadable = "verify failed: cannot read %s in %s (unzip exit %d)"
	msgVerifyOK         = "verify ok (%s: %d/%s)"
	msgCacheBuster      = "verify failed: index.html in %s does not contain ?v=%s (BUILD_ID check)"
	msgCacheBusterOK    = "verify ok (cache-buster ?v=%s found)"
	msgGateMissing      = "verify-module gate unavailable (not executable at %s); jars not verified (use --no-gate to override)"
	msgGate             = "gate: verify-module on %s"
	msgGateFailed       = "verify-module gate failed; jars not fit to deploy (use --no-gate to override)"
	msgGateOK           = "gate ok"
	// The warnings of a gate with no signer to expect: the texts build.sh prints (modjar), with the ways this
	// tool has to set the alias. warnDevSigner takes the development signer's name.
	warnNoSigner  = "WARN " + modjar.WhyNoSigner + signerRemedy
	warnDevSigner = "WARN " + modjar.WhyDevSigner + signerRemedy
	// warnSignerUnknown takes a station jar and why its signer cannot be told.
	warnSignerUnknown = "WARN the signer of %s cannot be told (%s), and the jars of <NIAGARA_HOME>/modules give none either, so the gate accepts any signature" + signerRemedy
	// warnSignerUnknownHome takes a station jar and why its signer cannot be told; the home gave the signer.
	warnSignerUnknownHome = "WARN the signer of %s cannot be told (%s), so the jars of <NIAGARA_HOME>/modules give the expected signer"
	signerRemedy          = " (set it: " + modjar.GradleEnvPrefix + modjar.AliasProperty + "=<alias> in the env file, or " + modjar.AliasProperty + "=<alias> in gradle.properties)"

	// the copy and its outcomes are in messages_copy.go; the deploy record and what follows it
	// msgRecordFailed takes the record file and one of the three reasons after it.
	msgRecordFailed = "jars copied, but the deployed commit was not recorded in %s: %s (the next annotation check cannot start at this deploy; SLOTOMATIC_DETECTION=off makes the record optional)"
	// msgRecordUnwritable is the same record found unwritable before the copy; it takes the file and whyNotWritten.
	msgRecordUnwritable = "the deployed commit cannot be recorded in %s (%s); nothing was copied (fix the file, or SLOTOMATIC_DETECTION=off makes the record optional)"
	// noticeNoRecord takes the record file, for a deploy from a directory that is not a git repository.
	noticeNoRecord  = "NOTICE: not a git repository; the deployed commit is not recorded in %s: the next annotation check has no starting point"
	whyNoHead       = "git rev-parse HEAD failed (exit %d)"
	whyHeadNotAName = "git rev-parse HEAD printed no full commit id"
	whyNotWritten   = "cannot write %s: %s"

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
