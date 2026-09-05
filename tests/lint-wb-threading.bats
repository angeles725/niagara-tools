#!/usr/bin/env bats
# RED-FIRST pins for lint-wb-threading.sh (campaign 8 PR11, WB-THREAD1 + WB-AGENT1, B809 §809.7).
# Swing thread-affinity + agent-breadth checks over a -wb src tree. Real shapes:
# BBatchLinkEditor.java:684-720 (doInvoke DFS on the UI thread) and :66-67 (@AgentOn baja:Component).
#
# SURFACE: lint-wb-threading.sh [--strict] <wb-src-dir>   (exit 0 clean / 1 any --strict WARN / 3 usage)
#   ui-thread-traversal WARN  a doInvoke( body calling getNavChildren|getNavNodes|BQL with no
#                             invokeLater|BJobService|JobThread in the SAME body (B809: human review, WARN)
#   agent-breadth       WARN  @AgentOn(types="baja:Component") with no comment containing justif|why|broad
#                             within 3 lines above
#   --strict turns any WARN into exit 1.
#
# RED today: lint-wb-threading.sh does not exist -> every pin fails for the right reason (tool absent).
# NAMED MUTATIONS (post-green): drop the invokeLater|BJobService|JobThread recognizer -> the guarded
# companion WARNs; drop the justification-comment exemption -> the justified agent WARNs.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LW="$KIT/toolbelt/lint-wb-threading.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/lint-wb-threading"
  ONE="$BATS_TEST_TMPDIR/one"; mkdir -p "$ONE"
}
only() { rm -rf "$ONE"; mkdir -p "$ONE"; cp "$FX/$1" "$ONE/"; }

@test "WBT1: a doInvoke DFS on the UI thread WARNs ui-thread-traversal, exit 0 (WARN not FAIL, B809)" {
  only Traversal.java
  run "$LW" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]] && [[ "$output" == *"ui-thread-traversal"* ]] && [[ "$output" == *"Traversal"* ]]
}

@test "WBT1c: the same traversal off-loaded via invokeLater does NOT WARN (mutation target)" {
  only GuardedTraversal.java
  run "$LW" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "WBT-strict: --strict turns the traversal WARN into exit 1" {
  only Traversal.java
  run "$LW" --strict "$ONE"
  [ "$status" -eq 1 ]
  [[ "$output" == *"ui-thread-traversal"* ]]
}

@test "WBA1: @AgentOn(baja:Component) with no justification comment WARNs agent-breadth" {
  only BroadAgent.java
  run "$LW" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]] && [[ "$output" == *"agent-breadth"* ]]
}

@test "WBA1c: the same broad agent justified in a comment within 3 lines does NOT WARN (mutation target)" {
  only JustifiedAgent.java
  run "$LW" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "WBT-usage: no <wb-src-dir> argument -> exit 3" {
  run "$LW"
  [ "$status" -eq 3 ]
}

@test "WBT-prune: a Traversal copy under .deploy-baseline/ is NOT flagged (D9b dot-dir prune)" {
  rm -rf "$ONE"; mkdir -p "$ONE/.deploy-baseline"
  cp "$FX/GuardedTraversal.java" "$ONE/"                 # a clean live wb source anchors the scan
  cp "$FX/Traversal.java" "$ONE/.deploy-baseline/Stale.java"
  run "$LW" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
  [[ "$output" != *"Stale"* ]]
}
