#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-subscribe-without-unsubscribe.sh
# A Subscriber.subscribe() in started() with no matching unsubscribe() in stopped() leaks the
# watched component and re-fires across enable cycles. [ev: retro module-hardening-failure-modes-deltas Δ3]

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  SWU="$KIT/toolbelt/lint-subscribe-without-unsubscribe.sh"
  mkdir -p "$TMPDIR_T/Mod/src/com/x"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "SWU-usage: no arg exits 3" {
  run "$SWU"
  [ "$status" -eq 3 ]
}

@test "SWU-nondir: a non-directory arg exits 3" {
  run "$SWU" "$TMPDIR_T/does-not-exist"
  [ "$status" -eq 3 ]
}

@test "SWU1: subscribe WITH a matching unsubscribe in the same file is clean (exit 0, no WARN)" {
  printf 'class A {\n  void started(){ mySub.subscribe(comp); }\n  void stopped(){ mySub.unsubscribe(comp); }\n}\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$SWU" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "SWU2: subscribe with NO unsubscribe anywhere is WARNed (exit 0 advisory)" {
  printf 'class A {\n  void started(){ mySub.subscribe(comp); }\n}\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$SWU" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"lint-subscribe-without-unsubscribe"* ]]
  [[ "$output" == *"A.java:2"* ]]
  # Named mutation SWU2: dropping the `unsubscribe`-absent guard makes SWU1 false-WARN.
}

@test "SWU2-strict: --strict promotes the WARN to exit 1" {
  printf 'class A {\n  void started(){ mySub.subscribe(comp); }\n}\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$SWU" --strict "$TMPDIR_T/Mod"
  [ "$status" -eq 1 ]
  [[ "$output" == *"WARN"* ]]
}

@test "SWU3: a file that never subscribes (uses lease) is clean" {
  printf 'class A {\n  void started(){ mySub.lease(comp, 30); }\n}\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$SWU" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "SWU4: commented-out subscribe does not trigger (comment strip)" {
  printf 'class A {\n  // mySub.subscribe(comp);\n  void f(){}\n}\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$SWU" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "SWU-locked: an unreadable sub-directory is an env error (exit 3), never a clean pass" {
  # Mutation: SWU-locked -- a raw find walk skips the locked directory and passes clean.
  mkdir -p "$TMPDIR_T/Mod/locked"
  chmod 000 "$TMPDIR_T/Mod/locked"
  if [ -r "$TMPDIR_T/Mod/locked" ]; then chmod 755 "$TMPDIR_T/Mod/locked"; skip "running as root"; fi
  run "$SWU" "$TMPDIR_T/Mod"
  chmod 755 "$TMPDIR_T/Mod/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"lint-subscribe-without-unsubscribe: cannot list every file under $TMPDIR_T/Mod: "* ]]
}

@test "SWU5: a subscribe in a block comment is clean; an unsubscribe in a comment or string never pairs a leak; a subscribe after a // inside a string is WARNed" {
  # Mutation: SWU5 -- matching the raw line (no comment/literal blanking) exempts C.java and D.java and WARNs A.java.
  # [ev: truth audit 2026-10-22 T3]
  printf 'class A {\n  /* sub.subscribe(c); */\n}\n' > "$TMPDIR_T/Mod/src/com/x/A.java"
  printf 'class B {\n  String u = "http://h"; void f() { sub.subscribe(c); }\n}\n' > "$TMPDIR_T/Mod/src/com/x/B.java"
  printf 'class C {\n  void f() { sub.subscribe(c); }\n  /* TODO unsubscribe */\n}\n' > "$TMPDIR_T/Mod/src/com/x/C.java"
  printf 'class D {\n  void f() { sub.subscribe(c); }\n  String s = "unsubscribe";\n}\n' > "$TMPDIR_T/Mod/src/com/x/D.java"
  run "$SWU" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-subscribe-without-unsubscribe  $TMPDIR_T/Mod/src/com/x/B.java:2  String u = \"http://h\"; void f() { sub.subscribe(c); }
WARN  lint-subscribe-without-unsubscribe  $TMPDIR_T/Mod/src/com/x/C.java:2  void f() { sub.subscribe(c); }
WARN  lint-subscribe-without-unsubscribe  $TMPDIR_T/Mod/src/com/x/D.java:2  void f() { sub.subscribe(c); }" ]
}
