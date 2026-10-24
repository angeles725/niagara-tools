#!/usr/bin/env bats
# lint-ndriver-tagless-correlation.bats — G67: NDriver messages with no getTag() and a comm config that keeps more
# than one request outstanding is one WARN row per module; the rule's golden cases live in
# go/internal/lint/rules/ndrivertaglesscorrelation. [ev: retro n4-course-drivers-bql-ords-files Δ3]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-ndriver-tagless-correlation.sh"
  DRV="$BATS_TEST_TMPDIR/acmeDriver"
  cp -r "$BATS_TEST_DIRNAME/fixtures/driver-lints/acmeDriver" "$DRV"
  RT="$DRV/acmeDriver-rt/src/com/example/drv"
  CFG="$RT/comm/BAcmeCommConfig.java"
}

MSG='no getTag() override and getMaxOutstandingTransactions() is not 1 -- a response can complete the wrong request'

@test "NTC-usage: no argument prints the usage and exits 3" {
  run "$L"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-ndriver-tagless-correlation.sh [--strict] <module-root>..." ]
}

@test "NTC1: the fixture module (no getTag(), the return 1 hint commented out) WARNs once on its first message" {
  # Mutation: NTC1 -- reading comments as code turns the commented-out hint into a fix (exit 0, no row)
  run "$L" "$DRV"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-ndriver-tagless-correlation  $RT/message/AcmeReq.java:6  $MSG" ]
  run "$L" --strict "$DRV"
  [ "$status" -eq 1 ]
}

@test "NTC2: a comm config returning 4 still WARNs; returning 1 is clean" {
  # Mutation: NTC2 -- accepting any returned value turns the return 4 case green
  sed -i 's#^  // public int getMax#  public int getMax#; s#return 1;#return 4;#' "$CFG"
  run "$L" "$DRV"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-ndriver-tagless-correlation  $RT/message/AcmeReq.java:6  $MSG" ]
  sed -i 's#return 4;#return 1;#' "$CFG"
  run "$L" --strict "$DRV"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "NTC-tag: a getTag() override in one message is clean; a module with no message is clean" {
  sed -i 's#public boolean isResponse() { return true; }#public Object getTag() { return seq; }#' "$RT/message/AcmeResp.java"
  run "$L" --strict "$DRV"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  rm -r "$RT/message"
  run "$L" --strict "$DRV"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "NTC-K20: a root with no -rt src/ *.java is the ERROR row, exit 3" {
  rm -r "$DRV/acmeDriver-rt"; mkdir -p "$DRV/acmeDriver-wb/src"; printf 'class A extends NMessage {}\n' > "$DRV/acmeDriver-wb/src/A.java"
  run "$L" "$DRV"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-ndriver-tagless-correlation  $DRV  no *.java under a -rt part's src/ (wrong path: pass the module root)" ]
}
