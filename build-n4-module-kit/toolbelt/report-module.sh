#!/usr/bin/env bash
# report-module.sh — aggregated conformance report for a Niagara N4 module (Campaign 7 PR8).
#
# Composes the campaign-6 toolbelt over every profile artifact under <module-root>.
# Invents no new checks — reuses each tool's own row text, prefixed by the artifact name.
# Per artifact (in order): verify-module.sh --src (SKIP if no built jar), slot-coverage.sh
# parse + dup-keys, lint-timers.sh <artifact>/src, --plano only when src/rc/index.html exists.
#
# Usage: report-module.sh <module-root> [--target-version x.y]
# Row:     <artifact>  PASS|FAIL|WARN|SKIP  <check>  <detail>
# Summary: report-module: N artifacts · p PASS · f FAIL · w WARN · s SKIP  ->  CLEAN|ISSUES
# Exit: 0 clean (zero FAIL) · 1 any FAIL · 3 env (member env fault)
# This script is VCS-free by design. version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
set -u

TOOLBELT="$(cd "${BASH_SOURCE[0]%/*}" && pwd)"

NPASS=0; NFAIL=0; NWARN=0; NSKIP=0
HAD_FAIL=0; HAD_ENV=0
MODULE_ROOT=""
TARGET_VERSION=""

usage_exit() {
  printf 'usage: report-module.sh <module-root> [--target-version x.y]\n' >&2
  exit 2
}

while [ $# -gt 0 ]; do
  case "$1" in
    --target-version)
      [ $# -ge 2 ] || usage_exit
      TARGET_VERSION="$2"; shift 2 ;;
    --) shift; break ;;
    -*) usage_exit ;;
    *)
      [ -z "$MODULE_ROOT" ] || usage_exit
      MODULE_ROOT="$1"; shift ;;
  esac
done
[ -n "$MODULE_ROOT" ] || usage_exit
[ -d "$MODULE_ROOT" ] || { printf 'report-module: not a directory: %s\n' "$MODULE_ROOT" >&2; exit 3; }

# emit <artifact> <STATUS> <check> <detail>
emit() {
  printf '%s  %s  %s  %s\n' "$1" "$2" "$3" "$4"
  case "$2" in
    PASS) NPASS=$((NPASS+1)) ;;
    FAIL) NFAIL=$((NFAIL+1)); HAD_FAIL=1 ;;
    WARN) NWARN=$((NWARN+1)) ;;
    SKIP) NSKIP=$((NSKIP+1)) ;;
  esac
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
  if [ -n "$JAR" ]; then
    VM_ARGS=()
    [ -n "$TARGET_VERSION" ] && VM_ARGS+=("--target-version" "$TARGET_VERSION")
    VM_ARGS+=("--src" "$MODULE_ROOT")
    vm_exit=0
    vm_out=$("$TOOLBELT/verify-module.sh" "${VM_ARGS[@]}" "$JAR" 2>&1) || vm_exit=$?
    if [ "$vm_exit" -eq 3 ]; then
      emit "$ANAME" ERROR verify-module "env fault (exit 3)"; HAD_ENV=1
    else
      while IFS= read -r _ln; do
        case "$_ln" in
          PASS*|FAIL*|SKIP*|WARN*)
            # Row format: STATUS(%-4s)  check(%-9s)  path  detail
            # awk: $1=status $2=check $3=path $4..=detail
            _parsed=$(printf '%s' "$_ln" | awk '{
              st=$1; chk=$2; det="";
              for(i=4;i<=NF;i++) det=(det? det " " : "") $i;
              print st "|" chk "|" det
            }')
            _st="${_parsed%%|*}"; _r="${_parsed#*|}"; _chk="${_r%%|*}"; _det="${_r#*|}"
            emit "$ANAME" "$_st" "$_chk" "$_det"
          ;;
        esac
      done <<< "$vm_out"
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
      # dup-keys: slot-coverage emits "slot-coverage: WARN dup-keys: <key>" per dup
      dup_ct=$(printf '%s\n' "$cov_out" | grep -c '^slot-coverage: WARN dup-keys:' || true)
      if [ "$dup_ct" -gt 0 ]; then
        emit "$ANAME" FAIL dup-keys "$dup_ct"
      else
        emit "$ANAME" PASS dup-keys "0"
      fi
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
            _bn="$(basename "$_fp")"
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
      while IFS= read -r _ln; do
        case "$_ln" in
          PASS*|FAIL*)
            # Row: STATUS(%-4s)  plano(%-9s)  path  detail
            _parsed=$(printf '%s' "$_ln" | awk '{
              st=$1; chk=$2; det="";
              for(i=4;i<=NF;i++) det=(det? det " " : "") $i;
              print st "|" chk "|" det
            }')
            _st="${_parsed%%|*}"; _r="${_parsed#*|}"; _chk="${_r%%|*}"; _det="${_r#*|}"
            emit "$ANAME" "$_st" "$_chk" "$_det"
          ;;
        esac
      done <<< "$pl_out"
    fi
  fi

done

# Summary
VERDICT="CLEAN"
[ "$HAD_FAIL" -eq 1 ] && VERDICT="ISSUES"
_s_sfx="$([ "$ARTIFACT_COUNT" -eq 1 ] && printf '' || printf 's')"
printf 'report-module: %d artifact%s · %d PASS · %d FAIL · %d WARN · %d SKIP  ->  %s\n' \
  "$ARTIFACT_COUNT" "$_s_sfx" "$NPASS" "$NFAIL" "$NWARN" "$NSKIP" "$VERDICT"

[ "$HAD_ENV" -eq 1 ] && exit 3
[ "$HAD_FAIL" -eq 1 ] && exit 1
exit 0
