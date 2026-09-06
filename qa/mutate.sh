#!/usr/bin/env bash
# qa/mutate.sh — named-mutation harness (campaign-9 verify terrain).
#
# Applies one row of a mutation table to a THROWAWAY detached worktree, runs the suite, proves the
# named pins flip GREEN -> RED, restores the file, and prints one verdict row per mutation. The
# mutant suite output is kept verbatim under --out (the "record verbatim output" the tasks require).
#
# Usage:
#   qa/mutate.sh --worktree <detached-worktree> --table <tsv> (--pr <PRn> | --id <Mid>) [--out <dir>]
#   qa/mutate.sh --table <tsv> --list [--pr <PRn>]
#
# Table (TSV; '#' comment lines; 10 columns):
#   id  pr  file  fmt  suite  flips  anchor  kind  expr  note
#   file   path RELATIVE TO THE WORKTREE ROOT of the file to mutate
#   fmt    bats | node | junit | exit   — how a failing pin is recognised in the suite output
#   suite  command run from the worktree root via bash -c; $QA (this dir) and $KIT (build-n4-module-kit)
#          are exported for it
#   flips  comma-separated tokens; after the mutation EVERY token must appear on a failure line
#          (bats/node: '^\s*not ok .*TOKEN'; junit: '^[0-9]+\) TOKEN'; exit: suite exit != 0)
#   anchor ERE that must match >= 1 line of <file> BEFORE mutating (content anchor, never a line number)
#   kind   sed | manual   expr = a `sed -E` script (kind=sed) or a prose description (kind=manual)
#
# Verdicts (one line per row):
#   OBSERVED        flip seen for every token, tree restored          (the evidence line to paste)
#   NO-FLIP         mutant suite did not fail on every named token    -> the pin does not bite
#   NO-OP-MUTATION  sed changed nothing                               -> re-anchor the expr at GREEN
#   ANCHOR-MISSING  anchor absent from the file                       -> implementation shape differs; hand-mutate
#   COMPILE-BROKEN  junit suite exit 2 (javac failed)                 -> ambiguous evidence, not a flip
#   BASELINE-RED    the suite is not green BEFORE mutating            -> nothing to prove
#   MANUAL          kind=manual rows are listed, never applied
# Exit: 0 every applied row OBSERVED (MANUAL rows allowed) · 1 any NO-FLIP/NO-OP/ANCHOR-MISSING/
#       COMPILE-BROKEN/BASELINE-RED · 3 usage / safety refusal
#
# Safety: refuses a worktree whose HEAD is on a branch (a real checkout), refuses a dirty tree, restores
#         with `git checkout -- <file>` and re-checks `git status --porcelain` is empty after every row.
#         Mutations never touch the real source: pass a `git worktree add --detach` copy.
set -uo pipefail

usage() { sed -n '2,40p' "$0" | sed 's/^# \{0,1\}//'; exit 3; }
die() { echo "mutate: $*" >&2; exit 3; }

WT=""; TABLE=""; PR=""; ID=""; OUT=""; LIST=0
while [ $# -gt 0 ]; do
  case "$1" in
    --worktree) WT=$2; shift 2 ;;
    --table)    TABLE=$2; shift 2 ;;
    --pr)       PR=$2; shift 2 ;;
    --id)       ID=$2; shift 2 ;;
    --out)      OUT=$2; shift 2 ;;
    --list)     LIST=1; shift ;;
    -h|--help)  usage ;;
    *) die "unknown arg: $1" ;;
  esac
done
[ -n "$TABLE" ] && [ -f "$TABLE" ] || die "--table <tsv> required (missing: '${TABLE:-}')"
TABLE=$(cd "$(dirname "$TABLE")" && pwd)/$(basename "$TABLE")
QA=$(cd "$(dirname "$0")" && pwd); export QA
KIT=$(cd "$QA/.." && pwd)/build-n4-module-kit; export KIT

# ---- row selection ----
select_rows() {
  grep -vE '^\s*(#|$)' "$TABLE" | awk -F'\t' -v pr="$PR" -v id="$ID" \
    '(id != "" && $1 == id) || (id == "" && pr != "" && $2 == pr) || (id == "" && pr == "")'
}
[ -n "$(select_rows)" ] || die "no rows selected (pr='${PR:-}' id='${ID:-}')"

if [ "$LIST" -eq 1 ]; then
  printf '%-6s %-5s %-7s %-14s %-32s %s\n' ID PR KIND FLIPS FILE MUTATION
  select_rows | while IFS=$'\t' read -r id pr file fmt suite flips anchor kind expr _note; do
    printf '%-6s %-5s %-7s %-14s %-32s %s\n' "$id" "$pr" "$kind" "$flips" "$(basename "$file")" "$expr"
  done
  exit 0
fi

# ---- safety ----
[ -n "$WT" ] && [ -d "$WT" ] || die "--worktree <dir> required for applying mutations"
cd "$WT" || die "cannot cd $WT"
git rev-parse --is-inside-work-tree >/dev/null 2>&1 || die "$WT is not a git worktree"
if git symbolic-ref -q HEAD >/dev/null 2>&1; then
  die "refusing: HEAD is on branch $(git symbolic-ref --short HEAD) — mutations run only on a DETACHED throwaway worktree"
fi
[ -z "$(git status --porcelain)" ] || die "refusing: worktree is dirty (mutations need a clean tree to prove restore)"
OUT=${OUT:-$(mktemp -d "${TMPDIR:-/tmp}/mutate.XXXXXX")}; mkdir -p "$OUT"
echo "mutate: worktree=$WT HEAD=$(git rev-parse --short HEAD) table=$(basename "$TABLE") out=$OUT"

declare -A BASE_RC
baseline() {  # $1 suite -> 0 green, else rc; cached per suite string
  local key=$1
  if [ -z "${BASE_RC[$key]:-}" ]; then
    bash -c "$key" >"$OUT/baseline-$(printf '%s' "$key" | md5sum | cut -c1-8).log" 2>&1
    BASE_RC[$key]=$?
  fi
  return "${BASE_RC[$key]}"
}

flip_seen() {  # $1 fmt $2 logfile $3 token $4 rc
  case "$1" in
    bats|node) grep -qE "^[[:space:]]*not ok .*$3" "$2" ;;
    junit)     grep -qE "^[0-9]+\) $3" "$2" ;;
    exit)      [ "$4" -ne 0 ] ;;
    *)         return 1 ;;
  esac
}

FAIL=0; N=0; MAN=0
printf '\n%-16s %-6s %-40s %s\n' VERDICT ID FILE DETAIL
while IFS=$'\t' read -r id pr file fmt suite flips anchor kind expr _note; do
  N=$((N+1))
  if [ "$kind" = "manual" ]; then
    MAN=$((MAN+1)); printf '%-16s %-6s %-40s %s\n' MANUAL "$id" "$file" "$expr"; continue
  fi
  if [ ! -f "$file" ]; then
    FAIL=1; printf '%-16s %-6s %-40s %s\n' ANCHOR-MISSING "$id" "$file" "file absent in worktree"; continue
  fi
  if ! grep -qE "$anchor" "$file"; then
    FAIL=1; printf '%-16s %-6s %-40s %s\n' ANCHOR-MISSING "$id" "$file" "anchor /$anchor/ not found — hand-mutate: $expr"; continue
  fi
  if ! baseline "$suite"; then
    FAIL=1; printf '%-16s %-6s %-40s %s\n' BASELINE-RED "$id" "$file" "suite exit ${BASE_RC[$suite]} before mutating (log: $OUT)"; continue
  fi
  sed -E -i "$expr" "$file"
  if git diff --quiet -- "$file"; then
    FAIL=1; printf '%-16s %-6s %-40s %s\n' NO-OP-MUTATION "$id" "$file" "sed '$expr' changed nothing"; continue
  fi
  git diff -- "$file" >"$OUT/$id.diff"
  bash -c "$suite" >"$OUT/$id.log" 2>&1; rc=$?
  git checkout -q -- "$file"
  [ -z "$(git status --porcelain)" ] || die "restore FAILED after $id — inspect $WT before anything else"
  if [ "$fmt" = junit ] && [ "$rc" -eq 2 ]; then
    FAIL=1; printf '%-16s %-6s %-40s %s\n' COMPILE-BROKEN "$id" "$file" "javac failed under the mutant (log: $OUT/$id.log)"; continue
  fi
  seen=0; total=0; missing=""
  IFS=',' read -ra TOK <<<"$flips"
  for t in "${TOK[@]}"; do
    total=$((total+1))
    if flip_seen "$fmt" "$OUT/$id.log" "$t" "$rc"; then seen=$((seen+1)); else missing="$missing $t"; fi
  done
  if [ "$seen" -eq "$total" ] && [ "$rc" -ne 0 ]; then
    printf '%-16s %-6s %-40s %s\n' OBSERVED "$id" "$file" "flips $seen/$total [$flips] exit=$rc restored (log: $OUT/$id.log)"
  else
    FAIL=1; printf '%-16s %-6s %-40s %s\n' NO-FLIP "$id" "$file" "flips $seen/$total exit=$rc missing:[${missing# }] (log: $OUT/$id.log)"
  fi
done < <(select_rows)

echo; echo "mutate: rows=$N manual=$MAN out=$OUT tree=$(git status --porcelain | wc -l | tr -d ' ')-dirty-paths"
exit "$FAIL"
