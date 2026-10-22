#!/usr/bin/env bats
# Kit doc pins for quality round 8 P1 unit "control" (register C32-C35, C50).
# KPC1: the old types/logic.md LC5 called the alarm source extension "notification-only". It owns the ALARM, FAULT and
# UNACKED_ALARM bits of out: it clears them and ORs its own (devguide source 4.15.6.20
# alarm-rt/javax/baja/alarm/ext/BAlarmSourceExt.java:1330-1331,1954; vineflower alarm-rt 4.14.0.162 :222).
# [ev: retro devguide415-runtime-api-corrections Δ6]

KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"

@test "KPC1: no kit doc calls the alarm extension notification-only" {
  run grep -rnE 'alarm-ext[^.;]*(is )?notification-only' "$KIT/types" "$KIT/BUILD-LOOP.md"
  [ "$status" -eq 1 ]
}

@test "KPC1 control: LC5 states the status bits the alarm extension owns" {
  run grep -nE 'OWNS the ALARM, FAULT and UNACKED_ALARM bits' "$KIT/types/logic.md"
  [ "$status" -eq 0 ]
}
