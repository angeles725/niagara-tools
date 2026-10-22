#!/usr/bin/env bats
# Pins the driver cards of the kit against the API they teach, so a wrong snippet cannot come back silently.
# KD1 regression: types/driver-authoring.md showed `public BObject doPing(Context cx)` with `return pingOk()`;
#     the API is `public abstract void doPing()` with void pingOk()/pingFail(String) (register C83,
#     retro n4-course-handson-ndriver Δ1; driver-rt BDevice.java:552,570,580).
# KD2 regression: types/logic-authoring.md said "enable `pingEnabled`"; it is ON by default (register C14,
#     retro devguide415-r3-drivers-integration Δ2; BPingMonitor.java:128 newProperty(0, true, null)).
# KD3 regression: types/driver-authoring.md said `pollEnabled` lives on the network; it is a poll-service
#     property (register C16, retro devguide415-r2-drivers-protocols Δ8; BAbstractPollService.java:62).
# KD4 regression: the readUnsubscribed example sent a poll request; it must stop polling (register C15,
#     retro devguide415-nav-subscriptions-tags-ords Δ1; BProxyExt.java:859-869).

setup() { KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"; }

@test "KD1: no kit doc shows a BObject-returning doPing or a returned pingOk/pingFail" {
  run grep -rnE 'BObject doPing|return ping(Ok|Fail)\(' "$KIT"/*.md "$KIT"/types/*.md
  [ "$status" -eq 1 ]     # grep: no match
}

@test "KD2: no kit doc tells the author to enable pingEnabled (it is on by default)" {
  run grep -rniE 'enable `?pingEnabled' "$KIT"/*.md "$KIT"/types/*.md
  [ "$status" -eq 1 ]
}

@test "KD3: types/driver-authoring.md does not place pollEnabled on the network" {
  run grep -nE 'pollEnabled`? on the network' "$KIT/types/driver-authoring.md"
  [ "$status" -eq 1 ]
}

@test "KD4: a readUnsubscribed example in the kit never sends, posts or polls" {
  run awk '/void readUnsubscribed\(/ { on = 1; next }
           on && /^[[:space:]]*}[[:space:]]*$/ { on = 0; next }
           on && /(send|post|poll)[A-Za-z]*\(/ { print FILENAME ":" FNR ": " $0 }' "$KIT"/types/*.md
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}
