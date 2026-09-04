#!/usr/bin/env bash
# shellcheck disable=SC2329
# why: the check_* functions are invoked indirectly by name from the loop `"$chk" "$JAR"` (file-wide directive)
# verify-module.sh — THE gate for built N4 module jars. A jar that has not passed it does not go to a station.
#
# Checks (each guards a failure seen on a real build — build-n4-module-kit/retros/):
#   bytecode  every .class has major 52 (Java 8). 65 = built with the default JDK 21.          [default]
#   signed    META-INF/NIAGARA4.SF present (gradle niagara-signing ran).                        [default]
#   types     every <type class=...> in META-INF/module.xml exists as a .class in the jar.      [default]
#   baja      module.xml baja vendorVersion <= --target-version (a 4.15 jar is rejected by 4.14). [--target-version]
#   stored    zero Deflated entries — required only when the jar must be re-signed in Workbench. [--stored]
#   typecount packaged <type> count == <module-dir>/<jar-basename>/module-include.xml count.     [--src]
#   facets    no BFacets.make(BFacets.MIN|MAX, <raw number>) under <module-dir>/<jar-basename>/src. [--src]
#   rcbackup  no editor/backup files (*~ *.orig *.bak*) packaged under rc/ — WARN, or FAIL under --strict. [default]
#
# Usage: verify-module.sh [--target-version X.Y] [--stored] [--src <module-dir>] [--strict] <jar>...
#   <module-dir> = the dir holding the profile dirs (e.g. .../Dashboard/DashboardPan); the profile is
#   derived from each jar's basename (DashboardPan-rt.jar -> DashboardPan-rt/).
# Report: one row per check per jar: PASS|FAIL|SKIP  <check>  <jar>  <detail>; a summary line closes it.
# Exit: 0 every executed check passed · 1 at least one check FAILED · 2 usage · 3 environment (tool missing, jar unreadable).
# Needs only unzip, od, grep, awk, sort — no JDK.
set -euo pipefail

usage() { sed -n '2,20p' "$0" | sed 's/^# \{0,1\}//'; }
TARGET=""; STORED=0; SRC=""; STRICT=0; JARS=()
while [ $# -gt 0 ]; do
  case "$1" in
    --target-version) [ $# -ge 2 ] || { usage >&2; exit 2; }; TARGET="$2"; shift 2 ;;
    --stored) STORED=1; shift ;;
    --strict) STRICT=1; shift ;;
    --src) [ $# -ge 2 ] || { usage >&2; exit 2; }; SRC="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    -*) echo "verify-module: unknown flag $1" >&2; usage >&2; exit 2 ;;
    *) JARS+=("$1"); shift ;;
  esac
done
[ ${#JARS[@]} -gt 0 ] || { usage >&2; exit 2; }
for t in unzip od awk grep sort; do command -v "$t" >/dev/null || { echo "verify-module: missing tool: $t" >&2; exit 3; }; done
[ -z "$SRC" ] || [ -d "$SRC" ] || { echo "verify-module: --src is not a directory: $SRC" >&2; exit 2; }
for j in "${JARS[@]}"; do case "$j" in *.jar) ;; *) echo "verify-module: not a jar: $j" >&2; exit 2 ;; esac; done

FAILED=0; NPASS=0; NFAIL=0; NSKIP=0; NWARN=0
row() { printf '%-4s  %-9s  %s  %s\n' "$1" "$2" "$3" "$4"; case "$1" in PASS) NPASS=$((NPASS+1));; FAIL) NFAIL=$((NFAIL+1));; SKIP) NSKIP=$((NSKIP+1));; WARN) NWARN=$((NWARN+1));; esac; }
# version compare: 0 when $1 <= $2 (dot-separated numeric)
ver_le() { [ "$(printf '%s\n%s\n' "$1" "$2" | sort -V | head -1)" = "$1" ]; }

check_bytecode_major() {  # every class, not just the first
  local jar="$1" n=0 bad=0 c m first=""
  while IFS= read -r c; do
    [ -n "$c" ] || continue; n=$((n+1))
    m=$(unzip -p "$jar" "$c" | od -An -t u1 -j6 -N2 | awk '{print $2}')
    [ "$m" = "52" ] || { bad=$((bad+1)); [ -n "$first" ] || first="$c major=$m"; }
  done <<< "$(printf '%s\n' "$LIST" | grep '\.class$' || true)"
  if [ "$n" -eq 0 ]; then row FAIL bytecode "$jar" "no .class entries"; return 1; fi
  if [ "$bad" -eq 0 ]; then row PASS bytecode "$jar" "$n classes, all major 52"; return 0; fi
  row FAIL bytecode "$jar" "$bad of $n classes not major 52 (expected Java 8); first: $first"; return 1
}
check_signed() {
  if printf '%s\n' "$LIST" | grep -q '^META-INF/NIAGARA4\.SF$'; then row PASS signed "$1" "META-INF/NIAGARA4.SF present"; return 0; fi
  row FAIL signed "$1" "META-INF/NIAGARA4.SF missing (jar not signed)"; return 1
}
check_types_have_classes() {
  local jar="$1" missing=0 tn=0 cls f firstmiss=""
  [ -n "$MX" ] || { row FAIL types "$jar" "META-INF/module.xml missing (not an N4 module jar)"; return 1; }
  while IFS= read -r cls; do
    [ -n "$cls" ] || continue; tn=$((tn+1))
    f="$(printf '%s' "$cls" | tr . /).class"
    printf '%s\n' "$LIST" | grep -qx "$f" || { missing=$((missing+1)); [ -n "$firstmiss" ] || firstmiss="$cls"; }
  done <<< "$TYPES"
  if [ "$missing" -eq 0 ]; then row PASS types "$jar" "$tn declared types resolve to classes"; return 0; fi
  row FAIL types "$jar" "$missing of $tn declared types have no class; first: $firstmiss"; return 1
}
check_baja_version() {
  local jar="$1" bv
  [ -n "$TARGET" ] || { row SKIP baja "$jar" "no --target-version"; return 0; }
  [ -n "$MX" ] || { row FAIL baja "$jar" "META-INF/module.xml missing"; return 1; }
  bv=$(printf '%s' "$MX" | grep -oE '<dependency [^>]*name="baja"[^>]*vendorVersion="[^"]+"' | sed -E 's/.*vendorVersion="([^"]+)".*/\1/' | head -1 || true)
  [ -n "$bv" ] || { row FAIL baja "$jar" "no baja dependency in module.xml"; return 1; }
  if ver_le "$bv" "$TARGET"; then row PASS baja "$jar" "stamped baja $bv <= target $TARGET"; return 0; fi
  row FAIL baja "$jar" "stamped baja $bv > target $TARGET (a $TARGET station rejects it; build against the $TARGET niagara_home)"; return 1
}
check_stored() {
  local jar="$1" d
  [ "$STORED" -eq 1 ] || { row SKIP stored "$jar" "no --stored"; return 0; }
  d=$(unzip -v "$jar" | grep -c 'Defl:' || true)
  if [ "$d" -eq 0 ]; then row PASS stored "$jar" "0 deflated entries (Workbench re-sign safe)"; return 0; fi
  row FAIL stored "$jar" "$d deflated entries (run stored-repack.sh before re-signing in Workbench)"; return 1
}
profile_dir() { local b; b=$(basename "$1" .jar); printf '%s/%s' "$SRC" "$b"; }
check_type_count() {
  local jar="$1" pd inc tn sn
  [ -n "$SRC" ] || { row SKIP typecount "$jar" "no --src"; return 0; }
  pd=$(profile_dir "$jar"); inc="$pd/module-include.xml"
  [ -f "$inc" ] || { row SKIP typecount "$jar" "no $inc"; return 0; }
  tn=$(printf '%s\n' "$TYPES" | grep -c . || true); sn=$(grep -c '<type ' "$inc" || true)
  if [ "$tn" = "$sn" ]; then row PASS typecount "$jar" "jar declares $tn types == module-include.xml"; return 0; fi
  row FAIL typecount "$jar" "jar declares $tn types, $inc declares $sn (stale build or missing slotomatic)"; return 1
}
check_raw_double_facets() {
  local jar="$1" pd hits
  [ -n "$SRC" ] || { row SKIP facets "$jar" "no --src"; return 0; }
  pd=$(profile_dir "$jar")
  [ -d "$pd/src" ] || { row SKIP facets "$jar" "no $pd/src"; return 0; }
  hits=$(grep -rnE 'BFacets\.make\(BFacets\.(MIN|MAX), *-?[0-9]' "$pd/src" --include='*.java' 2>/dev/null | head -1 || true)
  if [ -z "$hits" ]; then row PASS facets "$jar" "no raw-number MIN/MAX facet under $pd/src"; return 0; fi
  row FAIL facets "$jar" "raw-number MIN/MAX facet (wrap in BDouble.make): $hits"; return 1
}
check_rc_backup() {  # editor/backup files packaged under rc/ (bloat + servable). WARN by default; FAIL under --strict.
  local jar="$1" names
  names=$(printf '%s\n' "$LIST" | awk -F/ '/^rc\//{ b=$NF; if (b ~ /~$/ || b ~ /\.orig$/ || b ~ /\.bak.*$/) print $0 }' | tr '\n' ' ' | sed 's/ *$//')
  [ -n "$names" ] || return 0   # clean rc/ → silent, no row
  if [ "$STRICT" -eq 1 ]; then row FAIL rcbackup "$jar" "editor/backup files under rc/ (drop them): $names"; return 1; fi
  row WARN rcbackup "$jar" "editor/backup files under rc/ (drop them): $names"; return 0
}

for JAR in "${JARS[@]}"; do
  [ -f "$JAR" ] || { echo "verify-module: jar not readable: $JAR" >&2; exit 3; }
  LIST=$(unzip -Z1 "$JAR" 2>/dev/null) || { echo "verify-module: not a zip: $JAR" >&2; exit 3; }
  MX=""; TYPES=""
  if printf '%s\n' "$LIST" | grep -q '^META-INF/module\.xml$'; then
    MX=$(unzip -p "$JAR" META-INF/module.xml)
    TYPES=$(printf '%s' "$MX" | grep -oE '<type [^>]*class="[^"]+"' | sed -E 's/.*class="([^"]+)".*/\1/' || true)
  fi
  for chk in check_bytecode_major check_signed check_types_have_classes check_baja_version check_stored check_type_count check_raw_double_facets check_rc_backup; do
    if "$chk" "$JAR"; then :; else FAILED=1; fi
  done
done
printf 'verify-module: %d passed, %d failed, %d skipped, %d warned -> %s\n' "$NPASS" "$NFAIL" "$NSKIP" "$NWARN" "$([ "$FAILED" -eq 0 ] && echo ALL PASS || echo FAILED)"
exit "$FAILED"
