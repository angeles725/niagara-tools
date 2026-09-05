#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
# RED-FIRST tests for build-n4-module-kit/toolbelt/preflight.sh (Campaign 6 PR5b T5b.1/T5b.2).
# Contract: openspec/changes/build-n4-module-campaign6/{spec.md R5-3, design.md §5}.
#
# preflight.sh [--jvm-dir <d>] <niagara_home> <gradle-root>
#
# Checks (in order, all non-fatal — continue after FAIL):
#   jdk8      JDK 8 present under --jvm-dir (default /usr/lib/jvm) or JAVA_HOME; never $HOME
#   plugin-pin  <gradle-root>/settings.gradle.kts plugin version present in <nh>/etc/m2
#   jar-lock  lsof on <nh>/modules/*.jar (SKIP if lsof absent — never false PASS)
#   win-path  C:\ or \ in <niagara_home> or <gradle-root> -> FAIL + /mnt/c/... remedy
#
# Row format: PASS|FAIL|WARN|SKIP  <check>  <detail>
# Exit: 0 all PASS/WARN/SKIP · 1 any FAIL · 2 usage · 3 env
#
# Fixtures: tests/fixtures/preflight/{jvm/,niagara-home/,niagara-home-no-pin/,gradle-root/}
#
# Named mutations (each flips one test after GREEN; revert after proof):
#   PF1: plugin check always PASS -> PF1 exits 0 instead of 1
#   PF2: resolve JVM via $HOME -> PF2 output diverges under HOME=/nonexistent
#   PF4: lsof-absent path always PASS -> PF4 assertion [[ $output != *PASS* ]] flips

load helpers/n4-fixtures

setup() {
  TMPDIR_T="$(mktemp -d)"
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  PREFLIGHT="$KIT/toolbelt/preflight.sh"
  FIXDIR="$(cd "$BATS_TEST_DIRNAME" && pwd)/fixtures/preflight"

  # Stable fixture paths (not regenerated per-test — fixtures are pre-committed)
  JVMDIR="$FIXDIR/jvm"
  NH="$FIXDIR/niagara-home"         # has plugin pin 7.6.17 in etc/m2
  NH_NO_PIN="$FIXDIR/niagara-home-no-pin"  # etc/m2 is empty (no plugin)
  GR="$FIXDIR/gradle-root"          # settings.gradle.kts references 7.6.17
}

teardown() {
  rm -rf "$TMPDIR_T"
}

# ---------------------------------------------------------------------------
# PF1 — plugin pin missing -> FAIL row + exit 1
# Named mutation: plugin check always PASS -> PF1 exits 0
# ---------------------------------------------------------------------------
@test "PF1: missing plugin pin in niagara_home/etc/m2 -> FAIL row + exit 1" {
  run "$PREFLIGHT" --jvm-dir "$JVMDIR" "$NH_NO_PIN" "$GR"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
  [[ "$output" == *"plugin-pin"* ]]
}

# ---------------------------------------------------------------------------
# PF2 — identical output under HOME=/nonexistent (no $HOME coupling)
# Named mutation: resolve JVM via $HOME -> HOME=/nonexistent output diverges
# ---------------------------------------------------------------------------
@test "PF2: output is identical under HOME=/nonexistent (no \$HOME coupling)" {
  out_real=$(HOME=/tmp "$PREFLIGHT" --jvm-dir "$JVMDIR" "$NH_NO_PIN" "$GR" 2>&1) || true
  out_none=$(HOME=/nonexistent "$PREFLIGHT" --jvm-dir "$JVMDIR" "$NH_NO_PIN" "$GR" 2>&1) || true
  [ "$out_real" = "$out_none" ]
}

# ---------------------------------------------------------------------------
# PF3 — Windows-style path -> FAIL row + /mnt/c/... remedy text
# ---------------------------------------------------------------------------
@test "PF3: Windows-style path (C:\\\\...) -> FAIL row + /mnt/c/ remedy text" {
  run "$PREFLIGHT" --jvm-dir "$JVMDIR" 'C:\Niagara\niagara-4.14' "$GR"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
  [[ "$output" == *"win-path"* ]]
  [[ "$output" == *"/mnt/c/"* ]]
}

# ---------------------------------------------------------------------------
# PF4 — lsof absent -> SKIP row, never PASS
# Uses a restricted PATH that includes essential tools but not lsof.
# Named mutation: lsof-absent path -> PASS -> PF4 assertion flips
# ---------------------------------------------------------------------------
@test "PF4: lsof absent in PATH -> SKIP row, never PASS" {
  FAKEBIN="$TMPDIR_T/fakebin"
  mkdir -p "$FAKEBIN"
  # Symlink every tool the preflight script needs — but NOT lsof
  for t in bash sh env grep sed awk cut find paste sort uniq wc tr comm; do
    p=$(command -v "$t" 2>/dev/null) && ln -sf "$p" "$FAKEBIN/$t" || true
  done
  # Run with a PATH that has our fakebin first and then only dirs without lsof
  # lsof on this system: $(command -v lsof) — excluded by restricting PATH
  out=$(PATH="$FAKEBIN" "$PREFLIGHT" --jvm-dir "$JVMDIR" "$NH" "$GR" 2>&1) || true
  [[ "$out" == *"SKIP"* ]]
  [[ "$out" != *"PASS  jar-lock"* ]]
}
