#!/usr/bin/env bash
# commissioning-verify.sh — §6.b commissioning punch-list auditor (issue #114).
#
# Orchestrates the Wave 3 lints and bog-audit checks into a single §6.b commissioning
# punch-list. Emits rows STATUS  commissioning  <check>  <detail>; exits 0 clean (no FAIL)
# or 1 any FAIL, 3 usage/env.
#
# Usage:
#   commissioning-verify.sh <module-root> [--bog <config.bog>] [--module <MOD>]
#                           [--values-owed <file>] [--wiring-map <file>] [--strict]
#
# Source checks (static, VCS-free, no station required):
#   config-sanity:<rt>  lint-config-sanity.sh per -rt/src dir (CS1/CS2 FAIL, CS3 WARN)
#   status-parity:<rt>  lint-status-parity.sh per -rt/src dir (WARN, or FAIL under --strict)
#   recovery-path:<rt>  lint-recovery-path.sh per -rt/src dir (FAIL)
#   link-target-flags:<rt>  lint-link-target-flags.sh per -rt/src dir (LTF1/LTF2 FAIL), with
#                       --wiring-map <file> (default <module-root>/docs/wiring-map.md when present) so the
#                       Table 2 link-in targets are known. [ev: retro panccadia-commissioning-lessons Δ1]
#                       The default differs from report-module.sh (which also tries <module-root>/../docs)
#                       because this tool takes the repo root while build.sh passes <repo>/<MOD> to
#                       report-module; both resolve <repo>/docs/wiring-map.md. The chosen map is named in
#                       the row detail. Exit 3 is SKIP here (this script's env convention for every source
#                       lint; report-module maps it to ERROR + exit 3). A non-zero exit with no FAIL row is
#                       a FAIL row, never PASS.
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
# Values owed by the field (BUILD-LOOP §6.b table; default <module-root>/docs/values-owed.md):
#   values-owed  one MANUAL row per table row whose Status cell does not start with
#                filled/provided (columns: Slot | Owed by | Unit | Safe default | Status); a row
#                with fewer cells is a MANUAL "malformed" row (fails closed); alignment rows skip;
#                one PASS row when every value is provided; one MANUAL row when no table exists.
#   [ev: retro panccadia-commissioning-lessons Δ11] [ev: retro panccadia-version-defect-ledger Δ4]
#
# Manual-only footer (MANUAL rows — informational, never FAIL):
#   hot-reload-console       triage-console.sh clean after station restart
#   plant-control            station actually controlling the plant
#   per-instance-values      per-instance runtime values match physical setpoints
#   persisted-state-restart  persisted operator state seeded/saved before the first restart, re-read after
#   alarm-routing            alarm class + recipient per source; a test alarm reaches the console, acked
#   consumer-impact          downstream consumers notified of every changed point meaning
#   link-source-audit        obix-link-audit.sh: each declared link-in comes from its declared source
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
OWED=""
WMAP=""
STRICT=0

usage_exit() {
  printf 'usage: commissioning-verify.sh <module-root> [--bog <config.bog>] [--module <MOD>] [--values-owed <file>] [--wiring-map <file>] [--strict]\n' >&2
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
    --values-owed)
      [ $# -ge 2 ] || usage_exit
      OWED="$2"; shift 2 ;;
    --wiring-map)
      [ $# -ge 2 ] || usage_exit
      WMAP="$2"; shift 2 ;;
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

if [ -n "$OWED" ] && [ ! -f "$OWED" ]; then
  printf 'commissioning-verify: values-owed table not found: %s\n' "$OWED" >&2
  exit 3
fi
if [ -z "$OWED" ] && [ -f "$MODULE_ROOT/docs/values-owed.md" ]; then
  OWED="$MODULE_ROOT/docs/values-owed.md"
fi

if [ -n "$WMAP" ] && [ ! -f "$WMAP" ]; then
  printf 'commissioning-verify: wiring map not found: %s\n' "$WMAP" >&2
  exit 3
fi
if [ -z "$WMAP" ] && [ -f "$MODULE_ROOT/docs/wiring-map.md" ]; then
  WMAP="$MODULE_ROOT/docs/wiring-map.md"
fi

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
  sp_out=$("$TOOLBELT/lint-status-parity.sh" ${sp_args[@]+"${sp_args[@]}"} "$SRC" 2>&1) || sp_exit=$?
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

  # --- link-target-flags (LTF1 READONLY link-in target, LTF2 TRANSIENT operator mode/HOA: FAIL) ---
  ltf_args=()
  [ -n "$WMAP" ] && ltf_args+=("--wiring-map" "$WMAP")
  ltf_exit=0
  # ${A[@]+"${A[@]}"}: an empty "${A[@]}" is an unbound-variable error under set -u on bash < 4.4.
  ltf_out=$("$TOOLBELT/lint-link-target-flags.sh" ${ltf_args[@]+"${ltf_args[@]}"} "$SRC" 2>&1) || ltf_exit=$?
  # "(wiring-map Table 2)" is a contract with the LTF1 reason text of lint-link-target-flags.sh (CV-ltf6
  # pins it); pattern and replacement are quoted so an & in the path stays literal under bash 5.2
  # patsub_replacement. [polish-2026-10-02 P1c]
  [ -n "$WMAP" ] && ltf_out=${ltf_out//"(wiring-map Table 2)"/"(wiring-map Table 2: $WMAP)"}
  if [ "$ltf_exit" -eq 3 ]; then
    emit SKIP "link-target-flags:$LABEL" "env fault (exit 3) — run lint-link-target-flags.sh manually"
  elif printf '%s\n' "$ltf_out" | grep -q '^FAIL'; then
    while IFS= read -r _ln; do
      case "$_ln" in
        FAIL*) emit FAIL "link-target-flags:$LABEL" "${_ln#FAIL  lint-link-target-flags  }" ;;
      esac
    done <<< "$ltf_out"
  elif [ "$ltf_exit" -ne 0 ]; then
    # Fail closed: a non-zero exit with no FAIL row (127 = lint missing, a crash) is never PASS.
    # [polish-2026-10-02 P1b, R3-cv-ltf-fail-open-exit]
    emit FAIL "link-target-flags:$LABEL" \
      "lint-link-target-flags.sh exited $ltf_exit with no FAIL row (crash or unexpected output) -- run it manually"
  else
    _ltf_map="no wiring map"; [ -n "$WMAP" ] && _ltf_map="wiring map: $WMAP"
    emit PASS "link-target-flags:$LABEL" "LTF1/LTF2 clean ($_ltf_map)"
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
# Values owed by the field — a disabled/no-value-yet default stays an open MANUAL row
# until the table marks it filled/provided.
# ----------------------------------------------------------------
if [ -n "$OWED" ]; then
  # Fails closed [polish-2026-10-02 P3/P3b/P3c, #199 WU9]. The values-owed table is the one whose
  # header row starts with Slot. Another pipe block is a foreign table (revisions, notes: skipped)
  # only when its SECOND row is an alignment row; any other headerless block (an owed table split
  # by a blank line or a comment) is read as owed rows. A doc with table rows but no Slot header is
  # one MANUAL row (no PASS); a doc with no table at all (prose "none owed") stays a PASS. An owed
  # row is the header, an alignment row (each cell dashes with optional colons: ---, :---, :---:,
  # ---:), an all-empty row, a filled/provided row, or a MANUAL row; a row with fewer than five
  # cells or an empty Slot cell is a MANUAL "malformed" row, never skipped into the PASS.
  owed_rows=$(awk -v F="$OWED" '
    BEGIN { OUT = 0; OWED = 1; FOREIGN = 2; PEND = 3; st = OUT; pend = "" }   # block states
    function trim(x) { gsub(/^[ \t]+|[ \t]+$/, "", x); return x }
    function cells(r,    line, i) {   # sets c[1..n], n, blank, sep
      line = trim(r); sub(/^\|/, "", line); sub(/\|$/, "", line)
      n = split(line, c, "|"); blank = 1; sep = 1
      for (i = 1; i <= n; i++) {
        c[i] = trim(c[i]); gsub(/`/, "", c[i])
        if (c[i] != "") blank = 0
        if (c[i] !~ /^:?-+:?$/) sep = 0
      }
    }
    function owed(r) {
      cells(r)
      if (blank || sep) return
      if (n < 5 || c[1] == "") {
        printf "malformed values-owed row (needs Slot | Owed by | Unit | Safe default | Status): %s\n", trim(r)
        return
      }
      if (tolower(c[5]) ~ /^(filled|provided)/) return
      printf "%s: owed by %s, unit %s, still at safe default %s\n", c[1], c[2], c[3], c[4]
    }
    function flush() { if (pend != "") owed(pend); pend = "" }   # a lone buffered row is an owed row
    !/^[ \t]*\|/ { flush(); st = OUT; next }
    {
      pipes = 1
      if (st == OUT) {                 # first row of a block
        cells($0)
        if (tolower(c[1]) == "slot") { st = OWED; seen = 1; next }
        pend = $0; st = PEND; next     # a foreign header or an owed row: the second row decides
      }
      if (st == PEND) {
        cells($0)
        if (sep) { pend = ""; st = FOREIGN; next }
        flush(); st = OWED             # headerless block: owed rows (fail closed)
      }
      if (st == OWED) owed($0)
    }
    END {
      flush()
      if (pipes && !seen)
        printf "no values-owed table (header Slot | Owed by | Unit | Safe default | Status) in %s\n", F
    }' "$OWED")
  if [ -n "$owed_rows" ]; then
    while IFS= read -r _ln; do
      emit MANUAL "values-owed" "$_ln"
    done <<< "$owed_rows"
  else
    emit PASS "values-owed" "every value owed by the field is provided ($OWED)"
  fi
else
  emit MANUAL "values-owed" \
    "no docs/values-owed.md — declare the values owed by the field (slot, owed by, unit, safe default, status) or state none (BUILD-LOOP §6.b)"
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
emit MANUAL "persisted-state-restart" \
  "persisted operator state (HOA modes, setpoints, run hours, backups) seeded or saved BEFORE the first restart and re-read after it (BUILD-LOOP §6.b)"
emit MANUAL "alarm-routing" \
  "each alarm source has an alarm class and at least one recipient; a test alarm reaches the console and is acknowledged (BUILD-LOOP §6.b)"
emit MANUAL "consumer-impact" \
  "every point whose meaning changed (feature doc Consumer impact table) was notified to its downstream consumers (BUILD-LOOP §6.b)"
emit MANUAL "link-source-audit" \
  "run obix-link-audit.sh --map docs/wiring-map.md against the live facade: every declared link-in comes from its declared source (MATCH)"
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
