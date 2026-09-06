#!/usr/bin/env bash
# shellcheck disable=SC2329,SC2317
# why: the check_* functions are invoked indirectly by name from the loop `"$chk" "$JAR"` (file-wide directive).
# SC2329 (never invoked) and SC2317 (unreachable) are the same false positive — shellcheck can't see the
# indirect `"$chk" "$JAR"` call, so it flags the function bodies. Both disabled file-wide.
# verify-module.sh — THE gate for built N4 module jars. A jar that has not passed it does not go to a station.
#
# Checks (each guards a failure seen on a real build — build-n4-module-kit/retros/):
#   bytecode    every .class has major 52 (Java 8). 65 = built with the default JDK 21.          [default]
#   signed      META-INF/NIAGARA4.SF present (gradle niagara-signing ran).                       [default]
#               NOTE: `signed` checks PRESENCE only — the signer's cert may not be in the
#               station's trust store. A self-signed/untrusted-CA jar PASSES this gate yet a
#               station enforcing cert-chain validation loads it UNSIGNED (`Could not validate
#               cert chain`). Signed = jar has NIAGARA4.SF; trusted = station-side policy.
#               `[ev: corpus B800 §800.8]`
#               folded as code: verify-module.sh signed-check (cert-chain trust caveat, D8, R7.6).
#   types       every <type class=...> in META-INF/module.xml exists as a .class in the jar.     [default]
#   baja        module.xml baja vendorVersion <= --target-version (a 4.15 jar is rejected by 4.14). [--target-version]
#   stored      zero Deflated entries — required only when the jar must be re-signed in Workbench. [--stored]
#   typecount   packaged <type> count == <module-dir>/<jar-basename>/module-include.xml count.    [--src]
#   facets      no BFacets.make(BFacets.MIN|MAX, <raw number>) under <module-dir>/<jar-basename>/src. [--src]
#   facets-req  OPERATOR numeric slot without a facets key (WARN); setpoint/count-like slot without UNITS/PRECISION (WARN). [--src]
#   ord-literal Java string literal matching station:|local:|slot:/ under src (WARN); exempt: *OrdConstants*+comment, defaultValue=, srcTest/**. [--src]
#   rcbackup    no editor/backup files (*~ *.orig *.bak*) packaged under rc/ — WARN, or FAIL under --strict. [default]
#   palette     a module that declares types must not ship an EMPTY module.palette (nothing to drag in
#               Workbench) — WARN, or FAIL under --strict; SKIP when the jar has no module.palette.     [default]
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

# coverage subcommand — dispatched before the flag loop so 'coverage' is never
# mistaken for a jar path. Pure function: exits 0 (value printed) or 2 (bad args).
# Usage: verify-module.sh coverage <npass> <nfail> <nwarn> <nskip>
#   applicable = npass + nfail + nwarn  (SKIP is structurally not-applicable)
#   covered    = npass                  (only clean passes count as covered)
#   result     = integer-tenths rounding via t=(1000*P+A/2)/A; prints P.Q format
#              = "N/A" when applicable == 0 (never 100, that would be a false pass)
# No version control invoked; VCS-free by design.
if [ $# -ge 1 ] && [ "$1" = "coverage" ]; then
  shift
  if [ $# -ne 4 ]; then
    echo "usage: verify-module.sh coverage <npass> <nfail> <nwarn> <nskip>" >&2
    exit 2
  fi
  P="$1"; F="$2"; W="$3"; S="$4"
  for cov_arg in "$P" "$F" "$W" "$S"; do
    case "$cov_arg" in
      ''|*[!0-9]*) echo "usage: verify-module.sh coverage <npass> <nfail> <nwarn> <nskip>" >&2; exit 2 ;;
    esac
  done
  A=$(( P + F + W ))
  if [ "$A" -eq 0 ]; then
    echo "N/A"
  else
    t=$(( (1000 * P + A / 2) / A ))
    echo "$((t / 10)).$((t % 10))"
  fi
  exit 0
fi

# --plano subcommand — dispatched before the flag loop so '--plano <file>' is
# never mistaken for a jar operand. Mode: verify Rc==Rv==Ri and every numeric
# aspect-ratio==Rc (B797 §797.2, integer cross-multiplication only; auto exempt).
#   Rc = IMG_W/IMG_H constants; Ri = #planoImg intrinsic PNG; Rv = zones viewBox.
# Usage: verify-module.sh --plano <index.html|jar>
# Exits: 0 PASS · 1 FAIL · 2 usage · 3 env (tool missing, file unreadable).
# VCS-free by design. No $HOME.
if [ $# -ge 1 ] && [ "$1" = "--plano" ]; then
  shift
  if [ $# -ne 1 ]; then
    echo "usage: verify-module.sh --plano <index.html|jar>" >&2; exit 2
  fi
  PLANO_IN="$1"
  for _pt in base64 od; do
    command -v "$_pt" >/dev/null || { echo "verify-module: missing tool: $_pt" >&2; exit 3; }
  done
  PLANO_TMP=""
  if [ "${PLANO_IN%.jar}" != "$PLANO_IN" ]; then
    command -v unzip >/dev/null || { echo "verify-module: missing tool: unzip" >&2; exit 3; }
    [ -f "$PLANO_IN" ] || { echo "verify-module: file not readable: $PLANO_IN" >&2; exit 3; }
    PLANO_TMP=$(mktemp -d)
    _html="$PLANO_TMP/index.html"
    unzip -p "$PLANO_IN" 'rc/index.html' > "$_html" 2>/dev/null \
      || { rm -rf "$PLANO_TMP"; echo "verify-module: no rc/index.html in: $PLANO_IN" >&2; exit 3; }
  else
    _html="$PLANO_IN"
    [ -f "$_html" ] || { echo "verify-module: file not readable: $_html" >&2; exit 3; }
  fi
  _plano_cleanup() { [ -z "$PLANO_TMP" ] || rm -rf "$PLANO_TMP"; }
  trap _plano_cleanup EXIT
  _prow() { printf '%-4s  %-9s  %s  %s\n' "$1" "plano" "$PLANO_IN" "$2"; }
  _ceq() { local a="$1" b="$2" c="$3" d="$4"; [ "$(( a * d ))" = "$(( b * c ))" ]; }
  # Parse Rc = IMG_W / IMG_H
  _rc_w=$(grep -oE 'IMG_W[[:space:]]*=[[:space:]]*[0-9]+' "$_html" | grep -oE '[0-9]+$' | head -1 || true)
  _rc_h=$(grep -oE 'IMG_H[[:space:]]*=[[:space:]]*[0-9]+' "$_html" | grep -oE '[0-9]+$' | head -1 || true)
  if [ -z "$_rc_w" ] || [ -z "$_rc_h" ]; then
    _prow FAIL "Rc: IMG_W/IMG_H not found"; exit 1
  fi
  # Parse Rv = zones viewBox width / height (id="zonas" element; >=2 distinct -> FAIL)
  _vb=$(grep 'id="zonas"' "$_html" | grep -oE 'viewBox="[^"]*"' | head -1 \
    | sed 's/viewBox="//;s/"//' | awk '{if(NF>=4)print $3" "$4}' | sort -u || true)
  if [ -z "$_vb" ]; then _prow FAIL "Rv: no id=\"zonas\" viewBox found"; exit 1; fi
  _vb_n=$(printf '%s\n' "$_vb" | wc -l | tr -d ' ')
  if [ "$_vb_n" -gt 1 ]; then _prow FAIL "Rv: ambiguous — multiple distinct viewBox dimensions"; exit 1; fi
  _rv_w=$(printf '%s' "$_vb" | awk '{print $1}')
  _rv_h=$(printf '%s' "$_vb" | awk '{print $2}')
  # Parse Ri = #planoImg intrinsic PNG size via IHDR decode (first 64 b64 chars)
  _b64=$(grep 'id="planoImg"' "$_html" \
    | grep -oE 'data:image/png;base64,[A-Za-z0-9+/=]+' | head -1 \
    | sed 's/data:image\/png;base64,//' | head -c 64 || true)
  if [ -z "$_b64" ]; then _prow FAIL "Ri: no #planoImg with data:image/png;base64 found"; exit 1; fi
  _ri_w="0"; _ri_h="0"
  read -r _ri_w _ri_h <<< "$(printf '%s' "$_b64" | base64 -d 2>/dev/null | od -An -tu1 -N24 | \
    awk '{for(i=1;i<=NF;i++)a[++n]=$i} END{
      if(n<24){print "0 0";exit}
      if(a[1]!=137||a[2]!=80||a[3]!=78||a[4]!=71||a[5]!=13||a[6]!=10||a[7]!=26||a[8]!=10){print "0 0";exit}
      if(a[13]!=73||a[14]!=72||a[15]!=68||a[16]!=82){print "0 0";exit}
      print a[17]*16777216+a[18]*65536+a[19]*256+a[20]" "a[21]*16777216+a[22]*65536+a[23]*256+a[24]
    }' || true)" || true
  if [ -z "$_ri_w" ] || [ "$_ri_w" -eq 0 ] || [ -z "$_ri_h" ] || [ "$_ri_h" -eq 0 ]; then
    _prow FAIL "Ri: #planoImg PNG header invalid or unreadable"; exit 1
  fi
  PFAIL=0
  # Rc == Ri and Rc == Rv (cross-multiplication: a/b==c/d iff a*d==b*c)
  if ! _ceq "$_rc_w" "$_rc_h" "$_ri_w" "$_ri_h"; then
    _prow FAIL "Rc(${_rc_w}/${_rc_h}) != Ri(${_ri_w}/${_ri_h}): IMG_W/IMG_H vs intrinsic PNG"; PFAIL=1
  fi
  if ! _ceq "$_rc_w" "$_rc_h" "$_rv_w" "$_rv_h"; then
    _prow FAIL "Rv(${_rv_w}/${_rv_h}) != Rc(${_rc_w}/${_rc_h}): viewBox vs IMG_W/IMG_H"; PFAIL=1
  fi
  # Every numeric aspect-ratio must equal Rc; auto is exempt; anything else -> FAIL
  while IFS= read -r _ar_ln; do
    [ -n "$_ar_ln" ] || continue
    _ar_raw=$(printf '%s' "$_ar_ln" | grep -oE 'aspect-ratio:[[:space:]]*[^;{}]+' \
      | sed 's/aspect-ratio:[[:space:]]*//' | sed 's/[[:space:]]*$//' || true)
    [ -n "$_ar_raw" ] || continue
    _ar_v=$(printf '%s' "$_ar_raw" | tr -d ' \t')
    [ "$_ar_v" != "auto" ] || continue
    case "$_ar_v" in
      [0-9]*/[0-9]*)
        _ar_n="${_ar_v%%/*}"; _ar_d="${_ar_v##*/}"
        case "$_ar_n" in ''|*[!0-9]*) _prow FAIL "unparseable aspect-ratio: ${_ar_v}"; PFAIL=1; continue ;; esac
        case "$_ar_d" in ''|*[!0-9]*) _prow FAIL "unparseable aspect-ratio: ${_ar_v}"; PFAIL=1; continue ;; esac
        if ! _ceq "$_ar_n" "$_ar_d" "$_rc_w" "$_rc_h"; then
          _prow FAIL "aspect-ratio ${_ar_v} != Rc(${_rc_w}/${_rc_h})"; PFAIL=1
        fi ;;
      *) _prow FAIL "unparseable aspect-ratio: ${_ar_v}"; PFAIL=1 ;;
    esac
  done <<< "$(grep -E 'aspect-ratio:' "$_html" || true)"
  if [ "$PFAIL" -eq 0 ]; then
    _prow PASS "Rc=${_rc_w}/${_rc_h} Ri=${_ri_w}/${_ri_h} Rv=${_rv_w}/${_rv_h} — all agree"
  fi
  exit "$PFAIL"
fi

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
  local jar="$1" pd hits f
  [ -n "$SRC" ] || { row SKIP facets "$jar" "no --src"; return 0; }
  pd=$(profile_dir "$jar")
  [ -d "$pd/src" ] || { row SKIP facets "$jar" "no $pd/src"; return 0; }
  # D9b: prune dot-dirs (e.g. .deploy-baseline) so stale snapshots are never scanned.
  hits=""
  while IFS= read -r f; do
    h=$(grep -nE 'BFacets\.make\(BFacets\.(MIN|MAX), *-?[0-9]' "$f" 2>/dev/null | head -1 || true)
    if [ -n "$h" ]; then hits="$f:$h"; break; fi
  done < <(find "$pd/src" -type d -name '.*' -prune -o -name '*.java' -print)
  if [ -z "$hits" ]; then row PASS facets "$jar" "no raw-number MIN/MAX facet under $pd/src"; return 0; fi
  row FAIL facets "$jar" "raw-number MIN/MAX facet (wrap in BDouble.make): $hits"; return 1
}
check_facet_presence() {
  # D5 (campaign 8 PR4): WARN when an OPERATOR numeric @NiagaraProperty or newProperty slot
  # has no facets key at all, or when a setpoint/count-like slot lacks UNITS/PRECISION.
  # Presence-only: never reads a facet value (MIN=0 is a real MIN — slots with a facets key
  # are not flagged, even if only MIN is set). WARN never changes exit code (K20).
  # D9b: dot-dirs pruned.
  local jar="$1" pd warned=0 f hit
  [ -n "$SRC" ] || { row SKIP facets-req "$jar" "no --src"; return 0; }
  pd=$(profile_dir "$jar")
  [ -d "$pd/src" ] || { row SKIP facets-req "$jar" "no $pd/src"; return 0; }
  while IFS= read -r f; do
    while IFS= read -r hit; do
      [ -n "$hit" ] || continue
      row WARN facets-req "$jar" "$hit"
      warned=1
    done < <(awk -v FILE="$f" '
      {lines[NR]=$0}
      END {
        delete seen
        # pass 1: @NiagaraProperty annotations (handle single-line and multi-line)
        for (l=1; l<=NR; l++) {
          if (lines[l] !~ /@NiagaraProperty/) continue
          s = lines[l]; lstart = l
          if (lines[l] !~ /\)/) {
            for (j=l+1; j<=NR && j<=l+40; j++) {
              s = s " " lines[j]
              if (lines[j] ~ /^[[:space:]]*\)[[:space:]]*$/) break
            }
          }
          nm = ""
          if (match(s, /name[[:space:]]*=[[:space:]]*"[^"]*"/)) {
            nm = substr(s, RSTART, RLENGTH)
            sub(/name[[:space:]]*=[[:space:]]*"/, "", nm); sub(/".*/, "", nm)
          }
          if (nm == "" || (nm in seen)) continue
          typ = ""
          if (match(s, /type[[:space:]]*=[[:space:]]*"[^"]*"/)) {
            typ = substr(s, RSTART, RLENGTH)
            sub(/type[[:space:]]*=[[:space:]]*"/, "", typ); sub(/".*/, "", typ)
          }
          has_op = (s ~ /OPERATOR/)
          has_fc = (s ~ /[[:space:],]facets[[:space:]]*=/)
          reason = ""
          if (has_op && !has_fc) {
            if (typ == "" || typ ~ /^(double|float|BDouble|BFloat|BInteger|BLong|BRelTime|BStatusNumeric)$/)
              reason = "OPERATOR numeric without facets (slot=" nm ")"
          }
          if (reason == "" && nm ~ /[Ss]etpoint|Temp|[Ll]imit|[Bb]and|[Pp]si/) {
            if (!(s ~ /UNITS/)) reason = "setpoint-like slot " nm " missing UNITS"
          }
          if (reason == "" && nm ~ /demand|[Cc]ount|stages/) {
            if (!(s ~ /PRECISION/)) reason = "count-like slot " nm " missing PRECISION"
          }
          # Mark ALL annotation-defined slots in seen so pass 2 (newProperty) never
          # double-fires on the same slot (prevents false-positive on non-numeric
          # OPERATOR slots like boolean/BStatusBoolean that pass 1 correctly skips).
          seen[nm]=1
          if (reason != "") print FILE ":" lstart " " reason
        }
        # pass 2: old-style newProperty(... OPERATOR ..., value, null) with NO @NiagaraProperty
        for (l=1; l<=NR; l++) {
          if (lines[l] !~ /newProperty\(/ || lines[l] !~ /OPERATOR/) continue
          s = lines[l]
          for (j=l+1; j<=NR && j<=l+5 && s !~ /;/; j++) s = s " " lines[j]
          if (s !~ /, *null *[);]/) continue
          nm = s
          sub(/.*Property[[:space:]]+/, "", nm)
          sub(/[[:space:]]*=.*/, "", nm)
          gsub(/[[:space:]]/, "", nm)
          if (nm !~ /^[A-Za-z_][A-Za-z0-9_]*$/ || (nm in seen)) continue
          seen[nm]=1
          print FILE ":" l " OPERATOR numeric null facets (slot=" nm ")"
        }
      }
    ' "$f")
  done < <(find "$pd/src" -type d -name '.*' -prune -o -name '*.java' -print)
  [ "$warned" -eq 0 ] && row PASS facets-req "$jar" "no missing required facets under $pd/src"
  return 0
}
check_ord_literal() {
  # D5 (campaign 8 PR4): WARN on a Java string literal matching station:|local:|slot:/ under
  # <profile>/src. Three exemptions (D5, K2): (a) file name matches *OrdConstants* carrying a
  # justification comment, (b) the literal is inside a @NiagaraProperty defaultValue= attribute,
  # (c) the file is under srcTest/**. WARN never changes exit code (K20). D9b: dot-dirs pruned.
  local jar="$1" pd warned=0 f fname skip hit
  [ -n "$SRC" ] || { row SKIP ord-literal "$jar" "no --src"; return 0; }
  pd=$(profile_dir "$jar")
  [ -d "$pd/src" ] || { row SKIP ord-literal "$jar" "no $pd/src"; return 0; }
  while IFS= read -r f; do
    case "$f" in */srcTest/*) continue ;; esac
    fname="$(basename "$f" .java)"
    skip=0
    case "$fname" in
      *OrdConstants*)
        if grep -qE '(/\*|//)' "$f" 2>/dev/null; then skip=1; fi ;;
    esac
    [ "$skip" -eq 1 ] && continue
    while IFS= read -r hit; do
      [ -n "$hit" ] || continue
      case "$hit" in *defaultValue*) continue ;; esac
      row WARN ord-literal "$jar" "$f:$hit"
      warned=1
    done < <(grep -nE '"(station:|local:|slot:/)' "$f" 2>/dev/null || true)
  done < <(find "$pd/src" -type d -name '.*' -prune -o -name '*.java' -print)
  [ "$warned" -eq 0 ] && row PASS ord-literal "$jar" "no hardcoded ORD literals under $pd/src"
  return 0
}
check_rc_backup() {  # editor/backup files packaged under rc/ (bloat + servable). WARN by default; FAIL under --strict.
  local jar="$1" names
  names=$(printf '%s\n' "$LIST" | awk -F/ '/^rc\//{ b=$NF; if (b ~ /~$/ || b ~ /\.orig$/ || b ~ /\.bak.*$/) print $0 }' | tr '\n' ' ' | sed 's/ *$//')
  [ -n "$names" ] || return 0   # clean rc/ → silent, no row
  if [ "$STRICT" -eq 1 ]; then row FAIL rcbackup "$jar" "editor/backup files under rc/ (drop them): $names"; return 1; fi
  row WARN rcbackup "$jar" "editor/backup files under rc/ (drop them): $names"; return 0
}
check_palette() {  # an EMPTY module.palette on a module that declares types = nothing to drag in Workbench. WARN by default; FAIL under --strict.
  local jar="$1" pal entries types
  printf '%s\n' "$LIST" | grep -q '^module\.palette$' || { row SKIP palette "$jar" "no module.palette"; return 0; }
  pal=$(unzip -p "$jar" module.palette 2>/dev/null || true)
  # component entries carry a name (<p n="..."/>); the b:Folder root has no n= so it is naturally excluded
  entries=$(printf '%s' "$pal" | grep -c '<p n=' || true)
  types=$(printf '%s' "$MX" | grep -c '<type ' || true)
  if [ "$entries" -eq 0 ] && [ "$types" -ge 1 ]; then
    if [ "$STRICT" -eq 1 ]; then row FAIL palette "$jar" "empty palette but module declares $types type(s) — nothing to drag in Workbench (add one <p n=.../> per exposed type)"; return 1; fi
    row WARN palette "$jar" "empty palette but module declares $types type(s) — nothing to drag in Workbench (add one <p n=.../> per exposed type)"; return 0
  fi
  row PASS palette "$jar" "module.palette has $entries component entries"; return 0
}

for JAR in "${JARS[@]}"; do
  [ -f "$JAR" ] || { echo "verify-module: jar not readable: $JAR" >&2; exit 3; }
  LIST=$(unzip -Z1 "$JAR" 2>/dev/null) || { echo "verify-module: not a zip: $JAR" >&2; exit 3; }
  MX=""; TYPES=""
  if printf '%s\n' "$LIST" | grep -q '^META-INF/module\.xml$'; then
    MX=$(unzip -p "$JAR" META-INF/module.xml)
    TYPES=$(printf '%s' "$MX" | grep -oE '<type [^>]*class="[^"]+"' | sed -E 's/.*class="([^"]+)".*/\1/' || true)
  fi
  for chk in check_bytecode_major check_signed check_types_have_classes check_baja_version check_stored check_type_count check_raw_double_facets check_facet_presence check_ord_literal check_rc_backup check_palette; do
    if "$chk" "$JAR"; then :; else FAILED=1; fi
  done
done
printf 'verify-module: %d passed, %d failed, %d skipped, %d warned -> %s\n' "$NPASS" "$NFAIL" "$NSKIP" "$NWARN" "$([ "$FAILED" -eq 0 ] && echo ALL PASS || echo FAILED)"
exit "$FAILED"
