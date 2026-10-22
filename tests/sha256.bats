#!/usr/bin/env bats
# sha256.bats — scripts/lib/sha256.sh, the shared digest helper of install-skill.sh and
# check-skill-drift.sh (quality round 8 T8a: one copy instead of two).
# Every test sources the lib under a PATH it builds, so the host's hash tools never leak in.

setup() {
  ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  LIB="$ROOT/scripts/lib/sha256.sh"
  TMPDIR_T="$(mktemp -d)"
  BIN="$TMPDIR_T/bin"
  mkdir -p "$BIN"
  ln -s "$(command -v cut)" "$BIN/cut"
  printf 'abc\n' > "$TMPDIR_T/f"
}
teardown() { rm -rf "$TMPDIR_T"; }

# Run <snippet> in a fresh bash that sourced the lib, with PATH limited to $BIN.
_lib() { run env PATH="$BIN" /bin/bash -c ". '$LIB'; $1"; }

@test "SHA1: sha256_of prints the sha256sum digest of the file" {
  ln -s "$(command -v sha256sum)" "$BIN/sha256sum"
  want="$(sha256sum "$TMPDIR_T/f" | cut -d' ' -f1)"
  _lib "sha256_of '$TMPDIR_T/f'"
  [ "$status" -eq 0 ]
  [ "$output" = "$want" ]
}

@test "SHA2: without sha256sum, sha256_of falls back to shasum -a 256" {
  cat > "$BIN/shasum" <<'SH'
#!/bin/bash
[ "$1 $2" = "-a 256" ] && printf '%s  %s\n' stubdigest "$3"
SH
  chmod +x "$BIN/shasum"
  _lib "sha256_of '$TMPDIR_T/f'"
  [ "$status" -eq 0 ]
  [ "$output" = "stubdigest" ]
}

@test "SHA3: sha256_require with no hash tool returns 3 and names the caller's prefix" {
  # Mutation: SHA3 -- a require that returns 0 without a tool lets callers hash into empty digests.
  _lib "sha256_require my-script"
  [ "$status" -eq 3 ]
  [ "$output" = "my-script: no sha256 tool found (sha256sum or shasum)" ]
}

@test "SHA4: sha256_require with a hash tool returns 0 silently" {
  ln -s "$(command -v sha256sum)" "$BIN/sha256sum"
  _lib "sha256_require my-script"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "SHA5: the lib is clean under shellcheck (CI's scripts/*.sh glob does not reach lib/)" {
  command -v shellcheck >/dev/null 2>&1 || skip "shellcheck not installed"
  run shellcheck "$LIB"
  [ "$status" -eq 0 ]
}
