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

load helpers/symroot   # assert_symroot_same <dir> <cmd...> (issue #329)

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
  for t in bash sh env grep sed awk cut find paste sort uniq wc tr comm basename dirname head; do
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
  # A real lsof always lists its own process's files: a non-empty listing with no module jar in it.
  printf '#!/usr/bin/env bash\nprintf "p1\\nn/usr/bin/lsof\\n"\n' > "$FAKEBIN/lsof"
  chmod +x "$FAKEBIN/lsof"
  out=$(N4_FSTYPE_MOUNTS_FILE="$MOUNTS" PATH="$FAKEBIN" "$PREFLIGHT" --jvm-dir "$JVMDIR" "$NH" "$GR" 2>&1) || true
  [[ "$out" == *"PASS  jar-lock"* ]]
}

@test "PF-jarlock-lsof-fail: lsof that fails with an empty listing -> SKIP jar-lock, never PASS" {
  # Mutation: PF-jarlock-lsof-fail -- treating an empty lsof listing as "no locked jars" reports PASS
  # when lsof saw nothing at all (permission error, crash).
  MOUNTS="$TMPDIR_T/mounts-ext4b"
  printf '%s\n' "ext4 / ext4 rw 0 0" > "$MOUNTS"
  FAKEBIN="$TMPDIR_T/fakebin-fail"; _pf_fakebin "$FAKEBIN"
  printf '#!/usr/bin/env bash\necho "lsof: permission denied" >&2\nexit 1\n' > "$FAKEBIN/lsof"
  chmod +x "$FAKEBIN/lsof"
  out=$(N4_FSTYPE_MOUNTS_FILE="$MOUNTS" PATH="$FAKEBIN" "$PREFLIGHT" --jvm-dir "$JVMDIR" "$NH" "$GR" 2>&1) || true
  [[ "$out" == *"SKIP  jar-lock"* ]]
  [[ "$out" != *"PASS  jar-lock"* ]]
}

# ================= #432: jar-lock matches the jar set in ONE pass =================
# The single `lsof -Fn` call (Δ8 above) still forked `printf | grep -qxF` once per jar over the whole
# captured listing: 410 jars on a busy host hung preflight for more than 10 minutes. The jar set is now
# matched against the listing in one awk. The fixture is 400 jars and a 60000-line listing.
# [ev: retro r10-fork-per-line-again Δ1]

# _pf_bigjars <nh> <listing> <locked-index...>: creates 400 jars Mod001.jar..Mod400.jar under <nh>/modules
# and a 60000-line lsof -Fn listing in <listing> that holds the named jars (plus unrelated files).
_pf_bigjars() {
  local nh="$1" listing="$2"; shift 2
  mkdir -p "$nh/etc/m2" "$nh/modules"
  awk -v nh="$nh" -v locked="$*" 'BEGIN {
    for (i = 1; i <= 400; i++) { f = sprintf("%s/modules/Mod%03d.jar", nh, i); printf "" > f; close(f) }
    n = split(locked, L, " ")
    for (i = 1; i <= 30000; i++) { printf "p%d\nn/usr/lib/x86_64-linux-gnu/libfile%d.so\n", i, i }
    for (i = n; i >= 1; i--) printf "p9\nn%s/modules/Mod%03d.jar\n", nh, L[i]
  }' > "$listing"
}

# _pf_cat_lsof <fakebin> <listing>: a fake lsof that prints <listing>.
_pf_cat_lsof() {
  printf '#!/usr/bin/env bash\nexec awk 1 "%s"\n' "$2" > "$1/lsof"
  chmod +x "$1/lsof"
}

@test "PF-jarlock-bulk-order: 400 jars, 3 locked -> WARN lists exactly the locked basenames in glob order" {
  FAKEBIN="$TMPDIR_T/fakebin-bulk"; _pf_fakebin "$FAKEBIN"
  NH2="$TMPDIR_T/nh-bulk"; _pf_bigjars "$NH2" "$TMPDIR_T/listing" 7 250 399
  _pf_cat_lsof "$FAKEBIN" "$TMPDIR_T/listing"
  out=$(PATH="$FAKEBIN" "$PREFLIGHT" --jvm-dir "$JVMDIR" "$NH2" "$GR" 2>&1) || true
  # The listing names the jars in reverse order; the row keeps glob order.
  row=$(printf '%s\n' "$out" | grep 'jar-lock')
  [ "$(printf '%s\n' "$row" | wc -l)" -eq 1 ]
  [[ "$row" =~ ^WARN\ +jar-lock\ +jars\ held\ by\ a\ process\ \(station\ running\?\):\ Mod007\.jar\ Mod250\.jar\ Mod399\.jar$ ]]
}

@test "PF-jarlock-bulk-timeout: 400 jars against a 60000-line listing finish under 10 s" {
  # NAMED MUTATION: restore the per-jar `printf '%s\n' "$LSOF_OUT" | grep -qxF "n$jar"` loop -> 400 forks
  # over the whole listing -> this run exceeds the bound and timeout exits 124.
  FAKEBIN="$TMPDIR_T/fakebin-bulkt"; _pf_fakebin "$FAKEBIN"
  NH2="$TMPDIR_T/nh-bulkt"; _pf_bigjars "$NH2" "$TMPDIR_T/listing-t" 1
  _pf_cat_lsof "$FAKEBIN" "$TMPDIR_T/listing-t"
  TIMEOUT="$(command -v timeout)"
  rc=0; PATH="$FAKEBIN" "$TIMEOUT" 10 "$PREFLIGHT" --jvm-dir "$JVMDIR" "$NH2" "$GR" >"$TMPDIR_T/out" 2>&1 || rc=$?
  [ "$rc" -ne 124 ]
  grep -q 'WARN  jar-lock .*Mod001.jar' "$TMPDIR_T/out"
}

@test "PF-jarlock-bulk-pass: 400 jars, none in the listing -> PASS" {
  FAKEBIN="$TMPDIR_T/fakebin-bulkp"; _pf_fakebin "$FAKEBIN"
  NH2="$TMPDIR_T/nh-bulkp"; _pf_bigjars "$NH2" "$TMPDIR_T/listing-p"
  _pf_cat_lsof "$FAKEBIN" "$TMPDIR_T/listing-p"
  out=$(PATH="$FAKEBIN" "$PREFLIGHT" --jvm-dir "$JVMDIR" "$NH2" "$GR" 2>&1) || true
  [[ "$out" == *"PASS  jar-lock"*"no locked jars under $NH2/modules/"* ]]
}

@test "PF-jarlock-bulk-empty: 400 jars, empty lsof output -> SKIP, never PASS" {
  FAKEBIN="$TMPDIR_T/fakebin-bulke"; _pf_fakebin "$FAKEBIN"
  NH2="$TMPDIR_T/nh-bulke"; _pf_bigjars "$NH2" "$TMPDIR_T/listing-e"
  printf '#!/usr/bin/env bash\nexit 0\n' > "$FAKEBIN/lsof"; chmod +x "$FAKEBIN/lsof"
  out=$(PATH="$FAKEBIN" "$PREFLIGHT" --jvm-dir "$JVMDIR" "$NH2" "$GR" 2>&1) || true
  [[ "$out" == *"SKIP  jar-lock"* ]]
  [[ "$out" != *"PASS  jar-lock"* ]]
}

@test "PF-jarlock-whole-line: a listing path that is a prefix or extension of a jar path does not match" {
  FAKEBIN="$TMPDIR_T/fakebin-whole"; _pf_fakebin "$FAKEBIN"
  NH2="$TMPDIR_T/nh-whole"; mkdir -p "$NH2/etc/m2" "$NH2/modules"
  : > "$NH2/modules/Mod.jar"; : > "$NH2/modules/Other.jar"
  # Prefix of Mod.jar's path, a longer path that starts with it, and the path without the `n` field tag.
  printf '#!/usr/bin/env bash\nprintf "p1\\nn%s\\nn%s.bak\\n%s\\n"\n' \
    "$NH2/modules/Mod.ja" "$NH2/modules/Mod.jar" "$NH2/modules/Other.jar" > "$FAKEBIN/lsof"
  chmod +x "$FAKEBIN/lsof"
  out=$(PATH="$FAKEBIN" "$PREFLIGHT" --jvm-dir "$JVMDIR" "$NH2" "$GR" 2>&1) || true
  [[ "$out" == *"PASS  jar-lock"* ]]
}

@test "PF-jarlock-regular-only: a directory named *.jar is not a jar" {
  FAKEBIN="$TMPDIR_T/fakebin-dir"; _pf_fakebin "$FAKEBIN"
  NH2="$TMPDIR_T/nh-dir"; mkdir -p "$NH2/etc/m2" "$NH2/modules/Dir.jar"
  printf '#!/usr/bin/env bash\nprintf "p1\\nn%s\\n"\n' "$NH2/modules/Dir.jar" > "$FAKEBIN/lsof"
  chmod +x "$FAKEBIN/lsof"
  out=$(PATH="$FAKEBIN" "$PREFLIGHT" --jvm-dir "$JVMDIR" "$NH2" "$GR" 2>&1) || true
  [[ "$out" == *"PASS  jar-lock"* ]]
}

# ================= WU7 fold: [ev: retro continuous-fan-post-defrost-delay Δ1] =================
# The repo's settings.gradle.kts default (7.6.17 in the fixture) has drifted from the niagara_home
# actually configured (here: one that ships only 7.6.22). build.sh forwards -PniagaraPluginVersion,
# so preflight must check THAT version, not re-derive the stale default.
@test "PF-plugin-override: --plugin-version V checks V in etc/m2 instead of the settings default" {
  make_niagara_home "$TMPDIR_T/nh22" 7.6.22
  run "$PREFLIGHT" --jvm-dir "$JVMDIR" --plugin-version 7.6.22 "$TMPDIR_T/nh22" "$GR"
  [[ "$output" == *"PASS  plugin-pin"*"7.6.22"* ]]
  [[ "$output" == *"--plugin-version"* ]]
}

@test "PF-plugin-override-prefix: a version prefix does not match a longer version dir (7.6.2 vs 7.6.22)" {
  make_niagara_home "$TMPDIR_T/nh22" 7.6.22
  run "$PREFLIGHT" --jvm-dir "$JVMDIR" --plugin-version 7.6.2 "$TMPDIR_T/nh22" "$GR"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  plugin-pin"*"7.6.2 (from --plugin-version)"* ]]
}

@test "PF-plugin-default: without the override the same environment still FAILs on the stale default" {
  make_niagara_home "$TMPDIR_T/nh22" 7.6.22
  run "$PREFLIGHT" --jvm-dir "$JVMDIR" "$TMPDIR_T/nh22" "$GR"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  plugin-pin"*"7.6.17"* ]]
}

@test "PF-plugin-env: \$NIAGARA_PLUGIN_VERSION is honored like --plugin-version" {
  make_niagara_home "$TMPDIR_T/nh22" 7.6.22
  NIAGARA_PLUGIN_VERSION=7.6.22 run "$PREFLIGHT" --jvm-dir "$JVMDIR" "$TMPDIR_T/nh22" "$GR"
  [[ "$output" == *"PASS  plugin-pin"*"7.6.22"* ]]
}

@test "PF-plugin-override-missing: an override absent from etc/m2 still FAILs (the override is checked, not trusted)" {
  run "$PREFLIGHT" --jvm-dir "$JVMDIR" --plugin-version 9.9.9 "$NH" "$GR"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  plugin-pin"*"9.9.9"* ]]
}

@test "PF-plugin-usage: --plugin-version without a value exits 2" {
  run "$PREFLIGHT" --plugin-version
  [ "$status" -eq 2 ]
}

# ---------------------------------------------------------------------------
# --sample: a project the user did not generate (course zip, vendor sample) must be retargeted first.
# [ev: retro n4-course-signed-jars-gate-fix Δ6] — the shipped trafficLightServer.zip pointed at an
# absent 4.15.4.24 home, carried a <username> placeholder, had no root signing block and shipped build/ dirs.
# ---------------------------------------------------------------------------
# sample_tree <gr> <niagara_home-value> <user-home-value> <signing 1|0> <shipped-build 1|0>
sample_tree() {
  mkdir -p "$1"; cp "$GR/settings.gradle.kts" "$1/"
  printf 'niagara_home=%s\nniagara_user_home=%s\n' "$2" "$3" > "$1/gradle.properties"
  printf 'plugins {\n  id("com.tridium.niagara-signing")\n}\n' > "$1/build.gradle.kts"
  if [ "$4" = 1 ]; then
    printf 'signingServices {\n  signingProfileFactory {\n    allowDefaultProfile.set(false)\n  }\n}\n' >> "$1/build.gradle.kts"
    printf 'niagaraSigning {\n  aliases.set(listOf("CodeSigning"))\n  signingProfileFile.set(file("p.xml"))\n}\n' >> "$1/build.gradle.kts"
  fi
  [ "$5" = 0 ] || mkdir -p "$1/trafficLightServer-rt/build/libs"
}
# sample_nh: a niagara_home whose directory name carries the installed version (4.15.6.20)
sample_nh() { SNH="$TMPDIR_T/niagara-4.15.6.20"; cp -r "$NH" "$SNH"; }

@test "PF-sample-orig: the shipped course sample -> FAIL sample-home, sample-user, sample-sign + WARN sample-build" {
  sample_nh
  sample_tree "$TMPDIR_T/tls" 'c:\\niagara\\niagara-4.15.4.24' 'C:\\Users\\<username>\\Niagara4.15\\<brand>' 0 1
  run "$PREFLIGHT" --sample --jvm-dir "$JVMDIR" "$SNH" "$TMPDIR_T/tls"
  [ "$status" -eq 1 ]
  [ "$(printf '%s\n' "$output" | grep -cE '^FAIL  sample-')" -eq 3 ]
  printf '%s\n' "$output" | grep -E '^FAIL  sample-home ' | grep -qF '/mnt/c/niagara/niagara-4.15.4.24'
  printf '%s\n' "$output" | grep -E '^FAIL  sample-user ' | grep -qF '<username>'
  printf '%s\n' "$output" | grep -qE '^FAIL  sample-sign '
  printf '%s\n' "$output" | grep -E '^WARN  sample-build ' | grep -qF 'trafficLightServer-rt/build'
}

@test "PF-sample-retargeted: the retargeted tree PASSes every sample row, exit 0" {
  sample_nh
  sample_tree "$TMPDIR_T/tls" "$SNH" '/home/u/Niagara4.15/tridium' 1 0
  run "$PREFLIGHT" --sample --jvm-dir "$JVMDIR" "$SNH" "$TMPDIR_T/tls"
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | grep -cE '^PASS  sample-')" -eq 4 ]
  [ "$(printf '%s\n' "$output" | grep -cE '^(FAIL|WARN)  sample-')" -eq 0 ]
}

@test "PF-sample-version: a niagara_home that exists but is another version than the target FAILs sample-home" {
  sample_nh; mkdir -p "$TMPDIR_T/niagara-4.15.4.24"
  sample_tree "$TMPDIR_T/tls" "$TMPDIR_T/niagara-4.15.4.24" '/home/u/Niagara4.15/tridium' 1 0
  run "$PREFLIGHT" --sample --jvm-dir "$JVMDIR" "$SNH" "$TMPDIR_T/tls"
  [ "$status" -eq 1 ]
  printf '%s\n' "$output" | grep -E '^FAIL  sample-home ' | grep -qF '4.15.4.24'
}

@test "PF-sample-off: without --sample no sample row is printed (client trees keep their own gradle.properties)" {
  sample_nh
  sample_tree "$TMPDIR_T/tls" 'c:\\niagara\\niagara-4.15.4.24' 'C:\\Users\\<username>\\x' 0 1
  run "$PREFLIGHT" --jvm-dir "$JVMDIR" "$SNH" "$TMPDIR_T/tls"
  [[ "$output" != *"sample-"* ]]
}

# The review of #269 (R3) found three silent PASS channels in Check 6: sample-sign matched bare
# substrings (a set(true) or commented-out block PASSed), a missing niagara_user_home key PASSed
# sample-user, and a niagara_home with no X.Y.Z skipped the version compare without saying so.
# [ev: retro n4-course-sample-check-precision Δ1] [ev: retro n4-course-sample-check-precision Δ2]
@test "PF-sample-sign-true: allowDefaultProfile.set(true) FAILs sample-sign (the default profile still signs)" {
  sample_nh
  sample_tree "$TMPDIR_T/tls" "$SNH" '/home/u/Niagara4.15/tridium' 1 0
  sed -i 's/allowDefaultProfile.set(false)/allowDefaultProfile.set(true)/' "$TMPDIR_T/tls/build.gradle.kts"
  run "$PREFLIGHT" --sample --jvm-dir "$JVMDIR" "$SNH" "$TMPDIR_T/tls"
  [ "$status" -eq 1 ]
  printf '%s\n' "$output" | grep -E '^FAIL  sample-sign ' | grep -qF 'allowDefaultProfile.set(false)'
}

@test "PF-sample-sign-commented: a //-commented or /* */-commented signing block FAILs sample-sign" {
  sample_nh
  sample_tree "$TMPDIR_T/tls" "$SNH" '/home/u/Niagara4.15/tridium' 1 0
  sed -i '/^plugins/,/^}/!s|^|// |' "$TMPDIR_T/tls/build.gradle.kts"
  run "$PREFLIGHT" --sample --jvm-dir "$JVMDIR" "$SNH" "$TMPDIR_T/tls"
  [ "$status" -eq 1 ] || { echo "line comments: $output"; return 1; }
  printf '%s\n' "$output" | grep -qE '^FAIL  sample-sign '
  sample_tree "$TMPDIR_T/blk" "$SNH" '/home/u/Niagara4.15/tridium' 1 0
  sed -i -e '/^signingServices/i /*' -e '$a */' "$TMPDIR_T/blk/build.gradle.kts"
  run "$PREFLIGHT" --sample --jvm-dir "$JVMDIR" "$SNH" "$TMPDIR_T/blk"
  [ "$status" -eq 1 ] || { echo "block comment: $output"; return 1; }
  printf '%s\n' "$output" | grep -qE '^FAIL  sample-sign '
}

@test "PF-sample-sign-partial: a niagaraSigning block without signingProfileFile FAILs sample-sign, naming it" {
  sample_nh
  sample_tree "$TMPDIR_T/tls" "$SNH" '/home/u/Niagara4.15/tridium' 1 0
  sed -i '/signingProfileFile/d' "$TMPDIR_T/tls/build.gradle.kts"
  run "$PREFLIGHT" --sample --jvm-dir "$JVMDIR" "$SNH" "$TMPDIR_T/tls"
  [ "$status" -eq 1 ]
  printf '%s\n' "$output" | grep -E '^FAIL  sample-sign ' | grep -qF 'signingProfileFile'
}

@test "PF-sample-user-missing: no niagara_user_home key WARNs sample-user instead of PASSing" {
  sample_nh
  sample_tree "$TMPDIR_T/tls" "$SNH" '/home/u/Niagara4.15/tridium' 1 0
  sed -i '/^niagara_user_home=/d' "$TMPDIR_T/tls/gradle.properties"
  run "$PREFLIGHT" --sample --jvm-dir "$JVMDIR" "$SNH" "$TMPDIR_T/tls"
  printf '%s\n' "$output" | grep -E '^WARN  sample-user ' | grep -qF 'no niagara_user_home'
  [[ "$output" != *"PASS  sample-user"* ]]
}

@test "PF-sample-version-unknown: a niagara_home with no X.Y.Z WARNs that the version could not be compared" {
  sample_nh
  sample_tree "$TMPDIR_T/tls" "$NH" '/home/u/Niagara4.15/tridium' 1 0
  run "$PREFLIGHT" --sample --jvm-dir "$JVMDIR" "$SNH" "$TMPDIR_T/tls"
  printf '%s\n' "$output" | grep -E '^WARN  sample-home ' | grep -qF 'version could not be compared'
  [[ "$output" != *"PASS  sample-home"* ]]
}

# The review of the follow-up (R3-001/R4-001, R3-002): comment markers inside Kotlin strings are not
# comments ("**/*.jar" opened a block that never closed, "https://…" cut the line), and the Kotlin-DSL
# assignment `allowDefaultProfile = false` is the same setting as `.set(false)`.
# [ev: retro n4-course-sample-check-precision Δ1]
# sample_kts_prepend <gr> <line>: put <line> right after the plugins block, before the signing block
sample_kts_prepend() {
  L="$2" awk '{ print } !done && /^}/ { print ENVIRON["L"]; done = 1 }' "$1/build.gradle.kts" > "$1/kts.new"
  mv "$1/kts.new" "$1/build.gradle.kts"
}

@test "PF-sample-sign-strings: comment markers inside \"…\" and \"\"\"…\"\"\" strings do not hide the signing block" {
  sample_nh
  sample_tree "$TMPDIR_T/tls" "$SNH" '/home/u/Niagara4.15/tridium' 1 0
  sample_kts_prepend "$TMPDIR_T/tls" 'val jars = fileTree("lib") { include("**/*.jar") }'
  run "$PREFLIGHT" --sample --jvm-dir "$JVMDIR" "$SNH" "$TMPDIR_T/tls"
  printf '%s\n' "$output" | grep -qE '^PASS  sample-sign ' || { echo "glob: $output"; return 1; }
  sample_tree "$TMPDIR_T/url" "$SNH" '/home/u/Niagara4.15/tridium' 1 0
  sample_kts_prepend "$TMPDIR_T/url" 'repositories { maven { url = uri("https://repo.example/m2") } }; val q = "a \"/*\" b"; val r = """/* raw"""'
  run "$PREFLIGHT" --sample --jvm-dir "$JVMDIR" "$SNH" "$TMPDIR_T/url"
  printf '%s\n' "$output" | grep -qE '^PASS  sample-sign ' || { echo "url/escape/raw: $output"; return 1; }
  grep -qF 'https://repo.example' "$TMPDIR_T/url/build.gradle.kts"   # the fixture really carries the markers
}

@test "PF-sample-sign-assign: the Kotlin-DSL form allowDefaultProfile = false PASSes, = true FAILs" {
  sample_nh
  sample_tree "$TMPDIR_T/tls" "$SNH" '/home/u/Niagara4.15/tridium' 1 0
  sed -i 's/allowDefaultProfile.set(false)/allowDefaultProfile = false/' "$TMPDIR_T/tls/build.gradle.kts"
  run "$PREFLIGHT" --sample --jvm-dir "$JVMDIR" "$SNH" "$TMPDIR_T/tls"
  printf '%s\n' "$output" | grep -qE '^PASS  sample-sign ' || { echo "= false: $output"; return 1; }
  sed -i 's/allowDefaultProfile = false/allowDefaultProfile = true/' "$TMPDIR_T/tls/build.gradle.kts"
  run "$PREFLIGHT" --sample --jvm-dir "$JVMDIR" "$SNH" "$TMPDIR_T/tls"
  [ "$status" -eq 1 ]
  printf '%s\n' "$output" | grep -E '^FAIL  sample-sign ' | grep -qF 'allowDefaultProfile'
}

@test "PF-symroot: a symlinked gradle root lists its shipped build/ like find -H, same rows as the directory (issue #329)" {
  # Mutation: PF-symroot -- dropping -H from the sample-build find PASSes sample-build under a symlinked root.
  sample_nh
  sample_tree "$TMPDIR_T/tls" "c:\\\\niagara\\\\niagara-4.15.4.24" "C:\\\\Users\\\\<username>\\\\Niagara4.15\\\\<brand>" 0 1
  assert_symroot_same "$TMPDIR_T/tls" "$PREFLIGHT" --sample --jvm-dir "$JVMDIR" "$SNH" "$TMPDIR_T/tls"
  printf "%s\n" "$output" | grep -E "^WARN  sample-build " | grep -qF "trafficLightServer-rt/build"
}
