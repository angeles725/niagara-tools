#!/usr/bin/env bash
# station-snapshot.sh — Local audit-surface snapshot for a Niagara station directory.
# Campaign 8 PR9. Copies config.bog + console*.txt into <out-dir> and writes
# manifest.json with sha256, bytes, and a UTC timestamp, without touching the
# source directory. History/alarm db files are recorded as pointers (path + size)
# only — never copied. Feeds bog-audit and triage-console in the post-deploy flow.
#
# Usage:  station-snapshot.sh <station-dir> <out-dir>
#
#   Copies ONLY:  config.bog  console*.txt  (dot-dirs pruned; D9b)
#   Pointers:     history/**  (relative path + bytes; never copied)
#   Writes:       <out-dir>/manifest.json
#
#   manifest.json shape:
#     { "station": "<name>", "ts": "<UTC-stamp>",
#       "files": [ {"relpath":"…","sha256":"…","bytes":<n>} … ],
#       "pointers": [ {"relpath":"…","bytes":<n>} … ] }
#
#   Row format:  printf '%-4s  %-14s  %s  %s\n' STATUS station-snap relpath detail
#   Exits:       0 ok · 1 a listed file could not be copied · 3 usage/env (K20)
#
# Source is opened read-only (cp only; no write, no chmod, no temp file in station-dir).
# Refuses <out-dir> inside <station-dir> (exit 3).
# This script is VCS-free by design; kit-links.bats L2 enforces.
# [ev: retro campaign8-station-snapshot]
set -u
LC_ALL=C
export LC_ALL

FAILED=0

# ---------------------------------------------------------------------------
# Usage guard (K20: usage/env exits are 3)
# ---------------------------------------------------------------------------
if [ $# -lt 2 ]; then
    printf 'usage: station-snapshot.sh <station-dir> <out-dir>\n' >&2
    exit 3
fi

STATION_DIR="$1"
OUT_DIR="$2"

if [ ! -d "$STATION_DIR" ]; then
    printf 'station-snapshot: not a directory: %s\n' "$STATION_DIR" >&2
    exit 3
fi

# Guard: sha256sum must be present (threat matrix: external tool, D10)
if ! command -v sha256sum > /dev/null 2>&1; then
    printf 'station-snapshot: sha256sum not found in PATH\n' >&2
    exit 3
fi

# Canonical absolute path of the station dir (never writes here)
STATION_ABS="$(cd "$STATION_DIR" && pwd)"

# Refuse <out-dir> inside <station-dir>: derive canonical abs path before mkdir
_out_parent="$(dirname "$OUT_DIR")"
_out_base="$(basename "$OUT_DIR")"
if [ -d "$OUT_DIR" ]; then
    _out_abs="$(cd "$OUT_DIR" && pwd)"
elif [ -d "$_out_parent" ]; then
    _out_abs="$(cd "$_out_parent" && pwd)/$_out_base"
else
    _out_abs="$OUT_DIR"  # parent not yet created; use literal (edge case)
fi
case "$_out_abs" in
    "$STATION_ABS/"*|"$STATION_ABS")
        printf 'station-snapshot: <out-dir> must not be inside <station-dir>\n' >&2
        exit 3
        ;;
esac

mkdir -p "$OUT_DIR" || { printf 'station-snapshot: cannot create out-dir: %s\n' "$OUT_DIR" >&2; exit 3; }

STATION_NAME="$(basename "$STATION_ABS")"
STAMP="$(date -u '+%Y%m%dT%H%M%SZ')"

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT
_FILE_ENTRIES="$_TMP/file_entries"
_PTR_ENTRIES="$_TMP/ptr_entries"
: > "$_FILE_ENTRIES"
: > "$_PTR_ENTRIES"
COPY_COUNT=0

# ---------------------------------------------------------------------------
# _snap_file <relpath>  — copy one file, record a manifest entry
# ---------------------------------------------------------------------------
_snap_file() {
    local rel="$1"
    local src="$STATION_ABS/$rel"
    if [ ! -r "$src" ]; then
        printf '%-4s  %-14s  %s  %s\n' "FAIL" "station-snap" "$rel" "unreadable"
        FAILED=1
        return
    fi
    if ! cp -p "$src" "$OUT_DIR/$rel"; then
        printf '%-4s  %-14s  %s  %s\n' "FAIL" "station-snap" "$rel" "copy failed"
        FAILED=1
        return
    fi
    # Strip executable bit: cp -p preserves mtimes but copies source mode, which on NTFS/0777
    # WSL mounts makes every output executable. Contract (D10): outputs are never +x.
    chmod 0644 "$OUT_DIR/$rel"
    local sha bytes
    sha=$(sha256sum "$OUT_DIR/$rel" | cut -d' ' -f1)
    bytes=$(wc -c < "$OUT_DIR/$rel" | tr -d ' \t')
    printf '%-4s  %-14s  %s  sha256=%s bytes=%s\n' "PASS" "station-snap" "$rel" "$sha" "$bytes"
    printf '{"relpath": "%s", "sha256": "%s", "bytes": %s}\n' \
        "$rel" "$sha" "$bytes" >> "$_FILE_ENTRIES"
    COPY_COUNT=$((COPY_COUNT + 1))
}

# ---------------------------------------------------------------------------
# 1. config.bog
# ---------------------------------------------------------------------------
if [ -f "$STATION_ABS/config.bog" ]; then
    _snap_file "config.bog"
else
    printf '%-4s  %-14s  %s  %s\n' "FAIL" "station-snap" "config.bog" "not found"
    FAILED=1
fi

# ---------------------------------------------------------------------------
# 2. console*.txt  (shell glob; dot-dirs naturally excluded; D9b)
# ---------------------------------------------------------------------------
for _cfile in "$STATION_ABS"/console*.txt; do
    [ -f "$_cfile" ] || continue
    _snap_file "$(basename "$_cfile")"
done

# ---------------------------------------------------------------------------
# 3. Pointer entries: history/** (paths + sizes; never copied)
#    find with dot-dir prune (D9b); process substitution avoids subshell write loss
# ---------------------------------------------------------------------------
if [ -d "$STATION_ABS/history" ]; then
    while IFS= read -r _pfile; do
        _prel="${_pfile#"$STATION_ABS/"}"
        _pbytes=$(wc -c < "$_pfile" 2>/dev/null | tr -d ' \t')
        printf '{"relpath": "%s", "bytes": %s}\n' \
            "$_prel" "${_pbytes:-0}" >> "$_PTR_ENTRIES"
    done < <(find "$STATION_ABS/history" -type d -name '.*' -prune \
               -o -type f -print | LC_ALL=C sort)
fi

# ---------------------------------------------------------------------------
# 4. Assemble manifest.json from accumulated entry files
#    awk: join array elements with commas between them (last entry gets no trailing comma)
# ---------------------------------------------------------------------------
_join_entries() {
    local src="$1" indent="$2"
    if [ -s "$src" ]; then
        printf '\n'
        awk -v ind="$indent" \
            'NR > 1 { printf ",\n" } { printf "%s%s", ind, $0 }' "$src"
        printf '\n'
    fi
}

{
    printf '{\n'
    printf '  "station": "%s",\n' "$STATION_NAME"
    printf '  "ts": "%s",\n' "$STAMP"
    printf '  "files": ['
    _join_entries "$_FILE_ENTRIES" "    "
    printf '  ],\n'
    printf '  "pointers": ['
    _join_entries "$_PTR_ENTRIES" "    "
    printf '  ]\n'
    printf '}\n'
} > "$OUT_DIR/manifest.json"

printf 'station-snapshot: %d file(s) copied; manifest.json written to %s\n' \
    "$COPY_COUNT" "$OUT_DIR"

if [ "$FAILED" -eq 1 ]; then
    exit 1
fi
exit 0
