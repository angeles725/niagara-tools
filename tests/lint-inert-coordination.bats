#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-inert-coordination.sh
# A coordinator whose units() is structurally singleton while queue/token/stagger state is
# still declared is dead coordination machinery after a per-unit move (PR1c shape).
# [ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ5]

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  ICO="$KIT/toolbelt/lint-inert-coordination.sh"
  mkdir -p "$TMPDIR_T/Mod/src/com/x"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "ICO-usage: no arg exits 3" { run "$ICO"; [ "$status" -eq 3 ]; }

@test "ICO-nondir: a non-directory arg exits 3" { run "$ICO" "$TMPDIR_T/nope"; [ "$status" -eq 3 ]; }

_write_bad() {
  cat > "$TMPDIR_T/Mod/src/com/x/BDefrostController.java" << 'EOF'
class BDefrostController {
  private final Deque<Integer> waitingQueue = new ArrayDeque<Integer>();
  public static final Property staggerDelay = newProperty(0, BRelTime.make(240000), null);
  public BRelTime getStaggerDelay() { return (BRelTime)get(staggerDelay); }
  private List<BEvaporatorUnit> units() {
    Object parent = getParent();
    if (parent instanceof BEvaporatorUnit)
      return java.util.Collections.singletonList((BEvaporatorUnit)parent);
    return java.util.Collections.emptyList();
  }
  void acquireDefrostToken() {}
}
EOF
}

@test "ICO1: singleton units() + queue/token/stagger state still declared -> WARN" {
  _write_bad
  run "$ICO" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"lint-inert-coordination"* ]]
  [[ "$output" == *"waitingQueue"* ]]
}

@test "ICO1-strict: --strict promotes the WARN to exit 1" {
  _write_bad
  run "$ICO" --strict "$TMPDIR_T/Mod"
  [ "$status" -eq 1 ]
}

@test "ICO-multiunit: units() returns the parent's multi-unit list (not singleton) -> clean" {
  cat > "$TMPDIR_T/Mod/src/com/x/BDefrostController.java" << 'EOF'
class BDefrostController {
  private final Deque<Integer> waitingQueue = new ArrayDeque<Integer>();
  private List<BEvaporatorUnit> units() {
    Object parent = getParent();
    if (parent instanceof BColdRoom)
      return ((BColdRoom)parent).getUnits();
    return java.util.Collections.emptyList();
  }
}
EOF
  run "$ICO" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "ICO2: singleton units() with NO queue/token/stagger state declared -> clean" {
  cat > "$TMPDIR_T/Mod/src/com/x/Solo.java" << 'EOF'
class Solo {
  private List<Unit> units() {
    return java.util.Collections.singletonList((Unit) getParent());
  }
}
EOF
  run "$ICO" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
  # Named mutation: drop the coordination-state requirement -> ICO2 false-WARNs on this fixture
  # (a singleton units() with no queue/token/stagger state has nothing dead to flag).
}

@test "ICO-nounits: no units() method at all is clean" {
  printf 'class NoUnits {\n  void doStuff() {}\n}\n' > "$TMPDIR_T/Mod/src/com/x/NoUnits.java"
  run "$ICO" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "ICO-aslist: a one-element Arrays.asList(...) return also counts as structurally singleton" {
  cat > "$TMPDIR_T/Mod/src/com/x/Asl.java" << 'EOF'
class Asl {
  private final Queue<Integer> pending = new ArrayDeque<Integer>();
  int defrostToken;
  private List<Unit> units() {
    return java.util.Arrays.asList((Unit) getParent());
  }
}
EOF
  run "$ICO" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
}

@test "ICO-awkfail: an unreadable source file is an env error (exit 3, named on stderr), never a clean pass" {
  # Mutation: ICO-awkfail -- ignoring the awk exit status reports the unreadable file as clean (exit 0).
  mkdir -p "$(dirname "$TMPDIR_T/Mod/src/com/x/U.java")"
  printf 'x\n' > "$TMPDIR_T/Mod/src/com/x/U.java"
  chmod 000 "$TMPDIR_T/Mod/src/com/x/U.java"
  if [ -r "$TMPDIR_T/Mod/src/com/x/U.java" ]; then chmod 644 "$TMPDIR_T/Mod/src/com/x/U.java"; skip "running as root: chmod 000 does not block reads"; fi
  run "$ICO" "$TMPDIR_T/Mod"
  chmod 644 "$TMPDIR_T/Mod/src/com/x/U.java"
  [ "$status" -eq 3 ]
  [[ "$output" == *"Mod/src/com/x/U.java"* ]]
}

@test "ICO-lower: a lowerCamel staggerDelay / defrostToken field counts as coordination state" {
  # Mutation: ICO-lower -- requiring a capitalized Stagger/Token misses the staggerDelay field itself.
  cat > "$TMPDIR_T/Mod/src/com/x/C.java" << 'JEOF'
public class C {
  private double staggerDelay = 30;
  List<BComponent> units() {
    return Collections.singletonList(getParent());
  }
}
JEOF
  run "$ICO" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"C.java:3"* ]]
  [[ "$output" == *"staggerDelay"* ]]
}

@test "ICO-call: a units() CALL above the declaration is not taken for the declaration" {
  # Mutation: ICO-call -- accepting `return units()...;` as the declaration checks the wrong body (no WARN).
  cat > "$TMPDIR_T/Mod/src/com/x/D.java" << 'JEOF'
public class D {
  private Deque<BComponent> waitingQueue = new ArrayDeque<BComponent>();
  int count() {
    return units().size();
  }
  List<BComponent> units() {
    return Collections.singletonList(getParent());
  }
}
JEOF
  run "$ICO" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"D.java:6"* ]]
}
