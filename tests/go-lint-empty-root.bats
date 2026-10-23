#!/usr/bin/env bats
# An empty lint root is an error, not a clean pass (truth audit 2026-10-22 T6, R8-D14): every toolbelt
# wrapper over the Go engine (go/internal/lint Spec.EmptyError) prints an ERROR row naming the root and
# exits 3 when the root holds no *.java file. The bash side (lib/scan-files.sh) is fixed separately.

setup() {
  T="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit/toolbelt"
  E="$BATS_TEST_TMPDIR/empty"; mkdir -p "$E/sub"; printf 'not java\n' > "$E/sub/notes.txt"
}

@test "EMPTY-root: every Go-engine lint wrapper fails a root with no *.java file (ERROR row, exit 3)" {
  for lint in lint-timers lint-null-context-write lint-driver-scaffold lint-set-null-ord lint-service-types lint-wb-external-ord-value; do
    run "$T/$lint.sh" "$E"
    [ "$status" -eq 3 ] || { echo "$lint: exit $status"; false; }
    [ "$output" = "ERROR  $lint  $E  no *.java sources found (wrong path or empty source tree)" ] || { echo "$lint: $output"; false; }
  done
}

@test "EMPTY-pruned: a root whose only *.java lives under a pruned dot-dir is empty too" {
  mkdir -p "$E/.deploy-baseline"; printf 'class A {}\n' > "$E/.deploy-baseline/A.java"
  run "$T/lint-timers.sh" "$E"
  [ "$status" -eq 3 ]
  [[ "$output" == "ERROR  lint-timers  $E  no *.java sources found"* ]]
}

@test "EMPTY-nonempty: one *.java file under the root is enough (exit 0, no ERROR row)" {
  printf 'class A {}\n' > "$E/sub/A.java"
  run "$T/lint-null-context-write.sh" "$E"
  [ "$status" -eq 0 ]
  [[ "$output" != *"ERROR"* ]]
}
