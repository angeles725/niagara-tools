#!/usr/bin/env bats
# Tests for lint-uberjar-api-conflict.sh — a lib both uberjar()'d and api()'d = classloader conflict.
# [ev: retro 2026-09-19-module-wiring]
setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-uberjar-api-conflict.sh"
  mkdir -p "$TMPDIR_T/M"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "UAC-usage: no arg exits 3" { run "$L"; [ "$status" -eq 3 ]; }

@test "UAC1: uberjar + a DIFFERENT api dep is clean" {
  printf 'dependencies {\n  api(":baja")\n  uberjar("com.opencsv:opencsv:5.7.1")\n}\n' > "$TMPDIR_T/M/x-rt.gradle.kts"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" != *"WARN"* ]]
}
@test "UAC2: same coordinate uberjar'd AND api'd is flagged (WARN)" {
  printf 'dependencies {\n  api("com.fasterxml.jackson.core:jackson-core:2.13.1")\n  uberjar("com.fasterxml.jackson.core:jackson-core:2.13.1")\n}\n' > "$TMPDIR_T/M/x-rt.gradle.kts"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" == *"WARN"* ]]; [[ "$output" == *"jackson-core"* ]]
  # Named mutation: skip the comm -12 intersection -> UAC2 WARN vanishes.
}

@test "UAC-finderr: an unreadable sub-directory is an env error (exit 3, named on stderr), never a clean pass (audit A0)" {
  # Mutation: UAC-finderr -- ignoring the find exit status skips the unreadable directory's files and exits 0.
  # [ev: issue #226 R3-find-error-still-fail-open]
  mkdir -p "$TMPDIR_T/M/locked"
  printf '%s\n' 'x' > "$TMPDIR_T/M/locked/build.gradle.kts"
  chmod 000 "$TMPDIR_T/M/locked"
  if [ -r "$TMPDIR_T/M/locked" ]; then chmod 755 "$TMPDIR_T/M/locked"; skip "running as root: chmod 000 does not block reads"; fi
  run "$L" "$TMPDIR_T/M"
  chmod 755 "$TMPDIR_T/M/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"locked"* ]]
}

# Truth fix T1 (r10 Go port): the calls are read as Kotlin tokens (go/internal/ktssrc). The bash grep read
# raw text: a commented-out api(...) or one inside a string literal made a conflict, a call split over
# lines or written with blanks inside its parentheses was missed, and xapi( counted as api(.
@test "UAC3: comments and strings are not dependencies; multi-line and spaced calls are" {
  # Mutation: UAC3 -- matching raw text WARNs on the commented and string api() and misses the split call.
  cat > "$TMPDIR_T/M/x-rt.gradle.kts" <<'KTS'
dependencies {
  // api("g:cmt:1")
  uberjar("g:cmt:1")
  /* implementation("g:blk:1") */ uberjar("g:blk:1")
  val note = """api("g:str:1")"""
  uberjar("g:str:1")
  xapi("g:xapi:1"); uberjar("g:xapi:1")
  api(
    "g:ml:1"
  )
  uberjar("g:ml:1")
  implementation( "g:sp:1" )
  uberjar("g:sp:1")
}
KTS
  run "$L" "$TMPDIR_T/M"
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | grep -c 'is both')" -eq 2 ]
  [[ "$output" == *"g:ml is both"* && "$output" == *"g:sp is both"* ]]
}

# Truth fix T2: a coordinate held in a val (`val jackson = "g:a:1"`, then `api(jackson)`) is the same
# dependency as the literal; the bash grep saw only string arguments, so the conflict passed.
@test "UAC4: a coordinate in a val is resolved (uberjar(lib) + api(\"g:a\"))" {
  # Mutation: UAC4 -- not resolving a val argument misses the conflict of a coordinate held in a variable.
  cat > "$TMPDIR_T/M/x-rt.gradle.kts" <<'KTS'
val jackson = "com.fasterxml.jackson.core:jackson-core:2.13.1"
val csv: String = "com.opencsv:opencsv:5.7.1"
dependencies {
  uberjar(jackson)
  api("com.fasterxml.jackson.core:jackson-core:2.13.1")
  uberjar(csv)
  uberjar(unknown)
  api(unknown)
}
KTS
  run "$L" "$TMPDIR_T/M"
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | grep -c 'is both')" -eq 1 ]
  [[ "$output" == *"com.fasterxml.jackson.core:jackson-core is both"* ]]
}
