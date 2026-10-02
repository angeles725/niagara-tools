#!/usr/bin/env bash
# gen-lint-index.sh — generates toolbelt/INDEX.md, the lint index, from each toolbelt/lint-*.sh header.
#
# BUILD-LOOP §5/§6.b used to enumerate every lint inline; each new lint edited that enumeration
# (a merge-conflict hotspot) and the prose drifted. The index is now DERIVED: one row per
# toolbelt/lint-*.sh with its header description, Usage:, Exit contract, [ev: ...] tags and
# whether report-module.sh runs it automatically. Prose never enumerates lints or counts them.
# [ev: retro kit-meta-hygiene-2026-10-01 Δ3]
#
# Header contract (lines 1-60 of every lint-*.sh):
#   line 2      # <script-name> — <what it checks>   (continuation lines joined until a blank '#',
#               a keyword line or line 5: the description spans header lines 2-5 at most, so a
#               longer explanation belongs below the blank '#' and stays out of the index)
#   Usage line  # Usage: <script-name> [flags] <arg>
#   Exit line   # Exit: ... | # Exits: ... | # Exit <code> ...
# Every Usage and Exit line is kept, in order (joined with <br> in the cell). A '#' line whose text
# starts at or right of the value column of the Usage/Exit line above it is a wrapped continuation
# and is joined with a space; a less-indented line (an explanation) or a 'Word:' line is not.
# A lint missing any of the three is a contract violation (exit 1, nothing written).
# Auto = yes only when a non-comment line of report-module.sh invokes "$TOOLBELT/<script-name>"
# (a comment, a message string or a longer script name does not count).
#
# Mutation: GLI-stale -- dropping the generated-vs-committed compare makes --check vacuously clean
# Mutation: GLI-auto -- a plain substring grep marks a lint named only in a comment as Auto yes
# Mutation: GLI-wrap-exit -- keeping only the first Exit line truncates a wrapped exit contract
# Mutation: GLI-multi -- keeping only the first Usage/Exit line drops a second contract line
# Mutation: GLI-awkfail -- ignoring the awk exit status writes an index of empty rows
#
# Usage:  gen-lint-index.sh [--check] [<kit-root>]     (default kit root: this script's ../)
# Exits:  0 written / index fresh · 1 stale or missing index under --check, or header-contract
#         violation · 3 usage/env, or awk failed on a header (K20; nothing written)
# VCS-free by design (kit-links L2). Deterministic: LC_ALL=C sort, no timestamps.
set -u
LC_ALL=C
export LC_ALL

usage_exit() {
  printf 'usage: gen-lint-index.sh [--check] [<kit-root>]\n' >&2
  exit 3
}

CHECK=0
ROOT=""
while [ $# -gt 0 ]; do
  case "$1" in
    --check) CHECK=1; shift ;;
    -*) usage_exit ;;
    *) [ -z "$ROOT" ] || usage_exit; ROOT="$1"; shift ;;
  esac
done
if [ -z "$ROOT" ]; then
  ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fi
TB="$ROOT/toolbelt"
[ -d "$TB" ] || { printf 'gen-lint-index: no toolbelt/ under %s\n' "$ROOT" >&2; exit 3; }
OUT="$TB/INDEX.md"

# Extract "desc<TAB>usage<TAB>exit<TAB>ev" from one header, or "ERR<TAB><reason>".
# Several Usage (or Exit) lines are joined with "<br>"; the caller splits usages on it.
extract() {  # file name
  awk -v name="$2" '
    function clean(s) { gsub(/\|/, "\\|", s); gsub(/[[:space:]]+/, " ", s); sub(/^ /, "", s); sub(/ $/, "", s); return s }
    function strip_ev(s) { gsub(/\[ev: [^]]*\]/, "", s); return s }
    function addline(cur, v) { return cur == "" ? v : cur "<br>" v }
    NR > 60 { exit }
    {
      line = $0
      tmp = line
      while (match(tmp, /\[ev: [^]]*\]/)) {
        tag = substr(tmp, RSTART, RLENGTH)
        if (!(tag in seen)) { seen[tag] = 1; ev = ev (ev == "" ? "" : " ") tag }
        tmp = substr(tmp, RSTART + RLENGTH)
      }
    }
    NR == 2 {
      pre = "# " name
      if (index(line, pre) == 1) {
        rest = substr(line, length(pre) + 1)
        if (sub(/^[[:space:]]*(—|--|-)[[:space:]]*/, "", rest)) { desc = rest; joining = 1 }
      }
      next
    }
    # Description cap: header lines 3-5 only (documented in the header contract above).
    joining && NR <= 5 {
      if (line ~ /^#[[:space:]]*$/ || line !~ /^#/ || line ~ /^#[[:space:]]*(Usage|Exits?|Scope|Mutation|Row)/) { joining = 0 }
      else { t = line; sub(/^#[[:space:]]*/, "", t); desc = desc " " t }
    }
    # Wrapped continuation of the Usage/Exit line above: text at or right of its value column.
    cont != "" {
      if (line ~ /^#[[:space:]]*[^[:space:]]/ && line !~ /^#[[:space:]]*[A-Za-z][A-Za-z-]*:/ && match(line, /^#[[:space:]]*/) && RLENGTH + 1 >= vcol) {
        t = substr(line, RLENGTH + 1)
        if (cont == "u") usage = usage " " t; else exitc = exitc " " t
        next
      }
      cont = ""
    }
    line ~ /^#[[:space:]]*Usage:/ {
      match(line, /^#[[:space:]]*Usage:[[:space:]]*/); vcol = RLENGTH + 1
      usage = addline(usage, substr(line, RLENGTH + 1)); cont = "u"; next
    }
    line ~ /^#[[:space:]]*Exits?[:[:space:]]/ {
      match(line, /^#[[:space:]]*Exits?:?[[:space:]]*/); vcol = RLENGTH + 1
      exitc = addline(exitc, substr(line, RLENGTH + 1)); cont = "x"; next
    }
    END {
      if (desc == "")  { printf "ERR\tline 2 is not \"# %s — <what it checks>\"\n", name; exit }
      if (usage == "") { printf "ERR\tno \"# Usage:\" line in the header (lines 1-60)\n"; exit }
      if (exitc == "") { printf "ERR\tno \"# Exit:\"/\"# Exits:\" line in the header (lines 1-60)\n"; exit }
      printf "%s\t%s\t%s\t%s\n", clean(strip_ev(desc)), clean(usage), clean(exitc), clean(ev)
    }
  ' "$1"
}

# Auto: a non-comment line of report-module.sh invokes "$TOOLBELT/<name>" (or ${TOOLBELT}/<name>).
invoked_by_report() {  # name
  local rm="$TB/report-module.sh" esc
  [ -f "$rm" ] || return 1
  esc="${1//./\\.}"
  grep -vE '^[[:space:]]*#' "$rm" | grep -qE "[\$][{]?TOOLBELT[}]?/${esc}([\"'[:space:]]|\$)"
}

render() {
  cat <<'EOF'
<!-- GENERATED by toolbelt/gen-lint-index.sh from each toolbelt/lint-*.sh header. Do not edit by hand. -->
<!-- Regenerate: bash toolbelt/gen-lint-index.sh · CI freshness gate: bash toolbelt/gen-lint-index.sh --check -->
# Toolbelt lint index

One row per `toolbelt/lint-*.sh`, derived from the script's own header: the description (line 2,
`# <name> — <what it checks>`, up to header line 5), every `Usage:` line, every `Exit:`/`Exits:` line
(wrapped lines joined; several lines separated by a line break) and any `[ev: ...]` tags. **Auto** says
whether `report-module.sh` (run by `build.sh` after every successful build) invokes the lint; a `no`
lint is run by hand on the profiles its usage names.

To add a lint: write its header to this contract, regenerate, commit both. Never edit this file and
never enumerate or count lints in BUILD-LOOP.md prose. [ev: retro kit-meta-hygiene-2026-10-01 Δ3]

| Lint | What it checks | Usage | Exit contract | Auto | Evidence |
|---|---|---|---|---|---|
EOF
  local f name row desc usage exitc ev auto ucell
  for f in "$TB"/lint-*.sh; do
    [ -f "$f" ] || continue
    name="$(basename "$f")"
    if ! row="$(extract "$f" "$name")" || [ -z "$row" ]; then
      printf 'gen-lint-index: %s: awk failed reading the header\n' "$name" >&2; printf '%s\n' "$ENV_MARK"; continue
    fi
    case "$row" in
      ERR*) printf 'gen-lint-index: %s: %s\n' "$name" "${row#ERR?}" >&2; printf '%s\n' "$BAD_MARK"; continue ;;
    esac
    IFS='	' read -r desc usage exitc ev <<<"$row"
    auto=no
    if invoked_by_report "$name"; then auto=yes; fi
    # one code span per usage line (lines are joined with <br>); a usage already in backticks is kept
    case "$usage" in *'`'*) ucell="$usage" ;; *) ucell="\`${usage//"<br>"/"\`<br>\`"}\`" ;; esac
    # shellcheck disable=SC2016  # literal backticks are Markdown code-span delimiters
    printf '| `toolbelt/%s` | %s | %s | %s | %s | %s |\n' "$name" "$desc" "$ucell" "$exitc" "$auto" "${ev:-—}"
  done
}

BAD_MARK='@@gen-lint-index-contract-violation@@'
ENV_MARK='@@gen-lint-index-awk-failure@@'
CONTENT="$(render)"
case "$CONTENT" in *"$ENV_MARK"*) { printf 'gen-lint-index: awk failed; index not written\n' >&2; exit 3; } ;; esac
case "$CONTENT" in *"$BAD_MARK"*) { printf 'gen-lint-index: header-contract violation; index not written\n' >&2; exit 1; } ;; esac

if [ "$CHECK" -eq 1 ]; then
  if [ ! -f "$OUT" ]; then
    printf 'gen-lint-index: %s is missing; run gen-lint-index.sh\n' "$OUT" >&2
    exit 1
  fi
  if [ "$CONTENT" != "$(cat "$OUT")" ]; then
    printf 'gen-lint-index: %s is stale; run gen-lint-index.sh and commit the result\n' "$OUT" >&2
    diff <(printf '%s\n' "$CONTENT") "$OUT" >&2 || true
    exit 1
  fi
  printf 'gen-lint-index: %s is fresh\n' "$OUT"
  exit 0
fi

printf '%s\n' "$CONTENT" > "$OUT"
printf 'gen-lint-index: wrote %s\n' "$OUT"
