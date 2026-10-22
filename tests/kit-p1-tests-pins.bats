#!/usr/bin/env bats
# Kit doc pins for quality round 8 P1 unit "tests" (register C40, C41, C42).
# KPT1: templates/moduleTest-include.xml keeps the course shape: a <types> root, one <type> per test class, test classes
# under the 4.14+ "test." package root (devkit-wb 4.14.0.162 gradle/videodriver/BNfooVideoDriverTest.java.vm:4).
# [ev: retro testing-parity-btestng Δ6]
# KPT2: the bin/ext table records that 4.15.6 ships httpclient5, not httpclient 4.x. [ev: retro devguide415-r2-shipped-libraries Δ3]

KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"

@test "KPT1: the moduleTest-include template is well-formed, <types> root, every class under test." {
  T="$KIT/templates/moduleTest-include.xml"
  [ -f "$T" ]
  run python3 -c 'import sys,xml.etree.ElementTree as E; r=E.parse(sys.argv[1]).getroot(); t=r.findall("type"); print(r.tag, len(t), all(x.get("class","").startswith("test.") and x.get("name") for x in t))' "$T"
  [ "$status" -eq 0 ]
  [ "$output" = "types 1 True" ]
}

@test "KPT2: the shipped-library table names httpclient5 for 4.15.6.20 on the HttpClient row" {
  run grep -nE '^\| Apache HttpClient \|.*httpclient 4\.5\.13.*httpclient5 5\.3\.1' "$KIT/types/third-party-libraries.md"
  [ "$status" -eq 0 ]
}
