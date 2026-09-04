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

    # --- fake gradlew ---
    # Writes every invocation to gradlew.calls.log (one line per call, args space-separated)
    # and overwrites gradlew.args with the last invocation (backward-compat for T1-T17).
    cat > "$TMPDIR_T/fakebin/gradlew" << 'STUB'
#!/usr/bin/env bash
printf '%s\n' "$@" > "${TMPDIR_T}/gradlew.args"
printf '%s\n' "$*" >> "${TMPDIR_T}/gradlew.calls.log"
exit "${FAKE_GRADLEW_EXIT:-0}"
STUB
    chmod +x "$TMPDIR_T/fakebin/gradlew"

    # --- fake git ---
    # Dispatches by $1: diff → FAKE_GIT_DIFF_OUTPUT; rev-parse (--git-dir) → exit 0;
    # rev-parse HEAD → FAKE_GIT_REV_PARSE_OUTPUT; cat-file → FAKE_GIT_CAT_FILE_EXIT.
    cat > "$TMPDIR_T/fakebin/git" << 'STUB'
#!/usr/bin/env bash
case "$1" in
    diff)
        printf '%s\n' "${FAKE_GIT_DIFF_OUTPUT:-}"
        exit 0
        ;;
    rev-parse)
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

    # Write minimal .env.local for tests that need a valid environment
    # why: .env.local path is dynamic (TMPDIR_T); not following is expected
    # shellcheck disable=SC1090,SC1091
    cat > "$TMPDIR_T/dot_env_local" << ENVEOF
MODULE_NAME=test
GRADLEW_PATH=$TMPDIR_T/fakebin/gradlew
NIAGARA_HOME=/mnt/c/Niagara/iC-Niagara-4.13.2.18
NIAGARA_USER_HOME=/mnt/c/Users/equipo/Niagara4.13/test
JAVA_HOME=/usr/lib/jvm/java-8-openjdk-amd64
STATION_MODULES_DIR=$TMPDIR_T/modules
EXPECTED_RT_TYPES=9
EXPECTED_UX_TYPES=2
ENVEOF

    cd "$TMPDIR_T" || return 1
    # BATS_TEST_MODE is set per-test: =1 when sourcing functions directly,
    # =0 (or unset) when running the full script as a subprocess.
    export BATS_TEST_MODE=0
}

teardown() {
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
NIAGARA_HOME=/mnt/c/Niagara/iC-Niagara-4.13.2.18
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
    run bash "$SCRIPT" \
        --env-file "$TMPDIR_T/dot_env_local" \
        --no-deploy \
        --mode A
    # Even if exit is non-zero due to other issues, check that any _backups/ reference
    # in stdout matches the naming pattern
    [[ "$output" =~ _backups/test-pre-[0-9]{8}-[0-9]{6}\.tar\.gz ]]
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

# ---------------------------------------------------------------------------
# Test 10: verify_jar match returns 0
# ---------------------------------------------------------------------------
@test "verify_jar match returns 0" {
    # Source the script functions only (BATS_TEST_MODE=1 suppresses main)
    # TMPDIR_T is exported so inner shell can reference it; PATH prepend done inside
    FAKE_UNZIP_TYPES=9 BATS_TEST_MODE=1 \
    run bash -c "
        export PATH=\"\${TMPDIR_T}/fakebin:\${PATH}\"
        source '$SCRIPT'
        verify_jar 'fake.jar' 9
    "
    [ "$status" -eq 0 ]
}

# ---------------------------------------------------------------------------
# Test 11: verify_jar mismatch exits 50
# ---------------------------------------------------------------------------
@test "verify_jar mismatch exits 50" {
    FAKE_UNZIP_TYPES=8 BATS_TEST_MODE=1 \
    run bash -c "
        export PATH=\"\${TMPDIR_T}/fakebin:\${PATH}\"
        source '$SCRIPT'
        verify_jar 'fake.jar' 9
    "
    [ "$status" -eq 50 ]
}

# ---------------------------------------------------------------------------
# Test 12: verify_cachebuster skipped when BUILD_ID unset
# ---------------------------------------------------------------------------
@test "verify_cachebuster skipped when BUILD_ID unset" {
    BATS_TEST_MODE=1 \
    run bash -c "
        export PATH=\"\${TMPDIR_T}/fakebin:\${PATH}\"
        unset BUILD_ID
        source '$SCRIPT'
        verify_cachebuster 'fake-ux.jar'
    "
    [ "$status" -eq 0 ]
}

# ---------------------------------------------------------------------------
# Test 13: verify_cachebuster exits 50 when BUILD_ID set and not found
# ---------------------------------------------------------------------------
@test "verify_cachebuster exits 50 when BUILD_ID set and not found" {
    BUILD_ID=abc123 FAKE_UNZIP_TYPES=0 FAKE_INDEX_HTML=1 FAKE_BUILD_ID=WRONG BATS_TEST_MODE=1 \
    run bash -c "
        export PATH=\"\${TMPDIR_T}/fakebin:\${PATH}\"
        source '$SCRIPT'
        verify_cachebuster 'fake-ux.jar'
    "
    [ "$status" -eq 50 ]
}

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
    export FAKE_GIT_REV_PARSE_OUTPUT="abc1234abc1234abc1234abc1234abc1234abc1234"
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
NIAGARA_HOME=/mnt/c/Niagara/iC-Niagara-4.13.2.18
NIAGARA_USER_HOME=/mnt/c/Users/equipo/Niagara4.13/test
JAVA_HOME=/usr/lib/jvm/java-8-openjdk-amd64
STATION_MODULES_DIR=$TMPDIR_T/modules
EXPECTED_RT_TYPES=9
EXPECTED_UX_TYPES=9
ENVEOF
    run bash "$SCRIPT" \
        --env-file "$TMPDIR_T/env_t24" \
        --no-backup \
        --i-know-what-im-doing \
        --mode A \
        --no-gate
    [ "$status" -eq 0 ]
    [ -f "$TMPDIR_T/.last-deploy-sha" ]
    [ "$(cat "$TMPDIR_T/.last-deploy-sha")" = "abc1234abc1234abc1234abc1234abc1234abc1234" ]
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

_gate_env() {   # env whose expected type counts match the fake unzip (9) so verify_jar passes
    cat > "$TMPDIR_T/env_gate" <<ENVEOF
MODULE_NAME=test
GRADLEW_PATH=$TMPDIR_T/fakebin/gradlew
NIAGARA_HOME=/mnt/c/Niagara/iC-Niagara-4.13.2.18
NIAGARA_USER_HOME=/mnt/c/Users/equipo/Niagara4.13/test
JAVA_HOME=/usr/lib/jvm/java-8-openjdk-amd64
STATION_MODULES_DIR=$TMPDIR_T/modules
EXPECTED_RT_TYPES=9
EXPECTED_UX_TYPES=9
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

# ---------------------------------------------------------------------------
# Campaign 3 C3-PR3 (RED-first) — lesson B8 (ng-deploy-type-count)
# A real module.xml wraps N <type .../> in a <types> element. verify_jar counts with
# grep "<type" (NO space), which ALSO matches the <types> wrapper → off-by-one (found N+1),
# so a correct jar fails "expected N, found N+1". Fix: grep "<type " (WITH a space).
# Bite: put a <types> wrapper in the fake unzip output; a fixed count matches, the buggy one over-counts.
# ---------------------------------------------------------------------------
@test "verify_jar counts <type entries, NOT the <types> wrapper (lesson B8, off-by-one)" {
    # fake unzip that emits the real shape: a <types> wrapper around 3 <type .../> entries
    cat > "$TMPDIR_T/fakebin/unzip" <<'STUB'
#!/usr/bin/env bash
echo '<types>'
for i in 1 2 3; do printf '<type name="FakeType%s"/>\n' "$i"; done
echo '</types>'
exit 0
STUB
    chmod +x "$TMPDIR_T/fakebin/unzip"
    BATS_TEST_MODE=1 run bash -c "
        export PATH=\"\${TMPDIR_T}/fakebin:\${PATH}\"
        source '$SCRIPT'
        verify_jar 'fake.jar' 3
    "
    [ "$status" -eq 0 ]   # 3 real <type entries; the <types> wrapper must NOT be counted (buggy grep finds 4 → exit 50)
}

# ---------------------------------------------------------------------------
# Campaign 3 C3-PR3 — lesson B10 (ng-deploy-backup-liviano-y-autopurga)
# Promoted rule: lightweight backup DEFAULT (module's own jars only) + keep-N autopurge,
# with --no-backup (opt-in, gate removed above) and --full-backup (old whole-dir) as flags.
# NOTE: setup() fakes `tar` to a no-op; these tests set a REAL PATH so backup actually
# archives + is listable (the suite's other backup tests assert the message/argv, not contents).
# ---------------------------------------------------------------------------

@test "backup DEFAULT is lightweight — only the module's own jars, NOT a sibling module's jar (B10 i)" {
    mkdir -p "$TMPDIR_T/modules"
    : > "$TMPDIR_T/modules/test-rt.jar"; : > "$TMPDIR_T/modules/test-ux.jar"
    : > "$TMPDIR_T/modules/Other-rt.jar"      # a sibling module's jar — must NOT be in the backup
    BATS_TEST_MODE=1 run bash -c "
        export PATH=/usr/bin:/bin:/usr/local/bin
        export MODULE_NAME=test STATION_MODULES_DIR='$TMPDIR_T/modules'
        cd '$TMPDIR_T'
        source '$SCRIPT'
        backup
        tar tzf _backups/test-pre-*.tar.gz
    "
    [ "$status" -eq 0 ]
    [[ "$output" == *"test-rt.jar"* ]] && [[ "$output" == *"test-ux.jar"* ]]
    [[ "$output" != *"Other-rt.jar"* ]]          # RED today: whole-dir tar includes the sibling
}

@test "backup auto-purges to keep-N (default 3) — 5 backups collapse to the 3 newest, oldest gone (B10 ii)" {
    mkdir -p "$TMPDIR_T/modules" "$TMPDIR_T/_backups"; : > "$TMPDIR_T/modules/test-rt.jar"
    for n in 1 2 3 4; do touch -t "2026010${n}0000.00" "$TMPDIR_T/_backups/test-pre-2026010${n}-000000.tar.gz"; done
    BATS_TEST_MODE=1 run bash -c "
        export PATH=/usr/bin:/bin:/usr/local/bin
        export MODULE_NAME=test STATION_MODULES_DIR='$TMPDIR_T/modules'
        cd '$TMPDIR_T'
        source '$SCRIPT'
        backup                                   # the 5th; purge should keep only the 3 newest
        ls _backups/test-pre-*.tar.gz | wc -l
    "
    [ "$status" -eq 0 ]
    [ "$(printf '%s' "$output" | tail -1 | tr -d ' ')" -eq 3 ]                 # RED today: no purge → 5 remain
    [ ! -e "$TMPDIR_T/_backups/test-pre-20260101-000000.tar.gz" ]             # oldest purged
}

@test "--full-backup restores the whole-modules-dir backup (a sibling jar IS included) (B10 iv)" {
    mkdir -p "$TMPDIR_T/modules"
    : > "$TMPDIR_T/modules/test-rt.jar"; : > "$TMPDIR_T/modules/Other-rt.jar"
    BATS_TEST_MODE=1 run bash -c "
        export PATH=/usr/bin:/bin:/usr/local/bin
        export MODULE_NAME=test STATION_MODULES_DIR='$TMPDIR_T/modules' FULL_BACKUP=1
        cd '$TMPDIR_T'
        source '$SCRIPT'
        backup
        tar tzf _backups/test-pre-*.tar.gz
    "
    [ "$status" -eq 0 ]
    [[ "$output" == *"test-rt.jar"* ]] && [[ "$output" == *"Other-rt.jar"* ]]  # full = sibling included (guard; bites if --full-backup ignored)
}
