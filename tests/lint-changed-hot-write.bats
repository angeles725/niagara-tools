#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-changed-hot-write.sh
# A hot changed() (N>=6 `p ==` branches -> execute()) with no rate guard that writes a
# non-transient slot is the RUN8 callback-flood shape. [ev: retro live-diagnosis-hardening-deltas Δ1]

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  CHW="$KIT/toolbelt/lint-changed-hot-write.sh"
  mkdir -p "$TMPDIR_T/Mod/src/com/x"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "CHW-usage: no arg exits 3" { run "$CHW"; [ "$status" -eq 3 ]; }

@test "CHW-nondir: a non-directory arg exits 3" { run "$CHW" "$TMPDIR_T/nope"; [ "$status" -eq 3 ]; }

_write_hot() {
  # 7 branches, execute(), no rate guard, one non-transient property + setter.
  printf 'class A {\n' > "$TMPDIR_T/Mod/src/com/x/A.java"
  printf '  public static final Property xHours = newProperty(Flags.SUMMARY | Flags.READONLY, 0d, null);\n' \
    >> "$TMPDIR_T/Mod/src/com/x/A.java"
  printf '  public void changed(Property p, Context cx) {\n' >> "$TMPDIR_T/Mod/src/com/x/A.java"
  printf '    if (p == a || p == b || p == c || p == d || p == e || p == f || p == g)\n' \
    >> "$TMPDIR_T/Mod/src/com/x/A.java"
  printf '      execute();\n  }\n  void execute() {\n    setXHours(1.0);\n  }\n}\n' \
    >> "$TMPDIR_T/Mod/src/com/x/A.java"
}

@test "CHW1: hot changed() (7 branches) -> execute(), no rate guard, non-transient write -> WARN" {
  _write_hot
  run "$CHW" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"lint-changed-hot-write"* ]]
  [[ "$output" == *"xHours"* ]]
}

@test "CHW1-strict: --strict promotes the WARN to exit 1" {
  _write_hot
  run "$CHW" --strict "$TMPDIR_T/Mod"
  [ "$status" -eq 1 ]
}

@test "CHW2: fewer than 6 'p ==' branches is clean (no WARN)" {
  printf 'class A {\n  public static final Property xHours = newProperty(Flags.SUMMARY | Flags.READONLY, 0d, null);\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  printf '  public void changed(Property p, Context cx) {\n    if (p == a || p == b) execute();\n  }\n' \
    >> "$TMPDIR_T/Mod/src/com/x/A.java"
  printf '  void execute() { setXHours(1.0); }\n}\n' >> "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$CHW" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
  # Named mutation: drop the branch-count threshold (N>=6) -> CHW2 false-WARNs on this fixture.
}

@test "CHW3: a Clock.millis() rate guard in changed() suppresses the WARN" {
  printf 'class A {\n  public static final Property xHours = newProperty(Flags.SUMMARY | Flags.READONLY, 0d, null);\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  printf '  public void changed(Property p, Context cx) {\n    if (Clock.millis() - lastRun < 200) return;\n' \
    >> "$TMPDIR_T/Mod/src/com/x/A.java"
  printf '    if (p == a || p == b || p == c || p == d || p == e || p == f || p == g) execute();\n  }\n' \
    >> "$TMPDIR_T/Mod/src/com/x/A.java"
  printf '  void execute() { setXHours(1.0); }\n}\n' >> "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$CHW" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "CHW4: a TRANSIENT-only property (no persisted write) is clean" {
  printf 'class A {\n  public static final Property xHours = newProperty(Flags.TRANSIENT | Flags.SUMMARY, 0d, null);\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  printf '  public void changed(Property p, Context cx) {\n' >> "$TMPDIR_T/Mod/src/com/x/A.java"
  printf '    if (p == a || p == b || p == c || p == d || p == e || p == f || p == g) execute();\n  }\n' \
    >> "$TMPDIR_T/Mod/src/com/x/A.java"
  printf '  void execute() { setXHours(1.0); }\n}\n' >> "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$CHW" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "CHW5: no changed(Property...) override at all is clean" {
  printf 'class A {\n  void execute() { setXHours(1.0); }\n}\n' > "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$CHW" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}
