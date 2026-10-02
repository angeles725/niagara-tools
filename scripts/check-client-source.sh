#!/usr/bin/env bash
# check-client-source.sh — is this module root the declared SOURCE OF TRUTH checkout?
#
# Sessions built and read from the wrong client checkout for weeks because "where is the current
# source" lived only in session memory. BUILD-STATE.md now declares, per module,
#   source_of_truth: <path>@<branch>@<commit>      (or `unknown`)
# and this helper compares a module root against it. It is the version-control half of the build
# preflight: toolbelt scripts never call version control (kit-links L2), so it lives in scripts/ and
# BUILD-LOOP.md §0.b runs it next to toolbelt/preflight.sh. Local refs only — never fetches (no network).
# [ev: retro client-source-of-truth Δ1] [ev: retro client-source-of-truth Δ2]
#
# Usage:
#   check-client-source.sh [--declared <path>@<branch>@<commit>|unknown] <module-root>
#   check-client-source.sh --build-state <BUILD-STATE.md> --module <MOD> <module-root>
#
# Checks (rows: PASS|WARN|FAIL  source-of-truth  <detail>):
#   WARN  module root not inside a git work tree (it cannot answer "which commit is on the station")
#   WARN  source_of_truth undeclared or `unknown` (honest, but unchecked)
#   FAIL  detached HEAD
#   FAIL  the work tree is not the declared <path> (its top level does not end in /<path>)
#   FAIL  the checked-out branch is not the declared <branch>
#   FAIL  the declared <commit> is not in this checkout, or HEAD is BEHIND it (stale checkout)
#   PASS  HEAD at the declared commit, or ahead of it (new local work — update the field at close)
#   WARN  HEAD behind its upstream tracking ref as of the LAST fetch (count named; nothing fetched)
#   A <branch> or <commit> of `unknown` skips that comparison.
#
# Exit: 0 no FAIL · 1 any FAIL · 2 usage (incl. a malformed declared value) · 3 environment
#       (module root missing, BUILD-STATE file missing, git absent)
set -u

DECLARED=""
HAVE_DECLARED=0
BUILD_STATE=""
MODULE=""
FAILED=0

usage_exit() {
  printf 'usage: check-client-source.sh [--declared <path>@<branch>@<commit>|unknown] <module-root>\n' >&2
  printf '       check-client-source.sh --build-state <BUILD-STATE.md> --module <MOD> <module-root>\n' >&2
  exit 2
}

row() {
  printf '%-4s  source-of-truth  %s\n' "$1" "$2"
  case "$1" in FAIL) FAILED=1 ;; esac
}

while [ $# -gt 0 ]; do
  case "$1" in
    --declared)    [ $# -ge 2 ] || usage_exit; DECLARED="$2"; HAVE_DECLARED=1; shift 2 ;;
    --build-state) [ $# -ge 2 ] || usage_exit; BUILD_STATE="$2"; shift 2 ;;
    --module)      [ $# -ge 2 ] || usage_exit; MODULE="$2"; shift 2 ;;
    -h|--help)     sed -n '2,28p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    --) shift; break ;;
    -*) usage_exit ;;
    *) break ;;
  esac
done
[ $# -eq 1 ] || usage_exit
ROOT="$1"
[ -d "$ROOT" ] || { printf 'check-client-source: module root not found: %s\n' "$ROOT" >&2; exit 3; }
command -v git >/dev/null 2>&1 || { printf 'check-client-source: git not found in PATH\n' >&2; exit 3; }

# ---------------------------------------------------------------------------
# Resolve the declared value: --declared wins; else the module's build-state.v1 envelope.
# ---------------------------------------------------------------------------
if [ "$HAVE_DECLARED" -eq 0 ] && [ -n "$BUILD_STATE" ]; then
  [ -n "$MODULE" ] || usage_exit
  [ -f "$BUILD_STATE" ] || { printf 'check-client-source: BUILD-STATE not found: %s\n' "$BUILD_STATE" >&2; exit 3; }
  DECLARED="$(awk -v mod="$MODULE" '
    /^<!-- build-state\.v1 -->/   { inenv = 1; cur = ""; next }
    /^<!-- \/build-state\.v1 -->/ { inenv = 0; next }
    inenv && /^module:/           { v = $0; sub(/^module:[[:space:]]*/, "", v); sub(/[[:space:]]*#.*$/, "", v); cur = v; next }
    inenv && cur == mod && /^source_of_truth:/ {
      v = $0; sub(/^source_of_truth:[[:space:]]*/, "", v); sub(/[[:space:]]*#.*$/, "", v); print v; exit
    }' "$BUILD_STATE")"
fi

DECL_PATH=""; DECL_BRANCH="unknown"; DECL_COMMIT="unknown"
case "$DECLARED" in
  "")      ;;
  unknown) ;;
  *@*@*)
    DECL_PATH="${DECLARED%%@*}"
    _rest="${DECLARED#*@}"
    DECL_BRANCH="${_rest%%@*}"
    DECL_COMMIT="${_rest#*@}"
    case "$DECL_COMMIT" in *@*) usage_exit ;; esac
    if [ -z "$DECL_PATH" ] || [ -z "$DECL_BRANCH" ] || [ -z "$DECL_COMMIT" ]; then usage_exit; fi
    ;;
  *) printf 'check-client-source: malformed source_of_truth (want <path>@<branch>@<commit> or unknown): %s\n' "$DECLARED" >&2
     exit 2 ;;
esac

# ---------------------------------------------------------------------------
# Work-tree checks
# ---------------------------------------------------------------------------
if [ "$(git -C "$ROOT" rev-parse --is-inside-work-tree 2>/dev/null)" != "true" ]; then
  row WARN "$ROOT is not inside a git work tree — it cannot answer which commit is deployed; put the module in a repository"
  exit 0
fi
TOP="$(git -C "$ROOT" rev-parse --show-toplevel)"
HEAD_SHA="$(git -C "$ROOT" rev-parse HEAD 2>/dev/null || true)"
BRANCH="$(git -C "$ROOT" symbolic-ref --quiet --short HEAD 2>/dev/null || true)"

if [ -z "$BRANCH" ]; then
  row FAIL "detached HEAD at ${HEAD_SHA:-?} in $TOP — check out the source-of-truth branch before building"
fi

if [ -z "$DECL_PATH" ]; then
  if [ -z "$DECLARED" ]; then
    row WARN "source_of_truth undeclared for this module — record <path>@<branch>@<commit> in BUILD-STATE.md"
  else
    row WARN "source_of_truth is unknown — honest, but this checkout ($TOP) is unchecked"
  fi
else
  case "$TOP" in
    */"${DECL_PATH#/}"|"${DECL_PATH%/}") ;;
    *) row FAIL "work tree $TOP is not the declared source of truth $DECL_PATH — build from the declared checkout" ;;
  esac
  if [ -n "$BRANCH" ] && [ "$DECL_BRANCH" != "unknown" ] && [ "$BRANCH" != "$DECL_BRANCH" ]; then
    row FAIL "checked-out branch $BRANCH is not the declared source-of-truth branch $DECL_BRANCH"
  fi
  if [ "$DECL_COMMIT" != "unknown" ]; then
    DECL_SHA="$(git -C "$ROOT" rev-parse --verify --quiet "${DECL_COMMIT}^{commit}" 2>/dev/null || true)"
    if [ -z "$DECL_SHA" ]; then
      row FAIL "declared commit $DECL_COMMIT is not in this checkout — a stale clone (fetch it) or the wrong tree"
    elif [ "$DECL_SHA" = "$HEAD_SHA" ]; then
      row PASS "$TOP at the declared commit $DECL_COMMIT${BRANCH:+ on $BRANCH}"
    elif git -C "$ROOT" merge-base --is-ancestor "$DECL_SHA" HEAD 2>/dev/null; then
      row PASS "$TOP is ahead of the declared commit $DECL_COMMIT (new local work) — update source_of_truth at close"
    else
      row FAIL "HEAD ${HEAD_SHA:0:12} is behind or diverged from the declared commit $DECL_COMMIT — stale checkout"
    fi
  elif [ "$FAILED" -eq 0 ]; then
    row PASS "$TOP${BRANCH:+ on $BRANCH} matches the declared path/branch (commit unknown)"
  fi
fi

# Behind the upstream tracking ref as of the last fetch (no network): advisory only.
if [ -n "$BRANCH" ]; then
  _behind="$(git -C "$ROOT" rev-list --count "HEAD..@{upstream}" 2>/dev/null || true)"
  if [ -n "$_behind" ] && [ "$_behind" -gt 0 ]; then
    row WARN "$BRANCH is behind its upstream tracking ref by $_behind commit(s) as of the last fetch — pull before building"
  fi
fi

[ "$FAILED" -eq 0 ] || exit 1
exit 0
