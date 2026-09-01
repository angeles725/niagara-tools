#!/usr/bin/env bash
# Correct N4 module build: Java 8 + clean + slotomatic + jar.
# A `gradle :jar` with the default JDK is NOT valid (wrong bytecode + skips slotomatic).
# Usage: ./build.sh <module-root> <MOD-name> [niagara_home]
#   ./build.sh /home/cristian/niagara-modules/DashboardPan-Leon DashboardPan /home/cristian/dpan-niagara-home
set -euo pipefail

ROOT="${1:?module root}"; MOD="${2:?module name (e.g. DashboardPan)}"
NIAGARA_HOME="${3:-${niagara_home:-}}"
J8="${JAVA8:-/usr/lib/jvm/java-8-openjdk-amd64}"

[ -d "$J8" ] || { echo "ERROR: Java 8 no está en $J8 — revisa 'ls /usr/lib/jvm'"; exit 1; }
[ -n "$NIAGARA_HOME" ] || { echo "ERROR: pasa niagara_home (arg 3) o exporta niagara_home"; exit 1; }

cd "$ROOT"
# Detect which profiles exist (rt / ux / wb)
TASKS=(); for p in rt ux wb; do
  [ -d "$MOD/$MOD-$p" ] && TASKS+=(":$MOD-$p:clean" ":$MOD-$p:slotomatic" ":$MOD-$p:jar")
done
[ ${#TASKS[@]} -gt 0 ] || { echo "ERROR: no encuentro $MOD/$MOD-{rt,ux,wb}"; exit 1; }

echo "==> build (Java 8 + slotomatic): ${TASKS[*]}"
./gradlew "${TASKS[@]}" \
  -Pniagara_home="$NIAGARA_HOME" \
  -Porg.gradle.java.installations.paths="$J8"

echo "==> verificar bytecode (major 52 = Java 8) y firma:"
for p in rt ux wb; do
  J="$MOD/$MOD-$p/build/libs/$MOD-$p.jar"; [ -f "$J" ] || continue
  CLS=$(unzip -Z1 "$J" 2>/dev/null | grep -m1 '\.class$' || true)
  [ -n "$CLS" ] && printf "  %s major=%s\n" "$p" \
    "$(unzip -p "$J" "$CLS" | od -An -t d1 -j6 -N2 | awk '{print $2}')"
  unzip -l "$J" | grep -q NIAGARA4.SF && echo "    firmado" || echo "    SIN FIRMA"
done
