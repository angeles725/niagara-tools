#!/usr/bin/env bash
# kit-ticket.sh — Opens a kit defect issue (or writes the offline fallback when gh is absent).
#
# Usage: kit-ticket.sh "<one-line title>" [--from <retro-file>] [--repo <owner/repo>]
#
# Resolves the kit root from $KIT (env) → BASH_SOURCE parent → cwd.
#
# Behaviour:
#   With gh available: gh issue create --repo <kit-remote> --title "[kit] <title>"
#                      --label kit,from-run,campaign-9 --body-file <tmp>
#   Without gh:        print SKIP row; write retros/tickets/<date>-<slug>.md; exit 0
#                      (never fails a run — gh absence is a known offline situation)
#
# Output row (SKIP path):
#   ticket  SKIP  <retro-file-or-title>  gh absent; wrote retros/tickets/<date>-<slug>.md
#
# Exit codes: 0 ok (including gh absent) · 3 usage/env (K20)
# VCS-free by design; kit-links.bats L2 enforces.
# [ev: retro campaign8-retro-loop]
set -u
LC_ALL=C
export LC_ALL

# ---------------------------------------------------------------------------
# Kit root resolution (testable seam: $KIT env → BASH_SOURCE parent → cwd)
# ---------------------------------------------------------------------------
if [ -n "${KIT:-}" ] && [ -d "$KIT" ]; then
    KIT_ROOT="$KIT"
elif [ -d "$(dirname "${BASH_SOURCE[0]}")/.." ]; then
    KIT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
else
    KIT_ROOT="$(pwd)"
fi

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
if [ $# -lt 1 ]; then
    printf 'usage: kit-ticket.sh "<title>" [--from <retro-file>] [--repo <owner/repo>]\n' >&2
    exit 3
fi

TITLE="$1"; shift
RETRO_FILE=""
REPO=""

while [ $# -gt 0 ]; do
    case "$1" in
        --from)
            [ $# -ge 2 ] || { printf 'kit-ticket: --from requires a value\n' >&2; exit 3; }
            RETRO_FILE="$2"; shift 2
            ;;
        --repo)
            [ $# -ge 2 ] || { printf 'kit-ticket: --repo requires a value\n' >&2; exit 3; }
            REPO="$2"; shift 2
            ;;
        *)
            printf 'kit-ticket: unknown option: %s\n' "$1" >&2; exit 3
            ;;
    esac
done

# ---------------------------------------------------------------------------
# gh absent? — decide early so restricted-PATH environments never hit missing tools
# ---------------------------------------------------------------------------
GH_ABSENT=0
if ! command -v gh > /dev/null 2>&1; then
    GH_ABSENT=1
fi

# ---------------------------------------------------------------------------
# Derive date and slug (use only: date, sed, awk — no tr, no mktemp)
# ---------------------------------------------------------------------------
DATE="$(date +%Y-%m-%d)"

# Slugify: awk for lowercase, sed for charset replace + compress + trim
RAW_SLUG="$(printf '%s' "$TITLE" \
    | awk '{print tolower($0)}' \
    | sed 's/[^a-z0-9._-]/-/g' \
    | sed 's/--*/-/g;s/^-//;s/-$//')"
# Ensure slug meets the 6-char minimum required by sweep-fold-audit.sh
case "$RAW_SLUG" in
    ??????*) SLUG="$RAW_SLUG" ;;   # 6+ chars — ok
    *)       SLUG="${RAW_SLUG}-ticket" ;;
esac

TICKETS_DIR="$KIT_ROOT/retros/tickets"
TICKET_FILE="$TICKETS_DIR/$DATE-$SLUG.md"

# ---------------------------------------------------------------------------
# SKIP path: gh absent — write offline fallback file directly (no mktemp needed)
# ---------------------------------------------------------------------------
if [ "$GH_ABSENT" -eq 1 ]; then
    mkdir -p "$TICKETS_DIR" || { printf 'kit-ticket: cannot create tickets dir: %s\n' "$TICKETS_DIR" >&2; exit 3; }
    RETRO_REL="${RETRO_FILE:-<retro file — pass --from <path> to link>}"

    # Build and write the ticket body directly to the fallback file
    {
        printf '[kit] %s\n\n' "$TITLE"
        printf '**Retro**: %s\n' "$RETRO_REL"
        printf '**What happened**: <see retro>\n\n'
        printf '**Proposed kit deltas**\n'
        printf '| Δ | Delta | Target file / § | Token |\n'
        printf '|---|---|---|---|\n\n'
        # SC2016: backticks are literal markdown, not command substitution
        # shellcheck disable=SC2016
        printf 'Labels: `kit`, `from-run`, `campaign-9`\n'
    } > "$TICKET_FILE" || { printf 'kit-ticket: cannot write ticket file: %s\n' "$TICKET_FILE" >&2; exit 3; }

    printf 'ticket  SKIP  %s  gh absent; wrote retros/tickets/%s-%s.md\n' \
        "$RETRO_REL" "$DATE" "$SLUG"
    exit 0
fi

# ---------------------------------------------------------------------------
# gh is present — build the ticket body in a temp file and open the issue
# ---------------------------------------------------------------------------
_TMP_BODY="$TICKETS_DIR/.kit-ticket-tmp-$$-body.md"
mkdir -p "$TICKETS_DIR"

RETRO_REL="${RETRO_FILE:-<no retro>}"

{
    printf '[kit] %s\n\n' "$TITLE"
    printf '**Retro**: %s\n' "$RETRO_REL"

    if [ -n "$RETRO_FILE" ] && [ -f "$RETRO_FILE" ]; then
        WHAT="$(awk '/^## What happened/{f=1;next} f&&/^## /{exit} f{print}' "$RETRO_FILE" \
            | grep -v '^[[:space:]]*$' | head -5)"
        printf '**What happened**: %s\n\n' "${WHAT:-<see retro>}"

        DELTAS="$(awk '/^## Proposed kit deltas/{f=1;next} f&&/^## /{exit} f{print}' "$RETRO_FILE" \
            | grep -v '^[[:space:]]*$' | head -20)"
        printf '**Proposed kit deltas**\n%s\n\n' "${DELTAS:-| Δ | Delta | Target file / § | Token |}"
    else
        printf '**What happened**: <see retro>\n\n'
        printf '**Proposed kit deltas**\n'
        printf '| Δ | Delta | Target file / § | Token |\n'
        printf '|---|---|---|---|\n\n'
    fi
    # SC2016: backticks are literal markdown, not command substitution
    # shellcheck disable=SC2016
    printf 'Labels: `kit`, `from-run`, `campaign-9`\n'
} > "$_TMP_BODY"

REPO_FLAG=""
if [ -n "$REPO" ]; then
    REPO_FLAG="--repo $REPO"
fi

# shellcheck disable=SC2086
gh issue create \
    $REPO_FLAG \
    --title "[kit] $TITLE" \
    --label "kit,from-run,campaign-9" \
    --body-file "$_TMP_BODY"
_RC=$?
rm -f "$_TMP_BODY"
if [ $_RC -ne 0 ]; then
    printf 'kit-ticket: gh issue create failed\n' >&2
    exit 3
fi
