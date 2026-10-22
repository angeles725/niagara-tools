#!/usr/bin/env bash
# shellcheck disable=SC2329,SC2317
# why: the check_* functions are invoked indirectly by name from the loop `"$chk" "$JAR"` (file-wide directive).
# SC2329 (never invoked) and SC2317 (unreachable) are the same false positive — shellcheck can't see the
# indirect `"$chk" "$JAR"` call, so it flags the function bodies. Both disabled file-wide.
# verify-module.sh — THE gate for built N4 module jars. A jar that has not passed it does not go to a station.
#
# Checks (each guards a failure seen on a real build — build-n4-module-kit/retros/):
#   bytecode    every .class has major 52 (Java 8). 65 = built with the default JDK 21.          [default]
#               META-INF/versions/** (multi-release) is skipped and counted: Java 8 never loads it.
#   signed      a META-INF/<ALIAS>.SF with its <ALIAS>.RSA|.DSA|.EC block; the alias is reported. [default]
#               NOTE: `signed` checks PRESENCE only — the signer's cert may not be in the
#               station's trust store. A self-signed/untrusted-CA jar PASSES this gate yet a
#               station enforcing cert-chain validation loads it UNSIGNED (`Could not validate
#               cert chain`). Signed = jar has a *.SF block; trusted = station-side policy.
#               `[ev: corpus B800 §800.8]`
#               folded as code: verify-module.sh signed-check (cert-chain trust caveat, D8, R7.6).
#   types       every <type class=...> in META-INF/module.xml exists as a .class in the jar.     [default]
#   baja        module.xml baja vendorVersion <= --target-version (a 4.15 jar is rejected by 4.14). [--target-version]
#   stored      zero Deflated entries — required only when the jar must be re-signed in Workbench. [--stored]
#   typecount   packaged <type> count == <module-dir>/<jar-basename>/module-include.xml count.    [--src]
#   facets      no BFacets.make(BFacets.MIN|MAX, <raw number>) under <module-dir>/<jar-basename>/src. [--src]
#   facets-req  OPERATOR numeric slot without a facets key (WARN); setpoint/count-like slot without UNITS/PRECISION (WARN). [--src]
#               BFacets.makeNumeric(<unit>, ...) satisfies UNITS and any makeNumeric( satisfies PRECISION;
#               a precision-only makeNumeric(<int>) still WARNs missing UNITS, also when the precision is a
#               named constant (makeNumeric(PRECISION)). [ev: retro comppan-fase2-amps-alarms Δ1]
#               Mutation: VMN1 -- drop the makeNumeric(<unit>, ...) branch from the UNITS test -> VMN1 WARNs again
#               Mutation: VMN3 -- accept ANY makeNumeric( as a unit -> the precision-only makeNumeric(1) stops WARNing
#               Mutation: VMN4 -- accept a lone identifier argument as a unit -> makeNumeric(PRECISION) stops WARNing
#   ord-literal Java string literal matching station:|local:|slot:/ under src (WARN); exempt: *OrdConstants*+comment, defaultValue=, srcTest/**. [--src]
#   rcbackup    no editor/backup files (*~ *.orig *.bak*) packaged under rc/ — WARN, or FAIL under --strict. [default]
#   palette     a module that declares types must not ship an EMPTY module.palette (nothing to drag in
#               Workbench) — WARN, or FAIL under --strict; SKIP when the jar has no module.palette.     [default]
#               palette-type FAILs a t= that is not "<symbol>:<Type>" with an m=-declared symbol; palette-root
#               WARNs every baja Folder (root or nested, any alias); palette-parse FAILs an unterminated
#               comment. Entries: palette_scan below.
#   subscription-leak a TypeSubscriber subclass with no stopped() override — unsubscribeAll() is never
#               called; typeSubscriptionMap holds strong refs, leaking until station restart — WARN.    [--src]
#   slot-wall   more than 15 @NiagaraProperty on one class (flat slot wall) — WARN.              [--src]
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
    # If no plano overlay markers at all (no IMG_W/IMG_H, no id="zonas"/id="plano"),
    # this is a self-contained SPA (e.g. a 3D three.js dashboard) — not a plano module.
    # SKIP instead of FAIL so non-plano modules are not penalised. (D: plano-skip, B797-retro)
    _has_overlay=0
    grep -qE 'id="zonas"|id="plano"' "$_html" 2>/dev/null && _has_overlay=1
    if [ "$_has_overlay" -eq 0 ]; then
      _prow SKIP "not a plano dashboard (3D/self-contained SPA) — IMG_W/IMG_H and overlay markers absent"; exit 0
    fi
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
  # META-INF/versions/** is skipped, like lint-bundled-jar-class-version.sh: a Java 8 runtime never loads
  # multi-release classes (Jackson 2.21 ships major-55 ones and runs on N4). [ev: retro n4-course-signed-jars-gate-fix Δ3]
  local jar="$1" n=0 bad=0 mr=0 c m first="" ign=""
  while IFS= read -r c; do
    [ -n "$c" ] || continue
    case "$c" in META-INF/versions/*) mr=$((mr+1)); continue ;; esac
    n=$((n+1))
    m=$(unzip -p "$jar" "$c" | od -An -t u1 -j6 -N2 | awk '{print $2}')
    [ "$m" = "52" ] || { bad=$((bad+1)); [ -n "$first" ] || first="$c major=$m"; }
  done <<< "$(printf '%s\n' "$LIST" | grep '\.class$' || true)"
  [ "$mr" -eq 0 ] || ign="; $mr multi-release classes ignored"
  if [ "$n" -eq 0 ]; then row FAIL bytecode "$jar" "no .class entries${ign:+ outside META-INF/versions/}$ign"; return 1; fi
  if [ "$bad" -eq 0 ]; then row PASS bytecode "$jar" "$n classes, all major 52$ign"; return 0; fi
  row FAIL bytecode "$jar" "$bad of $n classes not major 52 (expected Java 8); first: $first$ign"; return 1
}
check_signed() {
  # The signature files are named after the signing ALIAS (NIAGARA4 default profile, CODESIGN course profile,
  # SERVER1 Honeywell); the station decides by verification mode + cert chain, never by the entry name.
  # A .SF counts only with its signature block (.RSA/.DSA/.EC). [ev: retro n4-course-signed-jars-gate-fix Δ1]
  # `sigs` is the META-INF signature entries (a here-string, never a pipe into grep: tests/pipefail-grep.bats),
  # matched in-shell so each alias lookup is one case pattern, not one grep per extension.
  local sigs sf alias ext ok="" orphan=""
  sigs=$'\n'$(grep -E '^META-INF/[^/]+\.(SF|RSA|DSA|EC)$' <<< "$LIST" || true)$'\n'
  while IFS= read -r sf; do
    case "$sf" in META-INF/*.SF) ;; *) continue ;; esac
    alias="${sf#META-INF/}"; alias="${alias%.SF}"
    for ext in RSA DSA EC; do
      case "$sigs" in *$'\n'"META-INF/$alias.$ext"$'\n'*) ok="$ok${ok:+, }$alias"; continue 2 ;; esac
    done
    orphan="$orphan${orphan:+, }$sf"
  done <<< "$sigs"
  if [ -n "$ok" ]; then row PASS signed "$1" "signed by alias $ok (META-INF/<ALIAS>.SF + signature block)"; return 0; fi
  if [ -n "$orphan" ]; then row FAIL signed "$1" "$orphan without its .RSA/.DSA/.EC signature block (jar not signed)"; return 1; fi
  row FAIL signed "$1" "no META-INF/*.SF (jar not signed)"; return 1
}
check_types_have_classes() {
  local jar="$1" missing=0 tn=0 cls f firstmiss=""
  [ -n "$MX" ] || { row FAIL types "$jar" "META-INF/module.xml missing (not an N4 module jar)"; return 1; }
  while IFS= read -r cls; do
    [ -n "$cls" ] || continue; tn=$((tn+1))
    f="$(printf '%s' "$cls" | tr . /).class"
    grep -qx "$f" <<< "$LIST" || { missing=$((missing+1)); [ -n "$firstmiss" ] || firstmiss="$cls"; }
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
# src_or_skip <check> <jar>: the shared opening of every --src check. Sets the CALLER's local `pd` to
# the jar's profile dir; without --src it prints the SKIP row and returns 1, so a caller writes
# `src_or_skip <check> "$jar" || return 0`.
src_or_skip() {
  [ -n "$SRC" ] || { row SKIP "$1" "$2" "no --src"; return 1; }
  pd=$(profile_dir "$2")
}
# src_dir_or_skip <check> <jar>: src_or_skip, then SKIP (return 1) when <pd>/src is not a directory.
src_dir_or_skip() {
  src_or_skip "$1" "$2" || return 1
  [ -d "$pd/src" ] || { row SKIP "$1" "$2" "no $pd/src"; return 1; }
}
# java_files <dir>: every *.java under <dir>, in find order. D9b: dot-dirs (e.g. .deploy-baseline) are
# pruned so stale snapshots are never scanned. Fail-open like the walk it replaced: find errors are not checked.
java_files() { find "$1" -type d -name '.*' -prune -o -name '*.java' -print; }
check_type_count() {
  local jar="$1" pd inc tn sn
  src_or_skip typecount "$jar" || return 0
  inc="$pd/module-include.xml"
  [ -f "$inc" ] || { row SKIP typecount "$jar" "no $inc"; return 0; }
  tn=$(printf '%s\n' "$TYPES" | grep -c . || true); sn=$(grep -c '<type ' "$inc" || true)
  if [ "$tn" = "$sn" ]; then row PASS typecount "$jar" "jar declares $tn types == module-include.xml"; return 0; fi
  row FAIL typecount "$jar" "jar declares $tn types, $inc declares $sn (stale build or missing slotomatic)"; return 1
}
check_raw_double_facets() {
  local jar="$1" pd hits f
  src_dir_or_skip facets "$jar" || return 0
  hits=""
  while IFS= read -r f; do
    h=$(grep -nE 'BFacets\.make\(BFacets\.(MIN|MAX), *-?[0-9]' "$f" 2>/dev/null | head -1 || true)
    if [ -n "$h" ]; then hits="$f:$h"; break; fi
  done < <(java_files "$pd/src")
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
  src_dir_or_skip facets-req "$jar" || return 0
  while IFS= read -r f; do
    while IFS= read -r hit; do
      [ -n "$hit" ] || continue
      row WARN facets-req "$jar" "$hit"
      warned=1
    done < <(awk -v FILE="$f" '
      {lines[NR]=$0}
      # mn_has_unit(s): 1 when the first BFacets.makeNumeric( call in s carries a unit. makeNumeric(<int>)
      # is the only one-argument overload, so a lone argument is a precision (a literal or a named
      # constant such as PRECISION) unless it names a unit (BUnit..., *UNIT*); a first argument that
      # starts with a letter followed by more arguments is the (<unit>, <precision>, ...) form.
      # [polish-2026-10-02 P2a, #199 WU6a has_nu]
      function mn_has_unit(s,    i, c, d, n, first, commas) {
        if (!match(s, /makeNumeric\(/)) return 0
        s = substr(s, RSTART + RLENGTH); n = length(s); d = 0; commas = 0; first = ""
        for (i = 1; i <= n; i++) {
          c = substr(s, i, 1)
          if (c == "(") d++
          else if (c == ")") { if (d == 0) break; d-- }
          else if (c == "," && d == 0) commas++
          if (commas == 0) first = first c
        }
        gsub(/^[[:space:]]+|[[:space:]]+$/, "", first)
        if (first !~ /^[A-Za-z_]/) return 0
        if (first ~ /[Uu]nit|UNIT/) return 1
        return (commas >= 1)
      }
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
          # BFacets.makeNumeric(<unit>, <precision>, ...) carries a unit without the UNITS token
          # and every makeNumeric overload carries a precision; a precision-only makeNumeric(<int>)
          # (first argument a number) still has no unit. [ev: retro comppan-fase2-amps-alarms Δ1]
          has_nu = mn_has_unit(s)
          has_np = (s ~ /makeNumeric\(/)
          if (reason == "" && nm ~ /[Ss]etpoint|Temp|[Ll]imit|[Bb]and|[Pp]si/) {
            if (!(s ~ /UNITS/) && !has_nu) reason = "setpoint-like slot " nm " missing UNITS"
          }
          if (reason == "" && nm ~ /demand|[Cc]ount|stages/) {
            if (!(s ~ /PRECISION/) && !has_np) reason = "count-like slot " nm " missing PRECISION"
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
  done < <(java_files "$pd/src")
  [ "$warned" -eq 0 ] && row PASS facets-req "$jar" "no missing required facets under $pd/src"
  return 0
}
check_ord_literal() {
  # D5 (campaign 8 PR4): WARN on a Java string literal matching station:|local:|slot:/ under
  # <profile>/src. Three exemptions (D5, K2): (a) file name matches *OrdConstants* carrying a
  # justification comment, (b) the literal is inside a @NiagaraProperty defaultValue= attribute,
  # (c) the file is under srcTest/**. WARN never changes exit code (K20). D9b: dot-dirs pruned.
  local jar="$1" pd warned=0 f fname skip hit
  src_dir_or_skip ord-literal "$jar" || return 0
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
  done < <(java_files "$pd/src")
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
# palette_scan: module.palette text on stdin -> one row per finding: ENTRY (a component entry), FOLDER <t> (a baja
# Folder under any alias), BADTYPE <t> <why> (plus BADENTRY when that element carries n=, so a count of named
# elements still sees it), UNTERM (a <!-- with no -->: the rest of the document is not read). A palette is a BOG document: t= is "<symbol>:<Type>" and the symbol must be declared
# by an m="<symbol>=<module>" on the same or an earlier element — else "Invalid typespec" / "Undeclared module
# symbol" (ValueDocDecoder.java:1071-1073,1091). An entry is a <p> with n= and a non-folder t= in ANY attribute
# order (Workbench saves n= last) and no v= (a <p n v/> child, t+v for a dynamic slot, is a slot override).
# XML comments are dropped first. [ev: retro n4-course-authoring-files-and-libraries Δ3]
# [ev: retro n4-course-handson-slotomatic-authoring Δ6]
palette_scan() {
  awk '
    # attr: the value of attribute a in element text e, or "\001" (a byte no attribute holds) when a is absent
    function attr(e, a,   v) {
      if (!match(e, "[[:space:]]" a "=\"[^\"]*\"")) return "\001"
      v = substr(e, RSTART, RLENGTH); sub(/^[[:space:]][a-z]+="/, "", v); sub(/"$/, "", v); return v
    }
    { doc = doc " " $0 }
    END {
      while ((i = index(doc, "<!--")) > 0) {
        j = index(substr(doc, i + 4), "-->")
        if (!j) { print "UNTERM"; doc = substr(doc, 1, i - 1); break }
        doc = substr(doc, 1, i - 1) substr(doc, i + 4 + j + 2)
      }
      n = split(doc, el, "<")
      for (k = 2; k <= n; k++) {
        e = el[k]; if (e !~ /^p[[:space:]\/>]/) continue
        e = " " substr(e, 2, index(e, ">") ? index(e, ">") - 2 : length(e))
        m = attr(e, "m"); if (m != "\001" && index(m, "=")) mod[substr(m, 1, index(m, "=") - 1)] = substr(m, index(m, "=") + 1)
        t = attr(e, "t"); if (t == "\001") continue
        c = index(t, ":"); sym = substr(t, 1, c - 1); ty = substr(t, c + 1); bad = ""
        if (index(t, ".")) bad = "is a Java class name, not <symbol>:<Type>"
        else if (c <= 1) bad = "has no <symbol>: prefix"
        else if (!(sym in mod)) bad = "uses symbol " sym " that no m=\"" sym "=<module>\" declared"
        if (bad != "") { print "BADTYPE " t " " bad; if (attr(e, "n") != "\001") print "BADENTRY"; continue }
        folder = (mod[sym] == "baja" && (ty == "Folder" || ty == "UnrestrictedFolder"))
        if (mod[sym] == "baja" && ty == "Folder") print "FOLDER " t
        if (!folder && attr(e, "n") != "\001" && attr(e, "v") == "\001") print "ENTRY"
      }
    }'
}
check_palette() {  # an EMPTY module.palette on a module that declares types = nothing to drag in Workbench. WARN by default; FAIL under --strict.
  # palette-type FAILs a malformed t= type spec; palette-parse FAILs an unterminated comment (no palette row then);
  # palette-root WARNs every baja Folder, root or nested, under any alias. [Δ1/B746 §746.1]
  local jar="$1" pal scan entries types folders bad rc=0
  grep -q '^module\.palette$' <<< "$LIST" || { row SKIP palette "$jar" "no module.palette"; return 0; }
  pal=$(unzip -p "$jar" module.palette 2>/dev/null || true)
  scan=$(palette_scan <<< "$pal")
  while IFS= read -r bad; do
    [ -n "$bad" ] || continue
    row FAIL palette-type "$jar" "module.palette t=\"${bad#BADTYPE }\" — write t=\"<symbol>:<Type>\" with m=\"<symbol>=<module>\" declared once [ev: retro n4-course-authoring-files-and-libraries Δ3]"
    rc=1
  done < <(grep '^BADTYPE ' <<< "$scan" || true)
  # Δ1/B746 §746.1: b:Folder is access-controlled (gated by category permission) — engineers without the
  # folder's category permission get "access denied" on expansion. b:UnrestrictedFolder is the idiomatic
  # ungated palette folder, root or nested. WARN independently of the entries count.
  folders=$(sed -n 's/^FOLDER //p' <<< "$scan" | sort -u | sed 's/.*/t="&"/' | paste -sd ' ' -)
  if [ -n "$folders" ]; then
    row WARN palette-root "$jar" "module.palette uses a baja Folder ($folders), root or nested: access-controlled — use the baja UnrestrictedFolder (t=\"b:UnrestrictedFolder\") for ungated palette expansion [ev: corpus B746 §746.1]"
  fi
  if grep -qx 'UNTERM' <<< "$scan"; then
    row FAIL palette-parse "$jar" "module.palette has an unterminated comment (<!-- with no -->): the rest of the document is not read"
    return 1
  fi
  entries=$(grep -c '^ENTRY$' <<< "$scan" || true)
  types=$(printf '%s' "$MX" | grep -c '<type ' || true)
  if [ "$entries" -eq 0 ] && [ "$types" -ge 1 ]; then
    if [ "$STRICT" -eq 1 ]; then row FAIL palette "$jar" "empty palette but module declares $types type(s) — nothing to drag in Workbench (add one <p n=.../> per exposed type)"; return 1; fi
    row WARN palette "$jar" "empty palette but module declares $types type(s) — nothing to drag in Workbench (add one <p n=.../> per exposed type)"; return "$rc"
  fi
  row PASS palette "$jar" "module.palette has $entries component entries"; return "$rc"
}
check_wb_scaffold() {
  # WB-SCAFFOLD1: a -wb jar with 0 .class entries AND 0 palette entries is an
  # empty scaffold — nothing packaged at all. WARN by default; FAIL under --strict.
  # Closes the :245-257 dead angle (the palette check only fires when types >=1;
  # a 0-class 0-palette jar with 0 declared types slips through silently otherwise).
  # Real shape: DashboardPan-wb (pure PX delegation, no compiled types, empty palette).
  # NAMED MUTATION: remove this check -> WB-SCAFFOLD1's WARN vanishes (a 0-class jar
  # reverts to a bytecode FAIL only, which does not carry the wb-scaffold label).
  local jar="$1" base n_classes pal entries
  base=$(basename "$jar" .jar)
  case "$base" in *-wb) ;; *) row SKIP wb-scaffold "$jar" "not a -wb jar"; return 0 ;; esac
  grep -q '^module\.palette$' <<< "$LIST" || { row SKIP wb-scaffold "$jar" "no module.palette"; return 0; }
  n_classes=$(printf '%s\n' "$LIST" | grep -c '\.class$' || true)
  pal=$(unzip -p "$jar" module.palette 2>/dev/null || true)
  entries=$(palette_scan <<< "$pal" | grep -cE '^(ENTRY|BADENTRY)$' || true)   # a named bad-type element is still an entry
  if [ "$n_classes" -eq 0 ] && [ "$entries" -eq 0 ]; then
    if [ "$STRICT" -eq 1 ]; then row FAIL wb-scaffold "$jar" "empty -wb jar: 0 classes, 0 palette entries — nothing packaged (add src or a palette entry)"; return 1; fi
    row WARN wb-scaffold "$jar" "empty -wb jar: 0 classes, 0 palette entries — nothing packaged (add src or a palette entry)"; return 0
  fi
  row PASS wb-scaffold "$jar" "ok ($n_classes classes, $entries palette entries)"
  return 0
}
check_phantom_dep() {
  # WB-DEP1 (--src): a <dependency> declared in META-INF/module.xml that has no
  # corresponding api(":X") or nre(":X") in the profile .gradle.kts is a phantom
  # dependency — it was pulled transitively during build but is not an explicit
  # contract; it may disappear after a Gradle upgrade and cause runtime CNFE.
  # NAMED MUTATION: drop this check -> WB-DEP1's phantom-dep WARN vanishes.
  local jar="$1" pd gkt dep gkt_deps phantom_list
  src_or_skip phantom-dep "$jar" || return 0
  gkt=$(find "$pd" -maxdepth 1 -name '*.gradle.kts' 2>/dev/null | sort | head -1)
  [ -n "$gkt" ] || { row SKIP phantom-dep "$jar" "no *.gradle.kts in $pd"; return 0; }
  [ -n "$MX" ]  || { row SKIP phantom-dep "$jar" "no META-INF/module.xml"; return 0; }
  # Dependency names declared in the gradle.kts (api, nre, or implementation).
  # Two forms are supported:
  #   api(":X")  / nre(":X")  — direct string dep
  #   api(project(":X")) / implementation(project(":X")) — project dep (multi-project layout)
  # Both forms declare the same runtime dependency; only the first was previously matched,
  # causing false phantom-dep WARNs for normal rt->ux client modules. (D: phantom-dep-project, C)
  gkt_deps=$(
    {
      grep -oE '(api|nre|implementation)[[:space:]]*\([[:space:]]*":[^"]*"' "$gkt" \
        | grep -oE '":[^"]*"' | sed 's/^"://;s/"$//'
      grep -oE '(api|nre|implementation)[[:space:]]*\([[:space:]]*project[[:space:]]*\([[:space:]]*":[^"]*"' "$gkt" \
        | grep -oE '":[^"]*"' | sed 's/^"://;s/"$//'
    } | sort -u
  )
  # Dependency names from module.xml <dependency name="X">
  phantom_list=""
  while IFS= read -r dep; do
    [ -z "$dep" ] && continue
    if ! grep -qxF "$dep" <<< "$gkt_deps"; then
      phantom_list="${phantom_list:+$phantom_list, }$dep"
    fi
  done < <(printf '%s' "$MX" | grep -oE '<dependency[[:space:]]+[^>]*\bname="[^"]*"' \
    | grep -oE '\bname="[^"]*"' | sed 's/^name="//;s/"$//' | sort -u)
  if [ -n "$phantom_list" ]; then
    row WARN phantom-dep "$jar" "module.xml dependency not declared in gradle.kts api()/nre(): $phantom_list"
    return 0  # WARN never changes exit code
  fi
  row PASS phantom-dep "$jar" "all module.xml dependencies declared in gradle.kts"
  return 0
}
check_moduletest_present() {
  # MODULETEST1 (--src): an -rt module whose source tree has production Java (src/**.java)
  # but NO moduleTest-include.xml declares zero BTestNg station-integration tests — the exact
  # gap found in our own CompPan-rt/ColdRoomPan-rt (see types/moduleTest.md). WARN by default;
  # FAIL under --strict. Requires --src (moduleTest-include.xml is a source file, not in the jar).
  # NAMED MUTATION: drop this check -> MODULETEST1's WARN vanishes. [ev: corpus B958 §958.4]
  local jar="$1" base pd njava mti
  base=$(basename "$jar" .jar)
  case "$base" in *-rt) ;; *) row SKIP moduletest "$jar" "not an -rt jar"; return 0 ;; esac
  src_or_skip moduletest "$jar" || return 0
  [ -d "$pd" ] || { row SKIP moduletest "$jar" "no source dir $pd"; return 0; }
  njava=$(find "$pd/src" -name '*.java' 2>/dev/null | grep -c . || true)
  [ "$njava" -ge 1 ] || { row SKIP moduletest "$jar" "no production .java under src/"; return 0; }
  mti=$(find "$pd" -name 'moduleTest-include.xml' 2>/dev/null | head -1 || true)
  if [ -z "$mti" ]; then
    if [ "$STRICT" -eq 1 ]; then row FAIL moduletest "$jar" "production .java in src/ but no moduleTest-include.xml — add BTestNg station-integration tests (see types/moduleTest.md)"; return 1; fi
    row WARN moduletest "$jar" "production .java in src/ but no moduleTest-include.xml — add BTestNg station-integration tests (see types/moduleTest.md)"; return 0
  fi
  row PASS moduletest "$jar" "moduleTest-include.xml present"; return 0
}

check_transient_operator() {
  # Δ9 (transient-operator-warn, --src): a @NiagaraProperty slot declared with BOTH
  # Flags.TRANSIENT and Flags.OPERATOR is almost certainly a bug — TRANSIENT means NOT
  # persisted to the .bog; OPERATOR means operator-writable config; together the operator
  # sets a setpoint, the station restarts, and the setpoint silently reverts to default.
  # WARN; SKIP without --src or no src/. D9b: dot-dirs pruned.
  # NAMED MUTATION: drop this check -> TRANSOP1's WARN vanishes.
  # [ev: corpus B755 §755.5, B4 §4.1.2]
  local jar="$1" pd warned=0 f hit
  src_dir_or_skip transient-operator "$jar" || return 0
  while IFS= read -r f; do
    while IFS= read -r hit; do
      [ -n "$hit" ] || continue
      row WARN transient-operator "$jar" "$f:$hit"
      warned=1
    done < <(grep -nE 'TRANSIENT[^;]*OPERATOR|OPERATOR[^;]*TRANSIENT' "$f" 2>/dev/null || true)
  done < <(java_files "$pd/src")
  [ "$warned" -eq 0 ] && row PASS transient-operator "$jar" "no TRANSIENT+OPERATOR slot combination under $pd/src"
  return 0
}
check_compact3_imports() {
  # Δ11 (compact3-imports, --src): rt/ux source that imports java.awt.*, javax.swing.*, or
  # java.sql.* compiles under a full JDK 8 but throws NoClassDefFoundError on the station —
  # the NRE ships the Compact 3 JRE subset, which excludes these packages. wb/se run the
  # full SE JRE and are exempt. WARN; SKIP without --src, no src/, or non-rt/ux profiles.
  # D9b: dot-dirs pruned.
  # NAMED MUTATION: drop this check -> CPT3-1's WARN vanishes.
  # [ev: corpus B756 §756.2]
  local jar="$1" base pd warned=0 f hit
  base=$(basename "$jar" .jar)
  case "$base" in *-rt|*-ux) ;; *) row SKIP compact3-import "$jar" "not an -rt or -ux jar"; return 0 ;; esac
  src_dir_or_skip compact3-import "$jar" || return 0
  while IFS= read -r f; do
    while IFS= read -r hit; do
      [ -n "$hit" ] || continue
      row WARN compact3-import "$jar" "$f:$hit"
      warned=1
    done < <(grep -nE 'import (java\.awt|javax\.swing|java\.sql)\.' "$f" 2>/dev/null || true)
  done < <(java_files "$pd/src")
  [ "$warned" -eq 0 ] && row PASS compact3-import "$jar" "no Compact3 non-safe imports under $pd/src"
  return 0
}
check_cross_module_type() {
  # CROSSMOD1 (--src): a @NiagaraProperty type= value that is a Java FQN whose
  # package is not javax.baja.*, com.tridium.*, com.tridiumx.*, com.honeywell.*,
  # nor the module's own packages (directories present under src/) compiles with
  # compileOnly(files(...)) but fails at station load as Missing class <mod>:<Type>.
  # WARN severity (may be intentional with a correct runtime dep). [ev: corpus B740 §740.2]
  # NAMED MUTATION: drop this check -> CROSSMOD1's cross-module-type WARN vanishes.
  local jar="$1" pd warned=0 t pkg pkg_path
  src_dir_or_skip cross-module-type "$jar" || return 0
  while IFS= read -r t; do
    [ -n "$t" ] || continue
    # FQN: must contain a dot; last segment must start with an uppercase letter
    case "$t" in *.* ) ;; *) continue ;; esac
    case "${t##*.}" in [A-Z]*) ;; *) continue ;; esac
    # Known-safe vendor prefixes
    case "$t" in javax.baja.*|com.tridium.*|com.tridiumx.*|com.honeywell.*) continue ;; esac
    # Own-package check: type's package dir exists under src/ -> not cross-module
    pkg="${t%.*}"; pkg_path="${pkg//.//}"
    [ -d "$pd/src/$pkg_path" ] && continue
    row WARN cross-module-type "$jar" "foreign type in @NiagaraProperty: $t"
    warned=1
  done < <(
    java_files "$pd/src" \
    | while IFS= read -r _f; do
        grep -oE 'type[[:space:]]*=[[:space:]]*"[^"]*"' "$_f" 2>/dev/null || true
      done \
    | sed -E 's/type[[:space:]]*=[[:space:]]*"([^"]*)"/\1/' \
    | sort -u
  )
  [ "$warned" -eq 0 ] && row PASS cross-module-type "$jar" "no cross-module type references in $pd/src"
  return 0
}

check_subscription_leak() {
  # Δ6 (subscription-leak, --src): a class that extends or uses TypeSubscriber but
  # has no stopped() override — subscriptions are never torn down (typeSubscriptionMap
  # holds strong refs per type per event-id on the space) = leak until station restart.
  # WARN (not FAIL: valid architectures exist where the owning space is torn down on stop
  # before the subscriber component). Companion to the existing Clock.Ticket cancel FAIL.
  # SKIP without --src or no src/. D9b: dot-dirs pruned.
  # NAMED MUTATION: drop this check -> SUBLEAK1's WARN row vanishes.
  # [ev: corpus B867 §867.2, B408 §408.4]
  local jar="$1" pd warned=0 f
  src_dir_or_skip subscription-leak "$jar" || return 0
  while IFS= read -r f; do
    grep -qE '(extends[[:space:]]+TypeSubscriber|TypeSubscriber[[:space:]]+)' "$f" 2>/dev/null || continue
    grep -qE 'void[[:space:]]+stopped[[:space:]]*\(' "$f" 2>/dev/null && continue
    row WARN subscription-leak "$jar" "$f: TypeSubscriber without stopped() — unsubscribeAll() never called, space-map leaks until restart"
    warned=1
  done < <(java_files "$pd/src")
  [ "$warned" -eq 0 ] && row PASS subscription-leak "$jar" "no TypeSubscriber without stopped() under $pd/src"
  return 0
}

check_slot_wall() {
  # SLOT-WALL-LINT1 (--src): a single @NiagaraType class with more than 15
  # @NiagaraProperty declarations is a flat slot wall — the property sheet and
  # Link picker sprawl above that count; group concerns into child BComponents
  # instead (logic.md L21). WARN; SKIP without --src or no src/. D9b: dot-dirs pruned.
  # NAMED MUTATION: drop this check -> SLOTWALL1's WARN vanishes.
  # [ev: corpus B737 §B.2-B.3]
  local jar="$1" pd warned=0 f count classname
  src_dir_or_skip slot-wall "$jar" || return 0
  while IFS= read -r f; do
    count=$(grep -c '@NiagaraProperty' "$f" 2>/dev/null || true)
    [ "$count" -gt 15 ] || continue
    classname=$(grep -oE '(public[[:space:]]+)?(class|interface)[[:space:]]+[A-Za-z_][A-Za-z0-9_]*' "$f" 2>/dev/null \
      | head -1 | awk '{print $NF}' || true)
    [ -z "$classname" ] && classname="$(basename "$f" .java)"
    row WARN slot-wall "$jar" "$count properties in class $classname ($f) — compose into child BComponents (logic.md L21)"
    warned=1
  done < <(java_files "$pd/src")
  [ "$warned" -eq 0 ] && row PASS slot-wall "$jar" "no flat slot wall (>15 @NiagaraProperty per class) under $pd/src"
  return 0
}

for JAR in "${JARS[@]}"; do
  [ -f "$JAR" ] || { echo "verify-module: jar not readable: $JAR" >&2; exit 3; }
  LIST=$(unzip -Z1 "$JAR" 2>/dev/null) || { echo "verify-module: not a zip: $JAR" >&2; exit 3; }
  MX=""; TYPES=""
  if grep -q '^META-INF/module\.xml$' <<< "$LIST"; then
    MX=$(unzip -p "$JAR" META-INF/module.xml)
    TYPES=$(printf '%s' "$MX" | grep -oE '<type [^>]*class="[^"]+"' | sed -E 's/.*class="([^"]+)".*/\1/' || true)
  fi
  for chk in check_bytecode_major check_signed check_types_have_classes check_baja_version check_stored check_type_count check_raw_double_facets check_facet_presence check_ord_literal check_rc_backup check_palette check_wb_scaffold check_phantom_dep check_moduletest_present check_cross_module_type check_transient_operator check_compact3_imports check_subscription_leak check_slot_wall; do
    if "$chk" "$JAR"; then :; else FAILED=1; fi
  done
done
printf 'verify-module: %d passed, %d failed, %d skipped, %d warned -> %s\n' "$NPASS" "$NFAIL" "$NSKIP" "$NWARN" "$([ "$FAILED" -eq 0 ] && echo ALL PASS || echo FAILED)"
exit "$FAILED"
