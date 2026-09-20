#!/usr/bin/env bash
# commissioning-verify.sh — §6.b commissioning punch-list auditor (issue #114).
#
# Orchestrates the Wave 3 lints and bog-audit checks into a single §6.b commissioning
# punch-list. Emits rows STATUS  commissioning  <check>  <detail>; exits 0 clean (no FAIL)
# or 1 any FAIL, 3 usage/env.
#
# Usage:
#   commissioning-verify.sh <module-root> [--bog <config.bog>] [--module <MOD>] [--strict]
#
# Source checks (static, VCS-free, no station required):
#   config-sanity:<rt>  lint-config-sanity.sh per -rt/src dir (CS1/CS2 FAIL, CS3 WARN)
#   status-parity:<rt>  lint-status-parity.sh per -rt/src dir (WARN, or FAIL under --strict)
#   recovery-path:<rt>  lint-recovery-path.sh per -rt/src dir (FAIL)
#
# Station checks (require --bog <config.bog> and --module <MOD>):
#   proxy-link-safety        bog-audit.sh CHECK11
#   station-logic:*          bog-audit.sh CHECK13-19
#   numeric-to-bool-direct   bog-audit.sh CHECK20
#   (SKIP rows emitted when --bog is absent)
#
# Facade slot inventory note (PASS informational row):
#   wiring-map  generate-wiring-map.sh scaffolds docs/wiring-map.md
#
# Manual-only footer (MANUAL rows — informational, never FAIL):
#   hot-reload-console    triage-console.sh clean after station restart
#   plant-control         station actually controlling the plant
#   per-instance-values   per-instance runtime values match physical setpoints
#
# Row format: STATUS  commissioning  <check>  <detail>
# Summary:    commissioning-verify: P PASS · F FAIL · W WARN · S SKIP · M MANUAL  ->  CLEAN|ISSUES
# Exit:       0 clean (zero FAIL) · 1 any FAIL · 3 usage/env
# VCS-free by design (kit-links.bats L2). Named in BUILD-LOOP.md (kit-links.bats L5).
# [ev: retro live-commissioning-verification-gaps]

set -u
LC_ALL=C; export LC_ALL

TOOLBELT="$(cd "${BASH_SOURCE[0]%/*}" && pwd)"

NPASS=0; NFAIL=0; NWARN=0; NSKIP=0; NMANUAL=0
HAD_FAIL=0

MODULE_ROOT=""
BOG=""
MOD=""
STRICT=0

usage_exit() {
  printf 'usage: commissioning-verify.sh <module-root> [--bog <config.bog>] [--module <MOD>] [--strict]\n' >&2
  exit 3
}

while [ $# -gt 0 ]; do
  case "$1" in
    --bog)
      [ $# -ge 2 ] || usage_exit
      BOG="$2"; shift 2 ;;
    --module)
      [ $# -ge 2 ] || usage_exit
      MOD="$2"; shift 2 ;;
    --strict) STRICT=1; shift ;;
    --) shift; break ;;
    -*) usage_exit ;;
    *)
      [ -z "$MODULE_ROOT" ] || usage_exit
      MODULE_ROOT="$1"; shift ;;
  esac
done
[ -n "$MODULE_ROOT" ] || usage_exit
[ -d "$MODULE_ROOT" ] || { printf 'commissioning-verify: not a directory: %s\n' "$MODULE_ROOT" >&2; exit 3; }

if [ -n "$BOG" ] && [ -z "$MOD" ]; then
  printf 'commissioning-verify: --bog requires --module\n' >&2
  usage_exit
fi

# emit <STATUS> <check> <detail>
emit() {
  local _st="$1" _chk="$2" _det="$3"
  printf '%s  commissioning  %s  %s\n' "$_st" "$_chk" "$_det"
  case "$_st" in
    PASS)   NPASS=$((NPASS+1)) ;;
    FAIL)   NFAIL=$((NFAIL+1)); HAD_FAIL=1 ;;
    WARN)   NWARN=$((NWARN+1)) ;;
    SKIP)   NSKIP=$((NSKIP+1)) ;;
    MANUAL) NMANUAL=$((NMANUAL+1)) ;;
  esac
}

# ----------------------------------------------------------------
# Discover -rt/src directories (glob: <module-root>/*/*-rt/src)
# Falls back to <module-root>/src when no -rt layout is found.
# ----------------------------------------------------------------
SRC_DIRS=()
while IFS= read -r _d; do
  SRC_DIRS+=("$_d")
done < <(find "$MODULE_ROOT" -mindepth 3 -maxdepth 3 -type d -name 'src' 2>/dev/null \
         | grep -- '-rt/src$' | sort)

if [ "${#SRC_DIRS[@]}" -eq 0 ] && [ -d "$MODULE_ROOT/src" ]; then
  SRC_DIRS=("$MODULE_ROOT/src")
fi

if [ "${#SRC_DIRS[@]}" -eq 0 ]; then
  printf 'commissioning-verify: no -rt/src or src/ found under %s\n' "$MODULE_ROOT" >&2
  exit 3
fi

# ----------------------------------------------------------------
# Per -rt/src static checks
# ----------------------------------------------------------------
for SRC in "${SRC_DIRS[@]}"; do
  LABEL="$(basename "$(dirname "$SRC")")"   # e.g. MinimalPan-rt, or src

  # --- config-sanity (CS1/CS2 FAIL, CS3 WARN) ---
  cs_exit=0
  cs_out=$("$TOOLBELT/lint-config-sanity.sh" "$SRC" 2>&1) || cs_exit=$?
  if [ "$cs_exit" -eq 3 ]; then
    emit SKIP "config-sanity:$LABEL" "env fault (exit 3) — run lint-config-sanity.sh manually"
  elif printf '%s\n' "$cs_out" | grep -qE '^FAIL|^WARN'; then
    while IFS= read -r _ln; do
      case "$_ln" in
        FAIL*) emit FAIL "config-sanity:$LABEL" "${_ln#FAIL  lint-config-sanity  }" ;;
        WARN*) emit WARN "config-sanity:$LABEL" "${_ln#WARN  lint-config-sanity  }" ;;
      esac
    done <<< "$cs_out"
  else
    emit PASS "config-sanity:$LABEL" "CS1/CS2/CS3 clean"
  fi

  # --- status-parity (WARN, or FAIL under --strict) ---
  sp_args=()
  [ "$STRICT" -eq 1 ] && sp_args+=("--strict")
  sp_exit=0
  sp_out=$("$TOOLBELT/lint-status-parity.sh" "${sp_args[@]}" "$SRC" 2>&1) || sp_exit=$?
  if [ "$sp_exit" -eq 3 ]; then
    emit SKIP "status-parity:$LABEL" "env fault (exit 3) — run lint-status-parity.sh manually"
  elif printf '%s\n' "$sp_out" | grep -qE '^FAIL|^WARN'; then
    while IFS= read -r _ln; do
      case "$_ln" in
        FAIL*) emit FAIL "status-parity:$LABEL" "${_ln#FAIL  lint-status-parity  }" ;;
        WARN*) emit WARN "status-parity:$LABEL" "${_ln#WARN  lint-status-parity  }" ;;
      esac
    done <<< "$sp_out"
  else
    emit PASS "status-parity:$LABEL" "N:M config/status parity clean"
  fi

  # --- recovery-path (FAIL when protection slot guarded-only) ---
  rp_exit=0
  rp_out=$("$TOOLBELT/lint-recovery-path.sh" "$SRC" 2>&1) || rp_exit=$?
  if [ "$rp_exit" -eq 3 ]; then
    emit SKIP "recovery-path:$LABEL" "env fault (exit 3) — run lint-recovery-path.sh manually"
  elif printf '%s\n' "$rp_out" | grep -q '^FAIL'; then
    while IFS= read -r _ln; do
      case "$_ln" in
        FAIL*) emit FAIL "recovery-path:$LABEL" "${_ln#FAIL  lint-recovery-path  }" ;;
      esac
    done <<< "$rp_out"
  else
    emit PASS "recovery-path:$LABEL" "protection-output recovery path clean"
  fi
done

# ----------------------------------------------------------------
# Facade slot inventory note (informational PASS)
# ----------------------------------------------------------------
emit PASS "wiring-map" \
  "run generate-wiring-map.sh <facade-src-dir> to scaffold docs/wiring-map.md before commissioning"

# ----------------------------------------------------------------
# Station checks (bog-coupled: CHECK11 + CHECK13-19 + CHECK20)
# ----------------------------------------------------------------
if [ -n "$BOG" ]; then
  [ -f "$BOG" ] || { printf 'commissioning-verify: bog not found: %s\n' "$BOG" >&2; exit 3; }
  ba_args=("$BOG" "--module" "$MOD")
  [ "$STRICT" -eq 1 ] && ba_args+=("--strict")
  ba_exit=0
  ba_out=$("$TOOLBELT/bog-audit.sh" "${ba_args[@]}" 2>&1) || ba_exit=$?
  if [ "$ba_exit" -eq 3 ]; then
    emit SKIP "station-checks" "bog-audit env fault (exit 3) — check python3 availability and bog path"
  else
    # Fold CHECK11 (proxy-link-safety)
    if printf '%s\n' "$ba_out" | grep -q '^CHECK11'; then
      while IFS= read -r _ln; do
        case "$_ln" in
          'CHECK11  FAIL'*) emit FAIL "proxy-link-safety" "${_ln#CHECK11  FAIL  }" ;;
          'CHECK11  PASS'*) emit PASS "proxy-link-safety" "${_ln#CHECK11  PASS  }" ;;
          'CHECK11  WARN'*) emit WARN "proxy-link-safety" "${_ln#CHECK11  WARN  }" ;;
        esac
      done <<< "$ba_out"
    fi
    # Fold CHECK13 (relay-double-source)
    if printf '%s\n' "$ba_out" | grep -q '^CHECK13'; then
      while IFS= read -r _ln; do
        case "$_ln" in
          'CHECK13  FAIL'*) emit FAIL "station-logic:relay-double-source" "${_ln#CHECK13  FAIL  }" ;;
          'CHECK13  PASS'*) emit PASS "station-logic:relay-double-source" "${_ln#CHECK13  PASS  }" ;;
        esac
      done <<< "$ba_out"
    fi
    # Fold CHECK14 (own-output-unlinked)
    if printf '%s\n' "$ba_out" | grep -q '^CHECK14'; then
      while IFS= read -r _ln; do
        case "$_ln" in
          'CHECK14  FAIL'*) emit FAIL "station-logic:own-output-unlinked" "${_ln#CHECK14  FAIL  }" ;;
          'CHECK14  WARN'*) emit WARN "station-logic:own-output-unlinked" "${_ln#CHECK14  WARN  }" ;;
          'CHECK14  PASS'*) emit PASS "station-logic:own-output-unlinked" "${_ln#CHECK14  PASS  }" ;;
        esac
      done <<< "$ba_out"
    fi
    # Fold CHECK15 (sensor-crossed)
    if printf '%s\n' "$ba_out" | grep -q '^CHECK15'; then
      while IFS= read -r _ln; do
        case "$_ln" in
          'CHECK15  FAIL'*) emit FAIL "station-logic:sensor-crossed" "${_ln#CHECK15  FAIL  }" ;;
          'CHECK15  PASS'*) emit PASS "station-logic:sensor-crossed" "${_ln#CHECK15  PASS  }" ;;
        esac
      done <<< "$ba_out"
    fi
    # Fold CHECK16 (defrost-sibling)
    if printf '%s\n' "$ba_out" | grep -q '^CHECK16'; then
      while IFS= read -r _ln; do
        case "$_ln" in
          'CHECK16  FAIL'*) emit FAIL "station-logic:defrost-sibling" "${_ln#CHECK16  FAIL  }" ;;
          'CHECK16  PASS'*) emit PASS "station-logic:defrost-sibling" "${_ln#CHECK16  PASS  }" ;;
        esac
      done <<< "$ba_out"
    fi
    # Fold CHECK17 (room-index-mismatch)
    if printf '%s\n' "$ba_out" | grep -q '^CHECK17'; then
      while IFS= read -r _ln; do
        case "$_ln" in
          'CHECK17  FAIL'*) emit FAIL "station-logic:room-index-mismatch" "${_ln#CHECK17  FAIL  }" ;;
          'CHECK17  PASS'*) emit PASS "station-logic:room-index-mismatch" "${_ln#CHECK17  PASS  }" ;;
        esac
      done <<< "$ba_out"
    fi
    # Fold CHECK18 (tile-number)
    if printf '%s\n' "$ba_out" | grep -q '^CHECK18'; then
      while IFS= read -r _ln; do
        case "$_ln" in
          'CHECK18  FAIL'*) emit FAIL "station-logic:tile-number" "${_ln#CHECK18  FAIL  }" ;;
          'CHECK18  PASS'*) emit PASS "station-logic:tile-number" "${_ln#CHECK18  PASS  }" ;;
        esac
      done <<< "$ba_out"
    fi
    # Fold CHECK19 (link-direction)
    if printf '%s\n' "$ba_out" | grep -q '^CHECK19'; then
      while IFS= read -r _ln; do
        case "$_ln" in
          'CHECK19  FAIL'*) emit FAIL "station-logic:link-direction" "${_ln#CHECK19  FAIL  }" ;;
          'CHECK19  PASS'*) emit PASS "station-logic:link-direction" "${_ln#CHECK19  PASS  }" ;;
        esac
      done <<< "$ba_out"
    fi
    # Fold CHECK20 (numeric-to-bool-direct)
    if printf '%s\n' "$ba_out" | grep -q '^CHECK20'; then
      while IFS= read -r _ln; do
        case "$_ln" in
          'CHECK20  FAIL'*) emit FAIL "numeric-to-bool-direct" "${_ln#CHECK20  FAIL  }" ;;
          'CHECK20  PASS'*) emit PASS "numeric-to-bool-direct" "${_ln#CHECK20  PASS  }" ;;
        esac
      done <<< "$ba_out"
    fi
  fi
else
  emit SKIP "proxy-link-safety" \
    "no --bog provided; pass --bog <config.bog> --module <MOD> to audit live station"
  emit SKIP "station-logic" \
    "no --bog provided; pass --bog <config.bog> --module <MOD> to audit live station"
  emit SKIP "numeric-to-bool-direct" \
    "no --bog provided; pass --bog <config.bog> --module <MOD> to audit live station"
fi

# ----------------------------------------------------------------
# Manual-only footer — live steps this tool cannot statically verify
# ----------------------------------------------------------------
emit MANUAL "hot-reload-console" \
  "after deploy: run triage-console.sh — confirm no own-module load failures before hand-off"
emit MANUAL "plant-control" \
  "confirm the station actually controls the plant (runtime only — not statically verifiable)"
emit MANUAL "per-instance-values" \
  "confirm per-instance runtime values match physical setpoints for every room/unit"
emit MANUAL "servlet-response-headers" \
  "BWebServlet module: curl -sI -H 'X-Requested-With: XMLHttpRequest' -u admin:pass http://<station>/<module>/api/equipment | grep -iE 'x-content-type-options|x-frame-options' — both headers must appear (ODA2-G1/G2; see types/security.md §7)"

# ----------------------------------------------------------------
# Summary
# ----------------------------------------------------------------
if [ "$HAD_FAIL" -eq 1 ]; then
  VERDICT="ISSUES"
else
  VERDICT="CLEAN"
fi
printf 'commissioning-verify: %d PASS · %d FAIL · %d WARN · %d SKIP · %d MANUAL  ->  %s\n' \
  "$NPASS" "$NFAIL" "$NWARN" "$NSKIP" "$NMANUAL" "$VERDICT"

[ "$HAD_FAIL" -eq 0 ]
