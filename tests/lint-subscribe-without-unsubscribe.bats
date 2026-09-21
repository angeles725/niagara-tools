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
