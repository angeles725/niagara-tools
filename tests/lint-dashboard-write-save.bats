#!/usr/bin/env bats
# lint-dashboard-write-save.sh — the wrapper contract and its mutation pins (LDWS1-LDWS4). The rule's own cases
# are the Go goldens (go/internal/lint/rules/dashboardsave/testdata). [ev: retro lint-dashboard-write-save Δ1]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LINT="$KIT/toolbelt/lint-dashboard-write-save.sh"
  MOD="$BATS_TEST_TMPDIR/Mod"
  UX="$MOD/Mod-ux/src/com/x/ux"
  mkdir -p "$UX"
}

# writer: a -ux helper that writes a slot with the request Context
writer() {
  printf '%s\n' 'package com.x.ux;' 'final class Writer {' '  static void write(BComponent point, BValue v, Context cx) {' \
    '    point.set("out", v, cx);' '  }' '}' > "$UX/Writer.java"
}

@test "LDWS1: a write with the request Context and no save FAILs (exit 1); a read-only list add is no write" {
  # Mutation: LDWS1 -- matching any last argument instead of a declared Context FAILs the list add on line 5
  writer
  printf '%s\n' 'package com.x.ux;' 'public final class BReadServlet extends BWebServlet {' '  public void doGet(WebOp op) {' \
    '    java.util.List<String> out = new java.util.ArrayList<>();' '    out.add(name(op, 1));' '  }' '}' > "$UX/BReadServlet.java"
  run "$LINT" "$MOD"
  [ "$status" -eq 1 ]
  [ "${#lines[@]}" -eq 1 ]
  [[ "${lines[0]}" == "FAIL  lint-dashboard-write-save  $UX/Writer.java:4  writes station state with the request Context"* ]]
  [[ "${lines[0]}" == *"types/dashboard.md § Write path"* ]]
}

@test "LDWS2: the same write with a BStation.save scheduled in the module is clean (no row, exit 0)" {
  # Mutation: LDWS2 -- dropping the save search FAILs a module that schedules BStation.save after its writes
  writer
  printf '%s\n' 'package com.x.ux;' 'public final class BDashServlet extends BWebServlet {' '  private Clock.Ticket saveTicket;' \
    '  void requestSave(BStation station, long d) {' \
    '    saveTicket = Clock.schedule(station, BRelTime.make(Math.max(d, 1L)), BStation.save, null);' '  }' '}' > "$UX/BDashServlet.java"
  run "$LINT" "$MOD"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "LDWS3: a non-empty persistence: line in MODULE-MAP.md passes; an empty one declares nothing" {
  # Mutation: LDWS3 -- counting an empty persistence: line passes an undeclared module
  writer
  printf '%s\n' 'modularity: strict' 'persistence:' > "$MOD/MODULE-MAP.md"
  run "$LINT" "$MOD"
  [ "$status" -eq 1 ]
  printf '%s\n' 'modularity: strict' 'persistence: the control module ConfigSaveScheduler' > "$MOD/MODULE-MAP.md"
  run "$LINT" "$MOD"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "LDWS4: a root with no *.java under a -ux part's src/ is an env error (exit 3), never a clean pass" {
  # Mutation: LDWS4 -- dropping the no -ux ErrEmptyRoot reports a wrong-path root as a clean exit 0
  mkdir -p "$MOD/Mod-rt/src/com/x"
  printf '%s\n' 'package com.x;' 'class BLogic { void a(BValue v, Context cx) { set("out", v, cx); } }' > "$MOD/Mod-rt/src/com/x/BLogic.java"
  run "$LINT" "$MOD"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-dashboard-write-save  $MOD  no *.java under a -ux part's src/ (wrong path: pass the module root; a module with no -ux part has no dashboard writes)" ]
}

@test "LDWS5: no argument prints the usage line (exit 3)" {
  run "$LINT"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-dashboard-write-save.sh [--strict] <module-root>" ]
}
