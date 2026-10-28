#!/usr/bin/env bats
# lint-slot-default-type.bats — an @NiagaraProperty whose defaultValue literal has a Java type its slot type refuses
# FAILs (the type fails to load at runtime). The rule's golden cases live in go/internal/lint/rules/slotdefaulttype.
# [ev: retro course-coverage Δ21]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-slot-default-type.sh"
  SRC="$BATS_TEST_TMPDIR/src"
  mkdir -p "$SRC/com/acme"
}

_row() { # <line> <name> <type> <default> <kind> <stored> <wanted>
  printf 'FAIL  lint-slot-default-type  %s:%s  @NiagaraProperty %s type="%s" defaultValue="%s": the default is %s, so Slot-o-Matic'"'"'s newProperty(flags, %s, ...) stores %s that is not a %s: the type fails to load ("Property default value is of wrong type") (types/structure.md § Slot defaults)' \
    "$SRC/com/acme/BA.java" "$1" "$2" "$3" "$4" "$5" "$4" "$6" "$7"
}

@test "SDT-usage: no argument prints the usage and exits 3" {
  run "$L"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-slot-default-type.sh <src-root>" ]
}

@test "SDT-fixture: the MinimalPan -rt sources are clean" {
  run "$L" "$KIT/fixtures/MinimalPan/MinimalPan/MinimalPan-rt/src"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "SDT1: 2d on an int slot FAILs; 0 on a double slot is clean (widened)" {
  # Mutation: SDT1 -- widening every numeric literal (not only an int on long/float/double) passes 2d on an int slot
  printf '%s\n' 'package com.acme;' '@NiagaraProperty(name = "lim", type = "double", defaultValue = "0")' \
    '@NiagaraProperty(name = "cnt", type = "int", defaultValue = "2d")' 'public class BA {}' > "$SRC/com/acme/BA.java"
  run "$L" "$SRC"
  [ "$status" -eq 1 ]
  [ "$output" = "$(_row 3 cnt int 2d 'a double' 'a BDouble' BInteger)" ]
}

@test "SDT2: BBoolean.TRUE on a double slot FAILs; BDouble.make(1) is clean" {
  # Mutation: SDT2 -- dropping the B-value kind passes BBoolean.TRUE on a double slot
  printf '%s\n' 'package com.acme;' '@NiagaraProperty(name = "g", type = "double", defaultValue = "BBoolean.TRUE")' \
    '@NiagaraProperty(name = "h", type = "double", defaultValue = "BDouble.make(1)")' 'public class BA {}' > "$SRC/com/acme/BA.java"
  run "$L" "$SRC"
  [ "$status" -eq 1 ]
  [ "$output" = "$(_row 2 g double BBoolean.TRUE 'a BBoolean' 'a BBoolean' BDouble)" ]
}

@test "SDT-K20: a root with no *.java is the ERROR row, exit 3" {
  run "$L" "$SRC"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-slot-default-type  $SRC  no *.java to scan (K20: wrong path or empty scaffold)" ]
}
