#!/usr/bin/env bash
# generate-wiring-map.sh — Scaffold a wiring-map.md template for a per-instance rearchitect.
#
# Scans @NiagaraProperty annotations in Java files under <facade-src-dir> and emits
# a markdown wiring-map template with three sections:
#
#   Table 1 — OPERATOR config slots (facade → control):
#     Setpoints, HOA modes, defrost settings, differentials that the facade writes
#     into the corresponding rt control slots.
#
#   Table 2 — SUMMARY display slots (control → facade):
#     Temperature readings, statuses, timer anchors that the integrator links
#     FROM the rt control module INTO the facade.
#
#   Table 3 — Per-instance commissioning checklist (BUILD-LOOP §6.b).
#
# Each table row carries the internal slot name, the Workbench display name and a full-ord
# column: the operator searches the live tree by the DISPLAY name, so a link table that names
# only the internal slot has to be redone by hand against Workbench. The display name is filled
# from the module lexicon (key "<Type>.<slot>" first, then the bare "<slot>" key; Type = class
# name without its leading B); a slot with no key shows "_(no lexicon key)_" — a lexicon gap
# slot-coverage.sh per-slot also reports. [ev: retro panccadia-commissioning-lessons Δ9]
#
# The developer fills in the RT-slot, Full-ord and "Physical instance / crossing notes"
# columns before handing off the module for commissioning. The operator must not hand-derive
# the crossing.
#
# Usage:  generate-wiring-map.sh <facade-src-dir> [--lexicon <module.lexicon>] [--output <file>]
#
#   <facade-src-dir>   Source tree of the facade (-rt) profile.
#   --lexicon <file>   Lexicon for the display-name column. Default: <facade-src-dir>/module.lexicon,
#                      else <facade-src-dir>/../module.lexicon; none found -> placeholder names.
#   --output <file>    Write to <file> instead of stdout.
#
# Exits: 0 ok · 3 usage/env (K20)
#
# [ev: retro live-commissioning-verification-gaps Δ6]
# Dot-directories excluded (D9b). VCS-free by design.

set -u
LC_ALL=C
export LC_ALL

# --- usage ---
if [ $# -lt 1 ]; then
  printf 'usage: generate-wiring-map.sh <facade-src-dir> [--lexicon <module.lexicon>] [--output <file>]\n' >&2
  exit 3
fi

SRC="$1"; shift
OUTFILE="-"   # stdout by default
LEXICON=""

while [ $# -gt 0 ]; do
  case "$1" in
    --output)
      [ $# -ge 2 ] || { printf 'generate-wiring-map: --output requires a filename\n' >&2; exit 3; }
      OUTFILE="$2"; shift 2 ;;
    --lexicon)
      [ $# -ge 2 ] || { printf 'generate-wiring-map: --lexicon requires a filename\n' >&2; exit 3; }
      LEXICON="$2"; shift 2
      [ -f "$LEXICON" ] || { printf 'generate-wiring-map: lexicon not found: %s\n' "$LEXICON" >&2; exit 3; } ;;
    *)
      printf 'generate-wiring-map: unknown option: %s\n' "$1" >&2; exit 3 ;;
  esac
done

[ -d "$SRC" ] || { printf 'generate-wiring-map: not a directory: %s\n' "$SRC" >&2; exit 3; }
if [ -z "$LEXICON" ]; then
  for _cand in "$SRC/module.lexicon" "$SRC/../module.lexicon"; do
    [ -f "$_cand" ] && { LEXICON="$_cand"; break; }
  done
fi

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT
_OP="$_TMP/operator.tsv"
_SUM="$_TMP/summary.tsv"
touch "$_OP" "$_SUM"

# ---------------------------------------------------------------------------
# Extract @NiagaraProperty slots from all *.java under <facade-src-dir>
# ---------------------------------------------------------------------------
while IFS= read -r f; do
  _type=$(basename "$f" .java)
  case "$_type" in B[A-Z]*) _type="${_type#B}" ;; esac
  awk -v OP="$_OP" -v SUM="$_SUM" -v TYPE="$_type" '
  BEGIN { in_prop = 0; prop_buf = ""; prop_line = 0 }
  FNR == 1 { in_prop = 0; prop_buf = ""; prop_line = 0 }

  # A single-line annotation must close on its own line (no "next" here): the old form glued it
  # to the following annotation and lost the second slot.
  !in_prop && index($0, "@NiagaraProperty") > 0 {
    in_prop = 1; prop_buf = ""; prop_line = FNR
  }
  in_prop {
    prop_buf = prop_buf " " $0
    depth = 0
    for (ci = 1; ci <= length(prop_buf); ci++) {
      c = substr(prop_buf, ci, 1)
      if (c == "(") depth++
      else if (c == ")") depth--
    }
    if (depth <= 0 && index(prop_buf, "(") > 0) {
      pname = ""
      if (match(prop_buf, /name[[:space:]]*=[[:space:]]*"[^"]+"/)) {
        seg = substr(prop_buf, RSTART)
        sub(/name[[:space:]]*=[[:space:]]*"/, "", seg)
        sub(/".*/, "", seg)
        pname = seg
      }
      if (pname != "") {
        is_operator = (index(prop_buf, "OPERATOR") > 0)
        if (is_operator) {
          printf "%s\t%s\n", pname, TYPE >> OP
        } else if (index(prop_buf, "SUMMARY") > 0 || index(prop_buf, "READONLY") > 0) {
          printf "%s\t%s\n", pname, TYPE >> SUM
        }
      }
      in_prop = 0; prop_buf = ""; prop_line = 0
    }
    next
  }
  ' "$f"
done < <(find "$SRC" -type d -name '.*' -prune -o -name '*.java' -print | sort)

# ---------------------------------------------------------------------------
# Display names: "<slot>\t<Type>" -> "<slot>\t<display>" from the lexicon (Type.slot, then slot).
# ---------------------------------------------------------------------------
display_names() {  # tsv-file
  awk -F'\t' -v LEX="$LEXICON" '
    BEGIN {
      if (LEX != "") {
        while ((getline ln < LEX) > 0) {
          if (ln ~ /^[[:space:]]*#/ || index(ln, "=") == 0) continue
          k = substr(ln, 1, index(ln, "=") - 1); v = substr(ln, index(ln, "=") + 1)
          gsub(/^[[:space:]]+|[[:space:]]+$/, "", k); gsub(/^[[:space:]]+|[[:space:]]+$/, "", v)
          gsub(/\|/, "\\|", v)
          if (k != "" && !(k in lex)) lex[k] = v
        }
      }
    }
    {
      d = "_(no lexicon key)_"
      if (($2 "." $1) in lex) d = lex[$2 "." $1]
      else if ($1 in lex) d = lex[$1]
      print $1 "\t" d
    }
  ' "$1"
}

# ---------------------------------------------------------------------------
# Render markdown
# SC2016 disabled: single-quoted printf format strings containing %s are intentional
# format specifiers, not shell expressions.
# shellcheck disable=SC2016
# ---------------------------------------------------------------------------
render() {
  local module_name
  module_name=$(basename "$SRC")

  printf '# Wiring Map — %s\n' "$module_name"
  printf '<!-- Generated by generate-wiring-map.sh from %s -->\n' "$SRC"
  printf '<!-- Fill in the RT-slot, Full-ord and Notes columns before commissioning hand-off. -->\n'
  printf '<!-- Rule: operator must not hand-derive the crossing  [ev: retro live-commissioning-verification-gaps Δ6] -->\n'
  printf '\n'

  printf '## Table 1 — OPERATOR config slots (facade → control)\n'
  printf '\n'
  printf 'These slots carry operator-visible setpoints and mode choices.\n'
  printf 'Each facade slot must be **linked facade→control** (the facade is the SOURCE, not the TARGET).\n'
  printf '\n'
  printf '| Facade slot | Workbench display name | Target RT slot | Full ord | Physical instance / crossing notes |\n'
  printf '|-------------|------------------------|----------------|----------|-------------------------------------|\n'
  if [ -s "$_OP" ]; then
    while IFS="$(printf '\t')" read -r slot disp; do
      printf '| `%s` | %s | _(fill)_ | _(fill)_ | |\n' "$slot" "$disp"
    done < <(display_names "$_OP")
  else
    printf '| _(no OPERATOR slots found — check src dir or slot flags)_ | | | | |\n'
  fi

  printf '\n'
  printf '## Table 2 — SUMMARY display slots (control → facade)\n'
  printf '\n'
  printf 'These slots display telemetry linked FROM the control module.\n'
  printf 'Each facade slot must be **linked control→facade** (the control point is the SOURCE).\n'
  printf '\n'
  printf '| Facade slot | Workbench display name | Source RT slot | Full ord | Physical instance / crossing notes |\n'
  printf '|-------------|------------------------|----------------|----------|-------------------------------------|\n'
  if [ -s "$_SUM" ]; then
    while IFS="$(printf '\t')" read -r slot disp; do
      printf '| `%s` | %s | _(fill)_ | _(fill)_ | |\n' "$slot" "$disp"
    done < <(display_names "$_SUM")
  else
    printf '| _(no SUMMARY display slots found — check src dir or slot flags)_ | | | | |\n'
  fi

  printf '\n'
  printf '## Per-instance commissioning checklist (BUILD-LOOP §6.b)\n'
  printf '\n'
  printf 'Run after wiring, before operator hand-off:\n'
  printf '\n'
  printf '%s\n' '- [ ] Every OPERATOR config slot linked facade→control (not to a display slot, not to a link-target side)'
  printf '%s\n' '- [ ] Every SUMMARY display slot linked control→facade'
  printf '%s\n' '- [ ] Physical mapping / crossing documented in the Notes columns above'
  printf '%s\n' '- [ ] Config sanity: interval > duration, setpoints ≠ 0, hasDefrost=true AND airDefrost=true on air-defrost units'
  printf '%s\n' '- [ ] After hot reload: triage-console.sh clean (no own-module load failures)'
  printf '%s\n' '- [ ] After hot reload: bog-audit.sh CHECK11 clean (proxy-link fallback set)'
  printf '%s\n' '- [ ] obix-nav.py facade<->rt link batch audit: all expected links present and values in range'
}

# ---------------------------------------------------------------------------
# Output
# ---------------------------------------------------------------------------
if [ "$OUTFILE" = "-" ]; then
  render
else
  render > "$OUTFILE"
  printf 'generate-wiring-map: wrote %s\n' "$OUTFILE"
fi
exit 0
