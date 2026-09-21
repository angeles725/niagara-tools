#!/usr/bin/env bash
# report-module.sh — aggregated conformance report for a Niagara N4 module (Campaign 8 PR8).
#
# Composes the campaign-6/8 toolbelt over every profile artifact under <module-root>.
# Invents no new checks — reuses each tool's own row text, prefixed by the artifact name.
# Per artifact (in order): verify-module.sh --src (SKIP if no built jar), slot-coverage.sh
# parse + dup-keys, lint-timers.sh <artifact>/src, --plano only when src/rc/index.html exists,
# lint-delays.sh <artifact>/src (SKIP if no src/), schema-risk.sh (SKIP if no .deploy-baseline/).
# Once per run: triage-console.sh --console-dir <dir> (SKIP row when --console-dir is absent).
#
# Usage: report-module.sh <module-root> [--target-version x.y] [--console-dir <dir>]
# Row:     <artifact>  PASS|FAIL|WARN|SKIP  <check>  <detail>
# Summary: report-module: N artifacts · p PASS · f FAIL · w WARN · s SKIP  ->  CLEAN|ISSUES
# Exit: 0 clean (zero FAIL) · 1 any FAIL · 3 env (member env fault)
# This script is VCS-free by design. version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
# [ev: retro campaign7-report-module]  [ev: retro campaign8-report-integration]
set -u

TOOLBELT="$(cd "${BASH_SOURCE[0]%/*}" && pwd)"

NPASS=0; NFAIL=0; NWARN=0; NSKIP=0
HAD_FAIL=0; HAD_ENV=0
MODULE_ROOT=""
TARGET_VERSION=""
CONSOLE_DIR=""

usage_exit() {
  printf 'usage: report-module.sh <module-root> [--target-version x.y] [--console-dir <dir>]\n' >&2
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
      # dup-keys: slot-coverage emits "slot-coverage: FAIL dup-keys: <key>" per dup (upgraded from WARN in A1/B792)
      dup_ct=$(printf '%s\n' "$cov_out" | grep -c '^slot-coverage: FAIL dup-keys:' || true)
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
          PASS*|FAIL*|SKIP*|WARN*)
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
  if [ -d "$ADIR/src" ]; then
    ld_exit=0
    ld_out=$("$TOOLBELT/lint-delays.sh" "$ADIR/src" 2>&1) || ld_exit=$?
    if [ "$ld_exit" -eq 3 ]; then
      emit "$ANAME" ERROR lint-delays "env fault (exit 3)"; HAD_ENV=1
    else
      while IFS= read -r _ln; do
        [ -z "$_ln" ] && continue
        case "$_ln" in
          FAIL*|WARN*)
            # Row: STATUS  lint-delays  /path/file.java:N  reason...
            # Fields are separated by two or more spaces (TAB-aligned in output).
            _parsed=$(printf '%s' "$_ln" | awk '{
              n = split($0, a, /[[:space:]]{2,}/)
              st = (n >= 1) ? a[1] : ""
              chk = (n >= 2) ? a[2] : ""
              site = (n >= 3) ? a[3] : ""
              reason = ""
              for (i = 4; i <= n; i++) reason = (reason == "" ? "" : reason "  ") a[i]
              colon = index(site, ":")
              fp = (colon > 0) ? substr(site, 1, colon - 1) : site
              lno = (colon > 0) ? substr(site, colon + 1) : ""
              nsplit = split(fp, parts, "/"); bn = parts[nsplit]
              print st "|" chk "|" bn ":" lno "  " reason
            }')
            _st="${_parsed%%|*}"
            _r="${_parsed#*|}"
            _chk="${_r%%|*}"
            _det="${_r#*|}"
            emit "$ANAME" "$_st" "$_chk" "$_det"
          ;;
        esac
      done <<< "$ld_out"
    fi
  else
    emit "$ANAME" SKIP lint-delays "no src/"
  fi

  # ----------------------------------------------------------------
  # 5b. lint-demand-scope.sh <artifact>/src (Campaign 9 PR2; SKIP if no src/)
  #     WARN-only lint: a control-decision method reading a process variable
  #     with no demand-shaped input in scope. Exit 0 = WARN-only; WARN rows
  #     aggregate as WARN (not FAIL) in the report; usage error = ERROR.
  # ----------------------------------------------------------------
  if [ -d "$ADIR/src" ]; then
    ds_exit=0
    ds_out=$("$TOOLBELT/lint-demand-scope.sh" "$ADIR/src" 2>&1) || ds_exit=$?
    if [ "$ds_exit" -eq 3 ]; then
      emit "$ANAME" ERROR lint-demand-scope "env fault (exit 3)"; HAD_ENV=1
    else
      while IFS= read -r _ln; do
        [ -z "$_ln" ] && continue
        case "$_ln" in
          WARN*)
            _parsed=$(printf '%s' "$_ln" | awk '{
              n = split($0, a, /[[:space:]]{2,}/)
              st  = (n >= 1) ? a[1] : ""
              chk = (n >= 2) ? a[2] : ""
              site = (n >= 3) ? a[3] : ""
              reason = ""
              for (i = 4; i <= n; i++) reason = (reason == "" ? "" : reason "  ") a[i]
              colon = index(site, ":")
              fp  = (colon > 0) ? substr(site, 1, colon - 1) : site
              lno = (colon > 0) ? substr(site, colon + 1) : ""
              nsplit = split(fp, parts, "/"); bn = parts[nsplit]
              print st "|" chk "|" bn ":" lno "  " reason
            }')
            _st="${_parsed%%|*}"
            _r="${_parsed#*|}"
            _chk="${_r%%|*}"
            _det="${_r#*|}"
            emit "$ANAME" "$_st" "$_chk" "$_det"
          ;;
        esac
      done <<< "$ds_out"
    fi
  else
    emit "$ANAME" SKIP lint-demand-scope "no src/"
  fi

  # ----------------------------------------------------------------
  # 5.5. lint-silent-protection.sh <artifact>/src (Campaign 9 PR3; D9b: SKIP if no src/)
  # ----------------------------------------------------------------
  if [ -d "$ADIR/src" ]; then
    lsp_exit=0
    lsp_out=$("$TOOLBELT/lint-silent-protection.sh" "$ADIR/src" 2>&1) || lsp_exit=$?
    if [ "$lsp_exit" -eq 3 ]; then
      emit "$ANAME" ERROR lint-silent-protection "env fault (exit 3)"; HAD_ENV=1
    else
      while IFS= read -r _ln; do
        [ -z "$_ln" ] && continue
        case "$_ln" in
          FAIL*|WARN*)
            _parsed=$(printf '%s' "$_ln" | awk '{
              n = split($0, a, /[[:space:]]{2,}/)
              st = (n >= 1) ? a[1] : ""
              chk = (n >= 2) ? a[2] : ""
              site = (n >= 3) ? a[3] : ""
              reason = ""
              for (i = 4; i <= n; i++) reason = (reason == "" ? "" : reason "  ") a[i]
              colon = index(site, ":")
              fp = (colon > 0) ? substr(site, 1, colon - 1) : site
              lno = (colon > 0) ? substr(site, colon + 1) : ""
              nsplit = split(fp, parts, "/"); bn = parts[nsplit]
              print st "|" chk "|" bn ":" lno "  " reason
            }')
            _st="${_parsed%%|*}"
            _r="${_parsed#*|}"
            _chk="${_r%%|*}"
            _det="${_r#*|}"
            emit "$ANAME" "$_st" "$_chk" "$_det"
          ;;
        esac
      done <<< "$lsp_out"
    fi
  else
    emit "$ANAME" SKIP lint-silent-protection "no src/"
  fi

  # ----------------------------------------------------------------
  # 5.6. lint-ext-writable-shape.sh <artifact>/src (Campaign 9 PR10; D9b: SKIP if no src/)
  # ----------------------------------------------------------------
  if [ -d "$ADIR/src" ]; then
    lew_exit=0
    lew_out=$("$TOOLBELT/lint-ext-writable-shape.sh" "$ADIR/src" 2>&1) || lew_exit=$?
    if [ "$lew_exit" -eq 3 ]; then
      emit "$ANAME" ERROR lint-ext-writable-shape "env fault (exit 3)"; HAD_ENV=1
    else
      while IFS= read -r _ln; do
        [ -z "$_ln" ] && continue
        case "$_ln" in
          FAIL*|WARN*)
            _parsed=$(printf '%s' "$_ln" | awk '{
              n = split($0, a, /[[:space:]]{2,}/)
              st = (n >= 1) ? a[1] : ""
              chk = (n >= 2) ? a[2] : ""
              site = (n >= 3) ? a[3] : ""
              reason = ""
              for (i = 4; i <= n; i++) reason = (reason == "" ? "" : reason "  ") a[i]
              colon = index(site, ":")
              fp = (colon > 0) ? substr(site, 1, colon - 1) : site
              lno = (colon > 0) ? substr(site, colon + 1) : ""
              nsplit = split(fp, parts, "/"); bn = parts[nsplit]
              print st "|" chk "|" bn ":" lno "  " reason
            }')
            _st="${_parsed%%|*}"
            _r="${_parsed#*|}"
            _chk="${_r%%|*}"
            _det="${_r#*|}"
            emit "$ANAME" "$_st" "$_chk" "$_det"
          ;;
        esac
      done <<< "$lew_out"
    fi
  else
    emit "$ANAME" SKIP lint-ext-writable-shape "no src/"
  fi

  # ----------------------------------------------------------------
  # 5.7. lint-no-system-out.sh <artifact>/src (FAIL; SKIP if no src/)
  # ----------------------------------------------------------------
  if [ -d "$ADIR/src" ]; then
    nso_exit=0
    nso_out=$("$TOOLBELT/lint-no-system-out.sh" "$ADIR/src" 2>&1) || nso_exit=$?
    if [ "$nso_exit" -eq 3 ]; then
      emit "$ANAME" ERROR lint-no-system-out "env fault (exit 3)"; HAD_ENV=1
    else
      _nso_had_fail=0
      while IFS= read -r _ln; do
        [ -z "$_ln" ] && continue
        case "$_ln" in
          FAIL*)
            _parsed=$(printf '%s' "$_ln" | awk '{
              n = split($0, a, /[[:space:]]{2,}/)
              st = (n >= 1) ? a[1] : ""
              chk = (n >= 2) ? a[2] : ""
              site = (n >= 3) ? a[3] : ""
              reason = ""
              for (i = 4; i <= n; i++) reason = (reason == "" ? "" : reason "  ") a[i]
              colon = index(site, ":")
              fp = (colon > 0) ? substr(site, 1, colon - 1) : site
              lno = (colon > 0) ? substr(site, colon + 1) : ""
              nsplit = split(fp, parts, "/"); bn = parts[nsplit]
              print st "|" chk "|" bn ":" lno "  " reason
            }')
            _st="${_parsed%%|*}"
            _r="${_parsed#*|}"
            _chk="${_r%%|*}"
            _det="${_r#*|}"
            emit "$ANAME" "$_st" "$_chk" "$_det"
            _nso_had_fail=1
          ;;
        esac
      done <<< "$nso_out"
      [ "$_nso_had_fail" -eq 0 ] && emit "$ANAME" PASS lint-no-system-out "clean"
    fi
  else
    emit "$ANAME" SKIP lint-no-system-out "no src/"
  fi

  # ----------------------------------------------------------------
  # 5.8. lint-clock-zero-floor.sh <artifact>/src (WARN; SKIP if no src/)
  # ----------------------------------------------------------------
  if [ -d "$ADIR/src" ]; then
    czf_exit=0
    czf_out=$("$TOOLBELT/lint-clock-zero-floor.sh" "$ADIR/src" 2>&1) || czf_exit=$?
    if [ "$czf_exit" -eq 3 ]; then
      emit "$ANAME" ERROR lint-clock-zero-floor "env fault (exit 3)"; HAD_ENV=1
    else
      while IFS= read -r _ln; do
        [ -z "$_ln" ] && continue
        case "$_ln" in
          WARN*)
            _parsed=$(printf '%s' "$_ln" | awk '{
              n = split($0, a, /[[:space:]]{2,}/)
              st = (n >= 1) ? a[1] : ""
              chk = (n >= 2) ? a[2] : ""
              site = (n >= 3) ? a[3] : ""
              reason = ""
              for (i = 4; i <= n; i++) reason = (reason == "" ? "" : reason "  ") a[i]
              colon = index(site, ":")
              fp = (colon > 0) ? substr(site, 1, colon - 1) : site
              lno = (colon > 0) ? substr(site, colon + 1) : ""
              nsplit = split(fp, parts, "/"); bn = parts[nsplit]
              print st "|" chk "|" bn ":" lno "  " reason
            }')
            _st="${_parsed%%|*}"
            _r="${_parsed#*|}"
            _chk="${_r%%|*}"
            _det="${_r#*|}"
            emit "$ANAME" "$_st" "$_chk" "$_det"
          ;;
        esac
      done <<< "$czf_out"
    fi
  else
    emit "$ANAME" SKIP lint-clock-zero-floor "no src/"
  fi

  # ----------------------------------------------------------------
  # 5.9. lint-null-context-write.sh <artifact>/src (WARN; SKIP if no src/)
  # ----------------------------------------------------------------
  if [ -d "$ADIR/src" ]; then
    ncw_exit=0
    ncw_out=$("$TOOLBELT/lint-null-context-write.sh" "$ADIR/src" 2>&1) || ncw_exit=$?
    if [ "$ncw_exit" -eq 3 ]; then
      emit "$ANAME" ERROR lint-null-context-write "env fault (exit 3)"; HAD_ENV=1
    else
      while IFS= read -r _ln; do
        [ -z "$_ln" ] && continue
        case "$_ln" in
          WARN*)
            _parsed=$(printf '%s' "$_ln" | awk '{
              n = split($0, a, /[[:space:]]{2,}/)
              st = (n >= 1) ? a[1] : ""
              chk = (n >= 2) ? a[2] : ""
              site = (n >= 3) ? a[3] : ""
              reason = ""
              for (i = 4; i <= n; i++) reason = (reason == "" ? "" : reason "  ") a[i]
              colon = index(site, ":")
              fp = (colon > 0) ? substr(site, 1, colon - 1) : site
              lno = (colon > 0) ? substr(site, colon + 1) : ""
              nsplit = split(fp, parts, "/"); bn = parts[nsplit]
              print st "|" chk "|" bn ":" lno "  " reason
            }')
            _st="${_parsed%%|*}"
            _r="${_parsed#*|}"
            _chk="${_r%%|*}"
            _det="${_r#*|}"
            emit "$ANAME" "$_st" "$_chk" "$_det"
          ;;
        esac
      done <<< "$ncw_out"
    fi
  else
    emit "$ANAME" SKIP lint-null-context-write "no src/"
  fi

  # ----------------------------------------------------------------
  # 5.10. lint-bql-string-concat.sh <artifact>/src (WARN; SKIP if no src/)
  # ----------------------------------------------------------------
  if [ -d "$ADIR/src" ]; then
    bsc_exit=0
    bsc_out=$("$TOOLBELT/lint-bql-string-concat.sh" "$ADIR/src" 2>&1) || bsc_exit=$?
    if [ "$bsc_exit" -eq 3 ]; then
      emit "$ANAME" ERROR lint-bql-string-concat "env fault (exit 3)"; HAD_ENV=1
    else
      while IFS= read -r _ln; do
        [ -z "$_ln" ] && continue
        case "$_ln" in
          WARN*)
            _parsed=$(printf '%s' "$_ln" | awk '{
              n = split($0, a, /[[:space:]]{2,}/)
              st = (n >= 1) ? a[1] : ""
              chk = (n >= 2) ? a[2] : ""
              site = (n >= 3) ? a[3] : ""
              reason = ""
              for (i = 4; i <= n; i++) reason = (reason == "" ? "" : reason "  ") a[i]
              colon = index(site, ":")
              fp = (colon > 0) ? substr(site, 1, colon - 1) : site
              lno = (colon > 0) ? substr(site, colon + 1) : ""
              nsplit = split(fp, parts, "/"); bn = parts[nsplit]
              print st "|" chk "|" bn ":" lno "  " reason
            }')
            _st="${_parsed%%|*}"
            _r="${_parsed#*|}"
            _chk="${_r%%|*}"
            _det="${_r#*|}"
            emit "$ANAME" "$_st" "$_chk" "$_det"
          ;;
        esac
      done <<< "$bsc_out"
    fi
  else
    emit "$ANAME" SKIP lint-bql-string-concat "no src/"
  fi

  # ----------------------------------------------------------------
  # 5.11. lint-arbitrary-ord.sh <artifact>/src (WARN; SKIP if no src/)
  # ----------------------------------------------------------------
  if [ -d "$ADIR/src" ]; then
    ao_exit=0
    ao_out=$("$TOOLBELT/lint-arbitrary-ord.sh" "$ADIR/src" 2>&1) || ao_exit=$?
    if [ "$ao_exit" -eq 3 ]; then
      emit "$ANAME" ERROR lint-arbitrary-ord "env fault (exit 3)"; HAD_ENV=1
    else
      while IFS= read -r _ln; do
        [ -z "$_ln" ] && continue
        case "$_ln" in
          WARN*)
            _parsed=$(printf '%s' "$_ln" | awk '{
              n = split($0, a, /[[:space:]]{2,}/)
              st = (n >= 1) ? a[1] : ""
              chk = (n >= 2) ? a[2] : ""
              site = (n >= 3) ? a[3] : ""
              reason = ""
              for (i = 4; i <= n; i++) reason = (reason == "" ? "" : reason "  ") a[i]
              colon = index(site, ":")
              fp = (colon > 0) ? substr(site, 1, colon - 1) : site
              lno = (colon > 0) ? substr(site, colon + 1) : ""
              nsplit = split(fp, parts, "/"); bn = parts[nsplit]
              print st "|" chk "|" bn ":" lno "  " reason
            }')
            _st="${_parsed%%|*}"
            _r="${_parsed#*|}"
            _chk="${_r%%|*}"
            _det="${_r#*|}"
            emit "$ANAME" "$_st" "$_chk" "$_det"
          ;;
        esac
      done <<< "$ao_out"
    fi
  else
    emit "$ANAME" SKIP lint-arbitrary-ord "no src/"
  fi

  # ----------------------------------------------------------------
  # 5.12. lint-se-display.sh <artifact>/src (FAIL; only -se artifacts; SKIP if no src/)
  # ----------------------------------------------------------------
  case "$ANAME" in
    *-se)
      if [ -d "$ADIR/src" ]; then
        sed_exit=0
        sed_out=$("$TOOLBELT/lint-se-display.sh" "$ADIR/src" 2>&1) || sed_exit=$?
        if [ "$sed_exit" -eq 3 ]; then
          emit "$ANAME" ERROR lint-se-display "env fault (exit 3)"; HAD_ENV=1
        else
          _sed_had_fail=0
          while IFS= read -r _ln; do
            [ -z "$_ln" ] && continue
            case "$_ln" in
              FAIL*)
                _parsed=$(printf '%s' "$_ln" | awk '{
                  n = split($0, a, /[[:space:]]{2,}/)
                  st = (n >= 1) ? a[1] : ""
                  chk = (n >= 2) ? a[2] : ""
                  site = (n >= 3) ? a[3] : ""
                  reason = ""
                  for (i = 4; i <= n; i++) reason = (reason == "" ? "" : reason "  ") a[i]
                  colon = index(site, ":")
                  fp = (colon > 0) ? substr(site, 1, colon - 1) : site
                  lno = (colon > 0) ? substr(site, colon + 1) : ""
                  nsplit = split(fp, parts, "/"); bn = parts[nsplit]
                  print st "|" chk "|" bn ":" lno "  " reason
                }')
                _st="${_parsed%%|*}"
                _r="${_parsed#*|}"
                _chk="${_r%%|*}"
                _det="${_r#*|}"
                emit "$ANAME" "$_st" "$_chk" "$_det"
                _sed_had_fail=1
              ;;
            esac
          done <<< "$sed_out"
          [ "$_sed_had_fail" -eq 0 ] && emit "$ANAME" PASS lint-se-display "clean"
        fi
      else
        emit "$ANAME" SKIP lint-se-display "no src/"
      fi
    ;;
  esac

  # ----------------------------------------------------------------
  # 5.13. lint-jasmine-ux.sh <artifact> (WARN; only -ux artifacts)
  # ----------------------------------------------------------------
  case "$ANAME" in
    *-ux)
      jux_exit=0
      jux_out=$("$TOOLBELT/lint-jasmine-ux.sh" "$ADIR" 2>&1) || jux_exit=$?
      if [ "$jux_exit" -eq 3 ]; then
        emit "$ANAME" ERROR lint-jasmine-ux "env fault (exit 3)"; HAD_ENV=1
      else
        while IFS= read -r _ln; do
          [ -z "$_ln" ] && continue
          case "$_ln" in
            WARN*)
              _parsed=$(printf '%s' "$_ln" | awk '{
                n = split($0, a, /[[:space:]]{2,}/)
                st = (n >= 1) ? a[1] : ""
                chk = (n >= 2) ? a[2] : ""
                det = ""
                for (i = 4; i <= n; i++) det = (det == "" ? "" : det "  ") a[i]
                print st "|" chk "|" det
              }')
              _st="${_parsed%%|*}"
              _r="${_parsed#*|}"
              _chk="${_r%%|*}"
              _det="${_r#*|}"
              emit "$ANAME" "$_st" "$_chk" "$_det"
            ;;
          esac
        done <<< "$jux_out"
      fi
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
      if [ -d "$ADIR/src" ]; then
        swu_exit=0
        swu_out=$("$TOOLBELT/lint-subscribe-without-unsubscribe.sh" "$ADIR/src" 2>&1) || swu_exit=$?
        if [ "$swu_exit" -eq 3 ]; then
          emit "$ANAME" ERROR lint-subscribe-without-unsubscribe "env fault (exit 3)"; HAD_ENV=1
        else
          while IFS= read -r _ln; do
            [ -z "$_ln" ] && continue
            case "$_ln" in
              WARN*)
                _parsed=$(printf '%s' "$_ln" | awk '{
                  n = split($0, a, /[[:space:]]{2,}/)
                  st = (n >= 1) ? a[1] : ""
                  chk = (n >= 2) ? a[2] : ""
                  site = (n >= 3) ? a[3] : ""
                  reason = ""
                  for (i = 4; i <= n; i++) reason = (reason == "" ? "" : reason "  ") a[i]
                  colon = index(site, ":")
                  fp = (colon > 0) ? substr(site, 1, colon - 1) : site
                  lno = (colon > 0) ? substr(site, colon + 1) : ""
                  nsplit = split(fp, parts, "/"); bn = parts[nsplit]
                  print st "|" chk "|" bn ":" lno "  " reason
                }')
                _st="${_parsed%%|*}"
                _r="${_parsed#*|}"
                _chk="${_r%%|*}"
                _det="${_r#*|}"
                emit "$ANAME" "$_st" "$_chk" "$_det"
              ;;
            esac
          done <<< "$swu_out"
        fi
      else
        emit "$ANAME" SKIP lint-subscribe-without-unsubscribe "no src/"
      fi

      # 5.15 lint-recovery-path (FAIL; exit 1 = FAIL rows)
      if [ -d "$ADIR/src" ]; then
        lrp_exit=0
        lrp_out=$("$TOOLBELT/lint-recovery-path.sh" "$ADIR/src" 2>&1) || lrp_exit=$?
        if [ "$lrp_exit" -eq 3 ]; then
          emit "$ANAME" ERROR lint-recovery-path "env fault (exit 3)"; HAD_ENV=1
        else
          _lrp_had_fail=0
          while IFS= read -r _ln; do
            [ -z "$_ln" ] && continue
            case "$_ln" in
              FAIL*)
                _parsed=$(printf '%s' "$_ln" | awk '{
                  n = split($0, a, /[[:space:]]{2,}/)
                  st = (n >= 1) ? a[1] : ""
                  chk = (n >= 2) ? a[2] : ""
                  site = (n >= 3) ? a[3] : ""
                  reason = ""
                  for (i = 4; i <= n; i++) reason = (reason == "" ? "" : reason "  ") a[i]
                  colon = index(site, ":")
                  fp = (colon > 0) ? substr(site, 1, colon - 1) : site
                  lno = (colon > 0) ? substr(site, colon + 1) : ""
                  nsplit = split(fp, parts, "/"); bn = parts[nsplit]
                  print st "|" chk "|" bn ":" lno "  " reason
                }')
                _st="${_parsed%%|*}"
                _r="${_parsed#*|}"
                _chk="${_r%%|*}"
                _det="${_r#*|}"
                emit "$ANAME" "$_st" "$_chk" "$_det"
                _lrp_had_fail=1
              ;;
            esac
          done <<< "$lrp_out"
          [ "$_lrp_had_fail" -eq 0 ] && emit "$ANAME" PASS lint-recovery-path "clean"
        fi
      else
        emit "$ANAME" SKIP lint-recovery-path "no src/"
      fi

      # 5.16 lint-config-sanity (CS1/CS2 FAIL; CS3 WARN)
      if [ -d "$ADIR/src" ]; then
        lcs_exit=0
        lcs_out=$("$TOOLBELT/lint-config-sanity.sh" "$ADIR/src" 2>&1) || lcs_exit=$?
        if [ "$lcs_exit" -eq 3 ]; then
          emit "$ANAME" ERROR lint-config-sanity "env fault (exit 3)"; HAD_ENV=1
        else
          _lcs_had_fail=0
          while IFS= read -r _ln; do
            [ -z "$_ln" ] && continue
            case "$_ln" in
              FAIL*|WARN*)
                _parsed=$(printf '%s' "$_ln" | awk '{
                  n = split($0, a, /[[:space:]]{2,}/)
                  st = (n >= 1) ? a[1] : ""
                  chk = (n >= 2) ? a[2] : ""
                  site = (n >= 3) ? a[3] : ""
                  reason = ""
                  for (i = 4; i <= n; i++) reason = (reason == "" ? "" : reason "  ") a[i]
                  colon = index(site, ":")
                  fp = (colon > 0) ? substr(site, 1, colon - 1) : site
                  lno = (colon > 0) ? substr(site, colon + 1) : ""
                  nsplit = split(fp, parts, "/"); bn = parts[nsplit]
                  print st "|" chk "|" bn ":" lno "  " reason
                }')
                _st="${_parsed%%|*}"
                _r="${_parsed#*|}"
                _chk="${_r%%|*}"
                _det="${_r#*|}"
                emit "$ANAME" "$_st" "$_chk" "$_det"
                [ "$_st" = "FAIL" ] && _lcs_had_fail=1
              ;;
            esac
          done <<< "$lcs_out"
          [ "$_lcs_had_fail" -eq 0 ] && emit "$ANAME" PASS lint-config-sanity "clean"
        fi
      else
        emit "$ANAME" SKIP lint-config-sanity "no src/"
      fi

      # 5.17 lint-status-parity (WARN; called without --strict)
      if [ -d "$ADIR/src" ]; then
        lsp2_exit=0
        lsp2_out=$("$TOOLBELT/lint-status-parity.sh" "$ADIR/src" 2>&1) || lsp2_exit=$?
        if [ "$lsp2_exit" -eq 3 ]; then
          emit "$ANAME" ERROR lint-status-parity "env fault (exit 3)"; HAD_ENV=1
        else
          while IFS= read -r _ln; do
            [ -z "$_ln" ] && continue
            case "$_ln" in
              WARN*)
                _parsed=$(printf '%s' "$_ln" | awk '{
                  n = split($0, a, /[[:space:]]{2,}/)
                  st = (n >= 1) ? a[1] : ""
                  chk = (n >= 2) ? a[2] : ""
                  site = (n >= 3) ? a[3] : ""
                  reason = ""
                  for (i = 4; i <= n; i++) reason = (reason == "" ? "" : reason "  ") a[i]
                  colon = index(site, ":")
                  fp = (colon > 0) ? substr(site, 1, colon - 1) : site
                  lno = (colon > 0) ? substr(site, colon + 1) : ""
                  nsplit = split(fp, parts, "/"); bn = parts[nsplit]
                  print st "|" chk "|" bn ":" lno "  " reason
                }')
                _st="${_parsed%%|*}"
                _r="${_parsed#*|}"
                _chk="${_r%%|*}"
                _det="${_r#*|}"
                emit "$ANAME" "$_st" "$_chk" "$_det"
              ;;
            esac
          done <<< "$lsp2_out"
        fi
      else
        emit "$ANAME" SKIP lint-status-parity "no src/"
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
      if [ -d "$ADIR/src" ]; then
        srv_exit=0
        srv_out=$("$TOOLBELT/lint-servlet.sh" "$ADIR/src" 2>&1) || srv_exit=$?
        if [ "$srv_exit" -eq 3 ]; then
          emit "$ANAME" ERROR lint-servlet "env fault (exit 3)"; HAD_ENV=1
        else
          _srv_had_row=0
          _srv_had_fail=0
          while IFS= read -r _ln; do
            [ -z "$_ln" ] && continue
            case "$_ln" in
              FAIL*|WARN*)
                _parsed=$(printf '%s' "$_ln" | awk '{
                  n = split($0, a, /[[:space:]]{2,}/)
                  st = (n >= 1) ? a[1] : ""
                  chk = (n >= 2) ? a[2] : ""
                  site = (n >= 3) ? a[3] : ""
                  reason = ""
                  for (i = 4; i <= n; i++) reason = (reason == "" ? "" : reason "  ") a[i]
                  colon = index(site, ":")
                  fp = (colon > 0) ? substr(site, 1, colon - 1) : site
                  lno = (colon > 0) ? substr(site, colon + 1) : ""
                  nsplit = split(fp, parts, "/"); bn = parts[nsplit]
                  print st "|" chk "|" bn ":" lno "  " reason
                }')
                _st="${_parsed%%|*}"
                _r="${_parsed#*|}"
                _chk="${_r%%|*}"
                _det="${_r#*|}"
                emit "$ANAME" "$_st" "$_chk" "$_det"
                _srv_had_row=1
                [ "$_st" = "FAIL" ] && _srv_had_fail=1
              ;;
            esac
          done <<< "$srv_out"
          if [ "$_srv_had_row" -eq 0 ]; then
            emit "$ANAME" SKIP lint-servlet "no BWebServlet"
          elif [ "$_srv_had_fail" -eq 0 ]; then
            emit "$ANAME" PASS lint-servlet "clean"
          fi
        fi
      else
        emit "$ANAME" SKIP lint-servlet "no src/"
      fi

      # 5.19 rc-scan (FAIL; only when src/rc/ exists)
      if [ -d "$ADIR/src/rc" ]; then
        rcs_exit=0
        rcs_out=$("$TOOLBELT/rc-scan.sh" "$ADIR" 2>&1) || rcs_exit=$?
        if [ "$rcs_exit" -eq 3 ]; then
          emit "$ANAME" ERROR rc-scan "env fault (exit 3)"; HAD_ENV=1
        else
          _rcs_had_fail=0
          while IFS= read -r _ln; do
            [ -z "$_ln" ] && continue
            case "$_ln" in
              FAIL*|WARN*)
                _parsed=$(printf '%s' "$_ln" | awk '{
                  n = split($0, a, /[[:space:]]{2,}/)
                  st = (n >= 1) ? a[1] : ""
                  chk = (n >= 2) ? a[2] : ""
                  site = (n >= 3) ? a[3] : ""
                  reason = ""
                  for (i = 4; i <= n; i++) reason = (reason == "" ? "" : reason "  ") a[i]
                  colon = index(site, ":")
                  fp = (colon > 0) ? substr(site, 1, colon - 1) : site
                  lno = (colon > 0) ? substr(site, colon + 1) : ""
                  nsplit = split(fp, parts, "/"); bn = parts[nsplit]
                  print st "|" chk "|" bn ":" lno "  " reason
                }')
                _st="${_parsed%%|*}"
                _r="${_parsed#*|}"
                _chk="${_r%%|*}"
                _det="${_r#*|}"
                emit "$ANAME" "$_st" "$_chk" "$_det"
                [ "$_st" = "FAIL" ] && _rcs_had_fail=1
              ;;
            esac
          done <<< "$rcs_out"
          [ "$_rcs_had_fail" -eq 0 ] && emit "$ANAME" PASS rc-scan "clean"
        fi
      else
        emit "$ANAME" SKIP rc-scan "no src/rc/"
      fi
    ;;
  esac

  # ----------------------------------------------------------------
  # 5.20. lint-wb-threading.sh (WARN; only *-wb artifacts with src/)
  #       Row format: <check>  WARN  <file>:<line>  <detail>
  #       (check is field 1, status is field 2 — awk swaps for emit)
  # ----------------------------------------------------------------
  case "$ANAME" in
    *-wb)
      if [ -d "$ADIR/src" ]; then
        wbt_exit=0
        wbt_out=$("$TOOLBELT/lint-wb-threading.sh" "$ADIR/src" 2>&1) || wbt_exit=$?
        if [ "$wbt_exit" -eq 3 ]; then
          emit "$ANAME" ERROR lint-wb-threading "env fault (exit 3)"; HAD_ENV=1
        else
          while IFS= read -r _ln; do
            [ -z "$_ln" ] && continue
            _parsed=$(printf '%s' "$_ln" | awk '{
              n = split($0, a, /[[:space:]]{2,}/)
              chk  = (n >= 1) ? a[1] : ""
              st   = (n >= 2) ? a[2] : ""
              site = (n >= 3) ? a[3] : ""
              reason = ""
              for (i = 4; i <= n; i++) reason = (reason == "" ? "" : reason "  ") a[i]
              colon = index(site, ":")
              fp  = (colon > 0) ? substr(site, 1, colon - 1) : site
              lno = (colon > 0) ? substr(site, colon + 1) : ""
              nsplit = split(fp, parts, "/"); bn = parts[nsplit]
              print st "|" chk "|" bn ":" lno "  " reason
            }')
            _st="${_parsed%%|*}"
            _r="${_parsed#*|}"
            _chk="${_r%%|*}"
            _det="${_r#*|}"
            case "$_st" in
              WARN) emit "$ANAME" "$_st" "$_chk" "$_det" ;;
            esac
          done <<< "$wbt_out"
        fi
      else
        emit "$ANAME" SKIP lint-wb-threading "no src/"
      fi
    ;;
  esac

  # ----------------------------------------------------------------
  # 6. schema-risk.sh <artifact>/.deploy-baseline <artifact>
  #    (Campaign 8 PR8 / D9a; SKIP if no .deploy-baseline/ snapshot)
  # ----------------------------------------------------------------
  SR_BASELINE="$ADIR/.deploy-baseline"
  if [ ! -d "$SR_BASELINE" ]; then
    emit "$ANAME" SKIP schema-risk "no .deploy-baseline"
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
aos_exit=0
aos_out=$("$TOOLBELT/lint-agent-on-shape.sh" "$MODULE_ROOT" 2>&1) || aos_exit=$?
if [ "$aos_exit" -eq 3 ]; then
  emit "(module)" ERROR lint-agent-on-shape "env fault (exit 3)"; HAD_ENV=1
else
  _aos_had_fail=0
  while IFS= read -r _ln; do
    [ -z "$_ln" ] && continue
    case "$_ln" in
      FAIL*)
        _parsed=$(printf '%s' "$_ln" | awk '{
          n = split($0, a, /[[:space:]]{2,}/)
          st = (n >= 1) ? a[1] : ""
          chk = (n >= 2) ? a[2] : ""
          site = (n >= 3) ? a[3] : ""
          reason = ""
          for (i = 4; i <= n; i++) reason = (reason == "" ? "" : reason "  ") a[i]
          colon = index(site, ":")
          fp = (colon > 0) ? substr(site, 1, colon - 1) : site
          lno = (colon > 0) ? substr(site, colon + 1) : ""
          nsplit = split(fp, parts, "/"); bn = parts[nsplit]
          print st "|" chk "|" bn ":" lno "  " reason
        }')
        _st="${_parsed%%|*}"
        _r="${_parsed#*|}"
        _chk="${_r%%|*}"
        _det="${_r#*|}"
        emit "(module)" "$_st" "$_chk" "$_det"
        _aos_had_fail=1
      ;;
    esac
  done <<< "$aos_out"
  [ "$_aos_had_fail" -eq 0 ] && emit "(module)" PASS lint-agent-on-shape "clean"
fi

# ----------------------------------------------------------------
# 9. lint-uberjar-api-conflict.sh <module-root> — once per run (WARN)
# ----------------------------------------------------------------
uac_exit=0
uac_out=$("$TOOLBELT/lint-uberjar-api-conflict.sh" "$MODULE_ROOT" 2>&1) || uac_exit=$?
if [ "$uac_exit" -eq 3 ]; then
  emit "(module)" ERROR lint-uberjar-api-conflict "env fault (exit 3)"; HAD_ENV=1
else
  while IFS= read -r _ln; do
    [ -z "$_ln" ] && continue
    case "$_ln" in
      WARN*)
        _parsed=$(printf '%s' "$_ln" | awk '{
          n = split($0, a, /[[:space:]]{2,}/)
          st = (n >= 1) ? a[1] : ""
          chk = (n >= 2) ? a[2] : ""
          site = (n >= 3) ? a[3] : ""
          reason = ""
          for (i = 4; i <= n; i++) reason = (reason == "" ? "" : reason "  ") a[i]
          nsplit = split(site, parts, "/"); bn = parts[nsplit]
          print st "|" chk "|" bn "  " reason
        }')
        _st="${_parsed%%|*}"
        _r="${_parsed#*|}"
        _chk="${_r%%|*}"
        _det="${_r#*|}"
        emit "(module)" "$_st" "$_chk" "$_det"
      ;;
    esac
  done <<< "$uac_out"
fi

# ----------------------------------------------------------------
# 10. lint-structure.sh <module-root> — once per run (FAIL)
# ----------------------------------------------------------------
lst_exit=0
lst_out=$("$TOOLBELT/lint-structure.sh" "$MODULE_ROOT" 2>&1) || lst_exit=$?
if [ "$lst_exit" -eq 3 ]; then
  emit "(module)" ERROR lint-structure "env fault (exit 3)"; HAD_ENV=1
else
  _lst_had_fail=0
  while IFS= read -r _ln; do
    [ -z "$_ln" ] && continue
    case "$_ln" in
      FAIL*|WARN*)
        _parsed=$(printf '%s' "$_ln" | awk '{
          n = split($0, a, /[[:space:]]{2,}/)
          st = (n >= 1) ? a[1] : ""
          chk = (n >= 2) ? a[2] : ""
          site = (n >= 3) ? a[3] : ""
          reason = ""
          for (i = 4; i <= n; i++) reason = (reason == "" ? "" : reason "  ") a[i]
          colon = index(site, ":")
          fp = (colon > 0) ? substr(site, 1, colon - 1) : site
          lno = (colon > 0) ? substr(site, colon + 1) : ""
          nsplit = split(fp, parts, "/"); bn = parts[nsplit]
          print st "|" chk "|" bn ":" lno "  " reason
        }')
        _st="${_parsed%%|*}"
        _r="${_parsed#*|}"
        _chk="${_r%%|*}"
        _det="${_r#*|}"
        emit "(module)" "$_st" "$_chk" "$_det"
        [ "$_st" = "FAIL" ] && _lst_had_fail=1
      ;;
    esac
  done <<< "$lst_out"
  [ "$_lst_had_fail" -eq 0 ] && emit "(module)" PASS lint-structure "clean"
fi

# ----------------------------------------------------------------
# 11. lint-write-path.sh <module-root> — once per run (FAIL on uncovered;
#     STALE/DRIFT advisory rows mapped to WARN, never blocking)
# ----------------------------------------------------------------
lwp_exit=0
lwp_out=$("$TOOLBELT/lint-write-path.sh" "$MODULE_ROOT" 2>&1) || lwp_exit=$?
if [ "$lwp_exit" -eq 3 ]; then
  emit "(module)" SKIP lint-write-path "no write-path-matrix.md"
else
  _lwp_had_fail=0
  while IFS= read -r _ln; do
    [ -z "$_ln" ] && continue
    case "$_ln" in
      FAIL*)
        _parsed=$(printf '%s' "$_ln" | awk '{
          n = split($0, a, /[[:space:]]{2,}/)
          st = (n >= 1) ? a[1] : ""
          chk = (n >= 2) ? a[2] : ""
          site = (n >= 3) ? a[3] : ""
          reason = ""
          for (i = 4; i <= n; i++) reason = (reason == "" ? "" : reason "  ") a[i]
          colon = index(site, ":")
          fp = (colon > 0) ? substr(site, 1, colon - 1) : site
          lno = (colon > 0) ? substr(site, colon + 1) : ""
          nsplit = split(fp, parts, "/"); bn = parts[nsplit]
          print st "|" chk "|" bn ":" lno "  " reason
        }')
        _st="${_parsed%%|*}"
        _r="${_parsed#*|}"
        _chk="${_r%%|*}"
        _det="${_r#*|}"
        emit "(module)" "$_st" "$_chk" "$_det"
        _lwp_had_fail=1
      ;;
      STALE*|DRIFT*)
        # Advisory rows: exit-0 but surface as WARN so they appear in the report
        _parsed=$(printf '%s' "$_ln" | awk '{
          n = split($0, a, /[[:space:]]{2,}/)
          chk = (n >= 2) ? a[2] : ""
          site = (n >= 3) ? a[3] : ""
          reason = ""
          for (i = 4; i <= n; i++) reason = (reason == "" ? "" : reason "  ") a[i]
          colon = index(site, ":")
          fp = (colon > 0) ? substr(site, 1, colon - 1) : site
          lno = (colon > 0) ? substr(site, colon + 1) : ""
          nsplit = split(fp, parts, "/"); bn = parts[nsplit]
          print chk "|" bn ":" lno "  " reason
        }')
        _chk="${_parsed%%|*}"
        _det="${_parsed#*|}"
        emit "(module)" WARN "$_chk" "$_det"
      ;;
    esac
  done <<< "$lwp_out"
  [ "$_lwp_had_fail" -eq 0 ] && emit "(module)" PASS lint-write-path "clean"
fi

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
[ "$HAD_FAIL" -eq 1 ] && VERDICT="ISSUES"
_s_sfx="$([ "$ARTIFACT_COUNT" -eq 1 ] && printf '' || printf 's')"
printf 'report-module: %d artifact%s · %d PASS · %d FAIL · %d WARN · %d SKIP  ->  %s\n' \
  "$ARTIFACT_COUNT" "$_s_sfx" "$NPASS" "$NFAIL" "$NWARN" "$NSKIP" "$VERDICT"

[ "$HAD_ENV" -eq 1 ] && exit 3
[ "$HAD_FAIL" -eq 1 ] && exit 1
exit 0
