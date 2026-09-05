#!/usr/bin/env bash
# preflight.sh — BUILD-LOOP §0.b environment preflight for Niagara N4 module builds.
#
# Automates the four pre-build checks so a broken environment is caught before
# gradle fails with a cryptic error. A FAIL row identifies the problem; the script
# continues checking the rest so all issues are visible in one run.
#
# Usage: preflight.sh [--jvm-dir <d>] <niagara_home> <gradle-root>
#   --jvm-dir <d>   scan this directory for a JDK 8 release (default /usr/lib/jvm, read-only)
#                   JAVA_HOME is also checked (fallback) — never $HOME
#   <niagara_home>  Niagara 4 installation root (must not be a Windows-style path)
#   <gradle-root>   gradle project root containing settings.gradle.kts
#
# Checks (order: win-path, jdk8, plugin-pin, jar-lock):
#   win-path    niagara_home or gradle-root contains C: or backslash
#               -> FAIL + "use /mnt/c/... in WSL" remedy
#   jdk8        JDK 8 found under --jvm-dir or JAVA_HOME (never $HOME)
#               -> PASS|FAIL row; detail names the found path or search dir
#   plugin-pin  settings.gradle.kts plugin version present in <niagara_home>/etc/m2
#               -> PASS|FAIL; version extracted from first "x.y.z" quoted string
#   jar-lock    lsof on <niagara_home>/modules/*.jar
#               -> PASS|WARN(locked)|SKIP(lsof absent) — never a false PASS
#
# Row format: PASS|FAIL|WARN|SKIP  <check>  <detail>
# Exit: 0 all PASS/WARN/SKIP · 1 any FAIL · 2 usage · 3 env (path not found)
# This script is VCS-free by design. version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
set -u

JVM_DIR="/usr/lib/jvm"
FAILED=0

row() {
  # printf: left-align status (4 chars) + check name (12 chars) + detail
  printf '%-4s  %-12s  %s\n' "$1" "$2" "$3"
  case "$1" in FAIL) FAILED=1 ;; esac
}

usage_exit() {
  printf 'usage: preflight.sh [--jvm-dir <d>] <niagara_home> <gradle-root>\n' >&2
  exit 2
}

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
while [ $# -gt 0 ]; do
  case "$1" in
    --jvm-dir)
      [ $# -ge 2 ] || usage_exit
      JVM_DIR="$2"; shift 2
      ;;
    --) shift; break ;;
    -*) usage_exit ;;
    *) break ;;
  esac
done

[ $# -eq 2 ] || usage_exit

NH="$1"
GR="$2"

# ---------------------------------------------------------------------------
# Check 1 — Windows-style path detection
# A path containing C: or a backslash is a Windows path pasted into WSL.
# This FAILS immediately and skips filesystem checks that need a valid path.
# ---------------------------------------------------------------------------
WIN_PATH_FAIL=0
case "$NH" in
  *C:*|*\\*)
    row FAIL "win-path" "niagara_home is a Windows path — use /mnt/c/... in WSL instead"
    WIN_PATH_FAIL=1
    ;;
esac
case "$GR" in
  *C:*|*\\*)
    row FAIL "win-path" "gradle-root is a Windows path — use /mnt/c/... in WSL instead"
    WIN_PATH_FAIL=1
    ;;
esac

# Validate paths exist (skip when Windows path detected — they won't resolve)
if [ "$WIN_PATH_FAIL" -eq 0 ]; then
  [ -d "$NH" ] || { printf 'preflight: niagara_home not found: %s\n' "$NH" >&2; exit 3; }
  [ -d "$GR" ] || { printf 'preflight: gradle-root not found: %s\n' "$GR" >&2; exit 3; }
fi

# ---------------------------------------------------------------------------
# Check 2 — JDK 8 detection
# Scans --jvm-dir (default /usr/lib/jvm) for a subdirectory whose release file
# declares JAVA_VERSION="1.8.*". Falls back to JAVA_HOME when set.
# Never uses $HOME — resolves only from explicit --jvm-dir or JAVA_HOME.
# ---------------------------------------------------------------------------
JDK8_FOUND=0
JDK8_PATH=""

if [ -d "$JVM_DIR" ]; then
  for jvm_candidate in "$JVM_DIR"/*/; do
    [ -f "$jvm_candidate/release" ] || continue
    if grep -q 'JAVA_VERSION="1\.8\.' "$jvm_candidate/release" 2>/dev/null; then
      JDK8_FOUND=1
      JDK8_PATH="$jvm_candidate"
      break
    fi
  done
fi

# Fallback: JAVA_HOME environment variable (not $HOME — different variable)
if [ "$JDK8_FOUND" -eq 0 ] && [ -n "${JAVA_HOME:-}" ]; then
  if [ -f "$JAVA_HOME/release" ] && grep -q 'JAVA_VERSION="1\.8\.' "$JAVA_HOME/release" 2>/dev/null; then
    JDK8_FOUND=1
    JDK8_PATH="$JAVA_HOME"
  fi
fi

if [ "$JDK8_FOUND" -eq 1 ]; then
  row PASS "jdk8" "JDK 8 found: $JDK8_PATH"
else
  row FAIL "jdk8" "JDK 8 not found under $JVM_DIR (or JAVA_HOME); use --jvm-dir to override"
fi

# ---------------------------------------------------------------------------
# Check 3 — gradle plugin pin in niagara_home/etc/m2
# Reads settings.gradle.kts for the first quoted "x.y.z" version string,
# then checks for a path matching that version under <niagara_home>/etc/m2.
# A missing pin means the build will fail with a cryptic repository error.
# ---------------------------------------------------------------------------
if [ "$WIN_PATH_FAIL" -eq 0 ]; then
  SETTINGS="$GR/settings.gradle.kts"
  if [ ! -f "$SETTINGS" ]; then
    row FAIL "plugin-pin" "settings.gradle.kts not found in $GR"
  else
    PLUGIN_VER=$(grep -oE '"[0-9]+\.[0-9]+\.[0-9]+"' "$SETTINGS" | head -1 | tr -d '"')
    if [ -z "$PLUGIN_VER" ]; then
      row FAIL "plugin-pin" "cannot determine plugin version from $SETTINGS"
    else
      M2_DIR="$NH/etc/m2"
      if find "$M2_DIR" -path "*${PLUGIN_VER}*" 2>/dev/null | grep -q .; then
        row PASS "plugin-pin" "plugin $PLUGIN_VER found in niagara_home/etc/m2"
      else
        row FAIL "plugin-pin" "plugin $PLUGIN_VER missing from $M2_DIR — copy from Niagara install"
      fi
    fi
  fi
fi

# ---------------------------------------------------------------------------
# Check 4 — jar lock via lsof
# Checks whether any jar in <niagara_home>/modules/ is held open by a running
# process (e.g. the station). Building over a locked jar can silently corrupt it.
# SKIP when lsof is not in PATH — never emit a false PASS.
# ---------------------------------------------------------------------------
if [ "$WIN_PATH_FAIL" -eq 0 ]; then
  if ! command -v lsof >/dev/null 2>&1; then
    row SKIP "jar-lock" "lsof not in PATH — install lsof to enable jar-lock detection"
  else
    LOCKED=""
    for jar in "$NH/modules/"*.jar; do
      [ -f "$jar" ] || continue
      if lsof "$jar" >/dev/null 2>&1; then
        LOCKED="$LOCKED $(basename "$jar")"
      fi
    done
    if [ -n "$LOCKED" ]; then
      row WARN "jar-lock" "jars held by a process (station running?):$LOCKED"
    else
      row PASS "jar-lock" "no locked jars under $NH/modules/"
    fi
  fi
fi

[ "$FAILED" -eq 1 ] && exit 1
exit 0
