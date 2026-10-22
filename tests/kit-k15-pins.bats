#!/usr/bin/env bats
# Kit doc pin for METHODOLOGY K15 (register C84). The old K15 said an empty pre-slotomatic skeleton "has no generated
# region" and that building it triggers "Found multiple metadata blocks"; the code throws that message for a region
# whose stamp-line count is not exactly one (organized/ devkit-wb 4.14.0.162 vineflower JavaUnit.java:270-272), i.e.
# for pre-written EMPTY markers, and slot-o-matic only logs it (Compiler.java:241).
# [ev: retro n4-course-handson-slotomatic-authoring Δ1]

@test "KK1: no kit doc blames 'Found multiple metadata blocks' on a source with no generated region" {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  run grep -rnE 'has no generated region; building it triggers' "$KIT/METHODOLOGY.md" "$KIT/types" "$KIT/BUILD-LOOP.md"
  [ "$status" -eq 1 ]
}
