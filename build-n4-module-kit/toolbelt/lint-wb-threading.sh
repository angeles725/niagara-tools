#!/usr/bin/env bash
# lint-wb-threading.sh — Workbench Swing-thread and agent-breadth checks (Campaign 8 PR11).
#
# Two checks over a -wb src tree (B809 §809.7):
#   ui-thread-traversal  WARN  a doInvoke body that directly calls
#                               getNavChildren|getNavNodes|BqlQuery without an
#                               invokeLater|BJobService|JobThread in the SAME body.
#                               Heuristic only: flags the obvious direct pattern for
#                               human review; deep call chains are out of scope (B809).
#   agent-breadth        WARN  @AgentOn(types="baja:Component") with no comment
#                               containing 'justif', 'why', or 'broad' within 3 lines above.
#
# --strict turns any WARN into exit 1 (useful for CI gates).
#
# Usage: lint-wb-threading.sh [--strict] <wb-src-dir>
#
#   Row format: <check>  WARN  <file>:<line>  <detail>
#   Exits: 0 clean or WARN-only (no --strict) · 1 any WARN under --strict · 3 usage/env (K20)
#
# Dot-directories (e.g. .deploy-baseline/) are excluded per D9b.
# VCS-free by design; kit-links.bats L2 enforces the no-VCS rule.
# [ev: retro campaign8-wb-audit]
set -u
LC_ALL=C
export LC_ALL

STRICT=0
SRC=""

if [ $# -lt 1 ]; then
  printf 'usage: lint-wb-threading.sh [--strict] <wb-src-dir>\n' >&2
  exit 3
fi

# Parse args — --strict may appear before or after the directory
while [ $# -gt 0 ]; do
  case "$1" in
    --strict) STRICT=1 ;;
    -*)
      printf 'lint-wb-threading: unknown option: %s\n' "$1" >&2
      exit 3
      ;;
    *)
      if [ -n "$SRC" ]; then
        printf 'lint-wb-threading: unexpected argument: %s\n' "$1" >&2
        exit 3
      fi
      SRC="$1"
      ;;
  esac
  shift
done

[ -n "$SRC" ] || { printf 'usage: lint-wb-threading.sh [--strict] <wb-src-dir>\n' >&2; exit 3; }
[ -d "$SRC" ] || { printf 'lint-wb-threading: not a directory: %s\n' "$SRC" >&2; exit 3; }

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT
_ROWS="$_TMP/rows.txt"
touch "$_ROWS"

# ---------------------------------------------------------------------------
# awk program: ui-thread-traversal
# Extracts each doInvoke method body by brace-counting, then checks whether
# the body calls getNavChildren|getNavNodes|BqlQuery without an off-load guard.
# Brace counting is heuristic (strings/comments are not parsed) — acceptable
# for a WARN-level B809 check that exists for human review, not a hard gate.
# ---------------------------------------------------------------------------
_AWK_THREAD="$_TMP/thread.awk"
cat > "$_AWK_THREAD" <<'AWK'
{
  lines[NR] = $0
}
END {
  for (i = 1; i <= NR; i++) {
    # Match a doInvoke method signature line (handles both
    # "void doInvoke(..." and "CommandArtifact doInvoke() throws ..."
    if (lines[i] !~ /[[:space:]]doInvoke[[:space:]]*\(/) continue
    # Extract the full method body by brace counting
    depth = 0
    body = ""
    start_line = i
    brace_seen = 0
    j = i
    while (j <= NR) {
      body = body "\n" lines[j]
      open_copy = lines[j]; gsub(/[^{]/, "", open_copy)
      clos_copy = lines[j]; gsub(/[^}]/, "", clos_copy)
      depth += length(open_copy) - length(clos_copy)
      if (index(lines[j], "{") > 0) brace_seen = 1
      if (brace_seen && depth <= 0) break
      j++
    }
    # Check: nav traversal call without an off-load guard in the same body?
    has_nav   = (body ~ /getNavChildren|getNavNodes|BqlQuery/)
    has_guard = (body ~ /invokeLater|BJobService|JobThread/)
    if (has_nav && !has_guard) {
      printf "ui-thread-traversal  WARN  %s:%d  doInvoke body calls nav traversal on the UI thread without invokeLater/BJobService (B809: flag for human review)\n", FILE, start_line
    }
    # Advance past this method to avoid re-scanning its inner lambdas/classes
    i = j
  }
}
AWK

# ---------------------------------------------------------------------------
# awk program: agent-breadth
# Reports @AgentOn(types="baja:Component") without a nearby justification
# comment (any comment containing 'justif', 'why', or 'broad' within 3 lines).
# ---------------------------------------------------------------------------
_AWK_AGENT="$_TMP/agent.awk"
cat > "$_AWK_AGENT" <<'AWK'
{
  lines[NR] = $0
}
END {
  for (i = 1; i <= NR; i++) {
    # Match @AgentOn with baja:Component (standalone or nested inside @NiagaraType)
    if (lines[i] !~ /@AgentOn/ || lines[i] !~ /baja:Component/) continue
    found = 0
    for (k = i - 1; k >= i - 3 && k >= 1; k--) {
      if (lines[k] ~ /justif|why|broad/) {
        found = 1
        break
      }
    }
    if (!found) {
      printf "agent-breadth  WARN  %s:%d  @AgentOn(baja:Component) without justification comment within 3 lines above (B809)\n", FILE, i
    }
  }
}
AWK

# ---------------------------------------------------------------------------
# Run both checks on every *.java under SRC (dot-dirs pruned per D9b)
# ---------------------------------------------------------------------------
while IFS= read -r f; do
  awk -v FILE="$f" -f "$_AWK_THREAD" "$f" >> "$_ROWS"
  awk -v FILE="$f" -f "$_AWK_AGENT"  "$f" >> "$_ROWS"
done < <(find "$SRC" -type d -name '.*' -prune -o -name '*.java' -print | sort)

# ---------------------------------------------------------------------------
# Emit rows and decide exit code
# ---------------------------------------------------------------------------
WARNED=0
if [ -s "$_ROWS" ]; then
  cat "$_ROWS"
  WARNED=1
fi

if [ "$STRICT" -eq 1 ] && [ "$WARNED" -eq 1 ]; then
  exit 1
fi

exit 0
