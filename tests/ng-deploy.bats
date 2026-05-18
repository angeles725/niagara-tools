#!/usr/bin/env bats
# tests/ng-deploy.bats — pure-logic unit tests for scripts/ng-deploy.sh
# Stub strategy: PATH-injected fakebin with controlled exit codes via env vars.
# No real gradlew, no real unzip to /mnt/c, no station dependency.

SCRIPT_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"
SCRIPT="$SCRIPT_DIR/scripts/ng-deploy.sh"

setup() {
    TMPDIR_T="$(mktemp -d)"
    export TMPDIR_T
    mkdir -p "$TMPDIR_T/fakebin" "$TMPDIR_T/modules"

    # --- fake gradlew ---
    cat > "$TMPDIR_T/fakebin/gradlew" << 'STUB'
#!/usr/bin/env bash
printf '%s\n' "$@" > "${TMPDIR_T}/gradlew.args"
exit "${FAKE_GRADLEW_EXIT:-0}"
STUB
    chmod +x "$TMPDIR_T/fakebin/gradlew"

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
# Test 3: --no-backup alone exits 20
# ---------------------------------------------------------------------------
@test "--no-backup alone exits 20" {
    run bash "$SCRIPT" --env-file "$TMPDIR_T/dot_env_local" --no-backup --mode A
    [ "$status" -eq 20 ]
    [[ "${output}${stderr}" == *"i-know-what-im-doing"* ]]
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
