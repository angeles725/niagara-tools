#!/usr/bin/env bash
# new-retro.sh — Atomic retro stub writer for the build-n4-module kit.
#
# Usage: new-retro.sh <module|kit> <slug> [--date YYYY-MM-DD] [--deltas N]
#
# Resolves the kit root from $KIT (env) → first positional arg if it is a directory
# with retros/INDEX.md → BASH_SOURCE parent → cwd.  This seam lets the test harness
# inject the kit root via KIT=<tmpdir> without passing a positional kit-root arg.
#
# Writes (all three in one atomic sequence — none if any staging step fails):
#   retros/<date>-<slug>.md    stub from template; refuses to overwrite (exit 3)
#   retros/INDEX.md            appends the row; idempotent guard skips if slug already present
#   BUILD-STATE.md             sets retro_pending: true (sed in place, only that key)
#
# Output rows:
#   retro    PASS  <file>  written
#   index    PASS|SKIP  <file>  appended|exists
#   envelope  PASS  BUILD-STATE.md  retro_pending: true
#
# Exit codes: 0 ok · 3 usage/env/file-already-exists (K20)
# VCS-free by design; kit-links.bats L2 enforces.
# [ev: retro campaign8-retro-loop]
set -u
LC_ALL=C
export LC_ALL

# ---------------------------------------------------------------------------
# Kit root resolution (testable seam: $KIT env → positional kit-root arg → BASH_SOURCE → cwd)
# ---------------------------------------------------------------------------
if [ -n "${KIT:-}" ] && [ -d "$KIT" ]; then
    KIT_ROOT="$KIT"
elif [ $# -ge 1 ] && [ -d "${1:-}" ] && [ -f "${1:-}/retros/INDEX.md" ]; then
    KIT_ROOT="$(cd "$1" && pwd)"
    shift
elif [ -d "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)" ]; then
    KIT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
else
    KIT_ROOT="$(pwd)"
fi

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
if [ $# -lt 2 ]; then
    printf 'usage: new-retro.sh <module|kit> <slug> [--date YYYY-MM-DD] [--deltas N]\n' >&2
    exit 3
fi

MODULE="$1"; shift
SLUG="$1";   shift

# Validate slug: [A-Za-z0-9][A-Za-z0-9._-]+  length >= 6
case "$SLUG" in
    [A-Za-z0-9][A-Za-z0-9._-][A-Za-z0-9._-][A-Za-z0-9._-][A-Za-z0-9._-]*)
        ;;  # at least 6 chars, valid charset
    *)
        printf 'new-retro: slug must be [A-Za-z0-9][A-Za-z0-9._-]+ and >= 6 chars: %s\n' "$SLUG" >&2
        exit 3
        ;;
esac

DATE="$(date +%Y-%m-%d)"
DELTAS=0

while [ $# -gt 0 ]; do
    case "$1" in
        --date)
            [ $# -ge 2 ] || { printf 'new-retro: --date requires a value\n' >&2; exit 3; }
            DATE="$2"; shift 2
            ;;
        --deltas)
            [ $# -ge 2 ] || { printf 'new-retro: --deltas requires a value\n' >&2; exit 3; }
            DELTAS="$2"; shift 2
            ;;
        *)
            printf 'new-retro: unknown option: %s\n' "$1" >&2; exit 3
            ;;
    esac
done

# ---------------------------------------------------------------------------
# Target paths
# ---------------------------------------------------------------------------
RETROS_DIR="$KIT_ROOT/retros"
RETRO_FILE="$RETROS_DIR/$DATE-$SLUG.md"
INDEX_FILE="$RETROS_DIR/INDEX.md"
BUILD_STATE="$KIT_ROOT/BUILD-STATE.md"

# Guard: required directories and files must exist
[ -d "$RETROS_DIR" ] || { printf 'new-retro: retros dir not found: %s\n' "$RETROS_DIR" >&2; exit 3; }
[ -f "$INDEX_FILE" ] || { printf 'new-retro: INDEX.md not found: %s\n' "$INDEX_FILE" >&2; exit 3; }
[ -f "$BUILD_STATE" ] || { printf 'new-retro: BUILD-STATE.md not found: %s\n' "$BUILD_STATE" >&2; exit 3; }

# Guard: refuse to overwrite existing retro file (primary idempotent guard, exits 3)
if [ -f "$RETRO_FILE" ]; then
    printf 'new-retro: retro already exists: %s\n' "$RETRO_FILE" >&2
    exit 3
fi

# ---------------------------------------------------------------------------
# Check INDEX for existing slug row (secondary idempotent guard — defense in depth)
# ---------------------------------------------------------------------------
INDEX_SKIP=0
if grep -qF "$DATE-$SLUG" "$INDEX_FILE" 2>/dev/null; then
    INDEX_SKIP=1
fi

# ---------------------------------------------------------------------------
# Stage all three edits in temp files (D13 atomicity: none land until all are ready)
# ---------------------------------------------------------------------------
_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT

# --- Stage 1: retro stub ---
RETRO_TMP="$_TMP/retro.md"

# Build the delta placeholder rows
DELTA_ROWS=""
i=1
while [ "$i" -le "$DELTAS" ]; do
    DELTA_ROWS="${DELTA_ROWS}| Δ${i} | <the change, one line> | \`<kit file>\` § \`<exact heading>\` | \`[ev: corpus B<n>]\` |
"
    i=$(( i + 1 ))
done

# INDEX row reference for the footer
INDEX_ROW="| $DATE-$SLUG.md | $MODULE | $DATE | pending | $DELTAS |"

cat > "$RETRO_TMP" << STUB_EOF
<!-- review-status: pending -->
# $DATE · $MODULE · $SLUG

**Session**: <campaign/PR context — one line>
**Delta count**: $DELTAS

## What happened
<one paragraph: the TRIGGER — what run/change/defect produced this retro>

## Evidence
- <console row / test name / commit / file:line>, each with a token: \`[ev: corpus B<n>]\` / \`[ev: <console file>]\` / \`[ev: <commit sha>]\`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
${DELTA_ROWS}
## Lessons
- <≤5 bullets; each a durable rule, not a narration>

---
**Status**: PENDING — INDEX row appended: \`$INDEX_ROW\`
STUB_EOF

# --- Stage 2: INDEX row ---
INDEX_TMP="$_TMP/index.md"
cp "$INDEX_FILE" "$INDEX_TMP" || { printf 'new-retro: failed to copy INDEX.md to temp\n' >&2; exit 1; }
if [ "$INDEX_SKIP" -eq 0 ]; then
    printf '| %s | %s | %s | pending | %s |\n' \
        "$DATE-$SLUG.md" "$MODULE" "$DATE" "$DELTAS" >> "$INDEX_TMP" \
        || { printf 'new-retro: failed to stage INDEX row\n' >&2; exit 1; }
fi

# --- Stage 3: BUILD-STATE retro_pending flip ---
BUILD_TMP="$_TMP/build-state.md"
sed 's/retro_pending:[[:space:]]*false/retro_pending: true/' "$BUILD_STATE" > "$BUILD_TMP" \
    || { printf 'new-retro: failed to stage BUILD-STATE.md\n' >&2; exit 1; }

# ---------------------------------------------------------------------------
# Atomic commit: copy all three staged files into place
# (staging failures above abort before this point; nothing was changed)
# ---------------------------------------------------------------------------
cp "$RETRO_TMP" "$RETRO_FILE" \
    || { printf 'new-retro: failed to write retro file\n' >&2; exit 1; }
cp "$INDEX_TMP" "$INDEX_FILE" \
    || { printf 'new-retro: failed to update INDEX.md\n' >&2; exit 1; }
cp "$BUILD_TMP" "$BUILD_STATE" \
    || { printf 'new-retro: failed to update BUILD-STATE.md\n' >&2; exit 1; }

# ---------------------------------------------------------------------------
# Output contract
# ---------------------------------------------------------------------------
printf 'retro    PASS  %s  written\n' "$RETRO_FILE"
if [ "$INDEX_SKIP" -eq 0 ]; then
    printf 'index    PASS  %s  appended\n' "$INDEX_FILE"
else
    printf 'index    SKIP  %s  exists\n' "$INDEX_FILE"
fi
printf 'envelope  PASS  BUILD-STATE.md  retro_pending: true\n'
