package buildmodule

import (
	"github.com/angeles725/niagara-tools/go/internal/modjar"
	"github.com/angeles725/niagara-tools/go/internal/reportmodule"
)

// Every line build.sh prints, byte for byte as the bash tool printed it. A message of several lines is one
// constant with its lines joined by "\n"; say and warn add the final newline.

// Usage and flags.
const (
	msgUIProfile   = "build.sh: --ui-profile must be hmi|lan|both|unknown (got: %s)"
	msgGradleProp  = "build.sh: --gradle-prop needs k=v (got: %s)"
	msgUnknownFlag = "build.sh: unknown flag %s"
)

// Inputs.
const (
	msgRootNotFound   = "build.sh: module root not found: %s"
	msgRootUnresolved = "build.sh: cannot enter module root: %s" // the path as typed
	msgGradlewNotExec = "build.sh: %[1]s/gradlew exists but is not executable (a fresh VCS worktree of a repo that stores it 100644)\n" +
		"  fix: chmod +x %[1]s/gradlew"
	msgNoGradlew  = "build.sh: no executable ./gradlew in %s or any ancestor (the module needs the gradle wrapper)"
	msgRootsBelow = "  found gradle root(s) BELOW %s — pass the GROUP dir that holds <MOD>/, not the repo root:"
	msgRootBelow  = "    %s"
	msgTryChmod   = "  try: chmod +x %s/gradlew"
	msgNoJava8    = "build.sh: Java 8 not found at %s — check 'ls /usr/lib/jvm' or set JAVA8"
	msgNoHome     = "build.sh: pass niagara_home (arg 3) or export niagara_home"
	msgNotAHome   = "build.sh: not a niagara_home (no etc/m2/repository): %s"
	msgRealHome   = "build.sh: %s is a real niagara_home (no .niagara-mirror): gradle's :jar step would replace its modules/ jars\n" +
		"  build against a mirror (toolbelt/mirror-niagara-home.sh <niagara_home> <mirror_dir>), or pass --install-into-home"
	msgNoGate = "build.sh: gate not found next to this script: %s"
)

// Build location and preflight.
const (
	msgSlowMount = "build.sh: WARN — %s (%s) is on a %s mount (WSL /mnt/<drive>) — builds run far slower than ext4.\n" +
		"  Clone the source tree under an ext4 path (e.g. ~/modulos_niagara_n4/Cliente/...) and mirror niagara_home with toolbelt/mirror-niagara-home.sh (see build-verify.md)."
	msgPreflight     = "==> preflight"
	msgPreflightFail = "build.sh: preflight FAILed — fix the environment (or --no-preflight to skip)"
)

// Gradle configuration.
const (
	msgPluginNotInM2 = "build.sh: WARN — gradlePluginVersion=%s not found under %s/etc/m2/repository\n" +
		"  SDK/plugin mismatch: the module was pinned for a different Niagara installation.\n" +
		"  If the build fails to resolve the niagara-module plugin, point niagara_home at the matching installation."
	// msgPluginPicked and msgPluginAbsent take the pin, the niagara_home and the versions it ships (pickMirrorPlugin).
	msgPluginPicked = "build.sh: NOTICE — niagaraPluginVersion=%s is not under %s/etc/m2/repository; using %s, the plugin version it ships"
	msgPluginAbsent = "build.sh: niagaraPluginVersion=%s is not under %s/etc/m2/repository and it ships %s — pass --plugin-version, " +
		"or point niagara_home at the installation the module is pinned for"
	msgSkipScaffold   = "==> skipping %s-%s: scaffold without sources (pass --profiles to force)"
	msgNoProfile      = "build.sh: no buildable profile under %[1]s/%[2]s/%[2]s-{rt,ux,wb}"
	msgProfileMissing = "build.sh: profile dir missing: %s"
	msgPluginDetected = "==> detected niagaraPluginVersion=%s from the module's gradle config"
	// What GNU grep prints for a file it cannot read, and for a match in a file holding a NUL byte.
	msgGrepUnreadable = "grep: %s: %s"
	msgGrepBinary     = "grep: %s: binary file matches"
)

// Mirror user.home isolation.
const (
	msgMirrorWhitespace = "build.sh: mirror niagara_home path contains whitespace: %s\n" +
		"  JAVA_TOOL_OPTIONS cannot carry -Duser.home=<path> with whitespace (the JVM splits it); move the mirror."
	msgHomeWhitespace = "build.sh: HOME contains whitespace: %s\n" +
		"  JAVA_TOOL_OPTIONS cannot carry -Dmaven.repo.local=$HOME/.m2/repository with whitespace (the JVM splits it);\n" +
		"  set -Dmaven.repo.local=<path without whitespace> in JAVA_TOOL_OPTIONS yourself, or use a HOME without whitespace."
	msgMirrorIsolated = "==> INFO: mirror niagara_home: user.home isolated to %s (GRADLE_USER_HOME=%s, --no-daemon) so the signing plugin cannot read the real ~/.tridium keystore"
)

// Baseline restore. %[1]s module-profile jar name without ".jar", %[2]s niagara_home.
const (
	msgRemoved          = "  Removed the newly-installed %[1]s.jar from %[2]s/modules (none was deployed before this build)."
	msgRemoveFailed     = "  FAILED TO REMOVE the newly-installed %[1]s.jar (%[3]s) — %[2]s/modules still holds the jar that failed the gate; remove it manually before retrying."
	msgRestored         = "  Restored the previously-deployed %[1]s.jar into %[2]s/modules%[3]s"
	msgRestoreFailedOld = "  FAILED TO RESTORE %[1]s.jar (%[3]s); %[2]s/modules still holds the pre-build jar, so a retry is still safe."
	msgRestoreFailedNew = "  FAILED TO RESTORE the previously-deployed %[1]s.jar (%[3]s) — %[2]s/modules no longer holds it (likely the NEW jar); restore it manually before retrying."
	restoreNoReason     = "the copy failed"
	restoredPlain       = "."
	restoredRetry       = " — a retry will catch this again."
)

// An interrupt between the baseline snapshot and a passing chain. %[1]s the signal, %[2]s niagara_home.
const (
	msgInterrupted           = "build.sh: interrupted (%[1]s) before the gate passed; putting modules/ back as it was before the build"
	msgInterruptedNoSnapshot = "build.sh: interrupted (%[1]s) under --no-drift-check: no pre-build snapshot, %[2]s/modules is left as the build left it"
)

// The mirror jar a build replaced (retro row B6). keepReplaced: %[1]s module-profile, %[2]s niagara_home, then
// the old and new vendorVersion and sha256 prefix, those of the first jar replaced this cycle, the build.sh path
// and that first kept jar.
const (
	msgReplaced = "==> mirror: %[1]s.jar in %[2]s/modules replaced (vendorVersion %[3]s sha256:%[4]s -> %[5]s sha256:%[6]s);" +
		" first replaced this cycle: vendorVersion %[7]s sha256:%[8]s; restore with: %[9]s --restore-mirror %[10]s"
	msgReplacedNoFirst = "==> mirror: %[1]s.jar in %[2]s/modules replaced (vendorVersion %[3]s sha256:%[4]s -> %[5]s sha256:%[6]s);" +
		" kept as %[7]s; the first jar replaced this cycle is not recorded: list %[8]s and restore the jar the station runs" +
		" with: %[9]s --restore-mirror <kept jar>"
	msgFirstUnrecorded     = "build.sh: WARN — the first %[1]s.jar replaced this cycle is not recorded (%[2]s: %[3]s); the line below names no jar as the one the station runs"
	msgKeepFailed          = "build.sh: WARN — the replaced %[1]s.jar was NOT kept in %[2]s (%[3]s); the mirror's modules/ no longer holds the jar it had before this build"
	msgNotKept             = "build.sh: --restore-mirror needs a jar a build kept: <mirror>/" + replacedDir + "/<MOD>-<profile>.jar.<sha12> (got: %s)"
	msgNoKept              = "build.sh: --restore-mirror: no such kept jar: %s"
	msgKeptChanged         = "build.sh: --restore-mirror refuses %s: its sha256 starts with %s, not %s (the jar was changed after it was kept)"
	msgRestoreNotMirror    = "build.sh: --restore-mirror refuses %s: no .niagara-mirror marker (only a mirror is restored)"
	msgRestoreMirrorFailed = "build.sh: --restore-mirror: could not write %s (%s)"
	// msgMirrorSource takes the mirror, the installation it was made from, that installation's Niagara version,
	// and the command that installs the jars built this run there (check the version, then copy).
	msgMirrorSource = "==> mirror: %[1]s was made from %[2]s (Niagara %[3]s); the jars built this run were NOT installed in %[2]s/modules" +
		" — to install them, stop the station and close Workbench of that installation, then run: %[4]s"
	// msgMirrorSigning takes the mirror, twice (the user home sits in it, and -Pniagara_home points at it).
	msgMirrorSigning = "build.sh: WARN — mirror build: the jar is signed with the throwaway key of the mirror user home (%[1]s/.mirror-user-home), not your signing profile, so it cannot be installed on a station.\n" +
		"  This build proves the module compiles. For a deployable jar run the same gradle tasks (clean, slotomatic, jar, Java 8 toolchain) with -Pniagara_home=%[2]s under your real HOME."
	msgMirrorRestored = "==> mirror: restored %[1]s/modules/%[2]s from %[3]s (vendorVersion %[4]s sha256:%[5]s)"
)

// Gradle run and its log.
const (
	msgBuild       = "==> build (Java 8 + slotomatic): %s"
	msgSkippedFile = "build.sh: Slot-o-Matic skipped the file(s) above, so the jar carries stale generated code (METHODOLOGY K15):\n" +
		"  the generated region needs exactly one /*@ ... @*/ stamp — delete the BEGIN/END BAJA AUTO GENERATED region and re-run slotomatic."
	msgDevKey     = "==> WARN: the jar is signed with a throwaway dev key created in the mirror user home (%s) — do not deploy it; add the signing block (templates/signing-root.gradle.kts, lint-structure L15)"
	msgCopyLocked = "build.sh: the plugin's post-jar copy into %s/modules failed — a local process (Workbench,\n" +
		"  a station) holds the old jar open (FileAlreadyExistsException). The build/libs jar is the artifact of record:"
	msgJarBuilt = "    %s (built this run)"
	msgJarStale = "    %s — not rebuilt this run (missing or stale): gradle stopped before this profile"
	// msgCopyLockedNext takes the niagara_home whose modules/ is locked.
	msgCopyLockedNext = "  Stop the station and close Workbench of the installation %s (on Windows the station is a service or station.exe), then re-run to refresh the modules/ copy — or use the build/libs\n" +
		"  jar directly (the verify gate was NOT run: run verify-module.sh on it before any deploy)."
	msgCleanLocked = "build.sh: :clean could not delete a modules/<jar> — a running station has it locked.\n" +
		"  Free the lock first: close Workbench, or stop the station, then build directly; or build against a\n" +
		"  mirror (mirror-niagara-home.sh); or just use the already-assembled build/libs jar (the modules/ copy\n" +
		"  is irrelevant when you are not deploying to that supervisor)."
	msgSlotErrors     = "build.sh: compileJava found slots slot-o-matic never generated."
	msgSlotErrorsFile = "  A file with empty BEGIN/END BAJA AUTO GENERATED markers is skipped (METHODOLOGY K15): delete the\n" +
		"  markers and rerun."
	msgSlotErrorsRerun = "  Rerun :<part>:slotomatic --info and read its output for 'Could not process' (METHODOLOGY K15)."
	msgWerror          = "build.sh: --werror build: javac warnings are errors (-Xlint:all -Werror). Fix each warning, or put a scoped\n" +
		"  @SuppressWarnings(\"<lint>\") on the smallest declaration that holds an unavoidable one (METHODOLOGY.md\n" +
		"  \"Compiles warning-clean\"); rebuild without --werror to tell a warning from a real compile error."
	msgGradleFailed = "build.sh: gradle failed"
)

// Deployed-baseline drift gate. %s after "verify" / "WARN —" is the module-profile name.
const (
	msgCannotVerify = "build.sh: FAIL — drift gate cannot verify %s: %s."
	whyUnlistable   = "cannot read the jar content (not a readable zip, no entry, a duplicate entry name, or an entry that fails its CRC)"
	whyNoVersion    = "shipped bytes changed but the module's own vendorVersion is unreadable (old='%s' new='%s')"
	msgWentDown     = "build.sh: WARN — %[1]s vendorVersion went down (%[2]s -> %[3]s) versus the deployed jar; fine for a deliberate rollback, otherwise bump defaultModuleVersion above %[2]s."
	msgDrift        = "build.sh: FAIL — %[1]s shipped bytes changed but vendorVersion is still %[2]s.\n" +
		"  Software Manager compares versions, not bytes — it will report \"Up to Date\" and SKIP installing this jar.\n" +
		"  Bump defaultModuleVersion(\"%[2]s\") in %[1]s's build.gradle.kts (patch for a fix, minor for a feature), then rebuild."
	msgDriftDetected   = "build.sh: deployed-baseline drift detected — bump the version above, then rebuild (or --no-drift-check if this niagara_home is not the deploy target)."
	msgDriftUnverified = "build.sh: deployed-baseline drift gate could not verify the jar(s) above — fix the cause (a damaged jar, an unreadable module.xml), or --no-drift-check if this niagara_home is not the deploy target."
)

// --schema-baseline-ref (baselineref.go).
const (
	msgBaselineRefFirstDeploy = "build.sh: --schema-baseline-ref and --first-deploy exclude each other (a never-deployed module has no deployed ref)"
	msgBaselineRefNotCommit   = "build.sh: --schema-baseline-ref %s: not a commit of the repository holding %s"
	msgBaselineRefNoModule    = "build.sh: --schema-baseline-ref %s: %s does not exist at that commit (pass the commit that was deployed)"
	msgBaselineRefExists      = "build.sh: --schema-baseline-ref: %s already exists — remove it to take the baseline from the ref, or drop the option to use it"
	msgBaselineRefNewArtifact = "build.sh: INFO — %s does not exist at %s: no schema-risk baseline for it (a first deploy of that artifact)"
	msgBaselineRefNotArtifact = "build.sh: INFO — %s is " + reportmodule.NotArtifactDetail + ": no schema-risk baseline for it"
	msgBaselineRefFailed      = "build.sh: --schema-baseline-ref %s: cannot extract %s (exit %d)"
	msgBaselineRefDangling    = "build.sh: INFO — removed %s, a dangling link a killed --schema-baseline-ref run left"
	msgBaselineRefLink        = "build.sh: --schema-baseline-ref: cannot link %s: %v"
	msgBaselineRefDone        = "==> schema-risk baseline: %s from %s (removed when the run ends)"
	msgBaselineRefVersion     = "build.sh: WARN — --schema-baseline-ref %s has defaultModuleVersion %s but the deployed %s is %s: the " +
		"schema-risk baseline is not what runs on the station; pass the ref of the deployed version"
)

// Verify gate and report-module.
const (
	msgGate       = "==> verify gate (verify-module.sh):"
	msgGateFailed = "build.sh: verify gate failed — do not deploy these jars"
	msgNoSigner   = "build.sh: WARN — " + modjar.WhyNoSigner + " (set it: --gradle-prop niagaraSigningAlias=<alias>, or gradle.properties)"
	// %s is the development signer's name.
	msgDevSignerDeployed = "build.sh: WARN — " + modjar.WhyDevSigner + " (set it: --gradle-prop niagaraSigningAlias=<alias>, or gradle.properties)"
	msgReport            = "==> report-module (hand-off punch-list)"
	msgReportEnv         = "build.sh: report-module environment error (exit 3)"
	msgReportIncomplete  = "build.sh: report-module verdict INCOMPLETE — a floor check (verify-module / schema-risk) did not run; extract the deployed baseline into <artifact>/.deploy-baseline (--schema-baseline-ref <deployed-ref> extracts it for the run; build-verify.md § Schema-risk deploy workflow) or pass --first-deploy for a never-deployed module"
	// msgSchemaRiskCrossRef ends the INCOMPLETE line when an artifact lacks .deploy-baseline (schema-risk SKIPped).
	msgSchemaRiskCrossRef = " (the schema-risk SKIP row above names the command that creates it)"
	msgReportFails        = "build.sh: report-module punch-list has FAILs — not hand-off-ready (--no-report to skip)"
)

// A command the chain could not start, or a step that failed outside any gate.
const (
	msgCannotRun = "build.sh: %s: %s"
	msgNoTemp    = "build.sh: cannot make a temporary %s in %s: %s"
	msgNoDir     = "build.sh: cannot create directory %s: %s"
	msgNoBackup  = "build.sh: cannot keep the pre-build jar: %s"
	whyNoCommand = "command not found"
	msgFault     = "build.sh: %s"
)
