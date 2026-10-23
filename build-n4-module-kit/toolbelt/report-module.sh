#!/usr/bin/env bash
# report-module.sh — aggregated conformance report for a Niagara N4 module (Campaign 8 PR8).
#
# Composes the campaign-6/8 toolbelt over every profile artifact under <module-root>.
# Invents no new checks — reuses each tool's own row text, prefixed by the artifact name.
# Per artifact (in order): verify-module.sh --src (SKIP if no built jar), slot-coverage.sh
# parse + dup-keys, lint-timers.sh <artifact>/src, --plano only when src/rc/index.html exists,
# lint-delays.sh <artifact>/src (SKIP if no src/), schema-risk.sh (SKIP if no .deploy-baseline/).
# Once per run: triage-console.sh --console-dir <dir> (SKIP row when --console-dir is absent).
# *-ux frontend members (WU4): rc-scan.sh receives --profile <ui_profile> and --legacy from this
# script's own flags (the caller reads the module's BUILD-STATE envelope; no envelope parsing here);
# ESLint with toolbelt/eslint.config.mjs on src/rc js (SKIP "no rc js" when only vendor/ext/min);
# lint-vendor-floor.sh on src/rc/vendor. A missing node/eslint/acorn is one SKIP row naming the
# tool (never a silent pass, never an env fault for the whole report).
# Tool resolution: KIT_ESLINT, else toolbelt/eslint/node_modules/.bin/eslint, else eslint on PATH
# (only at major >= 9, the flat-config major — an older PATH eslint is a SKIP row naming its version);
# node/acorn as in lint-vendor-floor.sh (KIT_NODE, KIT_ACORN). Install once:
# npm install --prefix build-n4-module-kit/toolbelt/eslint   (exact-pinned direct deps; no lockfile
# is committed — see toolbelt/eslint/.gitignore). A frontend member with WARN rows and no FAIL reports
# PASS "no FAIL (N WARN row(s) above)", never "clean".
# [ev: retro dashboard-frontend-standard Δ10] [ev: retro dashboard-frontend-standard Δ16]
# Link endpoints (polish-2026-10-02 P1): lint-link-target-flags.sh <artifact>/src per artifact, with
# --wiring-map <file> when given, else <module-root>/docs/wiring-map.md, else <module-root>/../docs/
# wiring-map.md (the build.sh layout <repo>/<MOD>); a named --wiring-map file that is missing is exit 3.
# The default differs from commissioning-verify.sh (<module-root>/docs only) because the two tools take
# different roots: build.sh passes <repo>/<MOD> here, commissioning-verify takes <repo>; both resolve
# <repo>/docs/wiring-map.md. In a multi-module repo the parent fallback may be another module's map, so
# the chosen map is named in the row detail. Exit 3 from the lint is ERROR + exit 3 here (this script's
# env convention); commissioning-verify maps it to SKIP (its convention). A non-zero exit with no FAIL
# row is a FAIL row, never PASS.
# [ev: retro panccadia-commissioning-lessons Δ1] slot-coverage FAIL lines other than dup-keys (facade
# type without lexicon, empty lexicon) are FAIL rows, not a WARN percentage.
# [ev: retro panccadia-commissioning-lessons Δ9] ADVISORY rows (lint-silent-protection console-only)
# keep their own severity: counted apart, never PASS, never FAIL, never change the verdict.
# [ev: retro alarm-console-design Δ3]
# Advisory members (deferred-lints-2026-10-03): lint-size.sh, lint-license-isoperational-gate.sh,
# lint-set-null-ord.sh and lint-ws-placement.sh [ev: retro ws-placement] on <artifact>/src — their WARN rows are relayed with a PASS verdict row; any non-zero exit
# of an advisory member is an ERROR row carrying the member's first non-row output line (exit 3), never PASS.
#
# Usage: report-module.sh <module-root> [--target-version x.y] [--console-dir <dir>]
#                         [--profile hmi|lan|both|unknown] [--legacy] [--wiring-map <file>]
# Row:     <artifact>  PASS|FAIL|WARN|SKIP|ADVISORY|ERROR  <check>  <detail>   (ERROR = member env fault, not counted; exit 3)
# Summary: report-module: N artifacts · p PASS · f FAIL · w WARN · s SKIP · a ADVISORY  ->  CLEAN|ISSUES|INCOMPLETE
# Floor (BUILD-LOOP §5): verify-module (needs a built jar) and schema-risk (needs <artifact>/.deploy-baseline) never
# read CLEAN when they SKIP — the verdict is INCOMPLETE, exit 4. --first-deploy: the module was never deployed, so
# schema-risk has no baseline to diff (N/A, not a floor gap). --lint-only: a punch-list run that is not the floor
# (no jar expected); floor SKIPs stay rows and are named on stderr. [ev: retro change-tier-time-budgets Δ3]
# Mutation: RM-floor -- dropping the floor-SKIP verdict reads a run that checked neither floor member as CLEAN
# Mutation: RM-floor-schema -- --first-deploy ignored keeps a never-deployed module INCOMPLETE forever
# Mutation: RM-wp-env -- mapping every exit 3 to SKIP no write-path-matrix.md hides a broken write-path scan
# Exit: 0 clean (zero FAIL) · 1 any FAIL · 2 usage · 3 env (member env fault) · 4 INCOMPLETE (a floor check — verify-module, schema-risk — SKIPped; --first-deploy / --lint-only, see below)
# This script is VCS-free by design. version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
# [ev: retro campaign7-report-module]  [ev: retro campaign8-report-integration]
set -u

TOOLBELT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

NPASS=0; NFAIL=0; NWARN=0; NSKIP=0; NADV=0
HAD_FAIL=0; HAD_ENV=0
MODULE_ROOT=""
TARGET_VERSION=""
CONSOLE_DIR=""
UI_PROFILE=""
LEGACY=0
WIRING_MAP=""
FIRST_DEPLOY=0
LINT_ONLY=0
FLOOR_SKIP=0

usage_exit() {
  printf 'usage: report-module.sh <module-root> [--target-version x.y] [--console-dir <dir>] [--profile hmi|lan|both|unknown] [--legacy] [--wiring-map <file>] [--first-deploy] [--lint-only]\n' >&2
  exit 2
}

# Mode note: report-module.sh invokes verify-module.sh with --src, which adds typecount, facets,
# and stored rows beyond the jar-mode baseline. A clean jar-mode run (B798: 7 PASS · 1 FAIL · 1 WARN)
# expands to 9 PASS · 1 FAIL · 1 WARN · 1 SKIP when --src is active — the FAIL and WARN are invariant.
while [ $# -gt 0 ]; do
  case "$1" in
    --target-version)
      [ $# -ge 2 ] || usage_exit
      TARGET_VERSION="$2"; shift 2 ;;
    --console-dir)
      [ $# -ge 2 ] || usage_exit
      CONSOLE_DIR="$2"; shift 2 ;;
    --profile)
      [ $# -ge 2 ] || usage_exit
      case "$2" in hmi|lan|both|unknown) ;; *) usage_exit ;; esac
      UI_PROFILE="$2"; shift 2 ;;
    --legacy) LEGACY=1; shift ;;
    --first-deploy) FIRST_DEPLOY=1; shift ;;
    --lint-only) LINT_ONLY=1; shift ;;
    --wiring-map)
      [ $# -ge 2 ] || usage_exit
      WIRING_MAP="$2"; shift 2 ;;
    --) shift; break ;;
    -*) usage_exit ;;
    *)
      [ -z "$MODULE_ROOT" ] || usage_exit
      MODULE_ROOT="$1"; shift ;;
  esac
done
[ -n "$MODULE_ROOT" ] || usage_exit
[ -d "$MODULE_ROOT" ] || { printf 'report-module: not a directory: %s\n' "$MODULE_ROOT" >&2; exit 3; }
# Wiring map for lint-link-target-flags (LTF1 Table 2 link-in targets): explicit, else discovered.
if [ -n "$WIRING_MAP" ]; then
  [ -f "$WIRING_MAP" ] || { printf 'report-module: --wiring-map file not found: %s\n' "$WIRING_MAP" >&2; exit 3; }
elif [ -f "$MODULE_ROOT/docs/wiring-map.md" ]; then
  WIRING_MAP="$MODULE_ROOT/docs/wiring-map.md"
elif [ -f "$MODULE_ROOT/../docs/wiring-map.md" ]; then
  WIRING_MAP="$MODULE_ROOT/../docs/wiring-map.md"
fi

# emit <artifact> <STATUS> <check> <detail>
emit() {
  printf '%s  %s  %s  %s\n' "$1" "$2" "$3" "$4"
  case "$2" in
    PASS) NPASS=$((NPASS+1)) ;;
    FAIL) NFAIL=$((NFAIL+1)); HAD_FAIL=1 ;;
    WARN) NWARN=$((NWARN+1)) ;;
    SKIP) NSKIP=$((NSKIP+1)) ;;
    ADVISORY) NADV=$((NADV+1)) ;;
  esac
}

# ROW_AWK — the one member-row parser, run ONCE over a member's whole output (never one awk fork per line).
# Selects the lines whose start matches ^(sel) (sel empty: every non-empty line) and prints one
# `<st>|<check>|<detail>` record per selected line. Fields of the row are split on runs of 2+ whitespace:
#   mode site    `ST  check  path:line  reason` -> detail `<basename>:<line>  <reason>`
#        swap    as site, but the member prints `check  ST  ...` (lint-wb-threading); only records whose
#                status (up to a '|', as the shell split reads it) equals keep are printed
#        base    `ST  check  path  reason` -> detail `<basename>  <reason>` (no :line split)
#        nosite  `ST  check  site  reason` -> detail `<reason>` (site dropped)
#        ws      whitespace-split `ST check path detail...` -> detail = the words from field 4, one space apart
#                (verify-module --src / --plano rows)
# Lines matching ^(asw) are relayed with status WARN instead (lint-write-path STALE/DRIFT advisories).
# shellcheck disable=SC2016  # awk program text, expanded by awk, not the shell
ROW_AWK='
function bname(p,   ns, parts) { ns = split(p, parts, "/"); return parts[ns] }
{
  forced = ""
  if (asw != "" && $0 ~ ("^(" asw ")")) forced = "WARN"
  else if (sel == "" ? $0 == "" : $0 !~ ("^(" sel ")")) next
  if (mode == "ws") {
    st = $1; chk = $2; det = ""
    for (i = 4; i <= NF; i++) det = (det ? det " " : "") $i
    print st "|" chk "|" det
    next
  }
  n = split($0, a, /[[:space:]]{2,}/)
  if (mode == "swap") { chk = (n >= 1) ? a[1] : ""; st = (n >= 2) ? a[2] : "" }
  else { st = (n >= 1) ? a[1] : ""; chk = (n >= 2) ? a[2] : "" }
  site = (n >= 3) ? a[3] : ""
  reason = ""
  for (i = 4; i <= n; i++) reason = (reason == "" ? "" : reason "  ") a[i]
  if (mode == "nosite") det = reason
  else if (mode == "base") det = bname(site) "  " reason
  else {
    colon = index(site, ":")
    fp = (colon > 0) ? substr(site, 1, colon - 1) : site
    lno = (colon > 0) ? substr(site, colon + 1) : ""
    det = bname(fp) ":" lno "  " reason
  }
  if (keep != "") { p = index(st, "|"); if ((p ? substr(st, 1, p - 1) : st) != keep) next }
  if (forced != "") st = forced
  print st "|" chk "|" det
}'

# member_rows <artifact> <member-output> <select-ERE> [<mode> [<keep-status> [<as-WARN-ERE>]]] — relay a
# member's rows (ROW_AWK grammar; mode defaults to site) as `<artifact>  <ST>  <check>  <detail>`.
# Sets MR_ROWS (rows relayed), MR_FAILP (rows whose status starts with FAIL), RELAY_FAIL=1 when a FAIL row
# was relayed, and RELAY_WARN / RELAY_ADV (WARN / ADVISORY rows).
member_rows() {
  local _parsed _p _st _r _chk _det
  MR_ROWS=0; MR_FAILP=0; RELAY_FAIL=0; RELAY_WARN=0; RELAY_ADV=0
  _parsed=$(printf '%s\n' "$2" | awk -v sel="$3" -v mode="${4:-site}" -v keep="${5:-}" -v asw="${6:-}" "$ROW_AWK")
  while IFS= read -r _p; do
    [ -z "$_p" ] && continue
    _st="${_p%%|*}"; _r="${_p#*|}"; _chk="${_r%%|*}"; _det="${_r#*|}"
    emit "$1" "$_st" "$_chk" "$_det"
    MR_ROWS=$((MR_ROWS+1))
    case "$_st" in
      FAIL) RELAY_FAIL=1; MR_FAILP=1 ;;
      FAIL*) MR_FAILP=1 ;;
      WARN) RELAY_WARN=$((RELAY_WARN+1)) ;;
      ADVISORY) RELAY_ADV=$((RELAY_ADV+1)) ;;
    esac
  done <<< "$_parsed"
  return 0
}

# relay_rows <artifact> <member-output> — relay `FAIL|WARN|ADVISORY  <check>  <path>:<line>  <reason>` rows
# (rc-scan / eslint / vendor-floor grammar) as `<artifact>  <ST>  <check>  <basename>:<line>  <reason>`.
# Sets RELAY_FAIL=1 when any FAIL row was relayed; RELAY_WARN / RELAY_ADV count the WARN / ADVISORY rows.
relay_rows() {
  member_rows "$1" "$2" 'FAIL|WARN|ADVISORY'
}

# pass_after_relay <artifact> <check> — the member verdict row after relay_rows: nothing when a FAIL row
# was relayed (the FAIL speaks), PASS "clean" only when no row was relayed, else PASS "no FAIL (N WARN
# row(s) above)" — a member with WARN rows is never reported "clean".
pass_after_relay() {
  local _n=$((RELAY_WARN + RELAY_ADV)) _what
  [ "$RELAY_FAIL" -eq 0 ] || return 0
  if [ "$_n" -eq 0 ]; then emit "$1" PASS "$2" "clean"; return 0; fi
  _what="$RELAY_WARN WARN"
  [ "$RELAY_ADV" -gt 0 ] && _what="$_what, $RELAY_ADV ADVISORY"
  if [ "$_n" -eq 1 ]; then _what="$_what row above"; else _what="$_what rows above"; fi
  emit "$1" PASS "$2" "no FAIL ($_what)"
}

# advisory_member <artifact> <check> <exit> <output> — an advisory (WARN-only, run without --strict) lint:
# exit 0 relays its WARN rows, then the pass_after_relay verdict row; ANY other exit (3 env, a crash, an
# unexpected code) is an ERROR row and an env fault, never a silent PASS (deferred-lints-2026-10-03 D2). The
# ERROR detail carries the member's first non-row output line (e.g. "cannot scan <file>: …") so the cause is
# actionable.
advisory_member() {
  local _why
  if [ "$3" -eq 0 ]; then
    relay_rows "$1" "$4"
    pass_after_relay "$1" "$2"
  else
    # the first output line that is not one of the member's own WARN/FAIL/ADVISORY rows (rows printed for earlier
    # files would otherwise hide the cause)
    _why=$(printf '%s\n' "$4" | awk 'NF && $0 !~ /^(WARN|FAIL|ADVISORY)  / { print; exit }')
    emit "$1" ERROR "$2" "env fault (exit $3)${_why:+: $_why}"; HAD_ENV=1
  fi
}

# lint_rows <artifact> <check> <exit> <output> <select-ERE> <verdict> [<mode> [<keep-status>]] — the common
# lint member shape: exit 3 is an ERROR env row (HAD_ENV=1); any other exit relays the selected rows
# (member_rows) and then the verdict row:
#   -     none (the member's own rows speak)
#   rows  PASS "clean" when no row was relayed
#   fail  PASS "clean" when no FAIL row was relayed (WARN rows do not block)
# The caller runs the member itself ("$TOOLBELT/<lint>.sh" in command position: gen-lint-index.sh reads
# those invocations for the INDEX.md Auto column).
lint_rows() {
  if [ "$3" -eq 3 ]; then emit "$1" ERROR "$2" "env fault (exit 3)"; HAD_ENV=1; return 0; fi
  member_rows "$1" "$4" "$5" "${7:-site}" "${8:-}"
  case "$6" in
    rows) [ "$MR_ROWS" -eq 0 ] && emit "$1" PASS "$2" "clean" ;;
    fail) [ "$RELAY_FAIL" -eq 0 ] && emit "$1" PASS "$2" "clean" ;;
  esac
  return 0
}

# have_src <check> — true when <artifact>/src exists; otherwise emits the `SKIP <check> no src/` row.
have_src() {
  [ -d "$ADIR/src" ] && return 0
  emit "$ANAME" SKIP "$1" "no src/"
  return 1
}

# Discover profile artifacts (all immediate subdirectories, sorted)
ARTIFACTS=()
while IFS= read -r _d; do
  ARTIFACTS+=("$_d")
done < <(find "$MODULE_ROOT" -maxdepth 1 -mindepth 1 -type d | sort)

ARTIFACT_COUNT="${#ARTIFACTS[@]}"
if [ "$ARTIFACT_COUNT" -eq 0 ]; then
  printf 'report-module: no artifacts found under %s\n' "$MODULE_ROOT" >&2
  exit 3
fi

for ADIR in "${ARTIFACTS[@]}"; do
  ANAME="$(basename "$ADIR")"

  # ----------------------------------------------------------------
  # 1. verify-module.sh --src  (SKIP if no built jar)
  # ----------------------------------------------------------------
  JAR=$(find "$ADIR/build/libs" -maxdepth 1 -name '*.jar' 2>/dev/null | head -1)
  if [ -z "$JAR" ]; then
    # verify-module is a floor check (BUILD-LOOP §5): no built jar means it did not run — INCOMPLETE, not CLEAN.
    emit "$ANAME" SKIP verify-module "no built jar under build/libs (floor check not run)"
    FLOOR_SKIP=1
  else
    VM_ARGS=()
    [ -n "$TARGET_VERSION" ] && VM_ARGS+=("--target-version" "$TARGET_VERSION")
    VM_ARGS+=("--src" "$MODULE_ROOT")
    vm_exit=0
    vm_out=$("$TOOLBELT/verify-module.sh" "${VM_ARGS[@]}" "$JAR" 2>&1) || vm_exit=$?
    if [ "$vm_exit" -eq 3 ]; then
      emit "$ANAME" ERROR verify-module "env fault (exit 3)"; HAD_ENV=1
    else
      # Row format: STATUS(%-4s)  check(%-9s)  path  detail -- whitespace fields, detail = fields 4..
      member_rows "$ANAME" "$vm_out" 'PASS|FAIL|SKIP|WARN' ws
    fi
  fi

  # ----------------------------------------------------------------
  # 2. slot-coverage.sh parse + dup-keys
  # ----------------------------------------------------------------
  MIXIN="$ADIR/module-include.xml"
  LEX="$ADIR/module.lexicon"
  if [ -f "$MIXIN" ] && [ -f "$LEX" ]; then
    cov_exit=0
    cov_out=$("$TOOLBELT/slot-coverage.sh" "$MIXIN" "$LEX" 2>&1) || cov_exit=$?
    if [ "$cov_exit" -eq 3 ]; then
      emit "$ANAME" ERROR slot-coverage "env fault (exit 3)"; HAD_ENV=1
    else
      # dup-keys: slot-coverage emits "slot-coverage: FAIL dup-keys: <key>" per dup (upgraded from WARN in A1/B792)
      dup_ct=$(printf '%s\n' "$cov_out" | grep -c '^slot-coverage: FAIL dup-keys:' || true)
      if [ "$dup_ct" -gt 0 ]; then
        emit "$ANAME" FAIL dup-keys "$dup_ct"
      else
        emit "$ANAME" PASS dup-keys "0"
      fi
      # Other slot-coverage FAIL lines (facade type without lexicon, empty lexicon) are FAIL rows —
      # the pct row below stays informational. [ev: retro panccadia-commissioning-lessons Δ9]
      while IFS= read -r _ln; do
        case "$_ln" in
          'slot-coverage: FAIL dup-keys:'*) ;;
          'slot-coverage: FAIL '*) emit "$ANAME" FAIL slot-coverage "${_ln#slot-coverage: FAIL }" ;;
        esac
      done <<< "$cov_out"
      # coverage: pct= line, e.g. "100.0 (type-set)" or "50.0" or "N/A"
      pct_line=$(printf '%s\n' "$cov_out" | grep '^pct=' | head -1)
      pct_val="${pct_line#pct=}"   # strip leading "pct="
      pct_num="${pct_val%% *}"     # strip trailing " (type-set)" label if present
      if [ "$pct_num" = "N/A" ]; then
        emit "$ANAME" SKIP slot-coverage "N/A (no types declared)"
      else
        pct_int="${pct_num%%.*}"
        miss_line=$(printf '%s\n' "$cov_out" | grep '^missing=' | head -1)
        miss_val="${miss_line#missing=}"
        if [ "$pct_int" -ge 100 ]; then
          emit "$ANAME" PASS slot-coverage "${pct_num}%"
        else
          if [ -n "$miss_val" ]; then
            emit "$ANAME" WARN slot-coverage "${pct_num}% (missing ${miss_val})"
          else
            emit "$ANAME" WARN slot-coverage "${pct_num}%"
          fi
        fi
      fi
    fi
  fi

  # ----------------------------------------------------------------
  # 3. lint-timers.sh <artifact>/src
  # ----------------------------------------------------------------
  if [ -d "$ADIR/src" ]; then
    lt_exit=0
    lt_out=$("$TOOLBELT/lint-timers.sh" "$ADIR/src" 2>&1) || lt_exit=$?
    if [ "$lt_exit" -eq 3 ]; then
      emit "$ANAME" ERROR lint-timers "env fault (exit 3)"; HAD_ENV=1
    else
      while IFS= read -r _ln; do
        [ -z "$_ln" ] && continue
        case "$_ln" in
          PASS*|FAIL*)
            # Row: STATUS  check  /path/file.java: message
            _st="${_ln%%  *}"
            _rest="${_ln#*  }"           # check  /path/file.java: msg
            _chk="${_rest%%  *}"
            _detail="${_rest#*  }"       # /path/file.java: msg
            _fp="${_detail%%:*}"         # /path/file.java
            _msg="${_detail#*: }"        # message after ": "
            # basename without a fork per row; basename(1) only where ${_fp##*/} differs (empty, trailing /, leading -)
            case "$_fp" in
              ''|*/|-*) _bn="$(basename "$_fp")" ;;
              *) _bn="${_fp##*/}" ;;
            esac
            emit "$ANAME" "$_st" "$_chk" "${_bn}: ${_msg}"
          ;;
        esac
      done <<< "$lt_out"
    fi
  fi

  # ----------------------------------------------------------------
  # 4. --plano: only when src/rc/index.html exists
  # ----------------------------------------------------------------
  PLANO_HTML="$ADIR/src/rc/index.html"
  if [ -f "$PLANO_HTML" ]; then
    pl_exit=0
    pl_out=$("$TOOLBELT/verify-module.sh" --plano "$PLANO_HTML" 2>&1) || pl_exit=$?
    if [ "$pl_exit" -eq 3 ]; then
      emit "$ANAME" ERROR plano "env fault (exit 3)"; HAD_ENV=1
    else
      # Row: STATUS(%-4s)  plano(%-9s)  path  detail
      member_rows "$ANAME" "$pl_out" 'PASS|FAIL|SKIP|WARN' ws
    fi
  fi

  # ----------------------------------------------------------------
  # 4b. lint-lexicon-ascii.sh <artifact> (D: lexicon-ascii, SKIP if no *.lexicon)
  #     Checks every *.lexicon under the artifact dir for non-ASCII bytes.
  #     Niagara reads lexicons as Latin-1; UTF-8 accents cause on-station mojibake.
  # ----------------------------------------------------------------
  _HAS_LEX=$(find "$ADIR" -maxdepth 2 -name '*.lexicon' -print -quit 2>/dev/null || true)
  if [ -z "$_HAS_LEX" ]; then
    emit "$ANAME" SKIP lint-lexicon-ascii "no *.lexicon under $ANAME"
  else
    la_exit=0
    la_out=$("$TOOLBELT/lint-lexicon-ascii.sh" "$ADIR" 2>&1) || la_exit=$?
    if [ "$la_exit" -eq 3 ]; then
      emit "$ANAME" ERROR lint-lexicon-ascii "env fault (exit 3)"; HAD_ENV=1
    else
      _la_had_fail=0
      while IFS= read -r _ln; do
        [ -z "$_ln" ] && continue
        case "$_ln" in
          FAIL*)
            _st="${_ln%%  *}"
            _rest="${_ln#*  }"
            _chk="${_rest%%  *}"
            _det="${_rest#*  }"
            emit "$ANAME" "$_st" "$_chk" "$_det"
            _la_had_fail=1
          ;;
        esac
      done <<< "$la_out"
      [ "$_la_had_fail" -eq 0 ] && emit "$ANAME" PASS lint-lexicon-ascii "all lexicon files are ASCII-clean"
    fi
  fi

  # ----------------------------------------------------------------
  # 5. lint-delays.sh <artifact>/src (Campaign 8 PR8; D9b: SKIP if no src/)
  # ----------------------------------------------------------------
  if have_src lint-delays; then
    # Row: STATUS  lint-delays  /path/file.java:N  reason... (fields separated by two or more spaces)
    _o=$("$TOOLBELT/lint-delays.sh" "$ADIR/src" 2>&1)
    lint_rows "$ANAME" lint-delays "$?" "$_o" 'FAIL|WARN' -
  fi

  # ----------------------------------------------------------------
  # 5b. lint-demand-scope.sh <artifact>/src (Campaign 9 PR2; SKIP if no src/)
  #     WARN-only lint: a control-decision method reading a process variable
  #     with no demand-shaped input in scope. Exit 0 = WARN-only; WARN rows
  #     aggregate as WARN (not FAIL) in the report; usage error = ERROR.
  # ----------------------------------------------------------------
  if have_src lint-demand-scope; then
    _o=$("$TOOLBELT/lint-demand-scope.sh" "$ADIR/src" 2>&1)
    lint_rows "$ANAME" lint-demand-scope "$?" "$_o" WARN -
  fi

  # ----------------------------------------------------------------
  # 5.5. lint-silent-protection.sh <artifact>/src (Campaign 9 PR3; D9b: SKIP if no src/)
  # ----------------------------------------------------------------
  if have_src lint-silent-protection; then
    # ADVISORY console-only keeps its own severity [ev: retro alarm-console-design Δ3]
    _o=$("$TOOLBELT/lint-silent-protection.sh" "$ADIR/src" 2>&1)
    lint_rows "$ANAME" lint-silent-protection "$?" "$_o" 'FAIL|WARN|ADVISORY' -
  fi

  # ----------------------------------------------------------------
  # 5.5b. lint-link-target-flags.sh [--wiring-map <map>] <artifact>/src (FAIL; SKIP if no src/)
  #       polish-2026-10-02 P1 [ev: retro panccadia-commissioning-lessons Δ1]
  # ----------------------------------------------------------------
  if [ -d "$ADIR/src" ]; then
    LTF_ARGS=()
    [ -n "$WIRING_MAP" ] && LTF_ARGS+=("--wiring-map" "$WIRING_MAP")
    ltf_exit=0
    # ${A[@]+"${A[@]}"}: an empty "${A[@]}" is an unbound-variable error under set -u on bash < 4.4.
    ltf_out=$("$TOOLBELT/lint-link-target-flags.sh" ${LTF_ARGS[@]+"${LTF_ARGS[@]}"} "$ADIR/src" 2>&1) || ltf_exit=$?
    if [ "$ltf_exit" -eq 3 ]; then
      emit "$ANAME" ERROR lint-link-target-flags "env fault (exit 3)"; HAD_ENV=1
    else
      # Name the chosen map in the Table 2 reason so a false FAIL from the parent-dir fallback is traceable.
      # "(wiring-map Table 2)" is a contract with the LTF1 reason text of lint-link-target-flags.sh
      # (`why`); RM43 pins it. Pattern and replacement are quoted: under bash 5.2 patsub_replacement an
      # unquoted & in the replacement would expand to the matched text. [polish-2026-10-02 P1c]
      [ -n "$WIRING_MAP" ] && ltf_out=${ltf_out//"(wiring-map Table 2)"/"(wiring-map Table 2: $WIRING_MAP)"}
      relay_rows "$ANAME" "$ltf_out"
      if [ "$RELAY_FAIL" -eq 0 ] && [ "$ltf_exit" -ne 0 ]; then
        # Fail closed: a non-zero exit with no FAIL row (127 = lint missing, a crash, an unexpected row
        # format) is never PASS. [polish-2026-10-02 P1b, R3-ltf-fail-open-exit]
        emit "$ANAME" FAIL lint-link-target-flags \
          "exited $ltf_exit with no FAIL row (crash or unexpected output) -- run lint-link-target-flags.sh manually"
      elif [ "$RELAY_FAIL" -eq 0 ]; then
        _ltf_map="no wiring map"; [ -n "$WIRING_MAP" ] && _ltf_map="wiring map: $WIRING_MAP"
        emit "$ANAME" PASS lint-link-target-flags "clean ($_ltf_map)"
      fi
    fi
  else
    emit "$ANAME" SKIP lint-link-target-flags "no src/"
  fi

  # ----------------------------------------------------------------
  # 5.6. lint-ext-writable-shape.sh <artifact>/src (Campaign 9 PR10; D9b: SKIP if no src/)
  # ----------------------------------------------------------------
  if have_src lint-ext-writable-shape; then
    _o=$("$TOOLBELT/lint-ext-writable-shape.sh" "$ADIR/src" 2>&1)
    lint_rows "$ANAME" lint-ext-writable-shape "$?" "$_o" 'FAIL|WARN' -
  fi

  # ----------------------------------------------------------------
  # 5.7. lint-no-system-out.sh <artifact>/src (FAIL; SKIP if no src/)
  # ----------------------------------------------------------------
  if have_src lint-no-system-out; then
    _o=$("$TOOLBELT/lint-no-system-out.sh" "$ADIR/src" 2>&1)
    lint_rows "$ANAME" lint-no-system-out "$?" "$_o" FAIL rows
  fi

  # ----------------------------------------------------------------
  # 5.8. lint-clock-zero-floor.sh <artifact>/src (WARN; SKIP if no src/)
  # ----------------------------------------------------------------
  if have_src lint-clock-zero-floor; then
    _o=$("$TOOLBELT/lint-clock-zero-floor.sh" "$ADIR/src" 2>&1)
    lint_rows "$ANAME" lint-clock-zero-floor "$?" "$_o" WARN -
  fi

  # ----------------------------------------------------------------
  # 5.9. lint-null-context-write.sh <artifact>/src (WARN; SKIP if no src/)
  # ----------------------------------------------------------------
  if have_src lint-null-context-write; then
    _o=$("$TOOLBELT/lint-null-context-write.sh" "$ADIR/src" 2>&1)
    lint_rows "$ANAME" lint-null-context-write "$?" "$_o" WARN -
  fi

  # ----------------------------------------------------------------
  # 5.10. lint-bql-string-concat.sh <artifact>/src (WARN; SKIP if no src/)
  # ----------------------------------------------------------------
  if have_src lint-bql-string-concat; then
    _o=$("$TOOLBELT/lint-bql-string-concat.sh" "$ADIR/src" 2>&1)
    lint_rows "$ANAME" lint-bql-string-concat "$?" "$_o" WARN -
  fi

  # ----------------------------------------------------------------
  # 5.11. lint-arbitrary-ord.sh <artifact>/src (WARN; SKIP if no src/)
  # ----------------------------------------------------------------
  if have_src lint-arbitrary-ord; then
    _o=$("$TOOLBELT/lint-arbitrary-ord.sh" "$ADIR/src" 2>&1)
    lint_rows "$ANAME" lint-arbitrary-ord "$?" "$_o" WARN -
  fi

  # ----------------------------------------------------------------
  # 5.12. lint-se-display.sh <artifact>/src (FAIL; only -se artifacts; SKIP if no src/)
  # ----------------------------------------------------------------
  case "$ANAME" in
    *-se)
      if have_src lint-se-display; then
        _o=$("$TOOLBELT/lint-se-display.sh" "$ADIR/src" 2>&1)
        lint_rows "$ANAME" lint-se-display "$?" "$_o" FAIL rows
      fi
    ;;
  esac

  # ----------------------------------------------------------------
  # 5.13. lint-jasmine-ux.sh <artifact> (WARN; only -ux artifacts)
  # ----------------------------------------------------------------
  case "$ANAME" in
    *-ux)
      _o=$("$TOOLBELT/lint-jasmine-ux.sh" "$ADIR" 2>&1)
      lint_rows "$ANAME" lint-jasmine-ux "$?" "$_o" WARN - nosite
    ;;
  esac

  # ----------------------------------------------------------------
  # 5.14–5.17. rt-specific static-source lints
  #    5.14 lint-subscribe-without-unsubscribe  WARN (called without --strict)
  #    5.15 lint-recovery-path                  FAIL
  #    5.16 lint-config-sanity                  FAIL (CS1/CS2) / WARN (CS3)
  #    5.17 lint-status-parity                  WARN (called without --strict)
  #    All gated to *-rt artifacts; SKIP if no src/.
  # ----------------------------------------------------------------
  case "$ANAME" in
    *-rt)
      # 5.14 lint-subscribe-without-unsubscribe
      if have_src lint-subscribe-without-unsubscribe; then
        _o=$("$TOOLBELT/lint-subscribe-without-unsubscribe.sh" "$ADIR/src" 2>&1)
        lint_rows "$ANAME" lint-subscribe-without-unsubscribe "$?" "$_o" WARN -
      fi

      # 5.15 lint-recovery-path (FAIL; exit 1 = FAIL rows)
      if have_src lint-recovery-path; then
        _o=$("$TOOLBELT/lint-recovery-path.sh" "$ADIR/src" 2>&1)
        lint_rows "$ANAME" lint-recovery-path "$?" "$_o" FAIL rows
      fi

      # 5.16 lint-config-sanity (CS1/CS2 FAIL; CS3 WARN)
      if have_src lint-config-sanity; then
        _o=$("$TOOLBELT/lint-config-sanity.sh" "$ADIR/src" 2>&1)
        lint_rows "$ANAME" lint-config-sanity "$?" "$_o" 'FAIL|WARN' fail
      fi

      # 5.17 lint-status-parity (WARN; called without --strict)
      if have_src lint-status-parity; then
        _o=$("$TOOLBELT/lint-status-parity.sh" "$ADIR/src" 2>&1)
        lint_rows "$ANAME" lint-status-parity "$?" "$_o" WARN -
      fi
    ;;
  esac

  # ----------------------------------------------------------------
  # 5.18–5.19. ux-specific static-source lints
  #    5.18 lint-servlet  FAIL rows block / WARN rows advisory;
  #                       self-SKIPs when no BWebServlet; needs src/
  #    5.19 rc-scan       FAIL; only when src/rc/ exists
  #    Both gated to *-ux artifacts.
  # ----------------------------------------------------------------
  case "$ANAME" in
    *-ux)
      # 5.18 lint-servlet
      if have_src lint-servlet; then
        srv_exit=0
        srv_out=$("$TOOLBELT/lint-servlet.sh" "$ADIR/src" 2>&1) || srv_exit=$?
        if [ "$srv_exit" -eq 3 ]; then
          emit "$ANAME" ERROR lint-servlet "env fault (exit 3)"; HAD_ENV=1
        else
          member_rows "$ANAME" "$srv_out" 'FAIL|WARN'
          # INFO rows (api-response-headers) prove a servlet was scanned; the report shows it as PASS.
          if [ "$MR_ROWS" -eq 0 ] && [[ $'\n'"$srv_out" != *$'\n'INFO* ]]; then
            emit "$ANAME" SKIP lint-servlet "no BWebServlet"
          elif [ "$RELAY_FAIL" -eq 0 ]; then
            emit "$ANAME" PASS lint-servlet "clean"
          fi
        fi
      fi

      # 5.19 rc-scan (FAIL; only when src/rc/ exists)
      if [ -d "$ADIR/src/rc" ]; then
        rcs_exit=0
        RCS_ARGS=("$ADIR")
        [ -n "$UI_PROFILE" ] && RCS_ARGS+=("--profile" "$UI_PROFILE")
        [ "$LEGACY" -eq 1 ] && RCS_ARGS+=("--legacy")
        rcs_out=$("$TOOLBELT/rc-scan.sh" "${RCS_ARGS[@]}" 2>&1) || rcs_exit=$?
        if [ "$rcs_exit" -eq 3 ]; then
          emit "$ANAME" ERROR rc-scan "env fault (exit 3)"; HAD_ENV=1
        else
          relay_rows "$ANAME" "$rcs_out"
          pass_after_relay "$ANAME" rc-scan
        fi
      else
        emit "$ANAME" SKIP rc-scan "no src/rc/"
      fi

      # 5.19b ESLint (kit flat config) on the artifact's own rc js (WU4)
      if [ -d "$ADIR/src/rc" ]; then
        _own_js=$(find "$ADIR/src/rc" \( -path '*/vendor' -o -path '*/ext' -o -name '.*' \) -prune -o \
          -type f -name '*.js' ! -name '*.min.js' -print 2>/dev/null | head -1)
        ESLINT_BIN="${KIT_ESLINT:-}"
        if [ -z "$ESLINT_BIN" ] && [ -x "$TOOLBELT/eslint/node_modules/.bin/eslint" ]; then
          ESLINT_BIN="$TOOLBELT/eslint/node_modules/.bin/eslint"
        fi
        _esl_old=""
        if [ -z "$ESLINT_BIN" ]; then
          # PATH fallback: an eslint on PATH is used only when its --version reports major >= 9 (the kit flat
          # config eslint.config.mjs needs it); any other or unreadable version is a SKIP row below. The probe is
          # bounded (KIT_ESLINT_PROBE_TIMEOUT seconds, default 15) when timeout(1) exists: a wedged wrapper is an
          # "unknown version" SKIP, never a hung report (#226 R4-002).
          ESLINT_BIN="$(command -v eslint 2>/dev/null || true)"
          if [ -n "$ESLINT_BIN" ] && [ -n "$_own_js" ]; then
            _esl_to=()
            command -v timeout >/dev/null 2>&1 && _esl_to=(timeout "${KIT_ESLINT_PROBE_TIMEOUT:-15}")
            _esl_ver=$("${_esl_to[@]}" "$ESLINT_BIN" --version 2>/dev/null | head -1) || true
            _esl_major=$(printf '%s' "$_esl_ver" | sed -n 's/^v\{0,1\}\([0-9][0-9]*\)\..*/\1/p')
            if [ -z "$_esl_major" ] || [ "$_esl_major" -lt 9 ]; then _esl_old="${_esl_ver:-unknown version}"; fi
          fi
        fi
        if [ -z "$_own_js" ]; then
          emit "$ANAME" SKIP eslint "no rc js"
        elif [ -z "$ESLINT_BIN" ] || [ ! -x "$ESLINT_BIN" ]; then
          emit "$ANAME" SKIP eslint "unavailable: eslint not installed (npm install --prefix toolbelt/eslint, or set KIT_ESLINT)"
        elif [ -n "$_esl_old" ]; then
          emit "$ANAME" SKIP eslint "unavailable: eslint on PATH is $_esl_old; the kit flat config needs ESLint >= 9 (npm install --prefix toolbelt/eslint, or set KIT_ESLINT)"
        else
          esl_exit=0
          esl_out=$(cd "$ADIR/src/rc" && "$ESLINT_BIN" --config "$TOOLBELT/eslint.config.mjs" \
            --format "$TOOLBELT/eslint/rows-formatter.cjs" --no-warn-ignored . 2>&1) || esl_exit=$?
          if [ "$esl_exit" -gt 1 ]; then
            emit "$ANAME" ERROR eslint "env fault (exit $esl_exit)"; HAD_ENV=1
          else
            relay_rows "$ANAME" "$esl_out"
            [ "$esl_exit" -eq 0 ] && pass_after_relay "$ANAME" eslint
            [ "$RELAY_FAIL" -eq 0 ] && [ "$esl_exit" -eq 1 ] && emit "$ANAME" FAIL eslint "exit 1 with no parsable row"
          fi
        fi
      fi

      # 5.19c lint-vendor-floor.sh on src/rc/vendor (WU4)
      if [ -d "$ADIR/src/rc/vendor" ]; then
        lvf_exit=0
        lvf_out=$("$TOOLBELT/lint-vendor-floor.sh" "$ADIR/src/rc" 2>&1) || lvf_exit=$?
        case "$lvf_exit" in
          0|1)
            relay_rows "$ANAME" "$lvf_out"
            pass_after_relay "$ANAME" vendor-floor
            ;;
          4)
            _lvf_why=$(printf '%s\n' "$lvf_out" | sed -n 's/.*unavailable: /unavailable: /p' | head -1)
            emit "$ANAME" SKIP vendor-floor "${_lvf_why:-unavailable}"
            ;;
          *)
            emit "$ANAME" ERROR vendor-floor "env fault (exit $lvf_exit)"; HAD_ENV=1
            ;;
        esac
      fi
    ;;
  esac

  # ----------------------------------------------------------------
  # 5.20. lint-wb-threading.sh (WARN; only *-wb artifacts with src/)
  #       Row format: <check>  WARN  <file>:<line>  <detail>
  #       (check is field 1, status is field 2 — ROW_AWK swap mode swaps them for emit)
  # ----------------------------------------------------------------
  case "$ANAME" in
    *-wb)
      if have_src lint-wb-threading; then
        _o=$("$TOOLBELT/lint-wb-threading.sh" "$ADIR/src" 2>&1)
        lint_rows "$ANAME" lint-wb-threading "$?" "$_o" '' - swap WARN
      fi
    ;;
  esac

  # ----------------------------------------------------------------
  # 5.21. lint-no-md5-credential-digest.sh (WARN; SKIP if no src/)
  # ----------------------------------------------------------------
  if have_src lint-no-md5-credential-digest; then
    _o=$("$TOOLBELT/lint-no-md5-credential-digest.sh" "$ADIR/src" 2>&1)
    lint_rows "$ANAME" lint-no-md5-credential-digest "$?" "$_o" WARN rows
  fi

  # ----------------------------------------------------------------
  # 5.22. lint-wb-file-chooser.sh (WARN; only *-wb artifacts with src/)
  #        A BWbFieldEditor.dialog(...BOrd.NULL) without BComponentChooser/targetType
  #        opens the file-system chooser (C:\), useless in a station context.
  # ----------------------------------------------------------------
  case "$ANAME" in
    *-wb)
      if have_src lint-wb-file-chooser; then
        _o=$("$TOOLBELT/lint-wb-file-chooser.sh" "$ADIR/src" 2>&1)
        lint_rows "$ANAME" lint-wb-file-chooser "$?" "$_o" WARN -
      fi
    ;;
  esac

  # ----------------------------------------------------------------
  # 5.23. lint-size.sh <artifact>/src (advisory WARN; SKIP if no src/) — deferred-lints D2
  # ----------------------------------------------------------------
  if have_src lint-size; then
    _o=$("$TOOLBELT/lint-size.sh" "$ADIR/src" 2>&1)
    advisory_member "$ANAME" lint-size "$?" "$_o"
  fi

  # ----------------------------------------------------------------
  # 5.24. lint-license-isoperational-gate.sh <artifact>/src (advisory WARN; SKIP if no src/) — deferred-lints D3
  # ----------------------------------------------------------------
  if have_src lint-license-isoperational-gate; then
    _o=$("$TOOLBELT/lint-license-isoperational-gate.sh" "$ADIR/src" 2>&1)
    advisory_member "$ANAME" lint-license-isoperational-gate "$?" "$_o"
  fi

  # ----------------------------------------------------------------
  # 5.25. lint-set-null-ord.sh <artifact>/src (advisory WARN; SKIP if no src/) — deferred-lints D4
  # ----------------------------------------------------------------
  if have_src lint-set-null-ord; then
    _o=$("$TOOLBELT/lint-set-null-ord.sh" "$ADIR/src" 2>&1)
    advisory_member "$ANAME" lint-set-null-ord "$?" "$_o"
  fi

  # ----------------------------------------------------------------
  # 5.26. lint-ws-placement.sh <artifact>/src (advisory WARN; SKIP if no src/) — ws-placement W3
  # ----------------------------------------------------------------
  if have_src lint-ws-placement; then
    _o=$("$TOOLBELT/lint-ws-placement.sh" "$ADIR/src" 2>&1)
    advisory_member "$ANAME" lint-ws-placement "$?" "$_o"
  fi

  # ----------------------------------------------------------------
  # 6. schema-risk.sh <artifact>/.deploy-baseline <artifact>
  #    (Campaign 8 PR8 / D9a; SKIP if no .deploy-baseline/ snapshot)
  # ----------------------------------------------------------------
  SR_BASELINE="$ADIR/.deploy-baseline"
  if [ ! -d "$SR_BASELINE" ]; then
    if [ "$FIRST_DEPLOY" -eq 1 ]; then
      emit "$ANAME" SKIP schema-risk "N/A (--first-deploy: never deployed, no baseline to diff)"
    else
      # schema-risk is a floor check: a deployed module with no baseline snapshot was never diffed.
      emit "$ANAME" SKIP schema-risk "no .deploy-baseline (floor check not run; extract the deployed baseline, build-verify.md § Schema-risk deploy workflow, or --first-deploy)"
      FLOOR_SKIP=1
    fi
  else
    sr_exit=0
    "$TOOLBELT/schema-risk.sh" "$SR_BASELINE" "$ADIR" >/dev/null 2>&1 || sr_exit=$?
    case "$sr_exit" in
      0) emit "$ANAME" PASS schema-risk "verdict=SAFE" ;;
      1) emit "$ANAME" WARN schema-risk "verdict=LOSSY" ;;
      2) emit "$ANAME" FAIL schema-risk "verdict=OUTAGE" ;;  # D9a: OUTAGE -> FAIL, never ERROR
      3|4) emit "$ANAME" ERROR schema-risk "env fault (exit ${sr_exit})"; HAD_ENV=1 ;;
      *) emit "$ANAME" ERROR schema-risk "unexpected exit ${sr_exit}"; HAD_ENV=1 ;;
    esac
  fi

done

# ----------------------------------------------------------------
# 8. lint-agent-on-shape.sh <module-root> — once per run (FAIL)
# ----------------------------------------------------------------
_o=$("$TOOLBELT/lint-agent-on-shape.sh" "$MODULE_ROOT" 2>&1)
lint_rows "(module)" lint-agent-on-shape "$?" "$_o" FAIL rows

# ----------------------------------------------------------------
# 9. lint-uberjar-api-conflict.sh <module-root> — once per run (WARN)
# ----------------------------------------------------------------
_o=$("$TOOLBELT/lint-uberjar-api-conflict.sh" "$MODULE_ROOT" 2>&1)
lint_rows "(module)" lint-uberjar-api-conflict "$?" "$_o" WARN - base

# ----------------------------------------------------------------
# 10. lint-structure.sh <module-root> — once per run (FAIL)
# ----------------------------------------------------------------
_o=$("$TOOLBELT/lint-structure.sh" "$MODULE_ROOT" 2>&1)
lint_rows "(module)" lint-structure "$?" "$_o" 'FAIL|WARN' fail

# ----------------------------------------------------------------
# 11. lint-write-path.sh <module-root> — once per run (FAIL on uncovered;
#     STALE/DRIFT advisory rows mapped to WARN, never blocking)
# ----------------------------------------------------------------
lwp_exit=0
lwp_out=$("$TOOLBELT/lint-write-path.sh" "$MODULE_ROOT" 2>&1) || lwp_exit=$?
if [ "$lwp_exit" -eq 3 ] && [[ "$lwp_out" == *"no write-path-matrix.md found"* ]]; then
  emit "(module)" SKIP lint-write-path "no write-path-matrix.md"
elif [ "$lwp_exit" -eq 3 ]; then
  # any OTHER exit 3 (a find error, usage) is an env fault: the write-path scan did not complete
  emit "(module)" ERROR lint-write-path "env fault (exit 3): $(printf '%s\n' "$lwp_out" | head -n 1)"; HAD_ENV=1
else
  # STALE/DRIFT advisory rows: exit-0 but surface as WARN so they appear in the report
  member_rows "(module)" "$lwp_out" FAIL site '' 'STALE|DRIFT'
  [ "$MR_FAILP" -eq 0 ] && emit "(module)" PASS lint-write-path "clean"
fi

# ----------------------------------------------------------------
# 12. lint-bundled-jar-class-version.sh <module-root> — once per run (FAIL)
#     Checks vendored/bundled ext-jars in the source tree for Java 9+ bytecode
#     (class-file major > 52). A bundled jar with major > 52 throws a raw
#     UnsupportedClassVersionError at module load, bypassing catch(Exception).
# ----------------------------------------------------------------
_o=$("$TOOLBELT/lint-bundled-jar-class-version.sh" "$MODULE_ROOT" 2>&1)
lint_rows "(module)" lint-bundled-jar-class-version "$?" "$_o" FAIL rows base

# ----------------------------------------------------------------
# 7. triage-console.sh — once per run (Campaign 8 PR8; D9)
#    Gated on --console-dir; SKIP row when flag is absent.
# ----------------------------------------------------------------
if [ -z "$CONSOLE_DIR" ]; then
  printf '(run)  SKIP  triage-console  no --console-dir\n'
  NSKIP=$((NSKIP+1))
else
  tc_exit=0
  tc_out=$("$TOOLBELT/triage-console.sh" --console-dir "$CONSOLE_DIR" 2>&1) || tc_exit=$?
  if [ "$tc_exit" -eq 3 ]; then
    printf '(run)  ERROR  triage-console  env fault (exit 3)\n'
    HAD_ENV=1
  else
    _tc_had_row=0
    while IFS= read -r _ln; do
      [ -z "$_ln" ] && continue
      printf '%s\n' "$_ln"
      _tc_had_row=1
      case "$_ln" in
        FAIL*) NFAIL=$((NFAIL+1)); HAD_FAIL=1 ;;
        WARN*) NWARN=$((NWARN+1)) ;;
        PASS*) NPASS=$((NPASS+1)) ;;
        SKIP*) NSKIP=$((NSKIP+1)) ;;
      esac
    done <<< "$tc_out"
    if [ "$_tc_had_row" -eq 0 ]; then
      printf '(run)  PASS  triage-console  0 rows\n'
      NPASS=$((NPASS+1))
    fi
  fi
fi

# Summary
VERDICT="CLEAN"
# A floor SKIP (verify-module / schema-risk did not run) is INCOMPLETE, never CLEAN — unless this run is
# declared --lint-only (a punch-list pass that is not the floor).
[ "$FLOOR_SKIP" -eq 1 ] && [ "$LINT_ONLY" -eq 0 ] && VERDICT="INCOMPLETE"
[ "$HAD_FAIL" -eq 1 ] && VERDICT="ISSUES"
[ "$LINT_ONLY" -eq 1 ] && [ "$FLOOR_SKIP" -eq 1 ] && printf 'report-module: --lint-only — floor checks (verify-module, schema-risk) were not run and are not part of this verdict\n' >&2
_s_sfx="$([ "$ARTIFACT_COUNT" -eq 1 ] && printf '' || printf 's')"
printf 'report-module: %d artifact%s · %d PASS · %d FAIL · %d WARN · %d SKIP · %d ADVISORY  ->  %s\n' \
  "$ARTIFACT_COUNT" "$_s_sfx" "$NPASS" "$NFAIL" "$NWARN" "$NSKIP" "$NADV" "$VERDICT"

[ "$HAD_ENV" -eq 1 ] && exit 3
[ "$HAD_FAIL" -eq 1 ] && exit 1
[ "$VERDICT" = "INCOMPLETE" ] && exit 4
exit 0
