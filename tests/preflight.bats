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

# ---------------------------------------------------------------------------
# PF5 — JDK 8 with no release file, bin/java reports 1.8 -> PASS (WSL fallback)
# Fixture: tests/fixtures/preflight/jvm-no-release/java-8-openjdk-wsl/bin/java
#          (fakebin printing openjdk version "1.8.0_412"; NO release file)
# Named mutation: remove the bin/java fallback scan -> PF5 exits 1 (FAIL jdk8)
# ---------------------------------------------------------------------------
# ---------------------------------------------------------------------------
# version-drift — sibling checkout at higher version triggers WARN; exit stays 0
# Named mutation: version-drift -- remove the sort -V compare -> PF-drift exits 0 silently
# ---------------------------------------------------------------------------
@test "version-drift: sibling dir at higher version -> WARN row + exit 0" {
  # Build a temp gradle-root hierarchy with two siblings.
  # module-a uses the real gradle-root fixture (has settings.gradle.kts + plugin pin)
  # but also adds build.gradle.kts with a low version; module-b only needs a kts with a higher version.
  DRIFT_ROOT="$TMPDIR_T/drift_test"
  mkdir -p "$DRIFT_ROOT/parent/module-a" "$DRIFT_ROOT/parent/module-b"
  # Copy the real gradle-root fixture so plugin-pin + settings.gradle.kts work
  cp -r "$FIXDIR/gradle-root/." "$DRIFT_ROOT/parent/module-a/"
  # module-a version: 1.0.0
  printf 'defaultModuleVersion("1.0.0")\n' >> "$DRIFT_ROOT/parent/module-a/settings.gradle.kts"
  # module-b: higher version sibling (no settings.gradle.kts needed — only build.gradle.kts)
  printf 'defaultModuleVersion("1.0.1")\n' > "$DRIFT_ROOT/parent/module-b/build.gradle.kts"
  run "$PREFLIGHT" --jvm-dir "$JVMDIR" "$NH" "$DRIFT_ROOT/parent/module-a"
  # exit must be 0 (WARN does not flip FAILED; jdk8/plugin-pin come from real fixtures)
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"version-drift"* ]]
  [[ "$output" == *"1.0.0"* ]]
  [[ "$output" == *"1.0.1"* ]]
}

@test "version-drift: no sibling with higher version -> no version-drift row" {
  DRIFT_ROOT2="$TMPDIR_T/drift_test2"
  mkdir -p "$DRIFT_ROOT2/parent/module-a" "$DRIFT_ROOT2/parent/module-b"
  cp -r "$FIXDIR/gradle-root/." "$DRIFT_ROOT2/parent/module-a/"
  # Both siblings at the same version
  printf 'defaultModuleVersion("2.1.0")\n' >> "$DRIFT_ROOT2/parent/module-a/settings.gradle.kts"
  printf 'defaultModuleVersion("2.1.0")\n' > "$DRIFT_ROOT2/parent/module-b/build.gradle.kts"
  run "$PREFLIGHT" --jvm-dir "$JVMDIR" "$NH" "$DRIFT_ROOT2/parent/module-a"
  [ "$status" -eq 0 ]
  [[ "$output" != *"version-drift"* ]]
}

@test "PF5: JDK 8 with no release file but bin/java reports 1.8 -> PASS jdk8 (WSL fallback)" {
  JVM_NO_RELEASE="$FIXDIR/jvm-no-release"
  run "$PREFLIGHT" --jvm-dir "$JVM_NO_RELEASE" "$NH" "$GR"
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS"* ]]
  [[ "$output" == *"jdk8"* ]]
  [[ "$output" != *"FAIL  jdk8"* ]]
}

# ---------------------------------------------------------------------------
# Δ8 (retro panccadia-defrost-sequencing-hmi-reload-deltas): jar-lock Check 4
# replaces a per-jar `lsof <jar>` loop (O(N) forks) with a single filtered
# `lsof -Fn` pass, and SKIPs the check entirely when niagara_home resolves to
# a WSL 9p/drvfs mount (lsof cannot see a Windows-side lock there anyway).
# A restricted PATH with a fake lsof isolates the call-count/timing/content
# assertions from the host's real lsof and mounts.
# ---------------------------------------------------------------------------

_pf_fakebin() {
  # Symlinks the tools preflight.sh needs (mirrors PF4, plus basename/head which the
  # jar-lock WARN row and other checks need) into $1, WITHOUT lsof — tests place their
  # own fake lsof at $1/lsof afterward.
  mkdir -p "$1"
  for t in bash sh env grep sed awk cut find paste sort uniq wc tr comm basename head; do
    p=$(command -v "$t" 2>/dev/null || true)
    [ -z "$p" ] || ln -sf "$p" "$1/$t"
  done
}

@test "PF-jarlock-single-call: lsof is invoked exactly ONCE regardless of jar count (Δ8 perf fix)" {
  FAKEBIN="$TMPDIR_T/fakebin-count"; _pf_fakebin "$FAKEBIN"
  CALLS="$TMPDIR_T/lsof.calls"; : > "$CALLS"
  NH2="$TMPDIR_T/nh-count"; mkdir -p "$NH2/etc/m2" "$NH2/modules"
  for i in 1 2 3 4 5; do : > "$NH2/modules/Mod$i.jar"; done
  printf '#!/usr/bin/env bash\necho call >> "%s"\n' "$CALLS" > "$FAKEBIN/lsof"
  chmod +x "$FAKEBIN/lsof"
  PATH="$FAKEBIN" "$PREFLIGHT" --jvm-dir "$JVMDIR" "$NH2" "$GR" >/dev/null 2>&1 || true
  # NAMED MUTATION: reintroduce the per-jar `for jar in ...; do lsof "$jar"; done` loop ->
  # this count becomes 5 (one per jar), so this assertion fails.
  [ "$(wc -l < "$CALLS")" -eq 1 ]
}

@test "PF-jarlock-timing: a single pass keeps runtime O(1) in jar count, not O(N) (Δ8 perf fix)" {
  FAKEBIN="$TMPDIR_T/fakebin-timing"; _pf_fakebin "$FAKEBIN"
  NH2="$TMPDIR_T/nh-timing"; mkdir -p "$NH2/etc/m2" "$NH2/modules"
  for i in $(seq 1 20); do : > "$NH2/modules/Mod$i.jar"; done
  printf '#!/usr/bin/env bash\nsleep 0.2\n' > "$FAKEBIN/lsof"
  chmod +x "$FAKEBIN/lsof"
  start=$(date +%s%N)
  PATH="$FAKEBIN" "$PREFLIGHT" --jvm-dir "$JVMDIR" "$NH2" "$GR" >/dev/null 2>&1 || true
  end=$(date +%s%N)
  elapsed_ms=$(( (end - start) / 1000000 ))
  # A per-jar loop over 20 jars at 0.2s each would take >=4000ms; a single pass stays well
  # under 1 lsof call's worth of overhead (a couple hundred ms plus script overhead).
  [ "$elapsed_ms" -lt 2000 ]
}

@test "PF-jarlock-detects-locked: the single -Fn pass still correctly flags a locked jar (functional parity)" {
  FAKEBIN="$TMPDIR_T/fakebin-detect"; _pf_fakebin "$FAKEBIN"
  NH2="$TMPDIR_T/nh-lock"; mkdir -p "$NH2/etc/m2" "$NH2/modules"
  : > "$NH2/modules/Locked.jar"
  : > "$NH2/modules/Free.jar"
  printf '#!/usr/bin/env bash\nprintf "p1234\\nn%s\\n"\n' "$NH2/modules/Locked.jar" > "$FAKEBIN/lsof"
  chmod +x "$FAKEBIN/lsof"
  out=$(PATH="$FAKEBIN" "$PREFLIGHT" --jvm-dir "$JVMDIR" "$NH2" "$GR" 2>&1) || true
  [[ "$out" == *"WARN"* ]]
  [[ "$out" == *"jar-lock"* ]]
  [[ "$out" == *"Locked.jar"* ]]
  [[ "$out" != *"Free.jar"* ]]
}

@test "PF-jarlock-skip-9p: niagara_home on a 9p/drvfs mount -> SKIP jar-lock, lsof never invoked" {
  MOUNTS="$TMPDIR_T/mounts-9p"
  printf '%s\n' "9p $NH 9p rw 0 0" > "$MOUNTS"
  FAKEBIN="$TMPDIR_T/fakebin-9p"; _pf_fakebin "$FAKEBIN"
  CALLS="$TMPDIR_T/lsof.calls.9p"; : > "$CALLS"
  printf '#!/usr/bin/env bash\necho call >> "%s"\n' "$CALLS" > "$FAKEBIN/lsof"
  chmod +x "$FAKEBIN/lsof"
  out=$(N4_FSTYPE_MOUNTS_FILE="$MOUNTS" PATH="$FAKEBIN" "$PREFLIGHT" --jvm-dir "$JVMDIR" "$NH" "$GR" 2>&1) || true
  [[ "$out" == *"SKIP"* ]]
  [[ "$out" == *"jar-lock"* ]]
  { [[ "$out" == *"9p"* ]] || [[ "$out" == *"drvfs"* ]]; }
  [ ! -s "$CALLS" ]
}

@test "PF-jarlock-no-9p: no 9p/drvfs mount -> jar-lock runs normally (not SKIPped by the new gate)" {
  MOUNTS="$TMPDIR_T/mounts-ext4"
  printf '%s\n' "ext4 / ext4 rw 0 0" > "$MOUNTS"
  FAKEBIN="$TMPDIR_T/fakebin-ext4"; _pf_fakebin "$FAKEBIN"
  printf '#!/usr/bin/env bash\n' > "$FAKEBIN/lsof"
  chmod +x "$FAKEBIN/lsof"
  out=$(N4_FSTYPE_MOUNTS_FILE="$MOUNTS" PATH="$FAKEBIN" "$PREFLIGHT" --jvm-dir "$JVMDIR" "$NH" "$GR" 2>&1) || true
  [[ "$out" == *"PASS  jar-lock"* ]]
}
