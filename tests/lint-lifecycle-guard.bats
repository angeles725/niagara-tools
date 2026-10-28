#!/usr/bin/env bats
# lint-lifecycle-guard.bats — lifecycle callbacks that work when the component is not running WARN (LG1, LG2). The
# rule's golden cases live in go/internal/lint/rules/lifecycleguard. [ev: retro course-coverage Δ23]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-lifecycle-guard.sh"
  SRC="$BATS_TEST_TMPDIR/src"
  mkdir -p "$SRC/com/acme"
}

@test "LG-usage: no argument prints the usage and exits 3" {
  run "$L"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-lifecycle-guard.sh [--strict] <src-root>" ]
}

@test "LG-fixture: the MinimalPan -rt sources are clean under --strict" {
  run "$L" --strict "$KIT/fixtures/MinimalPan/MinimalPan/MinimalPan-rt/src"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "LG1: changed() led by a slot dispatch WARNs; led by if (!isRunning()) return; it is clean" {
  # Mutation: LG1 -- accepting any leading if as the guard clears changed() whose first statement is if (p == in)
  printf '%s\n' 'package com.acme;' 'class BA extends BComponent {' \
    '  public void changed(Property p, Context cx) { if (!isRunning()) return; if (p == in) setOut(1); }' '}' > "$SRC/com/acme/BA.java"
  printf '%s\n' 'package com.acme;' 'class BB extends BComponent {' \
    '  public void changed(Property p, Context cx) { if (p == in) setOut(1); }' '}' > "$SRC/com/acme/BB.java"
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-lifecycle-guard  $SRC/com/acme/BB.java:3  LG1 changed() acts with no isRunning() guard: make its first statement if (!isRunning()) return; (types/logic.md § Tridium rt idioms)" ]
  run "$L" --strict "$SRC"
  [ "$status" -eq 1 ]
}

@test "LG2: stationStarted() work WARNs without started(); with started() it is clean" {
  # Mutation: LG2 -- dropping the started() lookup WARNs a stationStarted() whose class also has started()
  printf '%s\n' 'package com.acme;' 'class BA extends BComponent {' '  public void started() throws Exception { arm(); }' \
    '  public void stationStarted() throws Exception { refresh(); }' '}' > "$SRC/com/acme/BA.java"
  printf '%s\n' 'package com.acme;' 'class BB extends BComponent {' \
    '  public void stationStarted() throws Exception { refresh(); }' '}' > "$SRC/com/acme/BB.java"
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-lifecycle-guard  $SRC/com/acme/BB.java:3  LG2 stationStarted() does startup work and the class has no started(): it runs once at station boot, never for a component added to a running station; move the work to started() (types/logic.md § Tridium rt idioms)" ]
}

@test "LG-K20: a root with no *.java is the ERROR row, exit 3" {
  run "$L" "$SRC"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-lifecycle-guard  $SRC  no *.java to scan (K20: wrong path or empty scaffold)" ]
}
