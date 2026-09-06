#!/usr/bin/env bash
# slot-coverage.sh — MM2 exposed-set coverage (palette / lexicon / type-registration).
#
# Operationalizes three silent-deploy footguns as one set-difference metric before deploy:
#   - empty module.palette passes the gate yet nothing drags in Workbench (B5)
#   - missing module.lexicon key renders raw camelCase in operator views (T8)
#   - dangling module-include.xml <type> surfaces as "Missing class" live (B12)
#
# Usage (pure subcommand — QA-pinned via exact MM2 vectors):
#   slot-coverage.sh set-coverage <declared-csv> <required-csv>
#   ("" = empty set; each set deduped + lexicographically sorted)
#   stdout (exactly 3 lines, sets comma-joined, empty string after = when set is empty):
#       pct=<n.n|N/A>
#       missing=<sorted,csv>
#       extra=<sorted,csv>
#   exit 0   always for set-coverage
#   exit 2   argc != 2  +  "usage: slot-coverage.sh set-coverage ..." on stderr
#
# Usage (per-slot subcommand — Campaign 8 PR5, D6):
#   slot-coverage.sh per-slot <module-include.xml> <module.lexicon> <src-dir>
#   required   <- OPERATOR-flagged @NiagaraProperty slots from *.java under <src-dir>
#                 (dot-dirs pruned — D9b; non-OPERATOR slots are NOT required to have a key)
#   declared   <- lexicon slot names: bare key "fan" -> "fan"; "FanMode.fan" -> "fan" (after dot)
#   MISSING    <- OPERATOR slot with no lexicon key (renders raw camelCase in operator views)
#   STALE      <- lexicon key matching no "known key" — a truly dead translation.
#                 known keys = @NiagaraProperty slots (any flags) ∪ type display-name keys
#                 (from module-include.xml <type name="X">) ∪ @Range enum-tag values.
#                 A key for a READONLY slot, a type name, or a frozen-enum tag is live, NOT stale.
#   stdout:    pct=<n.n> (per-slot)   followed by MISSING <slot> and STALE <key> lines
#   exit 0   no MISSING slots
#   exit 1   any MISSING slot
#   exit 2   bad arity + "usage: slot-coverage.sh per-slot ..." on stderr
#   exit 3   file/dir not found
#
# Usage (parse subcommand — reads module files, calls the same pure function):
#   slot-coverage.sh [--strict] <module-include.xml> <module.lexicon>
#   required   <- <type name="X"> entries in module-include.xml
#   declared   <- key=value lines in module.lexicon (comments and blank lines ignored):
#                   bare key  (e.g. "fan")          -> declared name = fan
#                   Type.slot (e.g. "FanMode.fan")  -> declared name = FanMode (before the dot)
#                 Both forms name the same type; the part before the first dot is the type name.
#   |required|==0   -> N/A
#   empty lexicon + |required|>=1 -> pct=0.0 + "slot-coverage: FAIL empty lexicon ..." on stdout + exit 1
#   duplicate keys in lexicon -> "slot-coverage: WARN dup-keys: <key>" on stdout (B759/B780)
#   --strict   -> exit 1 when missing is non-empty
#   exit 0   clean or WARN-only (empty-lexicon FAIL always exits 1 regardless of --strict)
#   exit 2   wrong argc
#   exit 3   env (file missing / unreadable)
#
# Integer-tenths rounding: t=(1000*present+R/2)/R; print "$((t/10)).$((t%10))"
# (verbatim from verify-module.sh coverage — no awk/printf %.1f; no locale dependency)
# This script is VCS-free by design. version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
set -u

# ---------------------------------------------------------------------------
# Shared helpers
# ---------------------------------------------------------------------------

# csv_to_lines: convert comma-separated string to sorted, deduped newlines.
# Empty input -> no output at all (not a blank line).
csv_to_lines() {
  local v="$1"
  [ -z "$v" ] && return 0
  printf '%s\n' "$v" | tr ',' '\n' | grep -v '^$' | sort -u
}

# run_set_coverage <declared-csv> <required-csv>
# Prints exactly 3 lines: pct=, missing=, extra=
run_set_coverage() {
  local _decl="$1" _req="$2"
  local req_lines decl_lines R present t m e

  req_lines=$(csv_to_lines "$_req")
  decl_lines=$(csv_to_lines "$_decl")

  R=0
  [ -n "$req_lines" ] && R=$(printf '%s\n' "$req_lines" | wc -l | tr -d ' \t')

  # pct (N/A when |required| == 0 — never a false 100 for a scaffold module)
  if [ "$R" -eq 0 ]; then
    printf 'pct=N/A\n'
  else
    present=0
    if [ -n "$decl_lines" ]; then
      present=$(comm -12 \
        <(printf '%s\n' "$req_lines") \
        <(printf '%s\n' "$decl_lines") \
        | wc -l | tr -d ' \t')
    fi
    # integer-tenths rounding — verbatim from verify-module.sh coverage
    # integer-tenths rounding — verbatim from verify-module.sh coverage
    t=$(( (1000 * present + R / 2) / R ))
    printf 'pct=%d.%d\n' "$((t / 10))" "$((t % 10))"
  fi

  # missing = required - declared (the coverage gap — the footgun)
  if [ -z "$req_lines" ]; then
    printf 'missing=\n'
  elif [ -z "$decl_lines" ]; then
    printf 'missing=%s\n' "$(printf '%s\n' "$req_lines" | paste -sd ',' -)"
  else
    m=$(comm -23 \
      <(printf '%s\n' "$req_lines") \
      <(printf '%s\n' "$decl_lines") \
      | paste -sd ',' -)
    printf 'missing=%s\n' "$m"
  fi

  # extra = declared - required (dangling lint — reported, NOT scored)
  if [ -z "$decl_lines" ]; then
    printf 'extra=\n'
  elif [ -z "$req_lines" ]; then
    printf 'extra=%s\n' "$(printf '%s\n' "$decl_lines" | paste -sd ',' -)"
  else
    e=$(comm -23 \
      <(printf '%s\n' "$decl_lines") \
      <(printf '%s\n' "$req_lines") \
      | paste -sd ',' -)
    printf 'extra=%s\n' "$e"
  fi
}

# ---------------------------------------------------------------------------
# Per-slot subcommand (dispatched BEFORE flag loop — D6, Campaign 8 PR5).
# Compares OPERATOR @NiagaraProperty slot names against lexicon keys.
# ---------------------------------------------------------------------------
if [ $# -ge 1 ] && [ "$1" = "per-slot" ]; then
  shift
  if [ $# -ne 3 ]; then
    printf 'usage: slot-coverage.sh per-slot <module-include.xml> <module.lexicon> <src-dir>\n' >&2
    exit 2
  fi
  PS_XML="$1"; PS_LEX="$2"; PS_SRC="$3"
  [ -f "$PS_XML" ] || { printf 'slot-coverage: not found: %s\n' "$PS_XML" >&2; exit 3; }
  [ -f "$PS_LEX" ] || { printf 'slot-coverage: not found: %s\n' "$PS_LEX" >&2; exit 3; }
  [ -d "$PS_SRC" ] || { printf 'slot-coverage: not a directory: %s\n' "$PS_SRC" >&2; exit 3; }

  # Extract @NiagaraProperty slot names from Java files (dot-dirs pruned, D9b).
  # Uses paren-balance multi-line state machine — same technique as lint-delays.sh Pass 1.
  # Two outputs:
  #   op_slots_raw  — OPERATOR-flagged only (Flags.OPERATOR or "o"); gates MISSING (RED/K13)
  #   all_slots_raw — every @NiagaraProperty name, any flags; used to build known_sorted
  # STALE = lex_keys - known_sorted; known_sorted = all_annotation_slots ∪ type names ∪ @Range tags.
  # Type display-name keys and frozen-enum tag keys are live translations, not dead.
  # shellcheck disable=SC2016  # awk programs intentionally in single quotes (no shell expansion needed)
  _java_files=$(find "$PS_SRC" -type d -name '.*' -prune -o -name '*.java' -print | sort)
  # shellcheck disable=SC2016
  op_slots_raw=$(printf '%s\n' "$_java_files" | xargs awk '
    BEGIN { in_prop=0; prop_buf=""; prop_name=""; prop_op=0 }
    {
      ln = $0
      if (!in_prop) {
        if (index(ln, "@NiagaraProperty") > 0) {
          in_prop=1; prop_buf=ln; prop_name=""; prop_op=0
        }
      } else {
        prop_buf = prop_buf " " ln
      }
      if (in_prop) {
        if (prop_name == "" && match(prop_buf, /name[[:space:]]*=[[:space:]]*"[^"]*"/)) {
          seg = substr(prop_buf, RSTART)
          sub(/name[[:space:]]*=[[:space:]]*"/, "", seg)
          sub(/".*/, "", seg)
          prop_name = seg
        }
        if (!prop_op && (index(prop_buf, "OPERATOR") > 0 || index(prop_buf, "\"o\"") > 0)) {
          prop_op = 1
        }
        depth=0; n=length(prop_buf)
        for (ci=1; ci<=n; ci++) {
          c = substr(prop_buf, ci, 1)
          if      (c == "(") depth++
          else if (c == ")") depth--
        }
        if (depth <= 0 && index(prop_buf, "@NiagaraProperty") > 0) {
          if (prop_name != "" && prop_op) print prop_name
          in_prop=0; prop_buf=""; prop_name=""; prop_op=0
        }
      }
    }
    ' 2>/dev/null)
  # All annotated slots (any flags) — used for STALE computation only
  # shellcheck disable=SC2016
  all_slots_raw=$(printf '%s\n' "$_java_files" | xargs awk '
    BEGIN { in_prop=0; prop_buf=""; prop_name="" }
    {
      ln = $0
      if (!in_prop) {
        if (index(ln, "@NiagaraProperty") > 0) {
          in_prop=1; prop_buf=ln; prop_name=""
        }
      } else {
        prop_buf = prop_buf " " ln
      }
      if (in_prop) {
        if (prop_name == "" && match(prop_buf, /name[[:space:]]*=[[:space:]]*"[^"]*"/)) {
          seg = substr(prop_buf, RSTART)
          sub(/name[[:space:]]*=[[:space:]]*"/, "", seg)
          sub(/".*/, "", seg)
          prop_name = seg
        }
        depth=0; n=length(prop_buf)
        for (ci=1; ci<=n; ci++) {
          c = substr(prop_buf, ci, 1)
          if      (c == "(") depth++
          else if (c == ")") depth--
        }
        if (depth <= 0 && index(prop_buf, "@NiagaraProperty") > 0) {
          if (prop_name != "") print prop_name
          in_prop=0; prop_buf=""; prop_name=""
        }
      }
    }
    ' 2>/dev/null)

  # Lexicon slot keys: bare "fan" -> "fan"; "FanMode.fan" -> "fan" (part after last dot)
  lex_slots_raw=$(grep -vE '^[[:space:]]*#|^[[:space:]]*$' "$PS_LEX" \
    | grep '=' | cut -d'=' -f1 | sed 's/.*\.//' | sort -u)

  # Type names from module-include.xml: <type name="X" ...> -> "X"
  # These appear as top-level lexicon keys by Niagara convention (TypeName=Display Name).
  type_names_raw=$(grep -ohE '<type[[:space:]]+[^>]*\bname="[^"]*"' "$PS_XML" \
    | grep -oE '\bname="[^"]*"' | sed 's/^name="//;s/"$//')

  # @Range enum tags from Java sources (dot-dirs pruned): @Range("tag") -> "tag"
  # Frozen-enum types declare tags with @Range annotations; the tag names appear as
  # lexicon keys (e.g. "schedule=Schedule") for operator display of enum values.
  range_tags_raw=$(printf '%s\n' "$_java_files" \
    | xargs grep -ohE '@Range\("[^"]+"\)' 2>/dev/null \
    | sed 's/@Range("//;s/")//')

  # Sort and deduplicate each set
  if [ -n "$op_slots_raw" ]; then
    op_sorted=$(printf '%s\n' "$op_slots_raw" | sort -u)
  else
    op_sorted=""
  fi

  # known_sorted = @NiagaraProperty slots (any flags) ∪ type names ∪ @Range tags
  # This is the complete set of "live" lexicon keys — anything in lex_keys - known_sorted
  # is a genuinely dead translation (no matching annotation or type declaration).
  _known_parts=""
  [ -n "$all_slots_raw"  ] && _known_parts="$all_slots_raw"
  [ -n "$type_names_raw" ] && _known_parts="$_known_parts
$type_names_raw"
  [ -n "$range_tags_raw" ] && _known_parts="$_known_parts
$range_tags_raw"
  if [ -n "$_known_parts" ]; then
    known_sorted=$(printf '%s\n' "$_known_parts" | grep -v '^$' | sort -u)
  else
    known_sorted=""
  fi

  # MISSING = required OPERATOR slots with no lexicon key
  if [ -n "$op_sorted" ] && [ -n "$lex_slots_raw" ]; then
    missing_slots=$(comm -23 \
      <(printf '%s\n' "$op_sorted") \
      <(printf '%s\n' "$lex_slots_raw"))
  elif [ -n "$op_sorted" ]; then
    missing_slots="$op_sorted"
  else
    missing_slots=""
  fi

  # STALE = lex_keys - known_sorted (truly dead translations only).
  # known_sorted = @NiagaraProperty slots (any flags) ∪ type names ∪ @Range enum tags.
  # A key for a READONLY slot, a type display name, or a frozen-enum value tag is live.
  if [ -n "$lex_slots_raw" ] && [ -n "$known_sorted" ]; then
    stale_slots=$(comm -23 \
      <(printf '%s\n' "$lex_slots_raw") \
      <(printf '%s\n' "$known_sorted"))
  elif [ -n "$lex_slots_raw" ]; then
    stale_slots="$lex_slots_raw"
  else
    stale_slots=""
  fi

  # Compute pct (per-slot): covered = required - missing
  R_PS=0
  [ -n "$op_sorted" ] && R_PS=$(printf '%s\n' "$op_sorted" | grep -c '.')
  if [ "$R_PS" -eq 0 ]; then
    printf 'pct=N/A (per-slot)\n'
  else
    missing_count=0
    [ -n "$missing_slots" ] && missing_count=$(printf '%s\n' "$missing_slots" | grep -c '.')
    covered_ps=$(( R_PS - missing_count ))
    t_ps=$(( (1000 * covered_ps + R_PS / 2) / R_PS ))
    printf 'pct=%d.%d (per-slot)\n' "$(( t_ps / 10 ))" "$(( t_ps % 10 ))"
  fi

  # Emit MISSING rows (K14: label states what it measures)
  if [ -n "$missing_slots" ]; then
    while IFS= read -r ps; do
      [ -n "$ps" ] && printf 'MISSING %s\n' "$ps"
    done <<< "$missing_slots"
  fi

  # Emit STALE rows
  if [ -n "$stale_slots" ]; then
    while IFS= read -r ps; do
      [ -n "$ps" ] && printf 'STALE %s\n' "$ps"
    done <<< "$stale_slots"
  fi

  # Exit 1 if any MISSING (a missing operator slot renders raw camelCase — ship-blocker)
  [ -z "$missing_slots" ] && exit 0 || exit 1
fi

# ---------------------------------------------------------------------------
# Pure subcommand dispatch (before flag loop so 'set-coverage' is never
# mistaken for a file-path positional arg).
# ---------------------------------------------------------------------------
if [ $# -ge 1 ] && [ "$1" = "set-coverage" ]; then
  shift
  if [ $# -ne 2 ]; then
    printf 'usage: slot-coverage.sh set-coverage <declared-csv> <required-csv>\n' >&2
    exit 2
  fi
  run_set_coverage "$1" "$2"
  exit 0
fi

# ---------------------------------------------------------------------------
# Parse subcommand
# ---------------------------------------------------------------------------
STRICT=0
while [ $# -gt 0 ]; do
  case "$1" in
    --strict) STRICT=1; shift ;;
    --) shift; break ;;
    -*)
      printf 'slot-coverage: unknown flag: %s\n' "$1" >&2
      printf 'usage: slot-coverage.sh [--strict] <module-include.xml> <module.lexicon>\n' >&2
      exit 2
      ;;
    *) break ;;
  esac
done

[ $# -eq 2 ] || {
  printf 'usage: slot-coverage.sh [--strict] <module-include.xml> <module.lexicon>\n' >&2
  exit 2
}

XML="$1"; LEX="$2"
[ -f "$XML" ] || { printf 'slot-coverage: not found: %s\n' "$XML" >&2; exit 3; }

# WB-LEX1: a -wb module-include.xml with >=1 declared type and a MISSING lexicon
# is a FAIL (exit 1), not an env fault (exit 3). A missing lexicon means every
# operator-facing type name renders as raw camelCase — same severity as an empty
# lexicon (D6a). NAMED MUTATION: restore the exit-3 path here -> WB-LEX1 flips.
if [ ! -f "$LEX" ]; then
  _req_count=$(grep -ohE '<type[[:space:]]+[^>]*\bname="[^"]*"' "$XML" 2>/dev/null \
    | grep -cE '\bname="' || true)
  if [ "${_req_count:-0}" -ge 1 ]; then
    printf 'slot-coverage: FAIL missing lexicon with %d declared type(s)\n' "$_req_count"
    exit 1
  fi
  printf 'slot-coverage: not found: %s\n' "$LEX" >&2
  exit 3
fi

# Extract required type names from module-include.xml
# Pattern: <type ... name="TypeName" ...> — extract the name= attribute value
required_csv=$(grep -ohE '<type[[:space:]]+[^>]*\bname="[^"]*"' "$XML" \
  | grep -oE '\bname="[^"]*"' \
  | sed 's/^name="//;s/"$//' \
  | sort -u | paste -sd ',' -)

# Extract declared type names from lexicon key=value lines
# (skip comment lines starting with # and blank lines)
# bare key "fan" -> "fan"; "FanMode.fan" -> "FanMode" (before the dot)
declared_csv=$(grep -vE '^[[:space:]]*#|^[[:space:]]*$' "$LEX" \
  | grep '=' | cut -d'=' -f1 | sed 's/\..*//' | sort -u | paste -sd ',' -)

# Dup-key detection: duplicate raw keys before dot-stripping (operationalizes B759/B780)
dup_keys=$(grep -vE '^[[:space:]]*#|^[[:space:]]*$' "$LEX" \
  | grep '=' | cut -d'=' -f1 | sort | uniq -d)
HAS_WARN=0
if [ -n "$dup_keys" ]; then
  while IFS= read -r dk; do
    printf 'slot-coverage: WARN dup-keys: %s\n' "$dk"
  done <<< "$dup_keys"
  HAS_WARN=1
fi

# Empty-lexicon FAIL when required is non-empty (D6a, Campaign 8 PR5).
# Was WARN/exit-0 before PR5; now FAIL/exit-1 because every operator slot renders raw camelCase
# (T8 footgun is a ship-blocker; --strict-only gate leaves the very module that motivated it green).
lex_has_keys=0
grep -qE '^[^#[:space:]].*=' "$LEX" 2>/dev/null && lex_has_keys=1

R_COUNT=0
[ -n "$required_csv" ] && R_COUNT=$(printf '%s\n' "$required_csv" | tr ',' '\n' | grep -cv '^$')

EMPTY_LEX_FAIL=0
if [ "$lex_has_keys" -eq 0 ] && [ "$R_COUNT" -ge 1 ]; then
  printf 'slot-coverage: FAIL empty lexicon with %d declared type(s)\n' "$R_COUNT"
  EMPTY_LEX_FAIL=1
fi

# Run the pure function and print the 3 coverage lines.
# The parse mode measures TYPE-set coverage (a registered type counts as declared when it
# has >=1 lexicon key), NOT per-slot completeness. Label the pct= line accordingly so
# "100.0" cannot be misread as full per-slot coverage (B788 measured ~25% per-slot on
# DashboardPan-rt despite 100% type-set coverage). [ev: corpus B788; T6.11]
sc_out=$(run_set_coverage "$declared_csv" "$required_csv")
printf '%s\n' "$sc_out" | sed 's/^pct=\(.*\)$/pct=\1 (type-set)/'

# Empty-lexicon FAIL always exits 1 — not gated on --strict (D6a: ship-blocker behaviour change).
if [ "$EMPTY_LEX_FAIL" -eq 1 ]; then
  exit 1
fi

# --strict: exit 1 when missing is non-empty or WARN was emitted
if [ "$STRICT" -eq 1 ]; then
  missing_val=$(printf '%s\n' "$sc_out" | grep '^missing=' | sed 's/^missing=//')
  if [ -n "$missing_val" ] || [ "$HAS_WARN" -eq 1 ]; then
    exit 1
  fi
fi

exit 0
