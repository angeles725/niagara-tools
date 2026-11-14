package buildmodule

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
	msgRestored         = "  Restored the previously-deployed %[1]s.jar into %[2]s/modules%[3]s"
	msgRestoreFailedOld = "  FAILED TO RESTORE %[1]s.jar (%[3]s); %[2]s/modules still holds the pre-build jar, so a retry is still safe."
	msgRestoreFailedNew = "  FAILED TO RESTORE the previously-deployed %[1]s.jar (%[3]s) — %[2]s/modules no longer holds it (likely the NEW jar); restore it manually before retrying."
	restoreNoReason     = "cp failed"
	restoredPlain       = "."
	restoredRetry       = " — a retry will catch this again."
)

// Gradle run and its log.
const (
	msgBuild       = "==> build (Java 8 + slotomatic): %s"
	msgSkippedFile = "build.sh: Slot-o-Matic skipped the file(s) above, so the jar carries stale generated code (METHODOLOGY K15):\n" +
		"  the generated region needs exactly one /*@ ... @*/ stamp — delete the BEGIN/END BAJA AUTO GENERATED region and re-run slotomatic."
	msgDevKey     = "==> WARN: the jar is signed with a throwaway dev key created in the mirror user home (%s) — do not deploy it; add the signing block (templates/signing-root.gradle.kts, lint-structure L15)"
	msgCopyLocked = "build.sh: the plugin's post-jar copy into %s/modules failed — a local process (Workbench,\n" +
		"  a station) holds the old jar open (FileAlreadyExistsException). The build/libs jar is the artifact of record:"
	msgJarBuilt       = "    %s (built this run)"
	msgJarStale       = "    %s — not rebuilt this run (missing or stale): gradle stopped before this profile"
	msgCopyLockedNext = "  Close the process holding modules/<jar> and re-run to refresh the modules/ copy — or use the build/libs\n" +
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
	whyUnlistable   = "cannot list the jar content (unzip missing or failing, or an unreadable jar)"
	whyNoVersion    = "shipped bytes changed but the module's own vendorVersion is unreadable (old='%s' new='%s')"
	msgWentDown     = "build.sh: WARN — %[1]s vendorVersion went down (%[2]s -> %[3]s) versus the deployed jar; fine for a deliberate rollback, otherwise bump defaultModuleVersion above %[2]s."
	msgDrift        = "build.sh: FAIL — %[1]s shipped bytes changed but vendorVersion is still %[2]s.\n" +
		"  Software Manager compares versions, not bytes — it will report \"Up to Date\" and SKIP installing this jar.\n" +
		"  Bump defaultModuleVersion(\"%[2]s\") in %[1]s's build.gradle.kts (patch for a fix, minor for a feature), then rebuild."
	msgDriftDetected   = "build.sh: deployed-baseline drift detected — bump the version above, then rebuild (or --no-drift-check if this niagara_home is not the deploy target)."
	msgDriftUnverified = "build.sh: deployed-baseline drift gate could not verify the jar(s) above — fix the cause (install unzip, check module.xml), or --no-drift-check if this niagara_home is not the deploy target."
)

// Verify gate and report-module.
const (
	msgGate             = "==> verify gate (verify-module.sh):"
	msgGateFailed       = "build.sh: verify gate failed — do not deploy these jars"
	msgReport           = "==> report-module (hand-off punch-list)"
	msgReportEnv        = "build.sh: report-module environment error (exit 3)"
	msgReportIncomplete = "build.sh: report-module verdict INCOMPLETE — a floor check (verify-module / schema-risk) did not run; extract the deployed baseline into <artifact>/.deploy-baseline (build-verify.md § Schema-risk deploy workflow) or pass --first-deploy for a never-deployed module"
	msgReportFails      = "build.sh: report-module punch-list has FAILs — not hand-off-ready (--no-report to skip)"
)

// A command the chain could not start, or a step that failed outside any gate.
const (
	msgCannotRun = "build.sh: %s: %s"
	whyNoCommand = "command not found"
	msgFault     = "build.sh: %s"
)
