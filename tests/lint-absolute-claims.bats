#!/usr/bin/env bats
# lint-absolute-claims.bats — G38: absolute claims whose claim unit cites only corpus/retro inference are INFO rows;
# a code/course/devguide/CERT/javap tag or a .java: line cite in the unit clears them. The rule's golden cases live
# in go/internal/lint/rules/absoluteclaims. [ev: retro n4-course-as-exemplar-source Δ3]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-absolute-claims.sh"
  T="$BATS_TEST_TMPDIR/types"
  mkdir -p "$T"
}

@test "AC-usage: no argument prints the usage and exits 3" {
  run "$L"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-absolute-claims.sh [--strict] <types-dir>..." ]
}

@test "AC1: a code tag on the bullet's continuation line clears every line of the bullet" {
  # Mutation: AC1 -- reading the tags of the line alone lists a bullet whose code tag sits on its continuation line
  printf '%s\n' '- Never call X from a ticket,' '  it blocks the engine. `[ev: code toolbelt/lint-timers.sh:10]`' > "$T/a.md"
  run "$L" --strict "$T"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "AC2: a corpus-only claim is an INFO row and the report still exits 0 under --strict" {
  # Mutation: AC2 -- counting a corpus tag as strong drops the corpus-only claim from the report
  printf '%s\n' '- Never block the engine thread. [ev: corpus B9 §9.1]' > "$T/a.md"
  run "$L" --strict "$T"
  [ "$status" -eq 0 ]
  [ "$output" = "INFO  lint-absolute-claims  $T/a.md:1  never: - Never block the engine thread. [ev: corpus B9 §9.1]  [ev: corpus B9 §9.1]" ]
}

@test "AC-kit: the kit's own types/ scans with exit 0" {
  run "$L" "$KIT/types"
  [ "$status" -eq 0 ]
  [[ "$output" == INFO* ]]
}

@test "AC-K20: a root with no *.md is the ERROR row, exit 3" {
  printf 'x\n' > "$T/a.txt"
  run "$L" "$T"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-absolute-claims  $T  no *.md under the root (wrong path: pass build-n4-module-kit/types)" ]
}
