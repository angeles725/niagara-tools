#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-persist-hot-write.sh
# A non-transient property setter called every cycle from changed()/execute() with no
# cadence guard is the PER8 persisted-accumulator shape. [ev: retro live-diagnosis-hardening-deltas Δ3]

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  PHW="$KIT/toolbelt/lint-persist-hot-write.sh"
  mkdir -p "$TMPDIR_T/Mod/src/com/x"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "PHW-usage: no arg exits 3" { run "$PHW"; [ "$status" -eq 3 ]; }

@test "PHW-nondir: a non-directory arg exits 3" { run "$PHW" "$TMPDIR_T/nope"; [ "$status" -eq 3 ]; }

@test "PHW1: non-transient setter called every cycle from changed()->execute(), no guard -> WARN" {
  printf 'class A {\n  public static final Property xHours = newProperty(Flags.SUMMARY | Flags.READONLY, 0d, null);\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  printf '  public void changed(Property p, Context cx) { execute(); }\n' >> "$TMPDIR_T/Mod/src/com/x/A.java"
  printf '  void execute() {\n    setXHours(1.0);\n  }\n}\n' >> "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$PHW" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"lint-persist-hot-write"* ]]
  [[ "$output" == *"xHours"* ]]
}

@test "PHW1-strict: --strict promotes the WARN to exit 1" {
  printf 'class A {\n  public static final Property xHours = newProperty(Flags.SUMMARY | Flags.READONLY, 0d, null);\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  printf '  public void changed(Property p, Context cx) { execute(); }\n' >> "$TMPDIR_T/Mod/src/com/x/A.java"
  printf '  void execute() {\n    setXHours(1.0);\n  }\n}\n' >> "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$PHW" --strict "$TMPDIR_T/Mod"
  [ "$status" -eq 1 ]
}

@test "PHW2: a Clock.millis() cadence guard within 5 lines above the setter suppresses the WARN" {
  printf 'class A {\n  public static final Property xHours = newProperty(Flags.SUMMARY | Flags.READONLY, 0d, null);\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  printf '  public void changed(Property p, Context cx) { execute(); }\n' >> "$TMPDIR_T/Mod/src/com/x/A.java"
  printf '  void execute() {\n    if (Clock.millis() - lastCheckpoint < 60000) return;\n    setXHours(1.0);\n  }\n}\n' \
    >> "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$PHW" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
  # Named mutation: drop the cadence-guard lookback -> PHW2 false-WARNs on this fixture.
}

@test "PHW3: a TRANSIENT property setter is clean (not the leak shape)" {
  printf 'class A {\n  public static final Property xHours = newProperty(Flags.TRANSIENT | Flags.SUMMARY, 0d, null);\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  printf '  public void changed(Property p, Context cx) { execute(); }\n' >> "$TMPDIR_T/Mod/src/com/x/A.java"
  printf '  void execute() {\n    setXHours(1.0);\n  }\n}\n' >> "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$PHW" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "PHW4: no changed(Property...) override at all is clean" {
  printf 'class A {\n  public static final Property xHours = newProperty(Flags.SUMMARY | Flags.READONLY, 0d, null);\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  printf '  void execute() { setXHours(1.0); }\n}\n' >> "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$PHW" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "PHW5: the setter DECLARATION line itself (not a call) is not mistaken for a hot write" {
  printf 'class A {\n  public static final Property xHours = newProperty(Flags.SUMMARY | Flags.READONLY, 0d, null);\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  printf '  public void changed(Property p, Context cx) { execute(); }\n' >> "$TMPDIR_T/Mod/src/com/x/A.java"
  printf '  void execute() {\n    public void setXHours(double v) { setDouble(xHours, v, null); }\n  }\n}\n' \
    >> "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$PHW" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}
