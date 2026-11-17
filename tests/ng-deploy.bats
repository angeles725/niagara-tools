#!/usr/bin/env bats
# tests/ng-deploy.bats — pure-logic unit tests for scripts/ng-deploy.sh
# Stub strategy: PATH-injected fakebin with controlled exit codes via env vars.
# No real gradlew, no real unzip to /mnt/c, no station dependency.
# why: SC2030/SC2031 — each @test runs in a subshell; export is the correct
# mechanism to pass env vars to run-spawned subprocesses in bats. These are
# not real subshell-loss bugs. SC2154 — `$stderr` is a bats-provided `run` variable,
# not an unassigned reference.
# shellcheck disable=SC2030,SC2031,SC2154

SCRIPT_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"
SCRIPT="$SCRIPT_DIR/scripts/ng-deploy.sh"

setup() {
    TMPDIR_T="$(mktemp -d)"
    export TMPDIR_T
    mkdir -p "$TMPDIR_T/fakebin" "$TMPDIR_T/modules"
    # NIAGARA_HOME is a mirror of the test's own (the marker mirror-niagara-home.sh leaves): ng-deploy refuses a
    # niagara_home that is not one, and no test may name a real installation.
    mkdir -p "$TMPDIR_T/niagara-home"
    : > "$TMPDIR_T/niagara-home/.niagara-mirror"
    # The module names its signer (the gradle root is fakebin/), and no gradle.properties of the host is read: the
    # deploy gate is told to expect that signer, and without one the deploy prints a WARN.
    printf 'niagaraSigningAlias=testsigner\n' > "$TMPDIR_T/fakebin/gradle.properties"
    export GRADLE_USER_HOME="$TMPDIR_T/gradle-user-home"
    unset ORG_GRADLE_PROJECT_niagaraSigningAlias

    # --- fake gradlew ---
    # Writes every invocation to gradlew.calls.log (one line per call, args space-separated)
    # and overwrites gradlew.args with the last invocation (backward-compat for T1-T17).
    cat > "$TMPDIR_T/fakebin/gradlew" << 'STUB'
#!/usr/bin/env bash
printf '%s\n' "$@" > "${TMPDIR_T}/gradlew.args"
printf '%s\n' "$*" >> "${TMPDIR_T}/gradlew.calls.log"
# FAKE_GRADLEW_STDOUT: text a slotomatic invocation prints (a Slot-o-Matic skip report, K15)
if [[ -n "${FAKE_GRADLEW_STDOUT:-}" && "$*" == *slotomatic* ]]; then printf '%s\n' "$FAKE_GRADLEW_STDOUT"; fi
exit "${FAKE_GRADLEW_EXIT:-0}"
STUB
    chmod +x "$TMPDIR_T/fakebin/gradlew"

    # --- fake git ---
    # Dispatches by $1: diff → FAKE_GIT_DIFF_OUTPUT; rev-parse (--git-dir) → exit 0 (128 with FAKE_GIT_NOREPO);
    # rev-parse HEAD → FAKE_GIT_REV_PARSE_OUTPUT; cat-file → FAKE_GIT_CAT_FILE_EXIT.
    cat > "$TMPDIR_T/fakebin/git" << 'STUB'
#!/usr/bin/env bash
case "$1" in
    diff)
        printf '%s\n' "${FAKE_GIT_DIFF_OUTPUT:-}"
        exit 0
        ;;
    rev-parse)
        # FAKE_GIT_NOREPO: the working directory is not in a repository, as git answers there (exit 128)
        if [[ -n "${FAKE_GIT_NOREPO:-}" ]]; then
            echo "fatal: not a git repository (or any of the parent directories): .git" >&2
            exit 128
        fi
        if [[ "$2" == "--git-dir" ]]; then
            printf '%s\n' ".git"
            exit 0
        fi
        # rev-parse HEAD (or any other ref)
        printf '%s\n' "${FAKE_GIT_REV_PARSE_OUTPUT:-deadbeef1234567890abcdef1234567890abcdef}"
        exit 0
        ;;
    cat-file)
        exit "${FAKE_GIT_CAT_FILE_EXIT:-0}"
        ;;
    *)
        exit 0
        ;;
esac
STUB
    chmod +x "$TMPDIR_T/fakebin/git"

    # --- fake unzip ---
    # Outputs FAKE_UNZIP_TYPES lines containing "<type" (default 9)
    cat > "$TMPDIR_T/fakebin/unzip" << 'STUB'
#!/usr/bin/env bash
count="${FAKE_UNZIP_TYPES:-9}"
# audit A2: the module's own vendorVersion — FAKE_OLD_VERSION for a jar under modules/, FAKE_NEW_VERSION otherwise
case "$*" in
    */modules/*) [[ -n "${FAKE_OLD_VERSION:-}" ]] && printf '<module name="test-rt" vendorVersion="%s">\n' "$FAKE_OLD_VERSION" ;;
    *)           [[ -n "${FAKE_NEW_VERSION:-}" ]] && printf '<module name="test-rt" vendorVersion="%s">\n' "$FAKE_NEW_VERSION" ;;
esac
for i in $(seq 1 "$count"); do
    printf '<type name="FakeType%s"/>\n' "$i"
done
# if FAKE_INDEX_HTML is set, simulate index.html extraction
if [[ "${FAKE_INDEX_HTML:-}" == "1" ]]; then
    printf '?v=%s\n' "${FAKE_BUILD_ID:-NONE}"
fi
exit 0
STUB
    chmod +x "$TMPDIR_T/fakebin/unzip"

    # --- fake tar ---
    cat > "$TMPDIR_T/fakebin/tar" << 'STUB'
#!/usr/bin/env bash
exit "${FAKE_TAR_EXIT:-0}"
STUB
    chmod +x "$TMPDIR_T/fakebin/tar"

    # Prepend fakebin to PATH
    export PATH="$TMPDIR_T/fakebin:$PATH"
    # audit A2: every pre-A2 test deploys a never-deployed module (no schema baseline): the env files below carry
    # FIRST_DEPLOY=1 (an env-file key is an explicit opt-in; an ambient variable is ignored, #246 A2b), and the A2
    # tests drop it again with _no_first_deploy.
    unset FIRST_DEPLOY

    # Write minimal .env.local for tests that need a valid environment
    # why: .env.local path is dynamic (TMPDIR_T); not following is expected
    # shellcheck disable=SC1090,SC1091
    cat > "$TMPDIR_T/dot_env_local" << ENVEOF
MODULE_NAME=test
GRADLEW_PATH=$TMPDIR_T/fakebin/gradlew
NIAGARA_HOME=$TMPDIR_T/niagara-home
NIAGARA_USER_HOME=/mnt/c/Users/equipo/Niagara4.13/test
JAVA_HOME=/usr/lib/jvm/java-8-openjdk-amd64
STATION_MODULES_DIR=$TMPDIR_T/modules
EXPECTED_RT_TYPES=9
EXPECTED_UX_TYPES=2
FIRST_DEPLOY=1
ENVEOF

    cd "$TMPDIR_T" || return 1
}

teardown() {
    chmod -R u+w "$TMPDIR_T" 2>/dev/null || true   # a test may leave a directory read-only
    rm -rf "$TMPDIR_T"
}

# ---------------------------------------------------------------------------
# Test 1: flag --help exits 0 and prints usage
# ---------------------------------------------------------------------------
@test "flag --help exits 0 and prints usage" {
    run bash "$SCRIPT" --help
    [ "$status" -eq 0 ]
    [[ "$output" == *"Usage:"* ]]
}

# ---------------------------------------------------------------------------
# Test 2: missing MODULE_NAME exits 10
# ---------------------------------------------------------------------------
@test "missing MODULE_NAME exits 10" {
    # Write an env file with MODULE_NAME missing
    cat > "$TMPDIR_T/no_module_env" << ENVEOF
GRADLEW_PATH=$TMPDIR_T/fakebin/gradlew
NIAGARA_HOME=$TMPDIR_T/niagara-home
NIAGARA_USER_HOME=/mnt/c/Users/equipo/Niagara4.13/test
JAVA_HOME=/usr/lib/jvm/java-8-openjdk-amd64
STATION_MODULES_DIR=$TMPDIR_T/modules
EXPECTED_RT_TYPES=9
EXPECTED_UX_TYPES=2
ENVEOF
    run bash "$SCRIPT" --env-file "$TMPDIR_T/no_module_env" --mode A
    [ "$status" -eq 10 ]
}

# ---------------------------------------------------------------------------
# Test 3: --no-backup runs trivially but WARNS (Campaign 3 B10 iii, ng-deploy-backup)
# The --i-know-what-im-doing gate is dropped (default is now a safe lightweight backup, so
# --no-backup is a plain opt-in) — BUT it prints a one-line safety WARN instead of silently
# skipping. This is the WARN compromise: frictionless + a zero-cost rollback reminder, not a
# silent removal of a safety behavior. RED-first: today it exits 20 with the gate message.
# ---------------------------------------------------------------------------
@test "--no-backup runs without a gate but prints a safety WARN (B10 iii)" {
    run bash "$SCRIPT" --env-file "$TMPDIR_T/dot_env_local" --no-backup --no-deploy --mode A
    [ "$status" -ne 20 ]                                     # gate removed (was exit 20)
    [[ "${output}${stderr}" != *"i-know-what-im-doing"* ]]   # no gate message / no companion required
    [[ "${output}${stderr}" == *"backup skipped"* ]]         # the WARN fires
    [[ "${output}${stderr}" == *"rollback"* ]]               # ...and points at git rollback (mutation: drop the WARN → red)
}

# ---------------------------------------------------------------------------
# Test 4: --no-backup with --i-know-what-im-doing is allowed
# ---------------------------------------------------------------------------
@test "--no-backup with --i-know-what-im-doing is allowed" {
    run bash "$SCRIPT" \
        --env-file "$TMPDIR_T/dot_env_local" \
        --no-backup \
        --i-know-what-im-doing \
        --no-deploy \
        --mode A
    # Should NOT exit 20 (backup guard should pass)
    [ "$status" -ne 20 ]
}

# ---------------------------------------------------------------------------
# Test 5: backup filename matches expected format
# ---------------------------------------------------------------------------
@test "backup filename matches expected format" {
    : > "$TMPDIR_T/modules/test-rt.jar"   # a deployed jar: an archive is named only when one is written
    run bash "$SCRIPT" \
        --env-file "$TMPDIR_T/dot_env_local" \
        --no-deploy \
        --mode A
    # Even if exit is non-zero due to other issues, check that any _backups/ reference
    # in stdout matches the naming pattern
    [[ "$output" =~ _backups/test-pre-[0-9]{8}-[0-9]{6}\.tar\.gz ]]
}

@test "NG-backup-nothing: a module with no jar in the station names no archive and makes no _backups directory" {
    run bash "$SCRIPT" --env-file "$TMPDIR_T/dot_env_local" --no-deploy --mode A
    [ "$status" -eq 0 ]
    [[ "$output" == *"backup: no existing test-{rt,ux,wb}.jar in $TMPDIR_T/modules; nothing to archive"* ]]
    [[ "$output" != *"_backups/"* ]]
    [ ! -e "$TMPDIR_T/_backups" ]
}

# ---------------------------------------------------------------------------
# Test 6: mode A resolves to rt+ux gradle tasks
# ---------------------------------------------------------------------------
@test "mode A resolves to rt+ux gradle tasks" {
    run bash "$SCRIPT" \
        --env-file "$TMPDIR_T/dot_env_local" \
        --no-backup \
        --i-know-what-im-doing \
        --no-deploy \
        --mode A
    [ "$status" -eq 0 ]
    # gradlew.args should contain both rt and ux tasks
    run cat "$TMPDIR_T/gradlew.args"
    [[ "$output" == *"rt"* ]]
    [[ "$output" == *"ux"* ]]
}

# ---------------------------------------------------------------------------
# Test 7: mode B resolves to ux-only gradle tasks
# ---------------------------------------------------------------------------
@test "mode B resolves to ux-only gradle tasks" {
    run bash "$SCRIPT" \
        --env-file "$TMPDIR_T/dot_env_local" \
        --no-backup \
        --i-know-what-im-doing \
        --no-deploy \
        --mode B
    [ "$status" -eq 0 ]
    run cat "$TMPDIR_T/gradlew.args"
    [[ "$output" == *"ux"* ]]
    [[ "$output" != *":rt:"* ]]
}

# ---------------------------------------------------------------------------
# Test 8: mode C resolves to rt-only gradle tasks
# ---------------------------------------------------------------------------
@test "mode C resolves to rt-only gradle tasks" {
    run bash "$SCRIPT" \
        --env-file "$TMPDIR_T/dot_env_local" \
        --no-backup \
        --i-know-what-im-doing \
        --no-deploy \
        --mode C
    [ "$status" -eq 0 ]
    run cat "$TMPDIR_T/gradlew.args"
    [[ "$output" == *"rt"* ]]
    [[ "$output" != *":ux:"* ]]
}

# ---------------------------------------------------------------------------
# Test 9: default mode is A
# ---------------------------------------------------------------------------
@test "default mode is A" {
    run bash "$SCRIPT" \
        --env-file "$TMPDIR_T/dot_env_local" \
        --no-backup \
        --i-know-what-im-doing \
        --no-deploy
    [ "$status" -eq 0 ]
    run cat "$TMPDIR_T/gradlew.args"
    [[ "$output" == *"rt"* ]]
    [[ "$output" == *"ux"* ]]
}

# Tests 10-13 (verify_jar and verify_cachebuster called on their own) are Go tests now:
# go/internal/ngdeploy/verify_single_jar_test.go.

# ---------------------------------------------------------------------------
# Test 14: --no-deploy stops after build
# ---------------------------------------------------------------------------
@test "--no-deploy stops after build" {
    run bash "$SCRIPT" \
        --env-file "$TMPDIR_T/dot_env_local" \
        --no-backup \
        --i-know-what-im-doing \
        --no-deploy \
        --mode A
    [ "$status" -eq 0 ]
    # copy_jars should NOT have been called — no jars in modules dir
    # why: counting files only, no special chars in filenames expected here
    # shellcheck disable=SC2012
    [ "$(ls "$TMPDIR_T/modules/" 2>/dev/null | wc -l)" -eq 0 ]
}

# ---------------------------------------------------------------------------
# Test 15: unknown flag exits non-zero
# ---------------------------------------------------------------------------
@test "unknown flag exits non-zero" {
    run bash "$SCRIPT" --banana
    [ "$status" -ne 0 ]
}

# ---------------------------------------------------------------------------
# Test 16: --version exits 0 and prints semver
# ---------------------------------------------------------------------------
@test "--version exits 0 and prints semver" {
    run bash "$SCRIPT" --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
}

# ---------------------------------------------------------------------------
# Test 17: --version output equals VERSION file content (anti-drift)
# ---------------------------------------------------------------------------
@test "--version output equals VERSION file content (anti-drift)" {
    run bash "$SCRIPT" --version
    [ "$status" -eq 0 ]
    expected="$(<"$SCRIPT_DIR/VERSION")"
    expected="${expected%$'\n'}"
    [ "$output" = "$expected" ]
}

# ---------------------------------------------------------------------------
# T18: --with-slotomatic invoca slotomatic ANTES de build_jars
# La llamada a :test-rt:slotomatic debe aparecer en gradlew.calls.log
# ANTES de la llamada que contiene :test-rt:jar.
# ---------------------------------------------------------------------------
@test "T18: --with-slotomatic runs slotomatic before build_jars" {
    run bash "$SCRIPT" \
        --env-file "$TMPDIR_T/dot_env_local" \
        --no-backup \
        --i-know-what-im-doing \
        --no-deploy \
        --mode A \
        --with-slotomatic
    [ "$status" -eq 0 ]
    # The calls log must exist
    [ -f "$TMPDIR_T/gradlew.calls.log" ]
    # slotomatic line must appear before the build line in the log
    slotomatic_line=$(grep -n "slotomatic" "$TMPDIR_T/gradlew.calls.log" | head -1 | cut -d: -f1)
    build_line=$(grep -n ":test-rt:jar" "$TMPDIR_T/gradlew.calls.log" | head -1 | cut -d: -f1)
    [ -n "$slotomatic_line" ]
    [ -n "$build_line" ]
    [ "$slotomatic_line" -lt "$build_line" ]
}

# ---------------------------------------------------------------------------
# T19: gradlew slotomatic falla → exit 15, sin build
# ---------------------------------------------------------------------------
@test "T19: slotomatic failure exits 15 and does not build" {
    # gradlew exits 1 only for the slotomatic invocation
    # We make all gradlew calls fail so we can assert exit 15 specifically
    export FAKE_GRADLEW_EXIT=1
    run bash "$SCRIPT" \
        --env-file "$TMPDIR_T/dot_env_local" \
        --no-backup \
        --i-know-what-im-doing \
        --mode A \
        --with-slotomatic
    [ "$status" -eq 15 ]
    # build must NOT have been invoked after slotomatic fail
    # gradlew.calls.log should have exactly 1 line (the slotomatic call)
    [ -f "$TMPDIR_T/gradlew.calls.log" ]
    [ "$(wc -l < "$TMPDIR_T/gradlew.calls.log")" -eq 1 ]
    [[ "$(cat "$TMPDIR_T/gradlew.calls.log")" == *"slotomatic"* ]]
}

# ---------------------------------------------------------------------------
# T19b: slotomatic exits 0 but skipped a file (K15) → exit 15, file named, sin build
# The shared detector (go/internal/slotomaticskip) reads the captured slotomatic output.
# [ev: retro r10-slotomatic-skip-fail-open Δ1]
# Named mutation: drop the slotomaticskip.ScanFile call of the slotomatic step -> T19b exits 0 and builds.
# ---------------------------------------------------------------------------
@test "T19b: slotomatic that skipped a file exits 15, names it, and does not build" {
    export FAKE_GRADLEW_STDOUT="WARNING: Could not process /x/src/BHmiStatus.java: Found multiple metadata blocks"
    run bash "$SCRIPT" \
        --env-file "$TMPDIR_T/dot_env_local" \
        --no-backup \
        --mode A \
        --with-slotomatic
    [ "$status" -eq 15 ]
    [[ "$output" == *"FAIL slotomatic-skip /x/src/BHmiStatus.java"* ]]
    [[ "$output" == *"K15"* ]]
    [ "$(wc -l < "$TMPDIR_T/gradlew.calls.log")" -eq 1 ]
}

# ---------------------------------------------------------------------------
# T20: detección WARN — cambios de anotación sin --with-slotomatic → WARN en stderr, exit 0
# ---------------------------------------------------------------------------
@test "T20: annotation detection WARN when changes detected without --with-slotomatic" {
    export FAKE_GIT_DIFF_OUTPUT='+    @NiagaraType'
    export FAKE_GIT_CAT_FILE_EXIT=0
    run bash "$SCRIPT" \
        --env-file "$TMPDIR_T/dot_env_local" \
        --no-backup \
        --i-know-what-im-doing \
        --no-deploy \
        --mode A
    [ "$status" -eq 0 ]
    [[ "${output}${stderr}" == *"slotomatic"* ]]
}

# ---------------------------------------------------------------------------
# T21: --strict-slotomatic + cambios detectados → exit 15
# ---------------------------------------------------------------------------
@test "T21: --strict-slotomatic aborts with exit 15 when annotation changes detected" {
    export FAKE_GIT_DIFF_OUTPUT='+    @NiagaraProperty'
    export FAKE_GIT_CAT_FILE_EXIT=0
    run bash "$SCRIPT" \
        --env-file "$TMPDIR_T/dot_env_local" \
        --no-backup \
        --i-know-what-im-doing \
        --mode A \
        --strict-slotomatic
    [ "$status" -eq 15 ]
}

# ---------------------------------------------------------------------------
# T22: mode B + --with-slotomatic → WARN en stderr, slotomatic NO invocado, ux build normal
# ---------------------------------------------------------------------------
@test "T22: mode B + --with-slotomatic warns and skips slotomatic" {
    run bash "$SCRIPT" \
        --env-file "$TMPDIR_T/dot_env_local" \
        --no-backup \
        --i-know-what-im-doing \
        --no-deploy \
        --mode B \
        --with-slotomatic
    [ "$status" -eq 0 ]
    # WARN must appear in output (save output before any inner run call)
    local saved_output="$output"
    [[ "$saved_output" == *"slotomatic"* ]]
    # slotomatic must NOT appear in any gradlew invocation
    if [ -f "$TMPDIR_T/gradlew.calls.log" ]; then
        # why: using grep exit code only; not capturing output
        ! grep -q "slotomatic" "$TMPDIR_T/gradlew.calls.log"
    fi
}

# ---------------------------------------------------------------------------
# T23: git ausente → skip detección silencioso, deploy continúa, exit 0
# ---------------------------------------------------------------------------
@test "T23: git absent — detection skipped silently, deploy continues" {
    # Replace fakebin/git with a stub that does not exist (remove it)
    rm -f "$TMPDIR_T/fakebin/git"
    run bash "$SCRIPT" \
        --env-file "$TMPDIR_T/dot_env_local" \
        --no-backup \
        --i-know-what-im-doing \
        --no-deploy \
        --mode A
    [ "$status" -eq 0 ]
}

# ---------------------------------------------------------------------------
# T24: deploy exitoso → .last-deploy-sha escrito con HEAD SHA post-verify
# ---------------------------------------------------------------------------
@test "T24: successful deploy writes .last-deploy-sha after verify" {
    # a full 40-digit object name, as git prints it: anything else is not recorded (exit 41)
    export FAKE_GIT_REV_PARSE_OUTPUT="abc1234abc1234abc1234abc1234abc1234abc12"
    export FAKE_UNZIP_TYPES=9
    # Need to touch fake jar files so copy_jars and verify_jar succeed
    # gradlew path in .env.local points to TMPDIR_T/fakebin/gradlew
    # src_dir = dirname(GRADLEW_PATH) = TMPDIR_T/fakebin
    mkdir -p "$TMPDIR_T/fakebin/test/test-rt/build/libs" \
             "$TMPDIR_T/fakebin/test/test-ux/build/libs"
    touch "$TMPDIR_T/fakebin/test/test-rt/build/libs/test-rt.jar"
    touch "$TMPDIR_T/fakebin/test/test-ux/build/libs/test-ux.jar"
    # Use a custom env that sets EXPECTED_RT_TYPES=9, EXPECTED_UX_TYPES=9 to match fake unzip
    cat > "$TMPDIR_T/env_t24" << ENVEOF
MODULE_NAME=test
GRADLEW_PATH=$TMPDIR_T/fakebin/gradlew
NIAGARA_HOME=$TMPDIR_T/niagara-home
NIAGARA_USER_HOME=/mnt/c/Users/equipo/Niagara4.13/test
JAVA_HOME=/usr/lib/jvm/java-8-openjdk-amd64
STATION_MODULES_DIR=$TMPDIR_T/modules
EXPECTED_RT_TYPES=9
EXPECTED_UX_TYPES=9
FIRST_DEPLOY=1
ENVEOF
    run bash "$SCRIPT" \
        --env-file "$TMPDIR_T/env_t24" \
        --no-backup \
        --i-know-what-im-doing \
        --mode A \
        --no-gate
    [ "$status" -eq 0 ]
    [ -f "$TMPDIR_T/.last-deploy-sha" ]
    [ "$(cat "$TMPDIR_T/.last-deploy-sha")" = "abc1234abc1234abc1234abc1234abc1234abc12" ]
}

# ---------------------------------------------------------------------------
# T25: verify falla → .last-deploy-sha NO escrito
# ---------------------------------------------------------------------------
@test "T25: failed verify does not write .last-deploy-sha" {
    export FAKE_UNZIP_TYPES=99  # mismatched type count → verify fails
    local fake_src="$TMPDIR_T/fakebin"
    mkdir -p "$fake_src/test/test-rt/build/libs" "$fake_src/test/test-ux/build/libs"
    touch "$fake_src/test/test-rt/build/libs/test-rt.jar"
    touch "$fake_src/test/test-ux/build/libs/test-ux.jar"
    run bash "$SCRIPT" \
        --env-file "$TMPDIR_T/dot_env_local" \
        --no-backup \
        --i-know-what-im-doing \
        --mode A \
        --no-gate
    [ "$status" -eq 50 ]
    [ ! -f "$TMPDIR_T/.last-deploy-sha" ]
}

# ---------------------------------------------------------------------------
# T26: SLOTOMATIC_DETECTION=off desactiva detección completamente
# ---------------------------------------------------------------------------
@test "T26: SLOTOMATIC_DETECTION=off disables annotation detection" {
    export FAKE_GIT_DIFF_OUTPUT='+    @NiagaraType'
    export SLOTOMATIC_DETECTION=off
    run bash "$SCRIPT" \
        --env-file "$TMPDIR_T/dot_env_local" \
        --no-backup \
        --i-know-what-im-doing \
        --no-deploy \
        --mode A
    [ "$status" -eq 0 ]
    # No WARN about slotomatic in output when detection is off
    [[ "${output}${stderr}" != *"annotation"* ]]
}

# ===========================================================================
# P2 — ng-deploy runs :MOD-ux:slotomatic when the -ux profile has @Niagara
# annotations (today only :MOD-rt:slotomatic runs). Presence-based scan of the
# -ux source tree; requires --with-slotomatic and mode A (mode C has no ux).
# ===========================================================================

# T27: mode A + --with-slotomatic + annotated -ux source → :test-ux:slotomatic runs
@test "T27: ux slotomatic runs when the -ux profile carries a @Niagara annotation" {
    mkdir -p "$TMPDIR_T/fakebin/test/test-ux/src/com/x"
    printf 'package com.x;\n@NiagaraType\npublic class BFoo {}\n' \
        > "$TMPDIR_T/fakebin/test/test-ux/src/com/x/BFoo.java"
    run bash "$SCRIPT" \
        --env-file "$TMPDIR_T/dot_env_local" \
        --no-backup --i-know-what-im-doing --no-deploy \
        --mode A --with-slotomatic
    [ "$status" -eq 0 ]
    grep -q ":test-ux:slotomatic" "$TMPDIR_T/gradlew.calls.log"
    grep -q ":test-rt:slotomatic" "$TMPDIR_T/gradlew.calls.log"
}

# T28: mode A + --with-slotomatic + NO annotation in -ux → :test-ux:slotomatic NOT run
@test "T28: ux slotomatic is skipped when the -ux profile has no @Niagara annotation" {
    mkdir -p "$TMPDIR_T/fakebin/test/test-ux/src/com/x"
    printf 'package com.x;\npublic class Plain {}\n' \
        > "$TMPDIR_T/fakebin/test/test-ux/src/com/x/Plain.java"
    run bash "$SCRIPT" \
        --env-file "$TMPDIR_T/dot_env_local" \
        --no-backup --i-know-what-im-doing --no-deploy \
        --mode A --with-slotomatic
    [ "$status" -eq 0 ]
    run grep -q ":test-ux:slotomatic" "$TMPDIR_T/gradlew.calls.log"
    [ "$status" -ne 0 ]
    grep -q ":test-rt:slotomatic" "$TMPDIR_T/gradlew.calls.log"
}

# T29: mode C (rt-only) never runs :test-ux:slotomatic even if a ux tree exists
@test "T29: mode C never runs ux slotomatic (no ux profile in scope)" {
    mkdir -p "$TMPDIR_T/fakebin/test/test-ux/src/com/x"
    printf 'package com.x;\n@NiagaraType\npublic class BFoo {}\n' \
        > "$TMPDIR_T/fakebin/test/test-ux/src/com/x/BFoo.java"
    run bash "$SCRIPT" \
        --env-file "$TMPDIR_T/dot_env_local" \
        --no-backup --i-know-what-im-doing --no-deploy \
        --mode C --with-slotomatic
    [ "$status" -eq 0 ]
    run grep -q ":test-ux:slotomatic" "$TMPDIR_T/gradlew.calls.log"
    [ "$status" -ne 0 ]
    grep -q ":test-rt:slotomatic" "$TMPDIR_T/gradlew.calls.log"
}

# ===========================================================================
# P1 — ng-deploy runs the verify-module.sh gate on the built jars (default on
# for mode A/C, --no-gate to skip); a failing gate is exit 50. The gate binary
# is stubbed via VERIFY_MODULE_BIN so these test ng-deploy's wiring, not the
# gate's internals (verify-module.sh has its own suite).
# ===========================================================================

# helper: write a fake verify-module that logs its args and exits $1
_fake_gate() {   # _fake_gate <exit-code>
    cat > "$TMPDIR_T/fake-vm.sh" <<STUB
#!/usr/bin/env bash
printf '%s\n' "\$*" > "$TMPDIR_T/gate.args"
exit ${1:-0}
STUB
    chmod +x "$TMPDIR_T/fake-vm.sh"
    export VERIFY_MODULE_BIN="$TMPDIR_T/fake-vm.sh"
}

_no_first_deploy() { sed -i '/^FIRST_DEPLOY=/d' "$TMPDIR_T/env_gate"; }   # a module with a deployed baseline

_gate_env() {   # env whose expected type counts match the fake unzip (9) so verify_jar passes
    cat > "$TMPDIR_T/env_gate" <<ENVEOF
MODULE_NAME=test
GRADLEW_PATH=$TMPDIR_T/fakebin/gradlew
NIAGARA_HOME=$TMPDIR_T/niagara-home
NIAGARA_USER_HOME=/mnt/c/Users/equipo/Niagara4.13/test
JAVA_HOME=/usr/lib/jvm/java-8-openjdk-amd64
STATION_MODULES_DIR=$TMPDIR_T/modules
EXPECTED_RT_TYPES=9
EXPECTED_UX_TYPES=9
FIRST_DEPLOY=1
ENVEOF
    mkdir -p "$TMPDIR_T/fakebin/test/test-rt/build/libs" "$TMPDIR_T/fakebin/test/test-ux/build/libs"
    touch "$TMPDIR_T/fakebin/test/test-rt/build/libs/test-rt.jar" \
          "$TMPDIR_T/fakebin/test/test-ux/build/libs/test-ux.jar"
}

# T30: gate default-on for mode A; a failing gate makes ng-deploy exit 50
@test "T30: a failing verify-module gate aborts the deploy with exit 50" {
    _gate_env; _fake_gate 1
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" \
        --no-backup --i-know-what-im-doing --mode A
    [ "$status" -eq 50 ]
    [ -f "$TMPDIR_T/gate.args" ]
    grep -q -- "--src" "$TMPDIR_T/gate.args"
    grep -q "test-rt.jar" "$TMPDIR_T/gate.args"
    grep -q "test-ux.jar" "$TMPDIR_T/gate.args"
}

# T31: gate passes → deploy succeeds and the gate was invoked
@test "T31: a passing verify-module gate lets the deploy complete" {
    _gate_env; _fake_gate 0
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" \
        --no-backup --i-know-what-im-doing --mode A
    [ "$status" -eq 0 ]
    [ -f "$TMPDIR_T/gate.args" ]
}

# T32: --no-gate skips the gate entirely (binary never invoked), deploy proceeds
@test "T32: --no-gate skips the verify-module gate" {
    _gate_env; _fake_gate 1   # would fail IF invoked
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" \
        --no-backup --i-know-what-im-doing --mode A --no-gate
    [ "$status" -eq 0 ]
    [ ! -f "$TMPDIR_T/gate.args" ]
}

# T33: mode C gates the rt jar only (no ux jar passed)
@test "T33: mode C runs the gate on the rt jar only" {
    _gate_env; _fake_gate 0
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" \
        --no-backup --i-know-what-im-doing --mode C
    [ "$status" -eq 0 ]
    [ -f "$TMPDIR_T/gate.args" ]
    grep -q "test-rt.jar" "$TMPDIR_T/gate.args"
    run grep -q "test-ux.jar" "$TMPDIR_T/gate.args"
    [ "$status" -ne 0 ]
}

# Lesson B8 (the <types> wrapper is not a type entry) and lesson B10 (lightweight backup by default, keep-N
# purge, --full-backup) are Go tests now: go/internal/ngdeploy/verify_single_jar_test.go and
# backup_archive_test.go, which read the archive the host's tar wrote.

# ===========================================================================
# audit-2026-10-03 A2 — ng-deploy fails closed: a missing verify-module gate blocks (exit 50); a mode A/C deploy
# runs the schema-risk SAFE precheck on SCHEMA_BASELINE_DIR (or --first-deploy / FIRST_DEPLOY=1 for a module
# never deployed) and the version-bump precheck against the jar already in STATION_MODULES_DIR.
# ===========================================================================
_fake_sr() {   # _fake_sr <exit-code> — schema-risk stub recording its args
    cat > "$TMPDIR_T/fake-sr.sh" <<STUB
#!/usr/bin/env bash
printf '%s\n' "\$*" > "$TMPDIR_T/sr.args"
exit ${1:-0}
STUB
    chmod +x "$TMPDIR_T/fake-sr.sh"
    export SCHEMA_RISK_BIN="$TMPDIR_T/fake-sr.sh"
}

@test "NG-gate-missing: a missing verify-module gate blocks the deploy (exit 50), never a WARN pass (A2)" {
    # Mutation: NG-gate-missing -- the old WARN-and-return-0 path deploys an unverified jar.
    _gate_env
    export VERIFY_MODULE_BIN="$TMPDIR_T/no-such-verify-module.sh"
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 50 ]
    [[ "$output" == *"gate unavailable"* ]]
}

@test "NG-schema-required: mode A without SCHEMA_BASELINE_DIR or --first-deploy stops before the copy (exit 12) (A2)" {
    # Mutation: NG-schema-required -- skipping the precheck when no baseline is set deploys an un-diffed schema.
    _gate_env; _fake_gate 0; _no_first_deploy
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 12 ]
    [ ! -e "$TMPDIR_T/modules/test-rt.jar" ]
}

@test "NG-first-deploy-ambient: an inherited FIRST_DEPLOY=1 does not skip the schema-risk precheck (exit 12) (#246 A2b)" {
    # Mutation: NG-first-deploy-ambient -- reading FIRST_DEPLOY from the ambient environment lets a variable left
    # exported in the operator's shell silently skip the schema diff on a module that WAS deployed.
    _gate_env; _fake_gate 0; _no_first_deploy
    FIRST_DEPLOY=1 run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 12 ]
    [ ! -e "$TMPDIR_T/modules/test-rt.jar" ]
    FIRST_DEPLOY=1 run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A --first-deploy
    [ "$status" -eq 0 ]
}

@test "NG-schema-outage: a schema-risk OUTAGE (exit 2) or LOSSY (exit 1) blocks with exit 53; --accept-lossy passes LOSSY only (A2)" {
    # Mutation: NG-schema-outage -- ignoring the schema-risk verdict deploys a slot change that breaks saved data.
    _gate_env; _fake_gate 0; _no_first_deploy
    mkdir -p "$TMPDIR_T/baseline"; printf 'SCHEMA_BASELINE_DIR=%s\n' "$TMPDIR_T/baseline" >> "$TMPDIR_T/env_gate"
    _fake_sr 2
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A --accept-lossy
    [ "$status" -eq 53 ]
    _fake_sr 1
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 53 ]
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A --accept-lossy
    [ "$status" -eq 0 ]
}

@test "NG-schema-safe: a SAFE schema-risk diff (baseline vs the -rt module dir) lets the deploy complete (A2)" {
    _gate_env; _fake_gate 0; _no_first_deploy
    mkdir -p "$TMPDIR_T/baseline"; printf 'SCHEMA_BASELINE_DIR=%s\n' "$TMPDIR_T/baseline" >> "$TMPDIR_T/env_gate"
    _fake_sr 0
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 0 ]
    grep -q "$TMPDIR_T/baseline" "$TMPDIR_T/sr.args"
    grep -q "test/test-rt" "$TMPDIR_T/sr.args"
}

# helper: a real jar (a zip) at $1 whose one class holds $2, built at time $3: the build time is in module.xml, in
# the manifest's digest of module.xml and in the signature files, as in two builds by the vendor's plugin.
_real_jar() {
    local d
    d="$(mktemp -d "$TMPDIR_T/jar.XXXXXX")"
    mkdir -p "$d/META-INF" "$d/com/acme"
    printf 'Manifest-Version: 1.0\r\n\r\nName: META-INF/module.xml\r\nSHA-256-Digest: digest-%s\r\n\r\n' "$3" > "$d/META-INF/MANIFEST.MF"
    printf '<module name="test-rt" vendorVersion="1.0" buildMillis="%s" buildHost="H">\n</module>\n' "$3" > "$d/META-INF/module.xml"
    printf 'sf %s' "$3" > "$d/META-INF/SIGNER.SF"
    printf 'rsa %s' "$3" > "$d/META-INF/SIGNER.RSA"
    printf '%s' "$2" > "$d/com/acme/BTest.class"
    rm -f "$1"
    (cd "$d" && zip -q -r -X "$1" .)
    rm -r "$d"
}

@test "NG-version-same: changed jar bytes with the SAME vendorVersion as the deployed jar block the deploy (exit 51) (A2)" {
    # Mutation: NG-version-same -- skipping the version-bump precheck ships new bytes under an old version.
    _gate_env; _fake_gate 0
    _real_jar "$TMPDIR_T/modules/test-rt.jar" old 1788609301540
    _real_jar "$TMPDIR_T/fakebin/test/test-rt/build/libs/test-rt.jar" new 1788609309999
    cp "$TMPDIR_T/modules/test-rt.jar" "$TMPDIR_T/deployed.before"
    export FAKE_OLD_VERSION=2.1.0 FAKE_NEW_VERSION=2.1.0
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode C
    [ "$status" -eq 51 ]
    [[ "$output" == *"test-rt.jar bytes changed but vendorVersion is still 2.1.0"* ]]
    cmp -s "$TMPDIR_T/modules/test-rt.jar" "$TMPDIR_T/deployed.before"
    export FAKE_NEW_VERSION=2.1.1
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode C
    [ "$status" -eq 0 ]
}

@test "NG-version-unreadable: changed bytes with an unreadable vendorVersion block the deploy (exit 52) (A2)" {
    # Mutation: NG-version-unreadable -- an unreadable version read as "bumped" ships an unverified jar.
    _gate_env; _fake_gate 0
    printf 'old' > "$TMPDIR_T/modules/test-rt.jar"; printf 'new' > "$TMPDIR_T/fakebin/test/test-rt/build/libs/test-rt.jar"
    unset FAKE_OLD_VERSION FAKE_NEW_VERSION
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode C
    [ "$status" -eq 52 ]
}

# R10 shared compare (go/internal/vendorversion Cmp, the same helper build.sh's drift
# gate uses): the built version must be numerically HIGHER than the deployed one. 1.3 -> 1.3.0 is the same version
# (exit 51), a lower version is a downgrade (exit 54), and a differing non-numeric suffix cannot be ordered (exit 52).
@test "NG-version-trailing-zero: 1.3 -> 1.3.0 with changed bytes is not a bump (exit 51)" {
    # Mutation: NG-version-numeric -- a string compare reads '1.3' != '1.3.0' as a bump and deploys (exit 0).
    _gate_env; _fake_gate 0
    _real_jar "$TMPDIR_T/modules/test-rt.jar" old 1788609301540
    _real_jar "$TMPDIR_T/fakebin/test/test-rt/build/libs/test-rt.jar" new 1788609309999
    cp "$TMPDIR_T/modules/test-rt.jar" "$TMPDIR_T/deployed.before"
    export FAKE_OLD_VERSION=1.3 FAKE_NEW_VERSION=1.3.0
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode C
    [ "$status" -eq 51 ]
    cmp -s "$TMPDIR_T/modules/test-rt.jar" "$TMPDIR_T/deployed.before"
}

@test "NG-version-rebuild: a rebuild of unchanged sources under the same vendorVersion deploys; a jar that cannot be read is exit 52, never a pass" {
    # Mutation: NG-version-rebuild -- comparing whole jars refuses a retry (exit 51); reading an unreadable jar as unchanged passes it.
    _gate_env; _fake_gate 0
    _real_jar "$TMPDIR_T/modules/test-rt.jar" same 1788609301540
    _real_jar "$TMPDIR_T/fakebin/test/test-rt/build/libs/test-rt.jar" same 1788609309999
    ! cmp -s "$TMPDIR_T/modules/test-rt.jar" "$TMPDIR_T/fakebin/test/test-rt/build/libs/test-rt.jar" || return 1
    export FAKE_OLD_VERSION=1.0 FAKE_NEW_VERSION=1.0
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode C
    [ "$status" -eq 0 ]
    [[ "$output" == *"version-bump ok (test-rt.jar: vendorVersion still 1.0, and the same content as the deployed jar apart from the build time and host, the manifest digests and the signature files)"* ]]
    cmp -s "$TMPDIR_T/modules/test-rt.jar" "$TMPDIR_T/fakebin/test/test-rt/build/libs/test-rt.jar"

    printf 'not a zip' > "$TMPDIR_T/modules/test-rt.jar"
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode C
    [ "$status" -eq 52 ]
    [[ "$output" == *"version-bump precheck cannot verify test-rt.jar: cannot compare its content with the deployed jar: $TMPDIR_T/modules/test-rt.jar is not a readable jar ("* ]]
    [ "$(cat "$TMPDIR_T/modules/test-rt.jar")" = "not a zip" ]
}

@test "NG-version-downgrade: a LOWER vendorVersion (1.4 -> 1.3, 1.10 -> 1.9) is refused as a downgrade (exit 54); 1.3 -> 1.4 deploys" {
    # Mutation: NG-version-downgrade -- dropping the Cmp -1 branch counts a downgrade as a bump (exit 0).
    _gate_env; _fake_gate 0
    printf 'old' > "$TMPDIR_T/modules/test-rt.jar"; printf 'new' > "$TMPDIR_T/fakebin/test/test-rt/build/libs/test-rt.jar"
    local pair
    for pair in 1.4:1.3 1.10:1.9; do
        export FAKE_OLD_VERSION="${pair%%:*}" FAKE_NEW_VERSION="${pair##*:}"
        run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode C
        [ "$status" -eq 54 ]
        [[ "$output" == *"vendorVersion went down (${pair%%:*} -> ${pair##*:})"* ]]
        [ "$(cat "$TMPDIR_T/modules/test-rt.jar")" = "old" ]
    done
    export FAKE_OLD_VERSION=1.3 FAKE_NEW_VERSION=1.4
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode C
    [ "$status" -eq 0 ]
    [[ "$output" == *"version-bump ok (test-rt.jar: 1.3 -> 1.4)"* ]]
}

@test "NG-version-incomparable: a differing non-numeric suffix (1.3 -> 1.3-rc) cannot be ordered (exit 52), never a bump" {
    # Mutation: NG-version-incomparable -- treating an incomparable pair as 'different' deploys an unordered version.
    _gate_env; _fake_gate 0
    printf 'old' > "$TMPDIR_T/modules/test-rt.jar"; printf 'new' > "$TMPDIR_T/fakebin/test/test-rt/build/libs/test-rt.jar"
    export FAKE_OLD_VERSION=1.3 FAKE_NEW_VERSION=1.3-rc
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode C
    [ "$status" -eq 52 ]
    [[ "$output" == *"not comparable"* ]]
}

# ===========================================================================
# kit-backlog B5.0 — every gate and precheck runs on build/libs BEFORE the copy: a failing type-count verify,
# cache-buster check or verify-module gate leaves STATION_MODULES_DIR untouched (before the fix the gate and the
# verify ran after copy_jars, so an exit 50 still left the new jar installed).
# ===========================================================================
@test "NG-order-gate: a failing verify-module gate copies nothing to the station (mode A and C)" {
    # Mutation: NG-order-gate -- running the gate after copy_jars installs a jar the gate rejected.
    _gate_env; _fake_gate 1
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 50 ]
    [ -f "$TMPDIR_T/gate.args" ]
    [ ! -e "$TMPDIR_T/modules/test-rt.jar" ]
    [ ! -e "$TMPDIR_T/modules/test-ux.jar" ]
    [[ "$output" != *"copy ok"* ]]
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode C
    [ "$status" -eq 50 ]
    [ ! -e "$TMPDIR_T/modules/test-rt.jar" ]
}

@test "NG-order-verify: a failing type-count verify or cache-buster check copies nothing (mode A and B)" {
    # Mutation: NG-order-verify -- verifying the type count after copy_jars installs a jar with the wrong types.
    _gate_env; _fake_gate 0
    FAKE_UNZIP_TYPES=99 run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 50 ]
    [ ! -e "$TMPDIR_T/modules/test-rt.jar" ]
    [ ! -e "$TMPDIR_T/modules/test-ux.jar" ]
    BUILD_ID=b42 run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode B
    [ "$status" -eq 50 ]
    [[ "$output" == *"?v=b42"* ]]
    [ ! -e "$TMPDIR_T/modules/test-ux.jar" ]
}

@test "NG-order-pass: with every gate green the jars are copied after the gate ran" {
    _gate_env; _fake_gate 0
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 0 ]
    [ -e "$TMPDIR_T/modules/test-rt.jar" ]
    [ -e "$TMPDIR_T/modules/test-ux.jar" ]
    # the gate line precedes the copy line
    local gate_line copy_line
    gate_line="$(printf '%s\n' "$output" | grep -n 'gate ok' | cut -d: -f1)"
    copy_line="$(printf '%s\n' "$output" | grep -n 'copy ok' | cut -d: -f1)"
    [ -n "$gate_line" ]
    [ -n "$copy_line" ]
    [ "$gate_line" -lt "$copy_line" ]
}

# ===========================================================================
# The env file and the command line (retro go-ng-deploy-port Δ1), and the staged copy (Δ2)
# ===========================================================================

@test "NG-flag-wins: a flag wins over the same option variable in the env file" {
    printf 'MODE=B\nNO_DEPLOY=0\n' >> "$TMPDIR_T/dot_env_local"
    run bash "$SCRIPT" --env-file "$TMPDIR_T/dot_env_local" --no-backup --no-deploy --mode C
    [ "$status" -eq 0 ]
    [[ "$output" == *"build: mode C"* ]]
    [[ "$output" == *"no-deploy: jars in build/libs/"* ]]
    grep -q ":test-rt:jar" "$TMPDIR_T/gradlew.calls.log"
    run grep -q ":test-ux:" "$TMPDIR_T/gradlew.calls.log"
    [ "$status" -ne 0 ]
}

@test "NG-env-mode: without a flag the env file still sets the option" {
    printf 'MODE=C\n' >> "$TMPDIR_T/dot_env_local"
    run bash "$SCRIPT" --env-file "$TMPDIR_T/dot_env_local" --no-backup --no-deploy
    [ "$status" -eq 0 ]
    [[ "$output" == *"build: mode C"* ]]
}

@test "NG-env-guard-gate: MODE=B or VERIFY_MODULE_BIN in the env file is refused with file and line (exit 10) unless the flag is given" {
    _gate_env; _fake_gate 1   # the gate would fail if it ran
    printf 'MODE=B\n' >> "$TMPDIR_T/env_gate"
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup
    [ "$status" -eq 10 ]
    [[ "$output" == *"env file $TMPDIR_T/env_gate: line 10: MODE=B would switch the verify-module gate off"* ]]
    [[ "$output" == *"give --mode B on the command line"* ]]
    [ ! -e "$TMPDIR_T/gradlew.calls.log" ]
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --no-deploy --mode B
    [ "$status" -eq 0 ]
    [[ "$output" == *"build: mode B"* ]]

    _gate_env; _fake_gate 1; rm -f "$TMPDIR_T/gradlew.calls.log"
    printf 'VERIFY_MODULE_BIN=/bin/true\n' >> "$TMPDIR_T/env_gate"
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 10 ]
    [[ "$output" == *"env file $TMPDIR_T/env_gate: line 10: VERIFY_MODULE_BIN would switch the verify-module gate off"* ]]
    [[ "$output" == *"give --no-gate on the command line"* ]]
    [ ! -e "$TMPDIR_T/gradlew.calls.log" ]
    [ ! -e "$TMPDIR_T/modules/test-rt.jar" ]
}

@test "NG-env-guard: NO_GATE=1 or NO_BACKUP=1 in the env file is refused with file and line (exit 10) unless the flag is given" {
    _gate_env; _fake_gate 1   # the gate would fail if it ran
    printf 'NO_GATE=1\n' >> "$TMPDIR_T/env_gate"
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 10 ]
    [[ "$output" == *"env file $TMPDIR_T/env_gate: line 10: NO_GATE would switch the verify-module gate off"* ]]
    [[ "$output" == *"give --no-gate on the command line"* ]]
    [ ! -e "$TMPDIR_T/gradlew.calls.log" ]
    [ ! -e "$TMPDIR_T/modules/test-rt.jar" ]
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A --no-gate
    [ "$status" -eq 0 ]
    [ ! -f "$TMPDIR_T/gate.args" ]

    _gate_env; _fake_gate 0
    printf 'NO_BACKUP=1\n' >> "$TMPDIR_T/env_gate"
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --mode A
    [ "$status" -eq 10 ]
    [[ "$output" == *"env file $TMPDIR_T/env_gate: line 10: NO_BACKUP would switch the backup off"* ]]
    [[ "$output" == *"give --no-backup on the command line"* ]]
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --mode A --no-backup
    [ "$status" -eq 0 ]
}

@test "NG-copy-staged: a deploy replaces both jars and leaves no staged file in the station" {
    _gate_env; _fake_gate 0
    printf 'same' > "$TMPDIR_T/modules/test-rt.jar"
    printf 'same' > "$TMPDIR_T/fakebin/test/test-rt/build/libs/test-rt.jar"   # same bytes: no version to compare
    : > "$TMPDIR_T/modules/test-ux.jar.ng-deploy-new"                          # left by a killed run
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 0 ]
    [[ "$output" == *"copy ok"* ]]
    [ -e "$TMPDIR_T/modules/test-rt.jar" ]
    [ -e "$TMPDIR_T/modules/test-ux.jar" ]
    run find "$TMPDIR_T/modules" -name '*.ng-deploy-new'
    [ -z "$output" ]
}

@test "NG-copy-stale-old: a jar left aside by a deploy that did not finish is never deleted; the deploy refuses (exit 40) and changes nothing" {
    _gate_env; _fake_gate 0
    printf 'previous' > "$TMPDIR_T/modules/test-ux.jar.ng-deploy-old"
    : > "$TMPDIR_T/modules/test-ux.jar"   # the same bytes as the built jar: beside its jar the file aside still stops the deploy
    before="$(cd "$TMPDIR_T/modules" && ls -la --time-style=+ . && cat ./*)"
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 40 ]
    [[ "$output" == *"copy refused: a deploy that did not finish left previous jars aside in $TMPDIR_T/modules; nothing was copied or changed"* ]]
    [[ "$output" == *"test-ux.jar present, test-ux.jar.ng-deploy-old present"* ]]
    [[ "$output" == *"rename each .ng-deploy-old file over its jar"* ]]
    [[ "$output" == *"delete the .ng-deploy-old files"* ]]
    [ "$(cd "$TMPDIR_T/modules" && ls -la --time-style=+ . && cat ./*)" = "$before" ]

    rm "$TMPDIR_T/modules/test-ux.jar.ng-deploy-old"   # the operator keeps what is there
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 0 ]
    run find "$TMPDIR_T/modules" -name '*.ng-deploy-*'
    [ -z "$output" ]
}

@test "NG-gate-signer: the deploy gate gets the signer build.sh expects; with none configured the deploy WARNs and the exit code is the same" {
    # Mutation: NG-gate-signer -- a gate that is not told the signer passes a jar signed with any key.
    _gate_env; _fake_gate 0
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 0 ]
    [[ "$(cat "$TMPDIR_T/gate.args")" == "--src $TMPDIR_T/fakebin/test --expected-signer testsigner "*"test-rt.jar "*"test-ux.jar" ]]
    [[ "$output" != *"niagaraSigningAlias"* ]]

    rm "$TMPDIR_T/fakebin/gradle.properties"
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 0 ]
    [[ "$(cat "$TMPDIR_T/gate.args")" != *"--expected-signer"* ]]
    [[ "$output" == *"[ng-deploy] WARN no niagaraSigningAlias is configured and no deployed jar gives a signer, so the gate accepts any signature"* ]]
    [[ "$output" == *"(set it: ORG_GRADLE_PROJECT_niagaraSigningAlias=<alias> in the env file, or niagaraSigningAlias=<alias> in gradle.properties)"* ]]

    printf 'ORG_GRADLE_PROJECT_niagaraSigningAlias=fromenvfile\n' >> "$TMPDIR_T/env_gate"
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 0 ]
    grep -q -- "--expected-signer fromenvfile " "$TMPDIR_T/gate.args"
}

# ===========================================================================
# <NIAGARA_HOME>/modules: gradle's :jar step installs the built jars there before any check of the deploy runs.
# ===========================================================================

# helper: a gradlew that installs the built rt and ux jars into <NIAGARA_HOME>/modules on :jar, as the vendor's
# plugin does, then exits $FAKE_GRADLEW_EXIT
_installing_gradlew() {
    cat > "$TMPDIR_T/fakebin/gradlew" <<STUB
#!/usr/bin/env bash
printf '%s\n' "\$*" >> "$TMPDIR_T/gradlew.calls.log"
case "\$*" in *:jar*)
    for p in rt ux; do cp "$TMPDIR_T/fakebin/test/test-\$p/build/libs/test-\$p.jar" "$TMPDIR_T/niagara-home/modules/test-\$p.jar"; done ;;
esac
exit "\${FAKE_GRADLEW_EXIT:-0}"
STUB
    chmod +x "$TMPDIR_T/fakebin/gradlew"
}

@test "NG-home-real: a niagara_home that is not a mirror is refused before gradle runs (exit 10); --install-into-home opts in" {
    # Mutation: NG-home-real -- without the refusal gradle replaces the jars of a real installation before any check.
    _gate_env; _fake_gate 0
    mkdir -p "$TMPDIR_T/niagara-home/modules"
    printf 'installed' > "$TMPDIR_T/niagara-home/modules/test-rt.jar"
    rm "$TMPDIR_T/niagara-home/.niagara-mirror"
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --mode A --with-slotomatic
    [ "$status" -eq 10 ]
    [[ "$output" == "[ng-deploy] ERROR NIAGARA_HOME $TMPDIR_T/niagara-home is a real niagara_home (no .niagara-mirror): gradle's :jar step would replace the jars of its modules/ before any check runs; build against a mirror (toolbelt/mirror-niagara-home.sh <niagara_home> <mirror_dir>), or pass --install-into-home" ]]
    [ ! -e "$TMPDIR_T/gradlew.calls.log" ]
    [ ! -e "$TMPDIR_T/_backups" ]
    [ "$(cat "$TMPDIR_T/niagara-home/modules/test-rt.jar")" = "installed" ]

    printf 'INSTALL_INTO_HOME=1\n' >> "$TMPDIR_T/env_gate"   # the env file alone does not opt in
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 10 ]
    [[ "$output" == *"INSTALL_INTO_HOME would switch the refusal of a real niagara_home off, which an env file may not do: give --install-into-home on the command line"* ]]
    [ ! -e "$TMPDIR_T/gradlew.calls.log" ]

    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A --install-into-home
    [ "$status" -eq 0 ]
    [[ "$output" == *"copy ok"* ]]
}

@test "NG-home-restore: a check that fails after the build leaves <NIAGARA_HOME>/modules as it was before the build" {
    # Mutation: NG-home-restore -- without the restore the jars of a refused deploy stay in <NIAGARA_HOME>/modules.
    _gate_env; _fake_gate 1; _installing_gradlew
    mkdir -p "$TMPDIR_T/niagara-home/modules"
    printf 'previous-rt' > "$TMPDIR_T/niagara-home/modules/test-rt.jar"   # no ux jar was installed before
    printf 'new-rt' > "$TMPDIR_T/fakebin/test/test-rt/build/libs/test-rt.jar"
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 50 ]
    [[ "$output" == *"verify-module gate failed"* ]]
    [[ "$output" == *"NOTICE: the jars the build installed in $TMPDIR_T/niagara-home/modules were taken out again: it holds the module jars it held before the build"* ]]
    [ "$(cat "$TMPDIR_T/niagara-home/modules/test-rt.jar")" = "previous-rt" ]
    [ "$(ls "$TMPDIR_T/niagara-home/modules")" = "test-rt.jar" ]
    [ ! -e "$TMPDIR_T/modules/test-rt.jar" ]

    _fake_gate 0   # the same deploy with every check green keeps what the build installed
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 0 ]
    [[ "$output" != *"NOTICE: the jars the build installed"* ]]
    [ "$(cat "$TMPDIR_T/niagara-home/modules/test-rt.jar")" = "new-rt" ]
    [ "$(ls "$TMPDIR_T/niagara-home/modules")" = "test-rt.jar"$'\n'"test-ux.jar" ]
}

@test "NG-home-same: a STATION_MODULES_DIR that is <NIAGARA_HOME>/modules is verified, not copied, and still gets the version precheck" {
    # Mutation: NG-home-same -- comparing the build with the jar gradle just installed skips the version precheck.
    _gate_env; _fake_gate 0; _installing_gradlew
    mkdir -p "$TMPDIR_T/niagara-home/modules"
    printf 'previous-rt' > "$TMPDIR_T/niagara-home/modules/test-rt.jar"
    printf 'new-rt' > "$TMPDIR_T/fakebin/test/test-rt/build/libs/test-rt.jar"
    sed -i "s|^STATION_MODULES_DIR=.*|STATION_MODULES_DIR=$TMPDIR_T/niagara-home/modules/|" "$TMPDIR_T/env_gate"
    export FAKE_OLD_VERSION=1.4 FAKE_NEW_VERSION=1.3
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 54 ]
    [[ "$output" == *"STATION_MODULES_DIR is <NIAGARA_HOME>/modules ($TMPDIR_T/niagara-home/modules)"* ]]
    [ "$(cat "$TMPDIR_T/niagara-home/modules/test-rt.jar")" = "previous-rt" ]
    [ "$(ls "$TMPDIR_T/niagara-home/modules")" = "test-rt.jar" ]

    export FAKE_NEW_VERSION=1.5
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 0 ]
    [[ "$output" == *"version-bump ok (test-rt.jar: 1.4 -> 1.5)"* ]]
    [[ "$output" == *"copy: nothing to copy, the build installed test-rt.jar test-ux.jar in $TMPDIR_T/niagara-home/modules (the same bytes as build/libs)"* ]]
    [ "$(cat "$TMPDIR_T/niagara-home/modules/test-rt.jar")" = "new-rt" ]
    run find "$TMPDIR_T/niagara-home/modules" -name '*.ng-deploy-*'
    [ -z "$output" ]
}

@test "NG-next-steps: a deploy that went through ends with what it wrote, who signed, what to do next and how to undo; a failed one does not" {
    _gate_env; _fake_gate 0
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 0 ]
    [[ "$output" == *"[ng-deploy] station restart required (Java classes deployed)"$'\n'"[ng-deploy] done: test-rt.jar test-ux.jar copied to $TMPDIR_T/modules"$'\n'"[ng-deploy] signer: test-rt.jar carries no complete signature, test-ux.jar carries no complete signature; the gate checks who signed a jar, not whether the station trusts that signer: the station decides when it loads the module"$'\n'* ]]
    [[ "$output" == *"[ng-deploy] next: restart the station; ng-deploy did not look at whether one is running, and stops or starts none"* ]]
    [[ "$output" == *"toolbelt/triage-console.sh --package <the Java package of the module> --module-prefix test <station-dir>/console*.txt"* ]]
    [[ "${lines[-1]}" == "[ng-deploy] undo: to remove the module again, take its components out of the station, stop the station and delete $TMPDIR_T/modules/test-rt.jar $TMPDIR_T/modules/test-ux.jar" ]]

    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode B   # no Java classes: no restart to do
    [ "$status" -eq 0 ]
    [[ "$output" == *"[ng-deploy] done: test-ux.jar copied to $TMPDIR_T/modules"* ]]
    [[ "$output" != *"next: restart the station"* ]]

    _fake_gate 1
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 50 ]
    [[ "$output" != *"[ng-deploy] done:"* ]]
    [[ "$output" != *"[ng-deploy] undo:"* ]]
}

# helper: the previous rt jar of the module in <NIAGARA_HOME>/modules, and a gradlew that installs the built jars there
_home_with_previous_rt() {
    _gate_env; _installing_gradlew
    mkdir -p "$TMPDIR_T/niagara-home/modules"
    printf 'previous-rt' > "$TMPDIR_T/niagara-home/modules/test-rt.jar"
    printf 'new-rt' > "$TMPDIR_T/fakebin/test/test-rt/build/libs/test-rt.jar"
}

@test "NG-lock: a run lock another run holds, or a killed run left, is refused (exit 10) and never removed; a run removes its own" {
    # Mutation: NG-lock -- without the lock two runs on one <NIAGARA_HOME>/modules each put back or remove the other's jars.
    _home_with_previous_rt; _fake_gate 0
    printf 'pid 4194000\nhost elsewhere\nstarted 2020-01-02T03:04:05Z\n' > "$TMPDIR_T/niagara-home/modules/.ng-deploy.lock"
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 10 ]
    [[ "$output" == "[ng-deploy] ERROR another ng-deploy run holds $TMPDIR_T/niagara-home/modules/.ng-deploy.lock (pid 4194000 on host elsewhere, started 2020-01-02T03:04:05Z, "*" ago): two runs on one modules directory lose jars. A lock is never removed for the run that made it: when ps -p 4194000 -o comm= on elsewhere does not print ng-deploy, that run was killed; delete the file and deploy again" ]]
    [ ! -e "$TMPDIR_T/gradlew.calls.log" ]
    [ "$(cat "$TMPDIR_T/niagara-home/modules/test-rt.jar")" = "previous-rt" ]
    [ "$(sed -n 1p "$TMPDIR_T/niagara-home/modules/.ng-deploy.lock")" = "pid 4194000" ]

    mv "$TMPDIR_T/niagara-home/modules/.ng-deploy.lock" "$TMPDIR_T/modules/.ng-deploy.lock"   # the station's is refused too
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 10 ]
    [[ "$output" == *"another ng-deploy run holds $TMPDIR_T/modules/.ng-deploy.lock (pid 4194000 on host elsewhere"* ]]
    [ ! -e "$TMPDIR_T/gradlew.calls.log" ]
    [ ! -e "$TMPDIR_T/niagara-home/modules/.ng-deploy.lock" ]   # the lock this run took first is released

    rm "$TMPDIR_T/modules/.ng-deploy.lock"   # by hand, as the line says; then the deploy goes through and leaves none
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 0 ]
    run find "$TMPDIR_T" -name '.ng-deploy.lock'
    [ -z "$output" ]
}

@test "NG-home-interrupt: a SIGINT during the build puts the jars of <NIAGARA_HOME>/modules back and ends with 130" {
    # Mutation: NG-home-interrupt -- without the watch the signal ends the run with the built jars in the home and its lock left.
    _home_with_previous_rt; _fake_gate 0
    head -n -1 "$TMPDIR_T/fakebin/gradlew" > "$TMPDIR_T/gradlew.head"   # all but its last line, the exit
    cat "$TMPDIR_T/gradlew.head" - > "$TMPDIR_T/fakebin/gradlew" <<'STUB'
case "$*" in *:jar*) kill -INT $PPID; sleep 0.5; exit 130 ;; esac
STUB
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 130 ]
    [[ "$output" == *"[ng-deploy] ERROR interrupted (SIGINT) before the copy; nothing was copied to the station"* ]]
    [[ "$output" == *"NOTICE: the jars the build installed in $TMPDIR_T/niagara-home/modules were taken out again"* ]]
    [ "$(cat "$TMPDIR_T/niagara-home/modules/test-rt.jar")" = "previous-rt" ]
    [ "$(ls -A "$TMPDIR_T/niagara-home/modules")" = "test-rt.jar" ]
    [ -z "$(ls -A "$TMPDIR_T/modules")" ]
}

@test "NG-home-aside-left: a jar left aside in <NIAGARA_HOME>/modules is refused before gradle runs (exit 10) and not touched" {
    # Mutation: NG-home-aside-left -- without the refusal the move aside replaces the only copy of the previous jar.
    _home_with_previous_rt; _fake_gate 0
    printf 'the only copy' > "$TMPDIR_T/niagara-home/modules/test-rt.jar.ng-deploy-old"
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 10 ]
    [[ "$output" == *"build refused: a deploy that did not finish left previous jars aside in $TMPDIR_T/niagara-home/modules; nothing was built or changed"* ]]
    [ ! -e "$TMPDIR_T/gradlew.calls.log" ]
    [ "$(cat "$TMPDIR_T/niagara-home/modules/test-rt.jar.ng-deploy-old")" = "the only copy" ]
    [ "$(cat "$TMPDIR_T/niagara-home/modules/test-rt.jar")" = "previous-rt" ]
}

@test "NG-home-same-verify: the same directory spelled as a relative path is recognised, and a jar the build did not install there fails the copy (exit 40)" {
    # Mutation: NG-home-same-verify -- without the byte check a deploy 'goes through' with a jar the build never installed.
    _home_with_previous_rt; _fake_gate 0
    sed -i "s|^STATION_MODULES_DIR=.*|STATION_MODULES_DIR=./niagara-home/modules|" "$TMPDIR_T/env_gate"
    sed -i 's|for p in rt ux; do|for p in rt; do|' "$TMPDIR_T/fakebin/gradlew"   # the ux jar is built and not installed
    export FAKE_OLD_VERSION=1.4 FAKE_NEW_VERSION=1.5
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 40 ]
    [[ "$output" == *"STATION_MODULES_DIR is <NIAGARA_HOME>/modules ($TMPDIR_T/niagara-home/modules)"* ]]
    [[ "$output" == *"copy failed for test-ux.jar: STATION_MODULES_DIR is <NIAGARA_HOME>/modules and the jar the build left there is missing or is not the one in build/libs"* ]]
    [ "$(cat "$TMPDIR_T/niagara-home/modules/test-rt.jar")" = "previous-rt" ]
    [ "$(ls -A "$TMPDIR_T/niagara-home/modules")" = "test-rt.jar" ]

    mkdir "$TMPDIR_T/niagara-home/modules/test-ux.jar"   # a directory under a jar name: refused as the copy refuses it
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 40 ]
    [[ "$output" == *"copy failed for test-ux.jar: the station holds a directory under that name, not a jar file; nothing was replaced"* ]]
    [ -d "$TMPDIR_T/niagara-home/modules/test-ux.jar" ]
}

@test "NG-closing-quiet: --no-deploy prints no closing lines, and a build that installed nothing says its jars were moved back" {
    # Mutation: NG-closing-quiet -- closing lines after --no-deploy tell the author to restart a station nothing was copied to.
    _home_with_previous_rt; _fake_gate 0
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A --no-deploy
    [ "$status" -eq 0 ]
    [[ "${lines[-1]}" == "[ng-deploy] no-deploy: jars in build/libs/" ]]
    [[ "$output" != *"[ng-deploy] done:"* ]]
    [[ "$output" != *"[ng-deploy] next:"* ]]
    [[ "$output" != *"[ng-deploy] undo:"* ]]

    printf 'previous-rt' > "$TMPDIR_T/niagara-home/modules/test-rt.jar"; rm -f "$TMPDIR_T/niagara-home/modules/test-ux.jar"
    printf '#!/usr/bin/env bash\nexit 1\n' > "$TMPDIR_T/fakebin/gradlew"
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A
    [ "$status" -eq 30 ]
    [[ "$output" == *"NOTICE: the build installed no jar in $TMPDIR_T/niagara-home/modules; the module jars moved aside for it were moved back: it holds the module jars it held before the build"* ]]
    [[ "$output" != *"were taken out again"* ]]
    [ "$(cat "$TMPDIR_T/niagara-home/modules/test-rt.jar")" = "previous-rt" ]
}

@test "NG-no-permission: a home jar the directory's permissions refuse to move names the permissions, not a running station (exit 10)" {
    # Mutation: NG-no-permission -- an operator told to stop a running station for a permission problem fails again (retro D1, review R3/R4).
    if [ "$(id -u)" -eq 0 ]; then skip "root writes everywhere"; fi
    _home_with_previous_rt; _fake_gate 0
    # slotomatic runs before the move aside: it makes the directory read-only
    printf '#!/usr/bin/env bash\ncase "$*" in *slotomatic*) chmod 555 "%s/niagara-home/modules" ;; esac\nexit 0\n' "$TMPDIR_T" > "$TMPDIR_T/fakebin/gradlew"
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode A --with-slotomatic
    [ "$status" -eq 10 ]
    [[ "$output" == *"build not started: test-rt.jar could not be moved aside in $TMPDIR_T/niagara-home/modules (Permission denied)"* ]]
    [[ "$output" == *"[ng-deploy]   $TMPDIR_T/niagara-home/modules does not let this user create, rename or remove files (no write permission): fix its permissions or ownership, then run again; a running station is not the cause"* ]]
    [[ "$output" != *"Workbench"* ]]
    [ "$(cat "$TMPDIR_T/niagara-home/modules/test-rt.jar")" = "previous-rt" ]
}

@test "NG-outside-git: a deploy from a directory that is not a git repository ends 0 with a NOTICE, not exit 41" {
    # Mutation: NG-outside-git -- a deploy that went through ended with an ERROR and exit 41 (retro D9).
    _gate_env; _fake_gate 0
    export FAKE_GIT_NOREPO=1
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode C
    [ "$status" -eq 0 ]
    [[ "$output" == *"copy ok"* ]]
    [[ "$output" == *"NOTICE: not a git repository; the deployed commit is not recorded in .last-deploy-sha: the next annotation check has no starting point"* ]]
    [[ "$output" != *"ERROR"* ]]
    [ ! -e "$TMPDIR_T/.last-deploy-sha" ]
}

@test "NG-mirror-isolation: a mirror build gets build.sh's user.home isolation and --gradle-prop, on every gradlew call" {
    # Mutation: NG-mirror-isolation -- without it gradle read the real ~/.tridium keystore and the signing alias was set by hand (retro D3).
    _gate_env; _fake_gate 0
    # why: SC2016 — the line is inserted into the stub as text; the stub expands it when gradlew runs
    # shellcheck disable=SC2016
    sed -i '2i printf "%s\\n" "${JAVA_TOOL_OPTIONS-}" >> "${TMPDIR_T}/gradlew.jto"' "$TMPDIR_T/fakebin/gradlew"
    run bash "$SCRIPT" --env-file "$TMPDIR_T/env_gate" --no-backup --mode C \
        --gradle-prop niagaraSigningAlias=propsigner --gradle-prop niagaraSigningProfile=/p/signing.xml
    [ "$status" -eq 0 ]
    [[ "$output" == *"mirror niagara_home: user.home isolated to $TMPDIR_T/niagara-home/.mirror-user-home"* ]]
    [[ "$(cat "$TMPDIR_T/gradlew.calls.log")" == "-Pniagara_home=$TMPDIR_T/niagara-home -Pniagara_user_home="*" -Porg.gradle.java.installations.paths=/usr/lib/jvm/java-8-openjdk-amd64 -PniagaraSigningAlias=propsigner -PniagaraSigningProfile=/p/signing.xml :test-rt:clean :test-rt:jar --no-daemon" ]]
    [[ "$(cat "$TMPDIR_T/gradlew.jto")" == *"-Duser.home=$TMPDIR_T/niagara-home/.mirror-user-home -Dmaven.repo.local="* ]]
    [[ "$(cat "$TMPDIR_T/gate.args")" == *"--expected-signer propsigner"* ]]
    [ -d "$TMPDIR_T/niagara-home/.mirror-user-home" ]
}
