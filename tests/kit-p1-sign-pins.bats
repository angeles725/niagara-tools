#!/usr/bin/env bats
# Kit doc pins for quality round 8 P1 unit "sign" (register C47, C48).
# KPS1: the old types/logic-authoring.md § BProgram said an unsigned BRobotCode blob "is rejected at instantiation".
# BCode only verifies when a signature is present or program.requireSigning is true; otherwise it logs
# program.notSigned (vineflower program-rt 4.14.0.162 com/tridium/program/BCode.java:202-215).
# [ev: retro devguide415-r2-packaging-platform-signing Δ7]
# KPS2: templates/module-permissions.xml keeps the wizard shape (devkit-wb 4.14.0.162
# gradle/includes/niagara/module-permissions.xml.vm). [ev: retro n4-course-authoring-files-and-libraries Δ2]

KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"

@test "KPS1: no kit doc says an unsigned program/robot blob is rejected at instantiation" {
  run grep -rniE 'unsigned class blob is rejected' "$KIT/types" "$KIT/build-verify.md"
  [ "$status" -eq 1 ]
}

@test "KPS1 control: the BProgram passage names the program.requireSigning switch" {
  run grep -nE 'program\.requireSigning' "$KIT/types/logic-authoring.md"
  [ "$status" -eq 0 ]
}

@test "KPS2: the module-permissions template is well-formed with a <permissions> root and the three group types" {
  command -v python3 >/dev/null || skip "python3 not installed"
  T="$KIT/templates/module-permissions.xml"
  [ -f "$T" ]
  run python3 -c 'import sys,xml.etree.ElementTree as E; r=E.parse(sys.argv[1]).getroot(); print(r.tag, *sorted(g.get("type") for g in r.findall("niagara-permission-groups")))' "$T"
  [ "$status" -eq 0 ]
  [ "$output" = "permissions all station workbench" ]
}

@test "KPS3: every commented-out request in the template parses once uncommented (no raw < in placeholders)" {
  command -v python3 >/dev/null || skip "python3 not installed"
  run python3 -c '
import re, sys, xml.etree.ElementTree as E
n = 0
for c in re.findall(r"<!--(.*?)-->", open(sys.argv[1]).read(), re.S):
    m = re.search(r"<(req|opt)-permission>.*</(req|opt)-permission>", c, re.S)
    if m:
        E.fromstring("<x>" + m.group(0) + "</x>"); n += 1
print(n)' "$KIT/templates/module-permissions.xml"
  [ "$status" -eq 0 ]
  [ "$output" -ge 1 ]
}
