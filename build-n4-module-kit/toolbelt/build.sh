#!/usr/bin/env bash
# build.sh — full automatic chain: preflight → gradle (Java 8 + clean + slotomatic + jar) → verify gate → report-module.
# A `gradle :jar` with the default JDK is NOT a build (wrong bytecode major, slotomatic skipped).
# Deploying to a station is ng-deploy.sh's job (backup -> build -> copy -> type-count verify).
#
# Usage: build.sh [--profiles rt,ux,wb] [--target-version X.Y] [--plugin-version V] [--ui-profile P] [--legacy]
#                 [--no-preflight] [--no-report] [--no-drift-check] <module-root> <MOD> [niagara_home]
#   <module-root>   the dir holding ./gradlew and <MOD>/<MOD>-{rt,ux,wb}/ (the GROUP dir). When neither it nor
#                   an ancestor holds ./gradlew (the REPO root was passed), the error lists every gradle root
#                   found up to 3 levels BELOW it — descend into one. [ev: retro comppan-fase2-amps-alarms Δ2]
#   niagara_home    arg 3, else $niagara_home. On WSL use the /mnt/c/... mount or a mirror (mirror-niagara-home.sh).
#   --plugin-version / $NIAGARA_PLUGIN_VERSION   forwarded as -PniagaraPluginVersion (each install ships ONE
#                   niagara-module plugin: 4.13.2 -> 7.3.40, 4.14 -> 7.6.17, 4.15.3 -> 7.6.22); also forwarded to
#                   preflight.sh --plugin-version so its plugin-pin check tests the version gradle will actually
#                   use, not a drifted settings.gradle.kts default [ev: retro continuous-fan-post-defrost-delay Δ1]
#   --ui-profile P  hmi|lan|both|unknown — the module's BUILD-STATE ui_profile, forwarded to report-module.sh
#                   --profile (selects rc-scan's browser floor); --legacy forwards report-module.sh --legacy
#   --no-preflight  skip environment preflight (useful for inner rebuild loops when env is known-good)
#   --no-report     skip report-module punch-list at the end (useful for quick inner rebuild loops)
#   --first-deploy  forwarded to report-module.sh: a never-deployed module has no baseline for schema-risk
#                   (otherwise a missing <artifact>/.deploy-baseline is INCOMPLETE, exit 53) [ev: audit-2026-10-03 A2]
#   --no-drift-check  skip the deployed-baseline drift gate below (Δ2) — fix the version bump, don't skip, unless
#                   you have a real reason (e.g. this niagara_home was never the deploy target)
#   $JAVA8          JDK 8 path (default /usr/lib/jvm/java-8-openjdk-amd64)
# Profiles: by default every <MOD>-<p> dir that has a gradle file AND sources under src/ is built; a scaffold
#   (gradle file, no sources) is reported "skipped". --profiles replaces auto-detection entirely.
# After gradle, verify-module.sh (same dir) runs on every produced jar with --src <module-root>/<MOD>.
# Build-location precheck (Δ7): WARNs (non-fatal) when <module-root>'s gradle root or niagara_home sits on a
#   WSL 9p/drvfs mount (/mnt/<drive>) — Gradle's per-file work is far slower there than on ext4.
# Deployed-baseline drift gate (Δ2): before gradle's :jar task overwrites <niagara_home>/modules/<jar>, this
#   snapshots what is CURRENTLY installed there; after the build, if the new jar's shipped bytes differ from
#   that snapshot but the module's own vendorVersion did not change, the build FAILs (exit 51) — Software
#   Manager compares versions, not bytes, and would silently report "Up to Date" and skip installing the fix.
#   A jar the gate cannot list (unzip missing/failing) or a changed jar whose own vendorVersion cannot be
#   read FAILs closed with exit 52 ("cannot verify"), never a silent pass.
#   Retry-safe: gradle's :jar step already overwrote the deployed jar by the time this FAILs, so the gate
#   backs up the pre-build jar and restores it into modules/ on FAIL — a bare re-run then still sees the
#   OLD baseline and catches the same drift again, instead of silently "Up to Date"-passing on the retry.
# Post-jar copy lock (exit 32): the plugin's last step copies the jar into <niagara_home>/modules; a Windows
#   process (Workbench, station) holding the old jar open on /mnt/c makes it fail with FileAlreadyExistsException.
#   build.sh names the build/libs jar(s) rebuilt THIS run as the artifact of record (a stale one is flagged
#   "not rebuilt"). [ev: retro panccadia-restart-seq-comp-lockout-hours Δ3]
# Independent gradle groups (different gradle roots) may be built CONCURRENTLY: one build.sh per group in the
#   background, then wait — never two runs on the SAME gradle root (BUILD-LOOP.md §4).
#   [ev: retro change-tier-time-budgets Δ5]
# Mirror user.home isolation: when niagara_home holds the .niagara-mirror marker (mirror-niagara-home.sh), gradle
#   runs with JAVA_TOOL_OPTIONS += -Duser.home=<mirror>/.mirror-user-home, GRADLE_USER_HOME kept (default
#   ~/.gradle) and --no-daemon, so the signing plugin cannot read the real ~/.tridium keystore.
#   [ev: retro r8-mirror-userhome Δ1]
# Exit: 0 chain passed · 2 usage · 10 environment (preflight FAIL, no JDK 8, not a niagara_home, no profile) ·
#   30 gradle failed · 31 :clean locked · 32 post-jar modules/ copy locked · 50 gate or report-module FAIL ·
#   51 deployed-baseline drift (Δ2) · 52 drift gate cannot verify · 53 report-module INCOMPLETE (a floor check SKIPped)
set -euo pipefail

# usage: print the leading comment block (line 2 up to the first non-comment line), never code.
usage() { awk 'NR == 1 { next } /^#/ { sub(/^# ?/, ""); print; next } { exit }' "$0"; }
PROFILES=""; TARGET=""; PLUGIN="${NIAGARA_PLUGIN_VERSION:-}"; UI_PROFILE=""; LEGACY=0
SKIP_PREFLIGHT=0; SKIP_REPORT=0; SKIP_DRIFT_CHECK=0; FIRST_DEPLOY=0
while [ $# -gt 0 ]; do
  case "$1" in
    --profiles)       [ $# -ge 2 ] || { usage >&2; exit 2; }; PROFILES="$2"; shift 2 ;;
    --target-version) [ $# -ge 2 ] || { usage >&2; exit 2; }; TARGET="$2"; shift 2 ;;
    --plugin-version) [ $# -ge 2 ] || { usage >&2; exit 2; }; PLUGIN="$2"; shift 2 ;;
    --ui-profile)
      [ $# -ge 2 ] || { usage >&2; exit 2; }
      case "$2" in
        hmi|lan|both|unknown) UI_PROFILE="$2" ;;
        *) echo "build.sh: --ui-profile must be hmi|lan|both|unknown (got: $2)" >&2; exit 2 ;;
      esac
      shift 2 ;;
    --legacy)       LEGACY=1;         shift ;;
    --no-preflight) SKIP_PREFLIGHT=1; shift ;;
    --no-report)    SKIP_REPORT=1;    shift ;;
    --no-drift-check) SKIP_DRIFT_CHECK=1; shift ;;
    --first-deploy) FIRST_DEPLOY=1; shift ;;
    -h|--help) usage; exit 0 ;;
    -*) echo "build.sh: unknown flag $1" >&2; usage >&2; exit 2 ;;
    *) break ;;
  esac
done
if [ $# -lt 2 ] || [ $# -gt 3 ]; then usage >&2; exit 2; fi
ROOT="$1"; MOD="$2"
NIAGARA_HOME="${3:-${niagara_home:-}}"
J8="${JAVA8:-/usr/lib/jvm/java-8-openjdk-amd64}"
HERE="$(CDPATH='' cd -- "$(dirname "$0")" >/dev/null && pwd)"
# shellcheck disable=SC1091
. "$HERE/lib/fs-type.sh"

[ -d "$ROOT" ] || { echo "build.sh: module root not found: $ROOT" >&2; exit 10; }
# B7: gradlew may live at an ANCESTOR (client multi-project layout — the module dir is passed as ROOT). Walk up to find it.
# retro deltas 1924: resolve ROOT to an absolute path first — dirname of a relative "." is "." (endless loop) — and fail
# fast when a gradlew exists but is not executable (a worktree checkout of a 100644 gradlew): do not walk past it.
ROOT="$(CDPATH='' cd -- "$ROOT" >/dev/null && pwd)" || { echo "build.sh: cannot resolve module root: $ROOT" >&2; exit 10; }
GRADLE_ROOT="$ROOT"
while [ -n "$GRADLE_ROOT" ] && [ "$GRADLE_ROOT" != "/" ] && [ ! -x "$GRADLE_ROOT/gradlew" ]; do
    if [ -f "$GRADLE_ROOT/gradlew" ]; then
        echo "build.sh: $GRADLE_ROOT/gradlew exists but is not executable (a fresh VCS worktree of a repo that stores it 100644)" >&2
        echo "  fix: chmod +x $GRADLE_ROOT/gradlew" >&2
        exit 10
    fi
    GRADLE_ROOT="$(dirname "$GRADLE_ROOT")"
done
[ -x "$GRADLE_ROOT/gradlew" ] || {
    echo "build.sh: no executable ./gradlew in $ROOT or any ancestor (the module needs the gradle wrapper)" >&2
    # comppan-fase2-amps-alarms Δ2: the opposite mistake — the REPO root was passed, one or more levels
    # ABOVE the gradle root (a client repo nests several gradle groups as siblings). List them.
    _BELOW="$(find "$ROOT" -mindepth 2 -maxdepth 3 \( -type d \( -name '.*' -o -name node_modules -o -name build \) -prune \) \
        -o \( -type f -name gradlew -print \) 2>/dev/null | LC_ALL=C sort || true)"
    if [ -n "$_BELOW" ]; then
        echo "  found gradle root(s) BELOW $ROOT — pass the GROUP dir that holds <MOD>/, not the repo root:" >&2
        while IFS= read -r _gw; do echo "    $(dirname "$_gw")" >&2; done <<< "$_BELOW"
    fi
    echo "  try: chmod +x $ROOT/gradlew" >&2
    exit 10
}
[ -d "$J8" ] || { echo "build.sh: Java 8 not found at $J8 — check 'ls /usr/lib/jvm' or set JAVA8" >&2; exit 10; }
[ -n "$NIAGARA_HOME" ] || { echo "build.sh: pass niagara_home (arg 3) or export niagara_home" >&2; exit 10; }
[ -d "$NIAGARA_HOME/etc/m2/repository" ] || { echo "build.sh: not a niagara_home (no etc/m2/repository): $NIAGARA_HOME" >&2; exit 10; }
[ -x "$HERE/verify-module.sh" ] || { echo "build.sh: gate not found next to this script: $HERE/verify-module.sh" >&2; exit 10; }

# ---------------------------------------------------------------------------
# Build-location precheck (Δ7): WARN (non-fatal — the build still runs) when the
# gradle root or niagara_home sits on a WSL 9p/drvfs mount (/mnt/<drive>).
# Gradle's per-file stat/hash work multiplies a 9p mount's per-file latency —
# a measured file walk was 0.200s on 9p vs 0.020s on ext4 (10x); a 3-5s build
# on ext4 can balloon into minutes on 9p. [ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ7]
# ---------------------------------------------------------------------------
for _bl_label_path in "gradle-root:$GRADLE_ROOT" "niagara_home:$NIAGARA_HOME"; do
  _bl_label="${_bl_label_path%%:*}"; _bl_path="${_bl_label_path#*:}"
  if is_9p_or_drvfs "$_bl_path"; then
    echo "build.sh: WARN — $_bl_label ($_bl_path) is on a $(fs_type_for "$_bl_path") mount (WSL /mnt/<drive>) — builds run far slower than ext4." >&2
    echo "  Clone the source tree under an ext4 path (e.g. ~/modulos_niagara_n4/Cliente/...) and mirror niagara_home with toolbelt/mirror-niagara-home.sh (see build-verify.md)." >&2
  fi
done

if [ "$SKIP_PREFLIGHT" -eq 0 ]; then
  echo "==> preflight"
  PFARGS=(); [ -z "$PLUGIN" ] || PFARGS+=(--plugin-version "$PLUGIN")
  if "$HERE/preflight.sh" "${PFARGS[@]}" "$NIAGARA_HOME" "$GRADLE_ROOT"; then
    :
  else
    _PF=$?
    [ "$_PF" -eq 1 ] && echo "build.sh: preflight FAILed — fix the environment (or --no-preflight to skip)" >&2
    exit 10
  fi
fi

# D: plugin-m2-warn — WARN (non-fatal) when the gradlePluginVersion declared in
# settings.gradle.kts is absent from <niagara_home>/etc/m2.  A mismatch causes Gradle
# to fail looking up the plugin when retargeting to a different Niagara installation.
# This is advisory only and never blocks the build. (D-build-hardening)
_SETTINGS_KTS="$GRADLE_ROOT/settings.gradle.kts"
if [ -f "$_SETTINGS_KTS" ]; then
    _GPLUG_VER=$(LC_ALL=C grep -oE 'gradlePluginVersion[[:space:]]*:[[:space:]]*String[[:space:]]*=[[:space:]]*"[0-9][^"]*"' \
        "$_SETTINGS_KTS" 2>/dev/null | grep -oE '"[0-9][^"]*"' | tr -d '"' | head -1 || true)
    # an explicit override (flag/env) is the version gradle will use — check that one instead
    [ -z "$PLUGIN" ] || _GPLUG_VER="$PLUGIN"
    if [ -n "$_GPLUG_VER" ]; then
        _PLUG_IN_M2=$(find "$NIAGARA_HOME/etc/m2/repository" -maxdepth 8 -type d -name "$_GPLUG_VER" 2>/dev/null | head -1 || true)
        if [ -z "$_PLUG_IN_M2" ]; then
            echo "build.sh: WARN — gradlePluginVersion=$_GPLUG_VER not found under $NIAGARA_HOME/etc/m2/repository" >&2
            echo "  SDK/plugin mismatch: the module was pinned for a different Niagara installation." >&2
            echo "  If the build fails to resolve the niagara-module plugin, point niagara_home at the matching installation." >&2
        fi
    fi
fi

# profile selection: a gradle file alone is not a buildable profile (the DashboardPan-wb scaffold has one)
has_gradle()  { [ -f "$1/build.gradle" ] || [ -f "$1/build.gradle.kts" ] || compgen -G "$1/*.gradle.kts" >/dev/null; }
has_sources() { [ -d "$1/src" ] && find "$1/src" -type f \( -name '*.java' -o -name '*.js' -o -name '*.html' \) -print -quit | grep -q .; }
SEL=()
if [ -n "$PROFILES" ]; then
  IFS=, read -r -a SEL <<< "$PROFILES"
else
  for p in rt ux wb; do
    d="$ROOT/$MOD/$MOD-$p"; [ -d "$d" ] || continue
    if has_gradle "$d" && has_sources "$d"; then SEL+=("$p")
    else echo "==> skipping $MOD-$p: scaffold without sources (pass --profiles to force)"; fi
  done
fi
[ ${#SEL[@]} -gt 0 ] || { echo "build.sh: no buildable profile under $ROOT/$MOD/$MOD-{rt,ux,wb}" >&2; exit 10; }
TASKS=(); for p in "${SEL[@]}"; do
  [ -d "$ROOT/$MOD/$MOD-$p" ] || { echo "build.sh: profile dir missing: $ROOT/$MOD/$MOD-$p" >&2; exit 10; }
  TASKS+=(":$MOD-$p:clean" ":$MOD-$p:slotomatic" ":$MOD-$p:jar")
done
# B6: when no plugin version was given (flag/env), auto-detect the module's OWN pinned niagara plugin version.
if [ -z "$PLUGIN" ]; then
  if [ -f "$ROOT/gradle.properties" ]; then
    PLUGIN=$(grep -E '^[[:space:]]*niagaraPluginVersion[[:space:]]*=' "$ROOT/gradle.properties" | head -1 | sed -E 's/.*=[[:space:]]*//; s/[[:space:]]+$//' || true)
  fi
  if [ -z "$PLUGIN" ] && [ -f "$ROOT/settings.gradle.kts" ]; then
    PLUGIN=$(grep -oE 'niagaraPluginVersion[^)]*getOrElse\("[0-9][0-9.]*"\)' "$ROOT/settings.gradle.kts" | grep -oE '[0-9][0-9.]+' | head -1 || true)
  fi
  [ -z "$PLUGIN" ] || echo "==> detected niagaraPluginVersion=$PLUGIN from the module's gradle config"
fi
GARGS=(-Pniagara_home="$NIAGARA_HOME" -Porg.gradle.java.installations.paths="$J8")
[ -z "$PLUGIN" ] || GARGS+=(-PniagaraPluginVersion="$PLUGIN")

# Mirror user.home isolation [ev: retro r8-mirror-userhome Δ1]: org.gradle.jvmargs=-Duser.home never reaches an
# already-running daemon, so the signing plugin's default profile (user.home/.tridium/security) would read the
# REAL keystore. JAVA_TOOL_OPTIONS reaches every JVM gradle starts; --no-daemon stops reuse of a stale daemon.
# GRADLE_USER_HOME is pinned first, because gradle derives it from user.home and offline resolution needs it.
if [ -f "$NIAGARA_HOME/.niagara-mirror" ]; then
  _MIRROR_UH="$NIAGARA_HOME/.mirror-user-home"
  mkdir -p "$_MIRROR_UH"
  export GRADLE_USER_HOME="${GRADLE_USER_HOME:-$HOME/.gradle}"
  export JAVA_TOOL_OPTIONS="${JAVA_TOOL_OPTIONS:+$JAVA_TOOL_OPTIONS }-Duser.home=$_MIRROR_UH"
  GARGS+=(--no-daemon)
  echo "==> INFO: mirror niagara_home: user.home isolated to $_MIRROR_UH (GRADLE_USER_HOME=$GRADLE_USER_HOME, --no-daemon) so the signing plugin cannot read the real ~/.tridium keystore"
fi

# ---------------------------------------------------------------------------
# Deployed-baseline drift gate (Δ2), part 1 — pre-build snapshot + backup.
# Gradle's :jar task auto-installs the new jar into <niagara_home>/modules/ as
# its LAST step, so "what is currently deployed" must be captured BEFORE
# gradle runs — by the time gradle exits, modules/ already holds the NEW
# bytes and a post-build-only comparison would always see "no drift". A full
# copy of the pre-build jar (not just its hash) is kept too, so a drift FAIL
# below can restore modules/ to this exact baseline — without that restore, a
# bare re-run would snapshot the NEW jar gradle just installed as its OWN
# baseline, a deterministic rebuild would match it byte-for-byte, and the
# gate would silently pass on the retry (R4-drift-gate-not-retry-safe).
# [ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ2]
# ---------------------------------------------------------------------------
_content_hash() {
  # Deterministic hash of every non-META-INF jar entry's CONTENT (sorted by
  # name). META-INF/* (module.xml buildMillis, MANIFEST.MF, signatures) is
  # EXCLUDED on purpose: those bytes change on every rebuild even when nothing
  # actually shipped changed, which would otherwise make this gate false-fire
  # on every rebuild regardless of source changes.
  # Fails (non-zero, no output) when the jar cannot be listed (unzip missing or failing, an
  # unreadable or empty jar): two empty listings would hash EQUAL and the gate would silently
  # pass, so an unlistable jar is "cannot verify", never "unchanged" (fail closed).
  local jar="$1" list
  list="$(unzip -Z1 "$jar" 2>/dev/null)" || return 1
  [ -n "$list" ] || return 1
  # awk, not `grep -v`: a jar with only META-INF entries has no payload line, and grep's exit 1 would fail the
  # pipeline under pipefail and report an unchanged resource-only jar as "cannot verify" (#226).
  { printf '%s\n' "$list" | awk '!/^META-INF\//' | LC_ALL=C sort \
      | while IFS= read -r _entry; do unzip -p "$jar" "$_entry" 2>/dev/null; done
  } | sha256sum | awk '{print $1}'
}
_module_own_version() {
  # The MODULE's own vendorVersion lives on the <module ...> root tag itself —
  # NOT the <dependency name="baja" vendorVersion="..."/> floor, a different
  # number entirely (METHODOLOGY.md "two version concepts").
  { unzip -p "$1" META-INF/module.xml 2>/dev/null \
      | grep -oE '<module [^>]*vendorVersion="[0-9]+\.[0-9]+\.[0-9]+[^"]*"' \
      | head -1 | sed -E 's/.*vendorVersion="([^"]*)".*/\1/'
  } || true
}
# _restore_baseline <profile> <ok-suffix> — put the part-1 backup back into modules/ and say so. A failed copy
# prints cp's own reason and what modules/<jar> really holds now (the pre-build jar, or something else such as
# the new jar gradle's :jar step installed), because a bare retry would snapshot a new jar as its own baseline.
_restore_baseline() {
  local p="$1" ok="$2" dst="$NIAGARA_HOME/modules/$MOD-$1.jar" err
  if err="$(cp -p "$DRIFT_DIR/$p.old.jar" "$dst" 2>&1)"; then
    echo "  Restored the previously-deployed $MOD-$p.jar into $NIAGARA_HOME/modules$ok" >&2
  elif cmp -s "$DRIFT_DIR/$p.old.jar" "$dst"; then
    echo "  FAILED TO RESTORE $MOD-$p.jar (${err:-cp failed}); $NIAGARA_HOME/modules still holds the pre-build jar, so a retry is still safe." >&2
  else
    echo "  FAILED TO RESTORE the previously-deployed $MOD-$p.jar (${err:-cp failed}) — $NIAGARA_HOME/modules no longer holds it (likely the NEW jar); restore it manually before retrying." >&2
  fi
}
DRIFT_DIR=""
if [ "$SKIP_DRIFT_CHECK" -eq 0 ]; then
  DRIFT_DIR="$(mktemp -d)"
  trap 'rm -rf "$DRIFT_DIR"' EXIT
  for p in "${SEL[@]}"; do
    _old="$NIAGARA_HOME/modules/$MOD-$p.jar"
    [ -f "$_old" ] || continue
    _content_hash "$_old" > "$DRIFT_DIR/$p.old.hash" || : > "$DRIFT_DIR/$p.old.hash"
    _module_own_version "$_old" > "$DRIFT_DIR/$p.old.ver"
    cp -p "$_old" "$DRIFT_DIR/$p.old.jar"
  done
fi

echo "==> build (Java 8 + slotomatic): ${TASKS[*]}"
GLOG="$(mktemp)"
# Start-of-build marker: a build/libs jar NEWER than this was produced by this run (exit-32 report below).
GSTAMP="$(mktemp)"
if ( cd "$GRADLE_ROOT" && ./gradlew "${TASKS[@]}" "${GARGS[@]}" ) 2>&1 | tee "$GLOG"; then
  rm -f "$GLOG" "$GSTAMP"
else
  # restart-seq-comp-lockout-hours Δ3: the plugin's post-jar copy into <niagara_home>/modules failed because a
  # (Windows-side) process holds the old jar open. The jar task itself already assembled build/libs/<jar>.
  # Route-independent: the Windows gradlew.bat route prints backslash paths and the Kotlin message
  # ("Tried to overwrite the destination, but failed to delete it"). [ev: retro n4-course-signed-jars-gate-fix Δ5]
  if grep -qE 'FileAlreadyExistsException.*modules[/\\].*\.jar' "$GLOG"; then
    echo "build.sh: the plugin's post-jar copy into $NIAGARA_HOME/modules failed — a local process (Workbench," >&2
    echo "  a station) holds the old jar open (FileAlreadyExistsException). The build/libs jar is the artifact of record:" >&2
    for p in "${SEL[@]}"; do
      _lib="$ROOT/$MOD/$MOD-$p/build/libs/$MOD-$p.jar"
      if [ -n "$(find "$_lib" -newer "$GSTAMP" 2>/dev/null)" ]; then
        echo "    $_lib (built this run)" >&2
      else
        echo "    $_lib — not rebuilt this run (missing or stale): gradle stopped before this profile" >&2
      fi
    done
    echo "  Close the process holding modules/<jar> and re-run to refresh the modules/ copy — or use the build/libs" >&2
    echo "  jar directly (the verify gate was NOT run: run verify-module.sh on it before any deploy)." >&2
    rm -f "$GLOG" "$GSTAMP"; exit 32
  fi
  # soft-start: a running station LOCKS modules/<mod>.jar so :clean fails — tell the operator how to fix it.
  if grep -qE 'Unable to delete .*modules/.*\.jar' "$GLOG"; then
    echo "build.sh: :clean could not delete a modules/<jar> — a running station has it locked." >&2
    echo "  Free the lock first: close Workbench, or stop the station, then build directly; or build against a" >&2
    echo "  mirror (mirror-niagara-home.sh); or just use the already-assembled build/libs jar (the modules/ copy" >&2
    echo "  is irrelevant when you are not deploying to that supervisor)." >&2
    rm -f "$GLOG" "$GSTAMP"; exit 31
  fi
  # slotomatic-authoring Δ1 (K15): slot-o-matic logs "Could not process <file>: …" as a WARNING and its task
  # succeeds, so the failure surfaces later in compileJava ("… have you run slot-o-matic?"). Name the skipped file.
  # The capture ends at ".java" so a Windows drive letter (C:\...) does not cut the path short.
  if grep -q 'have you run slot-o-matic' "$GLOG"; then
    _SKIPPED=$(sed -n 's/.*Could not process \(.*\.java\): .*/  slot-o-matic skipped \1/p' "$GLOG" | sort -u)
    echo "build.sh: compileJava found slots slot-o-matic never generated." >&2
    if [ -n "$_SKIPPED" ]; then
      printf '%s\n' "$_SKIPPED" >&2
      echo "  A file with empty BEGIN/END BAJA AUTO GENERATED markers is skipped (METHODOLOGY K15): delete the" >&2
      echo "  markers and rerun." >&2
    else
      echo "  Rerun :<part>:slotomatic --info and read its output for 'Could not process' (METHODOLOGY K15)." >&2
    fi
  fi
  echo "build.sh: gradle failed" >&2; rm -f "$GLOG" "$GSTAMP"; exit 30
fi

# ---------------------------------------------------------------------------
# Deployed-baseline drift gate (Δ2), part 2 — post-build compare + restore.
# If the new jar's shipped-bytes content hash differs from the pre-build
# snapshot but the module's own vendorVersion is UNCHANGED, FAIL: Software
# Manager compares versions, not bytes, and would report "Up to Date" and
# silently skip installing this jar (the exact panccadia CompPan 2.1.0 leon
# vs leon2 trap). On FAIL, the backed-up pre-build jar (part 1) is restored
# into modules/ so the deployed baseline stays truthful and a bare re-run
# stays retry-safe (R4-drift-gate-not-retry-safe): it snapshots the SAME old
# baseline again and catches the drift a second time instead of passing.
# [ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ2]
# ---------------------------------------------------------------------------
if [ "$SKIP_DRIFT_CHECK" -eq 0 ]; then
  DRIFT_FAIL=0; DRIFT_UNVERIFIED=0
  for p in "${SEL[@]}"; do
    [ -f "$DRIFT_DIR/$p.old.hash" ] || continue
    _new="$ROOT/$MOD/$MOD-$p/build/libs/$MOD-$p.jar"
    [ -f "$_new" ] || continue
    _new_hash="$(_content_hash "$_new")" || _new_hash=""
    _old_hash="$(cat "$DRIFT_DIR/$p.old.hash")"
    _new_ver="$(_module_own_version "$_new")"
    _old_ver="$(cat "$DRIFT_DIR/$p.old.ver" 2>/dev/null || true)"
    # Fail closed: a jar the gate cannot list, or a changed jar whose own vendorVersion cannot be
    # read on either side, is UNVERIFIABLE (exit 52) -- never "unchanged" and never "bumped".
    _why=""
    if [ -z "$_new_hash" ] || [ -z "$_old_hash" ]; then
      _why="cannot list the jar content (unzip missing or failing, or an unreadable jar)"
    elif [ "$_new_hash" != "$_old_hash" ] && { [ -z "$_new_ver" ] || [ -z "$_old_ver" ]; }; then
      _why="shipped bytes changed but the module's own vendorVersion is unreadable (old='$_old_ver' new='$_new_ver')"
    fi
    if [ -n "$_why" ]; then
      echo "build.sh: FAIL — drift gate cannot verify $MOD-$p: $_why." >&2
      # Same retry-safety contract as the drift FAIL below.
      _restore_baseline "$p" "."
      DRIFT_UNVERIFIED=1
      continue
    fi
    [ "$_new_hash" = "$_old_hash" ] && continue   # nothing shipped actually changed
    if [ "$_new_ver" = "$_old_ver" ]; then
      echo "build.sh: FAIL — $MOD-$p shipped bytes changed but vendorVersion is still $_new_ver." >&2
      echo "  Software Manager compares versions, not bytes — it will report \"Up to Date\" and SKIP installing this jar." >&2
      echo "  Bump defaultModuleVersion(\"$_new_ver\") in $MOD-$p's build.gradle.kts (patch for a fix, minor for a feature), then rebuild." >&2
      # Restore the pre-build baseline (part 1 backup) so a bare re-run stays retry-safe: without this,
      # gradle's :jar step has already left the NEW jar in modules/, and a re-run would snapshot that as
      # its own baseline and silently pass. [R4-drift-gate-not-retry-safe]
      _restore_baseline "$p" " — a retry will catch this again."
      DRIFT_FAIL=1
    fi
  done
  # Precedence: a real drift FAIL (51) outranks a cannot-verify profile (52) in the same run — the bump it asks for
  # is needed either way, and the cannot-verify rows above stay on stderr.
  if [ "$DRIFT_FAIL" -eq 1 ]; then
    echo "build.sh: deployed-baseline drift detected — bump the version above, then rebuild (or --no-drift-check if this niagara_home is not the deploy target)." >&2
    exit 51
  fi
  if [ "$DRIFT_UNVERIFIED" -eq 1 ]; then
    echo "build.sh: deployed-baseline drift gate could not verify the jar(s) above — fix the cause (install unzip, check module.xml), or --no-drift-check if this niagara_home is not the deploy target." >&2
    exit 52
  fi
fi

echo "==> verify gate (verify-module.sh):"
JARS=(); for p in "${SEL[@]}"; do JARS+=("$ROOT/$MOD/$MOD-$p/build/libs/$MOD-$p.jar"); done
VARGS=(--src "$ROOT/$MOD"); [ -z "$TARGET" ] || VARGS+=(--target-version "$TARGET")
if "$HERE/verify-module.sh" "${VARGS[@]}" "${JARS[@]}"; then
  if [ "$SKIP_REPORT" -eq 0 ]; then
    echo "==> report-module (hand-off punch-list)"
    RARGS=("$ROOT/$MOD"); [ -z "$TARGET" ] || RARGS+=(--target-version "$TARGET")
    [ -z "$UI_PROFILE" ] || RARGS+=(--profile "$UI_PROFILE")
    [ "$LEGACY" -eq 0 ] || RARGS+=(--legacy)
    [ "$FIRST_DEPLOY" -eq 0 ] || RARGS+=(--first-deploy)
    if "$HERE/report-module.sh" "${RARGS[@]}"; then
      exit 0
    else
      _RM=$?
      if [ "$_RM" -eq 3 ]; then
        echo "build.sh: report-module environment error (exit 3)" >&2; exit 10
      fi
      if [ "$_RM" -eq 4 ]; then
        echo "build.sh: report-module verdict INCOMPLETE — a floor check (verify-module / schema-risk) did not run; extract the deployed baseline into <artifact>/.deploy-baseline (build-verify.md § Schema-risk deploy workflow) or pass --first-deploy for a never-deployed module" >&2; exit 53
      fi
      echo "build.sh: report-module punch-list has FAILs — not hand-off-ready (--no-report to skip)" >&2; exit 50
    fi
  else
    exit 0
  fi
fi
echo "build.sh: verify gate failed — do not deploy these jars" >&2; exit 50
