#!/usr/bin/env bats
# fixtures-minimalpan-offload.bats — the MinimalPan fixture's async-offload component (BPanOffload) keeps the shape
# types/actions.md § 2 teaches: ASYNC action, post() override handing `new Invocation(...)` to a module-owned
# executor, a not-running guard, shutdownNow() in stopped(), MANAGE_EXECUTION requested. The fixture cannot be built
# without a Niagara install, so these are structural checks plus the kit lints over a mutated copy.
# [ev: course Labs/Lab9 Multithreaded/BExecutorServiceExecutor.java:46-70] [ev: findings dev-course S6, B3]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  MOD="$BATS_TEST_TMPDIR/MinimalPan"
  cp -r "$KIT/fixtures/MinimalPan/MinimalPan" "$MOD"
  RT="$MOD/MinimalPan-rt"
  SRC="$RT/src/com/angeles/MinimalPan/BPanOffload.java"
  TESTS="$RT/srcTest/test/com/angeles/MinimalPan"
}

@test "FO-shape: BPanOffload is ASYNC, offloads through new Invocation, guards on isRunning and shuts down in stopped" {
  # Mutation: FO-shape -- dropping any one of the five elements below is a regression of the sanctioned offload
  grep -q 'flags = Flags.ASYNC' "$SRC"
  grep -q 'public IFuture post(Action action, BValue arg, Context cx)' "$SRC"
  grep -q 'new Invocation(this, action, arg, cx)' "$SRC"
  grep -q '!isRunning()' "$SRC"
  awk '/public void stopped\(\)/,/^  }/' "$SRC" | grep -q 'shutdownNow()'
  grep -q 'implements ThreadFactory' "$SRC"
  run bash -c "grep -v '^ *\\*' \"\$1\" | grep -c 'this::doExecute'" _ "$SRC"
  [ "$output" = "0" ]
}

@test "FO-register: the type, the permission and both new station tests are registered" {
  grep -q 'class="com.angeles.MinimalPan.BPanOffload" name="PanOffload"' "$RT/module-include.xml"
  grep -q 'PanOffload' "$RT/module.palette"
  grep -q '<name>MANAGE_EXECUTION</name>' "$RT/module-permissions.xml"
  grep -q 'BMinimalPanOffloadTest' "$RT/moduleTest-include.xml"
  grep -q 'BMinimalPanPermissionsTest' "$RT/moduleTest-include.xml"
  [ -f "$TESTS/BMinimalPanOffloadTest.java" ]
  [ -f "$TESTS/BMinimalPanPermissionsTest.java" ]
  grep -q 'userPermissionsScenario' "$TESTS/BMinimalPanPermissionsTest.java"
  grep -q 'actAsUser' "$TESTS/BMinimalPanPermissionsTest.java"
}

@test "FO-lints: the fixture is clean under the action and async lints and lint-timers has no FAIL" {
  run "$KIT/toolbelt/lint-action-handler.sh" --strict "$RT/src"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  run "$KIT/toolbelt/lint-async-blocking.sh" --strict "$RT/src"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  run "$KIT/toolbelt/lint-timers.sh" "$RT/src"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL "* ]]
}

@test "FO-direct: running the handler as this::doExecute inside post() is flagged by lint-async-blocking" {
  # Mutation: FO-direct -- the lab's own shape (course D6) must not pass the kit's lints (lint-action-handler leaves
  # a post(Action...) override to lint-async-blocking)
  sed -i 's/running.execute(new Invocation(this, action, arg, cx));/running.execute(this::doExecute);/' "$SRC"
  run "$KIT/toolbelt/lint-async-blocking.sh" --strict "$RT/src"
  [ "$status" -eq 1 ]
  [[ "$output" == *"this::doExecute"* ]]
}

@test "FO-permission: without MANAGE_EXECUTION the shutdownNow() in stopped() FAILs lint-timers" {
  # Mutation: FO-permission -- the empty <permissions/> the fixture had before leaves shutdownNow() denied at stop
  printf '%s\n' '<?xml version="1.0" encoding="UTF-8"?>' '<permissions/>' > "$RT/module-permissions.xml"
  run "$KIT/toolbelt/lint-timers.sh" "$RT/src"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  jdk-thread"*"BPanOffload.java"* ]]
}
