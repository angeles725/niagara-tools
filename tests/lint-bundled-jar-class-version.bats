#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-bundled-jar-class-version.sh
#
# N4 modules must target Java 8 (class-file major = 52). A bundled ext-jar with Java 9+
# bytecode (major > 52) throws a RAW UnsupportedClassVersionError at module load — a
# LinkageError that bypasses catch(Exception). [ev: retro module-hardening-failure-modes-deltas Δ11 BLD2]
#
# Fixtures build fake .class files: minimal class header = CAFEBABE + minor(2) + major(2).
#   Java 8 header: printf '\xca\xfe\xba\xbe\x00\x00\x00\x34'  (major = 0x34 = 52)
#   Java 9 header: printf '\xca\xfe\xba\xbe\x00\x00\x00\x35'  (major = 0x35 = 53)
# These 8-byte stubs are valid enough for od -j6 -N2 to read the major version field.
#
# All tests that build jars require `zip`; they skip with a clear reason when it is absent.

setup() {
    TMPDIR_T="$(mktemp -d)"
    export TMPDIR_T
    KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
    BJCV="$KIT/toolbelt/lint-bundled-jar-class-version.sh"
    mkdir -p "$TMPDIR_T/Mod/libs"
}

teardown() { rm -rf "$TMPDIR_T"; }

@test "BJCV-usage: no arg exits 3" {
    run "$BJCV"
    [ "$status" -eq 3 ]
}

@test "BJCV-nondir: a non-directory arg exits 3" {
    run "$BJCV" "$TMPDIR_T/does-not-exist"
    [ "$status" -eq 3 ]
}

@test "BJCV1: a module with a jar of only major-52 classes is clean (exit 0)" {
    if ! command -v zip >/dev/null 2>&1; then skip "zip not available"; fi
    # Java 8 class header: CAFEBABE 00 00 00 34 (major = 52)
    printf '\xca\xfe\xba\xbe\x00\x00\x00\x34\x00\x00' > "$TMPDIR_T/Foo.class"
    (cd "$TMPDIR_T" && zip -q Mod/libs/java8.jar Foo.class)
    run "$BJCV" "$TMPDIR_T/Mod"
    [ "$status" -eq 0 ]
    [[ "$output" != *"FAIL"* ]]
}

@test "BJCV2: a jar containing a major-53 class FAILs (exit 1, FAIL row present)" {
    if ! command -v zip >/dev/null 2>&1; then skip "zip not available"; fi
    # Java 9 class header: CAFEBABE 00 00 00 35 (major = 53)
    printf '\xca\xfe\xba\xbe\x00\x00\x00\x35\x00\x00' > "$TMPDIR_T/Bar.class"
    (cd "$TMPDIR_T" && zip -q Mod/libs/java9.jar Bar.class)
    run "$BJCV" "$TMPDIR_T/Mod"
    [ "$status" -eq 1 ]
    [[ "$output" == *"FAIL"* ]]
    [[ "$output" == *"lint-bundled-jar-class-version"* ]]
    [[ "$output" == *"Bar.class"* ]]
    [[ "$output" == *"major 53"* ]]
    # Named mutation: drop the major > 52 check -> BJCV2 exits 0 (no FAIL row).
}

@test "BJCV3: a module with NO jars is clean (exit 0, nothing to check)" {
    # Mod/libs/ is empty — no jars under module root
    run "$BJCV" "$TMPDIR_T/Mod"
    [ "$status" -eq 0 ]
    [[ "$output" != *"FAIL"* ]]
}

@test "BJCV4: a jar under build/ is ignored (build/ is pruned)" {
    if ! command -v zip >/dev/null 2>&1; then skip "zip not available"; fi
    # Java 9 class header in a jar placed under build/ — must NOT be flagged
    mkdir -p "$TMPDIR_T/Mod/build/libs"
    printf '\xca\xfe\xba\xbe\x00\x00\x00\x35\x00\x00' > "$TMPDIR_T/Baz.class"
    (cd "$TMPDIR_T" && zip -q Mod/build/libs/java9.jar Baz.class)
    run "$BJCV" "$TMPDIR_T/Mod"
    [ "$status" -eq 0 ]
    [[ "$output" != *"FAIL"* ]]
}
