#!/usr/bin/env bats
# lint-intrinsic-lock.bats — locks on a Niagara object's own monitor and nested locks WARN (IL1-IL3). The rule's
# golden cases live in go/internal/lint/rules/intrinsiclock. [ev: retro course-coverage Δ24]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-intrinsic-lock.sh"
  SRC="$BATS_TEST_TMPDIR/src"
  mkdir -p "$SRC/com/acme"
  W=" (types/logic.md § Tridium rt idioms)"
}

@test "IL-usage: no argument prints the usage and exits 3" {
  run "$L"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-intrinsic-lock.sh [--strict] <src-root>" ]
}

@test "IL-fixture: the MinimalPan -rt sources are clean under --strict" {
  run "$L" --strict "$KIT/fixtures/MinimalPan/MinimalPan/MinimalPan-rt/src"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "IL1: a synchronized method WARNs in a B class; in a plain helper it is clean" {
  # Mutation: IL1 -- dropping the Niagara-class test WARNs a synchronized method of a plain helper class
  printf '%s\n' 'package com.acme;' 'class Helper {' '  synchronized void a() { n++; }' '}' > "$SRC/com/acme/Helper.java"
  printf '%s\n' 'package com.acme;' 'class BA extends BComponent {' '  synchronized void a() { n++; }' '}' > "$SRC/com/acme/BA.java"
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-intrinsic-lock  $SRC/com/acme/BA.java:3  IL1 synchronized method locks the component, which the framework also locks: lock a private final Object or use Atomic*$W" ]
  run "$L" --strict "$SRC"
  [ "$status" -eq 1 ]
}

@test "IL2: synchronized (this) WARNs in a B class; synchronized (lock) is clean" {
  # Mutation: IL2 -- judging every synchronized block as this WARNs synchronized (lock) on a private final Object
  printf '%s\n' 'package com.acme;' 'class BA extends BComponent {' '  private final Object lock = new Object();' \
    '  void a() { synchronized (lock) { n++; } }' '  void b() { synchronized (this) { n++; } }' '}' > "$SRC/com/acme/BA.java"
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-intrinsic-lock  $SRC/com/acme/BA.java:5  IL2 synchronized (this) locks the component, which the framework also locks: lock a private final Object or use Atomic*$W" ]
}

@test "IL3: a block nested in a synchronized method WARNs, in any class" {
  # Mutation: IL3 -- dropping the method-body regions misses a block nested in a synchronized method
  printf '%s\n' 'package com.acme;' 'class Pool {' '  private final Object lock = new Object();' \
    '  synchronized void a() { synchronized (lock) { n++; } }' '}' > "$SRC/com/acme/Pool.java"
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-intrinsic-lock  $SRC/com/acme/Pool.java:4  IL3 synchronized block nested in another lock: nested locks can deadlock; take one lock at a time$W" ]
}

@test "IL-K20: a root with no *.java is the ERROR row, exit 3" {
  run "$L" "$SRC"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-intrinsic-lock  $SRC  no *.java to scan (K20: wrong path or empty scaffold)" ]
}
