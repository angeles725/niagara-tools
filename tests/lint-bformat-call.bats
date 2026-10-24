#!/usr/bin/env bats
# lint-bformat-call.sh — BFormat special calls written in a form BFormat never parses (register C07, rcf Δ6):
# time() is matched only as the exact call "time()" and takes no argument, and user() needs its parentheses;
# %time(<fmt>)% and %user% fall through to reflection and print %err:...% (S/util/BFormat.java:443-444).
# [ev: retro devguide415-r2-runtime-contracts-formats Δ6]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LINT="$KIT/toolbelt/lint-bformat-call.sh"
  SRC="$BATS_TEST_TMPDIR/src"; mkdir -p "$SRC"
}

@test "BF1: %time(<fmt>)% in a Java string WARNs (time() takes no argument)" {
  printf '%s\n' 'class A { String f = "at %time(HH:mm)%"; }' > "$SRC/A.java"
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  lint-bformat-call  $SRC/A.java:1"*"time()"* ]]
}

@test "BF2: %user% in a lexicon value WARNs (use %user()%)" {
  printf '%s\n' 'greeting=Hello %user%' > "$SRC/module.lexicon"
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]
  [[ "$output" == *"module.lexicon:1"*"user()"* ]]
}

@test "BF-ok: %time()%, %time().toDateString%, %user()% and a %%-escaped percent are clean" {
  printf '%s\n' 'class B { String f = "%time().toDateString% by %user()% at 100%% %time()%"; }' > "$SRC/B.java"
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "BF-comment: a pattern only in a Java comment is not reported" {
  printf '%s\n' 'class C {' '  // old: "%time(HH:mm)%"' '  /* "%user%" */' '  String f = "ok";' '}' > "$SRC/C.java"
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "BF-strict: --strict makes a WARN exit 1" {
  printf '%s\n' 'x=%user%' > "$SRC/m.lexicon"
  run "$LINT" --strict "$SRC"
  [ "$status" -eq 1 ]
}

@test "BF-usage: no root is a usage error (exit 3)" {
  run "$LINT"
  [ "$status" -eq 3 ]
}

@test "BF-xml: a .bog is read as XML: a comment is not read, an entity-encoded %user% is (truth fix 2026-10-24)" {
  # Mutation: BF-xml -- reading a .bog as text counts a commented-out pattern and misses an entity-encoded %user%
  printf '<bajaObjectGraph>\n<!-- <p v="%%time(x)%%"/> -->\n<p v="&#37;user&#37;"/>\n</bajaObjectGraph>\n' > "$SRC/config.bog"
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-bformat-call  $SRC/config.bog:3  BFC2 %user% is a reflective call that fails -- write %user()% (types/logic.md BFormat)" ]
}
