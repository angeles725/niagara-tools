#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-wb-external-ord-value.sh
# A manager method that resolves a per-row ORD to an object OUTSIDE the subscribed
# subtree, reads its live value/status, and never leases a refresh for that specific
# target is the B1140-G1 external-ORD gap. [ev: retro apillm-wb-subscription-refresh-and-points-deltas Δ3]

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  WEO="$KIT/toolbelt/lint-wb-external-ord-value.sh"
  mkdir -p "$TMPDIR_T/Mod/src/com/x"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "WEO-usage: no arg exits 3" { run "$WEO"; [ "$status" -eq 3 ]; }

@test "WEO-nondir: a non-directory arg exits 3" { run "$WEO" "$TMPDIR_T/nope"; [ "$status" -eq 3 ]; }

_write_bad() {
  # resolve external ORD -> read "out"/BStatusValue -> NO registerForComponentEvents anywhere.
  cat > "$TMPDIR_T/Mod/src/com/x/A.java" << 'EOF'
class A {
  String[] rowForBad(Object e) {
    Object o = e.getOrd().get(subject, null);
    if (o instanceof BComponent) {
      BComponent targetComp = (BComponent) o;
      Object out = targetComp.get("out");
      if (out instanceof BStatusValue) {
        BStatusValue sv = (BStatusValue) out;
      }
    }
    return null;
  }
}
EOF
}

@test "WEO1: external-ORD resolve + live value/status read, no register lease -> WARN" {
  _write_bad
  run "$WEO" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"lint-wb-external-ord-value"* ]]
  [[ "$output" == *"rowForBad"* ]]
}

@test "WEO1-strict: --strict promotes the WARN to exit 1" {
  _write_bad
  run "$WEO" --strict "$TMPDIR_T/Mod"
  [ "$status" -eq 1 ]
}

@test "WEO2: registerForComponentEvents(target,0) in the same method suppresses the WARN" {
  cat > "$TMPDIR_T/Mod/src/com/x/A.java" << 'EOF'
class A {
  String[] rowForGood(Object e) {
    Object o = e.getOrd().get(subject, null);
    if (o instanceof BComponent) {
      BComponent targetComp = (BComponent) o;
      if (!isRegisteredForComponentEvents(targetComp)) {
        registerForComponentEvents(targetComp, 0);
      }
      Object out = targetComp.get("out");
      if (out instanceof BStatusValue) {
        BStatusValue sv = (BStatusValue) out;
      }
    }
    return null;
  }
}
EOF
  run "$WEO" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
  # Named mutation: drop the registerForComponentEvents(-absence check -> WEO2 false-WARNs on this fixture.
}

@test "WEO3: no ORD-resolve at all is clean" {
  cat > "$TMPDIR_T/Mod/src/com/x/A.java" << 'EOF'
class A {
  void doNothing() {
    int x = 1 + 1;
  }
}
EOF
  run "$WEO" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "WEO4: an ORD resolve with no live-value read (container navigation only) is clean" {
  cat > "$TMPDIR_T/Mod/src/com/x/A.java" << 'EOF'
class A {
  BComponent resolveFolder(Object folderOrd) {
    Object obj = folderOrd.get(importer, null);
    if (obj instanceof BComponent) return (BComponent) obj;
    return null;
  }
}
EOF
  run "$WEO" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}
