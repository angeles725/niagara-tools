#!/usr/bin/env bats
# lint-adapter-cfg-default.bats — audit 2026-10-03 A7-D1: a pure-model Cfg field whose default differs from the
# default of the slot the adapter fills it from WARNs. The rule's golden cases live in
# go/internal/lint/rules/adaptercfgdefault.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-adapter-cfg-default.sh"
  MOD="$BATS_TEST_TMPDIR/Mod"
  SRC="$MOD/Mod-rt/src/com/example"
  mkdir -p "$SRC"
}

TAIL='a pure test built from new Cfg() runs a configuration the station never starts with -- initialize the field to the slot default'

@test "ACD-usage: no argument prints the usage and exits 3" {
  run "$L"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-adapter-cfg-default.sh [--strict] <module-root>..." ]
}

@test "ACD1: a BRelTime slot is compared through its getSeconds conversion" {
  # Mutation: ACD1 -- comparing a BRelTime slot in raw milliseconds (no getSeconds conversion) WARNs a 30 s field against BRelTime.makeSeconds(30)
  printf '%s\n' 'package com.example;' 'class Ctl { static class Cfg { long delaySec = 30; } }' > "$SRC/Ctl.java"
  printf '%s\n' 'package com.example;' '@NiagaraType' '@NiagaraProperty(name = "t", type = "BRelTime", defaultValue = "BRelTime.makeSeconds(30)")' \
    'class BCtl { void f(Ctl.Cfg c) { c.delaySec = getT().getSeconds(); } }' > "$SRC/BCtl.java"
  run "$L" --strict "$MOD"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "ACD2: a field with no initializer defaults to 0 and WARNs against a slot defaulting to 6; --strict exits 1" {
  # Mutation: ACD2 -- reading a field with no initializer as unjudged passes an int field the adapter fills from a slot defaulting to 6
  printf '%s\n' 'package com.example;' 'class Ctl {' '  static final class Cfg {' '    int maxStarts;' '  }' '}' > "$SRC/Ctl.java"
  printf '%s\n' 'package com.example;' '@NiagaraType' '@NiagaraProperty(name = "maxStarts", type = "int", defaultValue = "6")' \
    'class BCtl { void f(Ctl.Cfg c) { c.maxStarts = getMaxStarts(); } }' > "$SRC/BCtl.java"
  run "$L" "$MOD"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-adapter-cfg-default  $SRC/Ctl.java:4  Cfg.maxStarts defaults to 0 but slot maxStarts (BCtl.java:3) defaults to 6: $TAIL" ]
  run "$L" --strict "$MOD"
  [ "$status" -eq 1 ]
}

@test "ACD-K20: a root with no -rt src/ *.java is the ERROR row, exit 3" {
  rm -r "$MOD/Mod-rt"; mkdir -p "$MOD/Mod-ux/src"; printf 'class U {}\n' > "$MOD/Mod-ux/src/U.java"
  run "$L" "$MOD"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-adapter-cfg-default  $MOD  no *.java under a -rt part's src/ (wrong path: pass the module root)" ]
}
