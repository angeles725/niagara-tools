#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-no-md5-credential-digest.sh
# MD5 is broken for credential hashing; flag when MessageDigest.getInstance("MD5") appears
# in the same file as a credential context (BPassword, BCredentials, password/credential/pin).
# [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ13]

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  NMD="$KIT/toolbelt/lint-no-md5-credential-digest.sh"
  mkdir -p "$TMPDIR_T/Mod/src/com/x"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "NMD-usage: no arg exits 3" {
  run "$NMD"
  [ "$status" -eq 3 ]
}

@test "NMD-nondir: a non-directory arg exits 3" {
  run "$NMD" "$TMPDIR_T/does-not-exist"
  [ "$status" -eq 3 ]
}

@test "NMD1: file with MD5 but no credential context is clean (exit 0, no WARN)" {
  printf 'class BChksum {\n  byte[] h(byte[] d) throws Exception {\n    return java.security.MessageDigest.getInstance("MD5").digest(d);\n  }\n}\n' \
    > "$TMPDIR_T/Mod/src/com/x/BChksum.java"
  run "$NMD" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "NMD2: file with MD5 + a password token is WARNed (exit 0, advisory)" {
  printf 'class BAuth {\n  BPassword pwd;\n  byte[] h(String s) throws Exception { return java.security.MessageDigest.getInstance("MD5").digest(s.getBytes()); }\n}\n' \
    > "$TMPDIR_T/Mod/src/com/x/BAuth.java"
  run "$NMD" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"lint-no-md5-credential-digest"* ]]
  [[ "$output" == *"BAuth.java:3"* ]]
  # Named mutation NMD2: drop credential context guard -> NMD1 (MD5-only file) also WARNs.
}

@test "NMD2-strict: --strict promotes the WARN to exit 1" {
  printf 'class BAuth {\n  BPassword pwd;\n  byte[] h(String s) throws Exception { return java.security.MessageDigest.getInstance("MD5").digest(s.getBytes()); }\n}\n' \
    > "$TMPDIR_T/Mod/src/com/x/BAuth.java"
  run "$NMD" --strict "$TMPDIR_T/Mod"
  [ "$status" -eq 1 ]
  [[ "$output" == *"WARN"* ]]
}

@test "NMD3: SHA-256 + credentials is clean (exit 0, no WARN)" {
  printf 'class BAuth {\n  BPassword pwd;\n  byte[] h(String s) throws Exception { return java.security.MessageDigest.getInstance("SHA-256").digest(s.getBytes()); }\n}\n' \
    > "$TMPDIR_T/Mod/src/com/x/BAuth.java"
  run "$NMD" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "NMD4: commented-out MD5 does not trigger (comment strip)" {
  printf 'class BAuth {\n  BPassword pwd;\n  // java.security.MessageDigest.getInstance("MD5")\n  void f(){}\n}\n' \
    > "$TMPDIR_T/Mod/src/com/x/BAuth.java"
  run "$NMD" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
  # Named mutation: stop stripping // comments -> NMD4 would WARN.
}
