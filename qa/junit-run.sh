#!/usr/bin/env bash
# qa/junit-run.sh — compile + run ONE plain-JUnit test class of a client module offline in WSL.
#
# Usage: qa/junit-run.sh <module-dir> <fq.TestClass> [source.java ...]
#   <module-dir>   the profile dir (…/CompPan-rt, …/DashboardPan-ux). The run happens with cwd =
#                  <module-dir> because the source-structural tests resolve src/… and
#                  ../../build.gradle.kts from the cwd.
#   default sources: every src/**/*.java that never mentions javax.baja (the Baja-free cores). Pass
#                  explicit files to override. Baja-typed classes compile but cannot be instantiated
#                  offline (ExceptionInInitializerError) — those pins are harness-only.
# Env: N4 (default /mnt/c/Honeywell/OptimizerSupervisor-N4.14.0.162), JLIB (gradle 7.6 lib dir with
#      junit-4.13.2.jar + hamcrest-core-1.3.jar)
# Exit: JUnitCore's (0 green / 1 failures) · 2 compile failed (prints javac errors) · 3 usage/env
set -uo pipefail
[ $# -ge 2 ] || { sed -n '2,14p' "$0" | sed 's/^# \{0,1\}//'; exit 3; }
MOD=$1; CLS=$2; shift 2
N4=${N4:-/mnt/c/Honeywell/OptimizerSupervisor-N4.14.0.162}
JLIB=${JLIB:-$HOME/.gradle/wrapper/dists/gradle-7.6-bin/9l9tetv7ltxvx3i8an4pb86ye/gradle-7.6/lib}
CP="$JLIB/junit-4.13.2.jar:$JLIB/hamcrest-core-1.3.jar:$N4/modules/baja.jar:$N4/modules/control-rt.jar:$N4/modules/alarm-rt.jar:$N4/bin/ext/nre.jar"
for j in "$JLIB/junit-4.13.2.jar" "$JLIB/hamcrest-core-1.3.jar" "$N4/modules/baja.jar" "$N4/bin/ext/nre.jar"; do
  [ -f "$j" ] || { echo "junit-run: missing jar $j" >&2; exit 3; }
done
cd "$MOD" 2>/dev/null || { echo "junit-run: no module dir $MOD" >&2; exit 3; }
TF=$(find srcTest -name "${CLS##*.}.java" 2>/dev/null | head -1)
[ -n "$TF" ] || { echo "junit-run: test class ${CLS##*.} not under $MOD/srcTest" >&2; exit 3; }
if [ $# -eq 0 ]; then
  mapfile -t SRCS < <(grep -rL 'javax\.baja' src --include='*.java' 2>/dev/null)
else
  SRCS=("$@")
fi
OUT=$(mktemp -d "${TMPDIR:-/tmp}/junit.XXXXXX")
echo "junit-run: cwd=$PWD test=$TF sources=${#SRCS[@]} out=$OUT"
if ! javac -d "$OUT" -cp "$CP" -Xlint:none "${SRCS[@]}" "$TF" 2>"$OUT/javac.err"; then
  echo "junit-run: COMPILE FAILED"; cat "$OUT/javac.err"; exit 2
fi
java -cp "$OUT:$CP" org.junit.runner.JUnitCore "$CLS"
