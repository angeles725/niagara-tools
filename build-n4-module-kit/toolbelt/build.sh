#!/usr/bin/env bash
# build.sh — the recommended WSL build of an N4 module: Java 8 + clean + slotomatic + jar, then THE gate.
# A `gradle :jar` with the default JDK is NOT a build (wrong bytecode major, slotomatic skipped).
# Deploying to a station is ng-deploy.sh's job (backup -> build -> copy -> type-count verify).
#
# Usage: build.sh [--profiles rt,ux,wb] [--target-version X.Y] [--plugin-version V] <module-root> <MOD> [niagara_home]
#   <module-root>   the dir holding ./gradlew and <MOD>/<MOD>-{rt,ux,wb}/
#   niagara_home    arg 3, else $niagara_home. On WSL use the /mnt/c/... mount or a mirror (mirror-niagara-home.sh).
#   --plugin-version / $NIAGARA_PLUGIN_VERSION   forwarded as -PniagaraPluginVersion (each install ships ONE
#                   niagara-module plugin: 4.13.2 -> 7.3.40, 4.14 -> 7.6.17, 4.15.3 -> 7.6.22)
#   $JAVA8          JDK 8 path (default /usr/lib/jvm/java-8-openjdk-amd64)
# Profiles: by default every <MOD>-<p> dir that has a gradle file AND sources under src/ is built; a scaffold
#   (gradle file, no sources) is reported "skipped". --profiles replaces auto-detection entirely.
# After gradle, verify-module.sh (same dir) runs on every produced jar with --src <module-root>/<MOD>.
# Exit: 0 build + gate passed · 2 usage · 10 environment (no JDK 8, not a niagara_home, no profile) · 30 gradle failed · 50 gate failed (ng-deploy.sh 50 = verify failed)
set -euo pipefail

usage() { sed -n '2,15p' "$0" | sed 's/^# \{0,1\}//'; }
PROFILES=""; TARGET=""; PLUGIN="${NIAGARA_PLUGIN_VERSION:-}"
while [ $# -gt 0 ]; do
  case "$1" in
    --profiles)       [ $# -ge 2 ] || { usage >&2; exit 2; }; PROFILES="$2"; shift 2 ;;
    --target-version) [ $# -ge 2 ] || { usage >&2; exit 2; }; TARGET="$2"; shift 2 ;;
    --plugin-version) [ $# -ge 2 ] || { usage >&2; exit 2; }; PLUGIN="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    -*) echo "build.sh: unknown flag $1" >&2; usage >&2; exit 2 ;;
    *) break ;;
  esac
done
[ $# -ge 2 ] && [ $# -le 3 ] || { usage >&2; exit 2; }
ROOT="$1"; MOD="$2"
NIAGARA_HOME="${3:-${niagara_home:-}}"
J8="${JAVA8:-/usr/lib/jvm/java-8-openjdk-amd64}"
HERE="$(cd "$(dirname "$0")" && pwd)"

[ -d "$ROOT" ] || { echo "build.sh: module root not found: $ROOT" >&2; exit 10; }
[ -x "$ROOT/gradlew" ] || { echo "build.sh: no executable ./gradlew in $ROOT (the module needs the gradle wrapper)" >&2; exit 10; }
[ -d "$J8" ] || { echo "build.sh: Java 8 not found at $J8 — check 'ls /usr/lib/jvm' or set JAVA8" >&2; exit 10; }
[ -n "$NIAGARA_HOME" ] || { echo "build.sh: pass niagara_home (arg 3) or export niagara_home" >&2; exit 10; }
[ -d "$NIAGARA_HOME/etc/m2/repository" ] || { echo "build.sh: not a niagara_home (no etc/m2/repository): $NIAGARA_HOME" >&2; exit 10; }
[ -x "$HERE/verify-module.sh" ] || { echo "build.sh: gate not found next to this script: $HERE/verify-module.sh" >&2; exit 10; }

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
GARGS=(-Pniagara_home="$NIAGARA_HOME" -Porg.gradle.java.installations.paths="$J8")
[ -z "$PLUGIN" ] || GARGS+=(-PniagaraPluginVersion="$PLUGIN")

echo "==> build (Java 8 + slotomatic): ${TASKS[*]}"
if ! ( cd "$ROOT" && ./gradlew "${TASKS[@]}" "${GARGS[@]}" ); then
  echo "build.sh: gradle failed" >&2; exit 30
fi

echo "==> verify gate (verify-module.sh):"
JARS=(); for p in "${SEL[@]}"; do JARS+=("$ROOT/$MOD/$MOD-$p/build/libs/$MOD-$p.jar"); done
VARGS=(--src "$ROOT/$MOD"); [ -z "$TARGET" ] || VARGS+=(--target-version "$TARGET")
if "$HERE/verify-module.sh" "${VARGS[@]}" "${JARS[@]}"; then exit 0; fi
echo "build.sh: verify gate failed — do not deploy these jars" >&2; exit 50
