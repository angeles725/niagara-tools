#!/usr/bin/env bats
# Kit doc pins for quality round 8 P1 unit "web" (register C31, C39, C43, C46).
# KPW1: the old types/cloud-connector.md § 7 and types/issues-and-gotchas.md H2 said a non-driver BHttpClient
# "will receive"/"throws" a ServiceException. httpClient-rt has no such exception: permittedByService catches the
# refusal, writes a health fault and returns false (vineflower httpClient-rt 4.14.0.162 BHttpClientService.java:231-252;
# javap httpClient-rt.jar 4.15.6.20). [ev: retro devguide415-r3-drivers-integration Δ4]
# KPW2: the same two passages placed okhttp in net-rt; the NRE ships it in bin/ext (C27, WU-P0-9).

KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"

@test "KPW1: no kit doc says the non-driver HTTP gate raises a ServiceException" {
  # shellcheck disable=SC2016  # the backticks are Markdown code spans in the pattern, not command substitution
  run grep -rnE '(receive|throws?) an? `?ServiceException' "$KIT/types" "$KIT/BUILD-LOOP.md" "$KIT/build-verify.md"
  [ "$status" -eq 1 ]
}

@test "KPW1 control: cloud-connector § 7 states the health-fault refusal message" {
  run grep -cE 'Non driver clients are disabled in the HttpClientService' "$KIT/types/cloud-connector.md"
  [ "$status" -eq 0 ]
}

@test "KPW2: no kit doc places okhttp in net-rt as a usable copy" {
  # shellcheck disable=SC2016  # the backticks are Markdown code spans in the pattern, not command substitution
  run grep -rniE 'okhttp`? \(?from `?net-rt|okhttp from `?net-rt`? is' "$KIT/types"
  [ "$status" -eq 1 ]
}

@test "KPW2 control: the H2 fix routes okhttp through bin/ext" {
  # shellcheck disable=SC2016  # the backticks are Markdown code spans in the pattern, not command substitution
  run grep -nE 'okhttp from `bin/ext`' "$KIT/types/issues-and-gotchas.md"
  [ "$status" -eq 0 ]
}
