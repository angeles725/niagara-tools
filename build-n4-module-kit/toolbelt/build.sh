#!/usr/bin/env bash
# build.sh — full automatic chain: preflight → gradle (Java 8 + clean + slotomatic + jar) → verify gate → report-module.
# A `gradle :jar` with the default JDK is NOT a build (wrong bytecode major, slotomatic skipped).
# Deploying to a station is ng-deploy.sh's job (backup -> build -> copy -> type-count verify).
#
# Usage: build.sh [--profiles rt,ux,wb] [--target-version X.Y] [--plugin-version V] [--no-preflight] [--no-report] [--no-drift-check] <module-root> <MOD> [niagara_home]
#   <module-root>   the dir holding ./gradlew and <MOD>/<MOD>-{rt,ux,wb}/
#   niagara_home    arg 3, else $niagara_home. On WSL use the /mnt/c/... mount or a mirror (mirror-niagara-home.sh).
#   --plugin-version / $NIAGARA_PLUGIN_VERSION   forwarded as -PniagaraPluginVersion (each install ships ONE
#                   niagara-module plugin: 4.13.2 -> 7.3.40, 4.14 -> 7.6.17, 4.15.3 -> 7.6.22)
#   --no-preflight  skip environment preflight (useful for inner rebuild loops when env is known-good)
#   --no-report     skip report-module punch-list at the end (useful for quick inner rebuild loops)
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
# Exit: 0 chain passed · 2 usage · 10 environment (preflight FAIL, no JDK 8, not a niagara_home, no profile) ·
#   30 gradle failed · 31 :clean locked · 50 gate or report-module FAIL · 51 deployed-baseline drift (Δ2)
set -euo pipefail

usage() { sed -n '2,26p' "$0" | sed 's/^# \{0,1\}//'; }
PROFILES=""; TARGET=""; PLUGIN="${NIAGARA_PLUGIN_VERSION:-}"
SKIP_PREFLIGHT=0; SKIP_REPORT=0; SKIP_DRIFT_CHECK=0
while [ $# -gt 0 ]; do
  case "$1" in
    --profiles)       [ $# -ge 2 ] || { usage >&2; exit 2; }; PROFILES="$2"; shift 2 ;;
    --target-version) [ $# -ge 2 ] || { usage >&2; exit 2; }; TARGET="$2"; shift 2 ;;
    --plugin-version) [ $# -ge 2 ] || { usage >&2; exit 2; }; PLUGIN="$2"; shift 2 ;;
    --no-preflight) SKIP_PREFLIGHT=1; shift ;;
    --no-report)    SKIP_REPORT=1;    shift ;;
    --no-drift-check) SKIP_DRIFT_CHECK=1; shift ;;
    -h|--help) usage; exit 0 ;;
    -*) echo "build.sh: unknown flag $1" >&2; usage >&2; exit 2 ;;
    *) break ;;
  esac
done
if [ $# -lt 2 ] || [ $# -gt 3 ]; then usage >&2; exit 2; fi
ROOT="$1"; MOD="$2"
NIAGARA_HOME="${3:-${niagara_home:-}}"
J8="${JAVA8:-/usr/lib/jvm/java-8-openjdk-amd64}"
HERE="$(cd "$(dirname "$0")" && pwd)"
# shellcheck disable=SC1091
. "$HERE/lib/fs-type.sh"

[ -d "$ROOT" ] || { echo "build.sh: module root not found: $ROOT" >&2; exit 10; }
# B7: gradlew may live at an ANCESTOR (client multi-project layout — the module dir is passed as ROOT). Walk up to find it.
GRADLE_ROOT="$ROOT"
while [ -n "$GRADLE_ROOT" ] && [ "$GRADLE_ROOT" != "/" ] && [ ! -x "$GRADLE_ROOT/gradlew" ]; do GRADLE_ROOT="$(dirname "$GRADLE_ROOT")"; done
[ -x "$GRADLE_ROOT/gradlew" ] || {
    echo "build.sh: no executable ./gradlew in $ROOT or any ancestor (the module needs the gradle wrapper)" >&2
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
  if "$HERE/preflight.sh" "$NIAGARA_HOME" "$GRADLE_ROOT"; then
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

# ---------------------------------------------------------------------------
# Deployed-baseline drift gate (Δ2), part 1 — pre-build snapshot.
# Gradle's :jar task auto-installs the new jar into <niagara_home>/modules/ as
# its LAST step, so "what is currently deployed" must be captured BEFORE
# gradle runs — by the time gradle exits, modules/ already holds the NEW
# bytes and a post-build-only comparison would always see "no drift".
# [ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ2]
# ---------------------------------------------------------------------------
_content_hash() {
  # Deterministic hash of every non-META-INF jar entry's CONTENT (sorted by
  # name). META-INF/* (module.xml buildMillis, MANIFEST.MF, signatures) is
  # EXCLUDED on purpose: those bytes change on every rebuild even when nothing
  # actually shipped changed, which would otherwise make this gate false-fire
  # on every rebuild regardless of source changes.
  local jar="$1"
  {
    { unzip -Z1 "$jar" 2>/dev/null | grep -v '^META-INF/' | LC_ALL=C sort \
        | while IFS= read -r _entry; do unzip -p "$jar" "$_entry" 2>/dev/null; done
    } || true
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
DRIFT_DIR=""
if [ "$SKIP_DRIFT_CHECK" -eq 0 ]; then
  DRIFT_DIR="$(mktemp -d)"
  for p in "${SEL[@]}"; do
    _old="$NIAGARA_HOME/modules/$MOD-$p.jar"
    [ -f "$_old" ] || continue
    _content_hash "$_old" > "$DRIFT_DIR/$p.old.hash"
    _module_own_version "$_old" > "$DRIFT_DIR/$p.old.ver"
  done
fi

echo "==> build (Java 8 + slotomatic): ${TASKS[*]}"
GLOG="$(mktemp)"
if ( cd "$GRADLE_ROOT" && ./gradlew "${TASKS[@]}" "${GARGS[@]}" ) 2>&1 | tee "$GLOG"; then
  rm -f "$GLOG"
else
  # soft-start: a running station LOCKS modules/<mod>.jar so :clean fails — tell the operator how to fix it.
  if grep -qE 'Unable to delete .*modules/.*\.jar' "$GLOG"; then
    echo "build.sh: :clean could not delete a modules/<jar> — a running station has it locked." >&2
    echo "  Free the lock first: close Workbench, or stop the station, then build directly; or build against a" >&2
    echo "  mirror (mirror-niagara-home.sh); or just use the already-assembled build/libs jar (the modules/ copy" >&2
    echo "  is irrelevant when you are not deploying to that supervisor)." >&2
    rm -f "$GLOG"; exit 31
  fi
  echo "build.sh: gradle failed" >&2; rm -f "$GLOG"; exit 30
fi

# ---------------------------------------------------------------------------
# Deployed-baseline drift gate (Δ2), part 2 — post-build compare.
# If the new jar's shipped-bytes content hash differs from the pre-build
# snapshot but the module's own vendorVersion is UNCHANGED, FAIL: Software
# Manager compares versions, not bytes, and would report "Up to Date" and
# silently skip installing this jar (the exact panccadia CompPan 2.1.0 leon
# vs leon2 trap). [ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ2]
# ---------------------------------------------------------------------------
if [ "$SKIP_DRIFT_CHECK" -eq 0 ]; then
  DRIFT_FAIL=0
  for p in "${SEL[@]}"; do
    [ -f "$DRIFT_DIR/$p.old.hash" ] || continue
    _new="$ROOT/$MOD/$MOD-$p/build/libs/$MOD-$p.jar"
    [ -f "$_new" ] || continue
    _new_hash="$(_content_hash "$_new")"
    _old_hash="$(cat "$DRIFT_DIR/$p.old.hash")"
    [ "$_new_hash" = "$_old_hash" ] && continue   # nothing shipped actually changed
    _new_ver="$(_module_own_version "$_new")"
    _old_ver="$(cat "$DRIFT_DIR/$p.old.ver" 2>/dev/null || true)"
    if [ -n "$_new_ver" ] && [ -n "$_old_ver" ] && [ "$_new_ver" = "$_old_ver" ]; then
      echo "build.sh: FAIL — $MOD-$p shipped bytes changed but vendorVersion is still $_new_ver." >&2
      echo "  Software Manager compares versions, not bytes — it will report \"Up to Date\" and SKIP installing this jar." >&2
      echo "  Bump defaultModuleVersion(\"$_new_ver\") in $MOD-$p's build.gradle.kts (patch for a fix, minor for a feature), then rebuild." >&2
      DRIFT_FAIL=1
    fi
  done
  rm -rf "$DRIFT_DIR"
  if [ "$DRIFT_FAIL" -eq 1 ]; then
    echo "build.sh: deployed-baseline drift detected — bump the version above, then rebuild (or --no-drift-check if this niagara_home is not the deploy target)." >&2
    exit 51
  fi
fi

echo "==> verify gate (verify-module.sh):"
JARS=(); for p in "${SEL[@]}"; do JARS+=("$ROOT/$MOD/$MOD-$p/build/libs/$MOD-$p.jar"); done
VARGS=(--src "$ROOT/$MOD"); [ -z "$TARGET" ] || VARGS+=(--target-version "$TARGET")
if "$HERE/verify-module.sh" "${VARGS[@]}" "${JARS[@]}"; then
  if [ "$SKIP_REPORT" -eq 0 ]; then
    echo "==> report-module (hand-off punch-list)"
    RARGS=("$ROOT/$MOD"); [ -z "$TARGET" ] || RARGS+=(--target-version "$TARGET")
    if "$HERE/report-module.sh" "${RARGS[@]}"; then
      exit 0
    else
      _RM=$?
      if [ "$_RM" -eq 3 ]; then
        echo "build.sh: report-module environment error (exit 3)" >&2; exit 10
      fi
      echo "build.sh: report-module punch-list has FAILs — not hand-off-ready (--no-report to skip)" >&2; exit 50
    fi
  else
    exit 0
  fi
fi
echo "build.sh: verify gate failed — do not deploy these jars" >&2; exit 50
