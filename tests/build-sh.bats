#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/build.sh — preflight, profile selection, gate call.
# gradle is never run: a fake gradlew records its argv. verify-module.sh is stubbed to isolate build.sh.

load helpers/n4-fixtures

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  # copy build.sh next to a stub gate so the sibling lookup resolves to the stub
  mkdir -p "$TMPDIR_T/kit"; cp "$KIT/toolbelt/build.sh" "$TMPDIR_T/kit/build.sh"
  # build.sh sources lib/fs-type.sh (Δ7/Δ2) relative to its own dir — copy the real lib
  # alongside the copied build.sh so that sibling lookup resolves too.
  mkdir -p "$TMPDIR_T/kit/lib"; cp "$KIT/toolbelt/lib/fs-type.sh" "$TMPDIR_T/kit/lib/fs-type.sh"
  # shellcheck disable=SC2016
  # why: $* and ${FAKE_VERIFY_EXIT} must reach the generated stub unexpanded
  printf '#!/usr/bin/env bash\nprintf "%%s\\n" "$*" > "%s/verify.args"\nexit "${FAKE_VERIFY_EXIT:-0}"\n' "$TMPDIR_T" > "$TMPDIR_T/kit/verify-module.sh"
  chmod +x "$TMPDIR_T/kit/verify-module.sh"
  # Stub preflight.sh + report-module.sh next to build.sh so the $HERE lookups resolve to
  # stubs (isolate build.sh). Default exit 0; override with FAKE_PREFLIGHT_EXIT / FAKE_REPORT_EXIT.
  # shellcheck disable=SC2016
  # why: ${FAKE_PREFLIGHT_EXIT} must reach the generated stub unexpanded
  printf '#!/usr/bin/env bash\nexit "${FAKE_PREFLIGHT_EXIT:-0}"\n' > "$TMPDIR_T/kit/preflight.sh"
  chmod +x "$TMPDIR_T/kit/preflight.sh"
  # shellcheck disable=SC2016
  # why: ${FAKE_REPORT_EXIT} must reach the generated stub unexpanded
  printf '#!/usr/bin/env bash\nexit "${FAKE_REPORT_EXIT:-0}"\n' > "$TMPDIR_T/kit/report-module.sh"
  chmod +x "$TMPDIR_T/kit/report-module.sh"
  B="$TMPDIR_T/kit/build.sh"
  ROOT="$TMPDIR_T/mod"; mkdir -p "$ROOT"
  make_profile "$ROOT" Foo rt 1; make_profile "$ROOT" Foo ux 1; make_profile "$ROOT" Foo wb 0
  make_fake_gradlew "$ROOT"
  make_niagara_home "$TMPDIR_T/nh" 7.6.17
  mkdir -p "$TMPDIR_T/j8"; export JAVA8="$TMPDIR_T/j8"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "B1: stub -wb (gradle file, no sources) is skipped; rt+ux get clean+slotomatic+jar on Java 8 (DashboardPan-wb regression)" {
  run "$B" "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 0 ]
  [[ "$output" == *"skipping Foo-wb"* ]]
  args="$(cat "$TMPDIR_T/gradlew.calls.log")"
  [[ "$args" == *":Foo-rt:clean :Foo-rt:slotomatic :Foo-rt:jar :Foo-ux:clean :Foo-ux:slotomatic :Foo-ux:jar"* ]]
  [[ "$args" != *":Foo-wb:"* ]]
  [[ "$args" == *"-Porg.gradle.java.installations.paths=$TMPDIR_T/j8"* ]]
  [[ "$args" == *"-Pniagara_home=$TMPDIR_T/nh"* ]]
}

@test "B2: no args prints usage and exits 2 (bare \${1:?} abort)" {
  run "$B"
  [ "$status" -eq 2 ]
  [[ "$output" == *"Usage:"* ]]
  [ ! -e "$TMPDIR_T/gradlew.calls.log" ]
}

@test "B3: a target without etc/m2 exits 10 before gradle (plugin-not-found failure)" {
  mkdir -p "$TMPDIR_T/fake"
  run "$B" "$ROOT" Foo "$TMPDIR_T/fake"
  [ "$status" -eq 10 ]
  [[ "$output" == *"not a niagara_home"* ]]
  [ ! -e "$TMPDIR_T/gradlew.calls.log" ]
}

@test "B4: --profiles overrides detection entirely (only rt tasks run)" {
  run "$B" --profiles rt "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 0 ]
  args="$(cat "$TMPDIR_T/gradlew.calls.log")"
  [[ "$args" == *":Foo-rt:jar"* ]] && [[ "$args" != *":Foo-ux:"* ]] && [[ "$args" != *":Foo-wb:"* ]]
}

@test "B5: verify-module.sh runs after gradle with --src, --target-version and the jars; its failure exits 50 (gate skip)" {
  run "$B" --target-version 4.14 --plugin-version 7.6.17 "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 0 ]
  v="$(cat "$TMPDIR_T/verify.args")"
  [[ "$v" == *"--src $ROOT/Foo"* ]] && [[ "$v" == *"--target-version 4.14"* ]]
  [[ "$v" == *"Foo-rt/build/libs/Foo-rt.jar"* ]] && [[ "$v" == *"Foo-ux/build/libs/Foo-ux.jar"* ]]
  [[ "$(cat "$TMPDIR_T/gradlew.calls.log")" == *"-PniagaraPluginVersion=7.6.17"* ]]
  FAKE_VERIFY_EXIT=1 run "$B" "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 50 ]
  [[ "$output" == *"do not deploy"* ]]
}

@test "B6: gradle failure exits 30 and the gate is not invoked" {
  FAKE_GRADLEW_EXIT=1 run "$B" "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 30 ]
  [ ! -e "$TMPDIR_T/verify.args" ]
}

@test "B7: --help prints only the header comment, no code line leaks (usage sed range drift)" {
  run "$B" --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage:"* ]]
  [[ "$output" != *"set -euo pipefail"* ]]
  [[ "$output" != *"usage()"* ]]
}

# ================= Campaign 3 C3-PR2 (RED-first) =================
# Three owed build.sh impls. Each mutation-bites; fake gradlew logs argv (setup), verify-module stubbed.

@test "B8 (lesson B6, module-palette-and-build-target): auto-detect the plugin version from the module's OWN gradle files" {
  # gradle.properties can lie about the target, but the pinned niagara plugin version is the module's own
  # config — build.sh must read it and forward -PniagaraPluginVersion WITHOUT requiring --plugin-version.
  printf 'niagaraPluginVersion=7.6.22\n' > "$ROOT/gradle.properties"
  run "$B" "$ROOT" Foo "$TMPDIR_T/nh"                       # NO --plugin-version
  [ "$status" -eq 0 ]
  [[ "$(cat "$TMPDIR_T/gradlew.calls.log")" == *"-PniagaraPluginVersion=7.6.22"* ]]   # detected, not empty
  # explicit --plugin-version still WINS over the detected one (precedence)
  : > "$TMPDIR_T/gradlew.calls.log"
  run "$B" --plugin-version 7.3.40 "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 0 ]
  [[ "$(cat "$TMPDIR_T/gradlew.calls.log")" == *"-PniagaraPluginVersion=7.3.40"* ]]
  [[ "$(cat "$TMPDIR_T/gradlew.calls.log")" != *"7.6.22"* ]]
}

@test "B9 (lesson B7, dashboardpan-ux-direct-build): walk UP to the gradle root when ROOT has no ./gradlew (nested multi-project)" {
  # A client repo keeps ./gradlew at the repo root; the module dir passed as ROOT has none.
  # Today build.sh exits 10 'no executable ./gradlew'; it must walk up to find gradlew + settings.gradle*.
  proj="$TMPDIR_T/proj"; mkdir -p "$proj"; make_fake_gradlew "$proj"; : > "$proj/settings.gradle.kts"
  root2="$proj/client"; mkdir -p "$root2"; make_profile "$root2" Foo rt 1
  run "$B" "$root2" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 0 ]
  [[ "$(cat "$TMPDIR_T/gradlew.calls.log")" == *":Foo-rt:jar"* ]]   # gradle ran from the resolved root
}

@test "B10 (lesson soft-start): a modules/<jar> clean-lock prints an ACTIONABLE message + distinct exit 31, not a raw exit 30" {
  # A running station locks modules/<mod>.jar so :clean fails with an IOException. build.sh must detect
  # that specific failure and tell the operator how to fix it (free the lock / build/libs / mirror),
  # instead of the generic 'gradle failed' exit 30.
  cat > "$ROOT/gradlew" <<'GRADLEW'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$TMPDIR_T/gradlew.calls.log"
echo "> Task :Foo-rt:clean FAILED" >&2
echo "Unable to delete file '$TMPDIR_T/nh/modules/Foo-rt.jar'" >&2
exit 1
GRADLEW
  chmod +x "$ROOT/gradlew"
  run "$B" "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 31 ]                                   # distinct from generic gradle-fail 30
  { [[ "$output" == *"locked"* ]] || [[ "$output" == *"Unable to delete"* ]]; }
  { [[ "$output" == *"Workbench"* ]] || [[ "$output" == *"mirror"* ]] || [[ "$output" == *"build/libs"* ]]; }
  [ ! -e "$TMPDIR_T/verify.args" ]                       # gate not reached on a failed build
}

# ---------------------------------------------------------------------------
# Campaign 8 PR14 — exit-31 regression pin: a running station holds a lock on modules/<jar>, so gradle
# :clean cannot delete it. build.sh (:82-88) detects the "Unable to delete ...modules/...jar" gradle
# message and exits 31 with a mirror-niagara-home.sh hint (never a bare 30). Green-on-arrival regression
# guard; the named mutation below proves it bites. The lock is SIMULATED via a fake gradlew — the real
# Niagara home is never touched.
@test "BS-lock: :clean blocked by a station lock on modules/<jar> -> exit 31 (not 30)" {
  printf '#!/usr/bin/env bash\necho "Unable to delete %s/modules/Foo-rt.jar"\nexit 1\n' "$TMPDIR_T/nh" > "$ROOT/gradlew"
  chmod +x "$ROOT/gradlew"
  run "$B" "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 31 ]
  [[ "$output" == *"locked"* ]]
  # NAMED MUTATION: delete the `Unable to delete ...` branch in build.sh -> this exits 30, so BS-lock fails.
}

@test "BS-lock-hint: the exit-31 message points the operator at mirror-niagara-home.sh" {
  printf '#!/usr/bin/env bash\necho "Unable to delete %s/modules/Foo-rt.jar"\nexit 1\n' "$TMPDIR_T/nh" > "$ROOT/gradlew"
  chmod +x "$ROOT/gradlew"
  run "$B" "$ROOT" Foo "$TMPDIR_T/nh"
  [[ "$output" == *"mirror-niagara-home.sh"* ]]
}

# BS-gradlew-hint: when no executable gradlew is found, the error message must include
# a "try: chmod +x" hint pointing at the module root, so the operator knows what to fix.
# Named mutation: remove the hint echo line -> BS-gradlew-hint fails (hint absent from output).
@test "BS-gradlew-hint: no-gradlew error prints chmod +x hint (D-build-hardening)" {
  rm "$ROOT/gradlew"    # remove the fake gradlew so the check triggers
  run "$B" "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 10 ]
  [[ "$output" == *"chmod +x"* ]]
}

# BS-plugin-m2-warn: when the gradlePluginVersion in settings.gradle.kts is absent from
# the niagara_home etc/m2 tree, build.sh must print a WARN (never fail the build).
# make_niagara_home creates the m2 dir for a specific plugin version (7.6.17 in setup);
# a settings.gradle.kts with a different version (9.9.9) will not be found.
# Named mutation: remove the plugin m2 WARN block -> BS-plugin-m2-warn fails (WARN absent).
@test "BS-plugin-m2-warn: settings.gradle.kts gradlePluginVersion absent from m2 emits WARN (non-fatal)" {
  # Write a settings.gradle.kts with a version NOT in the m2 (setup installs 7.6.17)
  printf 'val gradlePluginVersion: String = "9.9.9"\n' > "$ROOT/settings.gradle.kts"
  run "$B" "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 0 ]   # WARN only — build is not failed
  [[ "$output" == *"WARN"* ]] && [[ "$output" == *"9.9.9"* ]]
}

# Automation audit PR-B: build.sh auto-chains preflight (start) + report-module (end).
@test "BS-preflight-fail: a preflight FAIL (exit 1) aborts before gradle with exit 10" {
  FAKE_PREFLIGHT_EXIT=1 run "$B" "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 10 ]
  [ ! -e "$TMPDIR_T/gradlew.calls.log" ]   # preflight blocks before gradle
}

@test "BS-preflight-skip: --no-preflight bypasses a failing preflight and the build proceeds" {
  FAKE_PREFLIGHT_EXIT=1 run "$B" --no-preflight "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 0 ]
  [ -e "$TMPDIR_T/gradlew.calls.log" ]      # gradle ran despite the failing preflight stub
}

@test "BS-report-fail: a report-module FAIL (exit 1) after a passing verify gate exits 50 (not hand-off-ready)" {
  FAKE_REPORT_EXIT=1 run "$B" "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 50 ]
  [[ "$output" == *"report-module"* ]]
}

@test "BS-report-skip: --no-report skips the report-module gate; a passing verify exits 0" {
  FAKE_REPORT_EXIT=1 run "$B" --no-report "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 0 ]
}

# ================= Δ7 (retro panccadia-defrost-sequencing-hmi-reload-deltas) =================
# Build-location precheck: WARN (non-fatal) when gradle-root or niagara_home is on a WSL
# 9p/drvfs mount. N4_FSTYPE_MOUNTS_FILE (toolbelt/lib/fs-type.sh) overrides `df -T` with a
# /proc/mounts-formatted fixture so these tests never depend on the host's real mounts.

@test "BS-9p-warn-root: gradle-root on a 9p/drvfs mount -> WARN (non-fatal), build still proceeds" {
  MOUNTS="$TMPDIR_T/mounts-root"
  printf '%s\n' "drvfs $ROOT drvfs rw 0 0" > "$MOUNTS"
  N4_FSTYPE_MOUNTS_FILE="$MOUNTS" run "$B" "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"drvfs"* ]]
  [[ "$output" == *"gradle-root"* ]]
  [ -e "$TMPDIR_T/gradlew.calls.log" ]   # WARN is non-fatal — the build still ran
}

@test "BS-9p-warn-nh: niagara_home on a 9p mount -> WARN naming niagara_home" {
  MOUNTS="$TMPDIR_T/mounts-nh"
  printf '%s\n' "9p $TMPDIR_T/nh 9p rw 0 0" > "$MOUNTS"
  N4_FSTYPE_MOUNTS_FILE="$MOUNTS" run "$B" "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"9p"* ]]
  [[ "$output" == *"niagara_home"* ]]
}

@test "BS-9p-none: no 9p/drvfs mount -> no build-location WARN" {
  MOUNTS="$TMPDIR_T/mounts-none"
  printf '%s\n' "ext4 / ext4 rw 0 0" > "$MOUNTS"
  N4_FSTYPE_MOUNTS_FILE="$MOUNTS" run "$B" "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 0 ]
  [[ "$output" != *"9p/drvfs mount"* ]]
}

# ================= Δ2 (retro panccadia-defrost-sequencing-hmi-reload-deltas) =================
# Deployed-baseline drift gate: a rebuild that changes shipped bytes but leaves the module's
# own vendorVersion unchanged FAILs (exit 51) before the verify gate runs. Software Manager
# compares versions, not bytes, and would otherwise report "Up to Date" and skip the fix
# (the exact CompPan 2.1.0 leon-vs-leon2 trap this retro documents).
#
# _mkjar_ver <jar> <vendorVersion> <payload> — a minimal N4-module-shaped jar with the
# MODULE's OWN vendorVersion on the <module ...> root tag itself (not the <dependency
# name="baja" vendorVersion="..."/> floor) plus one payload file whose content varies the
# scenario. buildMillis is randomized every call, exactly like a real rebuild, to prove the
# gate ignores META-INF noise and only reacts to an ACTUAL shipped-bytes change.
_mkjar_ver() {
  local jar="$1" ver="$2" payload="$3" d
  d="$(mktemp -d)"
  mkdir -p "$d/META-INF"
  printf '<module name="X" vendor="Angeles" vendorVersion="%s" buildMillis="%s%s">\n<dependencies/>\n<types/>\n</module>\n' \
    "$ver" "$RANDOM" "$RANDOM" > "$d/META-INF/module.xml"
  printf '%s' "$payload" > "$d/payload.txt"
  mkdir -p "$(dirname "$jar")"
  (cd "$d" && zip -q -r "$jar" .)
  rm -rf "$d"
}

@test "BS-drift-fail: rebuild changes shipped bytes but vendorVersion is unchanged -> exit 51 before verify-module runs" {
  mkdir -p "$TMPDIR_T/nh/modules"
  _mkjar_ver "$TMPDIR_T/nh/modules/Foo-rt.jar" "1.0.0" "old-behavior"
  mkdir -p "$ROOT/Foo/Foo-rt/build/libs"
  _mkjar_ver "$ROOT/Foo/Foo-rt/build/libs/Foo-rt.jar" "1.0.0" "NEW-behavior-fix"
  run "$B" "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 51 ]
  [[ "$output" == *"vendorVersion"* ]]
  [[ "$output" == *"Up to Date"* ]]
  [ ! -e "$TMPDIR_T/verify.args" ]   # the verify gate is never reached
}

@test "BS-drift-pass: rebuild changes shipped bytes AND vendorVersion is bumped -> no drift FAIL, verify gate reached" {
  mkdir -p "$TMPDIR_T/nh/modules"
  _mkjar_ver "$TMPDIR_T/nh/modules/Foo-rt.jar" "1.0.0" "old-behavior"
  mkdir -p "$ROOT/Foo/Foo-rt/build/libs"
  _mkjar_ver "$ROOT/Foo/Foo-rt/build/libs/Foo-rt.jar" "1.0.1" "NEW-behavior-fix"
  run "$B" "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 0 ]
  [ -e "$TMPDIR_T/verify.args" ]
}

@test "BS-drift-identical-content: rebuild with IDENTICAL payload (only buildMillis differs) -> no drift FAIL (false-positive guard)" {
  mkdir -p "$TMPDIR_T/nh/modules"
  _mkjar_ver "$TMPDIR_T/nh/modules/Foo-rt.jar" "1.0.0" "same-behavior"
  mkdir -p "$ROOT/Foo/Foo-rt/build/libs"
  _mkjar_ver "$ROOT/Foo/Foo-rt/build/libs/Foo-rt.jar" "1.0.0" "same-behavior"
  run "$B" "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 0 ]
  [ -e "$TMPDIR_T/verify.args" ]
}

@test "BS-drift-no-baseline: no jar under modules/ yet (first deploy) -> no drift check, build proceeds" {
  mkdir -p "$ROOT/Foo/Foo-rt/build/libs"
  _mkjar_ver "$ROOT/Foo/Foo-rt/build/libs/Foo-rt.jar" "1.0.0" "first-ever-build"
  run "$B" "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 0 ]
  [ -e "$TMPDIR_T/verify.args" ]
}

@test "BS-drift-skip-flag: --no-drift-check bypasses a real drift condition" {
  mkdir -p "$TMPDIR_T/nh/modules"
  _mkjar_ver "$TMPDIR_T/nh/modules/Foo-rt.jar" "1.0.0" "old-behavior"
  mkdir -p "$ROOT/Foo/Foo-rt/build/libs"
  _mkjar_ver "$ROOT/Foo/Foo-rt/build/libs/Foo-rt.jar" "1.0.0" "NEW-behavior-fix"
  run "$B" --no-drift-check "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 0 ]
  [ -e "$TMPDIR_T/verify.args" ]
}

# ================= R4-drift-gate-not-retry-safe (RDD resilience correction) =================
# The stock make_fake_gradlew stub above never touches modules/, so it cannot exercise the real
# failure mode: gradle's :jar task installs the new jar into modules/ as its OWN last step (see
# build.sh's Δ2 comment), BEFORE the drift FAIL below fires. These two tests use a fake gradlew
# that also performs that install, so a rebuild-then-restore lifecycle can actually be observed.
_mk_gradlew_installs_jar() {
  # Simulates gradle's :jar step installing the already-built Foo-rt.jar (planted by the test via
  # _mkjar_ver) into niagara_home/modules/ as its last step.
  cat > "$ROOT/gradlew" <<GRADLEW
#!/usr/bin/env bash
printf '%s\n' "\$*" >> "$TMPDIR_T/gradlew.calls.log"
cp -p "$ROOT/Foo/Foo-rt/build/libs/Foo-rt.jar" "$TMPDIR_T/nh/modules/Foo-rt.jar"
exit 0
GRADLEW
  chmod +x "$ROOT/gradlew"
}

@test "BS-drift-restore: a drift FAIL restores modules/<jar> byte-identical to the pre-build baseline" {
  mkdir -p "$TMPDIR_T/nh/modules"
  _mkjar_ver "$TMPDIR_T/nh/modules/Foo-rt.jar" "1.0.0" "old-behavior"
  cp "$TMPDIR_T/nh/modules/Foo-rt.jar" "$TMPDIR_T/pre-build-baseline.jar"
  mkdir -p "$ROOT/Foo/Foo-rt/build/libs"
  _mkjar_ver "$ROOT/Foo/Foo-rt/build/libs/Foo-rt.jar" "1.0.0" "NEW-behavior-fix"
  _mk_gradlew_installs_jar
  run "$B" --profiles rt "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 51 ]
  [[ "$output" == *"Restored"* ]]
  cmp -s "$TMPDIR_T/nh/modules/Foo-rt.jar" "$TMPDIR_T/pre-build-baseline.jar"
}

@test "BS-drift-retry-safe: a SECOND run after a drift FAIL still FAILs (R4-drift-gate-not-retry-safe)" {
  mkdir -p "$TMPDIR_T/nh/modules"
  _mkjar_ver "$TMPDIR_T/nh/modules/Foo-rt.jar" "1.0.0" "old-behavior"
  mkdir -p "$ROOT/Foo/Foo-rt/build/libs"
  _mkjar_ver "$ROOT/Foo/Foo-rt/build/libs/Foo-rt.jar" "1.0.0" "NEW-behavior-fix"
  _mk_gradlew_installs_jar
  run "$B" --profiles rt "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 51 ]
  # SECOND run, same inputs, same not-actually-rebuilt build/libs jar (a deterministic rebuild
  # reproduces identical non-META-INF bytes) — without the restore above, part 1 would snapshot
  # the NEW jar gradle just installed as ITS OWN baseline and this would silently exit 0.
  run "$B" --profiles rt "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 51 ]
  [[ "$output" == *"vendorVersion"* ]]
}
