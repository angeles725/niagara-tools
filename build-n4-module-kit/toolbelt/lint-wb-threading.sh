#!/usr/bin/env bash
# lint-wb-threading.sh — Workbench Swing-thread and agent-breadth checks (Campaign 8 PR11).
#
# Two checks over a -wb src tree (B809 §809.7):
#   ui-thread-traversal  WARN  a doInvoke body (or any same-class private/protected
#                               method reachable within 3 call levels) calls
#                               getNavChildren|getNavNodes|BqlQuery without an
#                               invokeLater|BJobService|JobThread anywhere on the
#                               expanded chain.  Bodies are extracted by brace-counting;
#                               the expansion is cycle-safe (visited set).
#                               Heuristic only: WARN for human review (B809).
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
# Mutation: WBT1c -- changes the invokeLater exemption, making an invokeLater-wrapped traversal false-WARN
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
#
# Phase 1 — extract bodies of all private/protected methods in the file into
#   mb[name].  Brace-counted from the signature line (Allman and K&R safe).
#
# Phase 2 — for each doInvoke method:
#   a. Extract the method body by brace-counting.
#   b. Recursively expand same-class callee bodies (methods found in mb[])
#      up to depth 3.  Expansion is cycle-safe via a visited set.
#   c. If the expanded body contains getNavChildren|getNavNodes|BqlQuery
#      AND no invokeLater|BJobService|JobThread appears anywhere, WARN.
#   d. Compute the call chain for the detail column via a DFS (chain_from).
#
# NOTE: 'sub' is a built-in awk name; local variables use distinct names
# (mbody, callee) throughout to avoid awk syntax errors.
# ---------------------------------------------------------------------------
_AWK_THREAD="$_TMP/thread.awk"
cat > "$_AWK_THREAD" <<'AWK'
{ lines[NR] = $0 }

# brace_body: extract the method body starting at line 'start' by counting {/}.
# Sets global g_end to the last line included.
function brace_body(start,    j,depth,body,brace_seen,oc,cc) {
  depth=0; body=""; brace_seen=0; j=start
  while (j <= NR) {
    body = body "\n" lines[j]
    oc = lines[j]; gsub(/[^{]/,"",oc)
    cc = lines[j]; gsub(/[^}]/,"",cc)
    depth += length(oc) - length(cc)
    if (index(lines[j],"{") > 0) brace_seen=1
    if (brace_seen && depth <= 0) break
    j++
  }
  g_end = j
  return body
}

# get_ids: extract all lowercase-starting identifier tokens from src into out[].
function get_ids(src, out,    tmp, n, arr, i, w) {
  delete out
  tmp = src; gsub(/[^a-zA-Z_0-9]/," ",tmp)
  n = split(tmp,arr)
  for (i=1; i<=n; i++) {
    w = arr[i]
    if (w ~ /^[a-z][a-zA-Z0-9_]+$/) out[w] = 1
  }
}

# expand_body: return body augmented with callee bodies reachable within d levels.
# vis[] is the visited set (passed by reference; modified in-place).
# NOTE: 'sub' is reserved in awk; local var is named 'mbody'.
function expand_body(body, d, vis,    ids, callee, mbody) {
  if (d <= 0) return body
  get_ids(body,ids)
  for (callee in ids) {
    if ((callee in mb) && !(callee in vis)) {
      vis[callee] = 1
      mbody = expand_body(mb[callee],d-1,vis)
      body = body "\n" mbody
    }
  }
  return body
}

# chain_from: DFS to find the call path from body to a nav call.
# Returns "getNavChildren" when nav is directly in body,
#   "callee() -> ..." when reached via a callee, or "?" when not reachable.
function chain_from(body, d, vis,    ids, callee, mbody) {
  if (body ~ /getNavChildren|getNavNodes|BqlQuery/) return "getNavChildren"
  if (d <= 0) return "?"
  get_ids(body,ids)
  for (callee in ids) {
    if ((callee in mb) && !(callee in vis)) {
      vis[callee] = 1
      mbody = chain_from(mb[callee],d-1,vis)
      if (mbody != "?") return callee "() -> " mbody
    }
  }
  return "?"
}

END {
  # --- Phase 1: collect private/protected method bodies into mb[] ----------
  for (i=1; i<=NR; i++) {
    if (lines[i] !~ /^[[:space:]]+(private|protected)[[:space:]]/) continue
    # Extract method name: last word before the first '('
    n = split(lines[i],pts,"(")
    if (n < 2) continue
    n2 = split(pts[1],wds)
    if (n2 == 0) continue
    mname = wds[n2]
    # Method names start lowercase (skip constructors and type names)
    if (mname !~ /^[a-z][a-zA-Z0-9_]*$/) continue
    mb[mname] = brace_body(i)
  }

  # --- Phase 2: check each doInvoke body -----------------------------------
  for (i=1; i<=NR; i++) {
    if (lines[i] !~ /[[:space:]]doInvoke[[:space:]]*\(/) continue
    start_line = i
    body = brace_body(i)
    j = g_end

    # Expand same-class callees to depth 3 (cycle-safe)
    delete vis; vis["doInvoke"] = 1
    expanded = expand_body(body,3,vis)

    has_nav   = (expanded ~ /getNavChildren|getNavNodes|BqlQuery/)
    has_guard = (expanded ~ /invokeLater|BJobService|JobThread/)

    if (has_nav && !has_guard) {
      # Compute call chain for the detail column
      delete vis2
      ch = chain_from(body,3,vis2)
      if (ch == "?") ch = "..."
      printf "ui-thread-traversal  WARN  %s:%d  doInvoke -> %s without invokeLater/BJobService (B809: flag for human review)\n",
             FILE, start_line, ch
    }
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
