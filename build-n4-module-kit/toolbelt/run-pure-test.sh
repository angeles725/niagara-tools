#!/usr/bin/env bash
# run-pure-test.sh — run a ZERO-Baja pure-logic JUnit test standalone in WSL.
#
# WHY: niagaraTest does not run in WSL (needs native bin/test + a dev license), and
# plugin 7.6.17 discovers 0 tests anyway (build-verify.md §Unit tests). The only
# runnable coverage of control/safety logic is a Baja-free `final class` decision core
# tested with JUnit 4 via JUnitCore. The recipe was documented but the jar paths were
# re-hunted every session (retro 2026-09-04-junit-standalone-cached-jar-locations...).
# This removes the hunt and the per-session copy-paste.
#
# USAGE: run-pure-test.sh <module-rt-dir> <pkg.TestClass>
#   <module-rt-dir>  e.g. .../CompPan/CompPan-rt   (holds src/ and srcTest/)
#   <pkg.TestClass>  e.g. com.angeles.CompPan.CompressorControlTest
#
# Compiles the test + any pure sources it references (via -sourcepath) into a TEMP dir
# — never writes into the module tree (a parallel session's working tree is off-limits)
# — then runs JUnitCore.
#
# EXIT: 0 tests passed · 1 a test FAILED (the bite) · 2 usage · 3 environment
#   (no JDK / junit not in the gradle cache — run one gradle build to fetch it).

set -euo pipefail

die() { printf '%s\n' "run-pure-test: $2" >&2; exit "$1"; }

[ $# -eq 2 ] || die 2 "usage: run-pure-test.sh <module-rt-dir> <pkg.TestClass>"
rt=$1
testfqcn=$2

[ -d "$rt" ] || die 3 "module-rt-dir not found: $rt"
command -v javac >/dev/null 2>&1 || die 3 "javac not on PATH (need a JDK 8)"

# JUnit 4.13.2 + Hamcrest 1.3 live in the Gradle cache/dist, not the repo or the N4 install.
JU=$(find "$HOME/.gradle" -name 'junit-4.13.2.jar' 2>/dev/null | head -1)
HC=$(find "$HOME/.gradle" -name 'hamcrest-core-1.3.jar' 2>/dev/null | head -1)
[ -n "$JU" ] || die 3 "junit-4.13.2.jar not in ~/.gradle — run one './gradlew :<mod>-rt:compileTestJava' to fetch it"
[ -n "$HC" ] || die 3 "hamcrest-core-1.3.jar not in ~/.gradle — run one gradle build to fetch it"

pkgpath=${testfqcn%.*}; pkgpath=${pkgpath//.//}
testsimple=${testfqcn##*.}

# The test source root is either srcTest/test/ (the ColdRoomPan/CompPan/DashboardPan
# layout, package under an extra test/ segment) or srcTest/ directly. Detect it.
testroot=""
for cand in "$rt/srcTest/test" "$rt/srcTest"; do
  if [ -f "$cand/$pkgpath/$testsimple.java" ]; then testroot=$cand; break; fi
done
[ -n "$testroot" ] || die 3 "test source not found: $rt/srcTest[/test]/$pkgpath/$testsimple.java"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# -sourcepath pulls in the pure sibling classes the test references, compiled together
# so package-private members stay reachable. Nothing is written under $rt.
javac -source 8 -target 8 -nowarn -cp "$JU" \
  -sourcepath "$rt/src:$testroot" \
  -d "$tmp" "$testroot/$pkgpath/$testsimple.java"

java -cp "$tmp:$JU:$HC" org.junit.runner.JUnitCore "$testfqcn"
