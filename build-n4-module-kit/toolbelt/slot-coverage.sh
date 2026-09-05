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
# Usage (parse subcommand — reads module files, calls the same pure function):
#   slot-coverage.sh [--strict] <module-include.xml> <module.lexicon>
#   required   <- <type name="X"> entries in module-include.xml
#   declared   <- key=value lines in module.lexicon (comments and blank lines ignored):
#                   bare key  (e.g. "fan")          -> declared name = fan
#                   Type.slot (e.g. "FanMode.fan")  -> declared name = FanMode (before the dot)
#                 Both forms name the same type; the part before the first dot is the type name.
#   |required|==0   -> N/A
#   empty lexicon + |required|>=1 -> pct=0.0 + "slot-coverage: WARN empty lexicon ..." on stdout
#   duplicate keys in lexicon -> "slot-coverage: WARN dup-keys: <key>" on stdout (B759/B780)
#   --strict   -> exit 1 when missing is non-empty
#   exit 0   clean or WARN-only
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
[ -f "$LEX" ] || { printf 'slot-coverage: not found: %s\n' "$LEX" >&2; exit 3; }

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

# Empty-lexicon WARN when required is non-empty (CompPan T8 footgun: slots render raw camelCase)
lex_has_keys=0
grep -qE '^[^#[:space:]].*=' "$LEX" 2>/dev/null && lex_has_keys=1

R_COUNT=0
[ -n "$required_csv" ] && R_COUNT=$(printf '%s\n' "$required_csv" | tr ',' '\n' | grep -cv '^$')

if [ "$lex_has_keys" -eq 0 ] && [ "$R_COUNT" -ge 1 ]; then
  printf 'slot-coverage: WARN empty lexicon with %d declared type(s)\n' "$R_COUNT"
  HAS_WARN=1
fi

# Run the pure function and print the 3 coverage lines
sc_out=$(run_set_coverage "$declared_csv" "$required_csv")
printf '%s\n' "$sc_out"

# --strict: exit 1 when missing is non-empty or WARN was emitted
if [ "$STRICT" -eq 1 ]; then
  missing_val=$(printf '%s\n' "$sc_out" | grep '^missing=' | sed 's/^missing=//')
  if [ -n "$missing_val" ] || [ "$HAS_WARN" -eq 1 ]; then
    exit 1
  fi
fi

exit 0
