#!/usr/bin/env bash
# rc-scan.sh — Browser-resource lint for Niagara rc/ assets (Campaign 8 PR6).
#
# Scans **/rc/** *.html *.js *.css under <artifact-dir> for defects the
# Java-side lints never see (from the real DashboardPan-ux rc/index.html):
#   ord-literal   FAIL  hardcoded station:|local:|slot:/|h:/ ORD string
#   host-literal  FAIL  http:// non-namespace host OR bare IPv4 literal
#   bare-catch    WARN  .catch(() => {}) swallowing write errors [--strict FAIL]
#   null-branch   WARN  ? null : display branch on process field [--strict FAIL]
#
# Exclusions: rc/ext/**, *.min.js, srcTest/**, dot-directories (D9b),
#             comment-only lines (// /* * <!--).
# W3C namespace URIs (http://www.w3.org/) are NOT flagged as hosts (root cause:
#   xmlns/w3.org namespace strings are not network hosts — exempted by type).
# NOTE: ORD pattern uses h:/ (not h:) to avoid false positives from CSS
#   properties ending in 'h' followed by ':' (e.g. width:). Niagara history
#   ORDs always carry a path component: h:/station/HistoryService/path.
#
# Usage:  rc-scan.sh <artifact-dir> [--strict]
#
#   Row format:  FAIL|WARN  rc-scan  <file>:<line>  <check-label>: <reason>
#   Exits:       0  no FAIL (WARN-only is still 0) · 1  any FAIL · 3  usage/env (K20)
#
# This script is VCS-free by design; version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
# [ev: retro campaign8-rc-scan]
set -u

FAILED=0
STRICT=0

if [ $# -lt 1 ]; then
    printf 'usage: rc-scan.sh <artifact-dir> [--strict]\n' >&2
    exit 3
fi
ARTIFACT_DIR="$1"
shift
while [ $# -gt 0 ]; do
    case "$1" in
        --strict) STRICT=1 ;;
        *) printf 'rc-scan: unknown option: %s\n' "$1" >&2; exit 3 ;;
    esac
    shift
done

if [ ! -d "$ARTIFACT_DIR" ]; then
    printf 'rc-scan: not a directory: %s\n' "$ARTIFACT_DIR" >&2
    exit 3
fi

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT
_ROWS="$_TMP/rows.txt"

# ---------------------------------------------------------------------------
# Inline awk program written to a temp file to avoid shell-quoting issues
# with single quotes inside a '...' awk script.
# ---------------------------------------------------------------------------
cat > "$_TMP/scan.awk" << 'AWKEOF'
# Skip comment-only lines: // single-line JS/CSS, /* block-comment open,
# * block-comment interior, and <!-- HTML comment opener.
/^[[:space:]]*(\/\/|\/\*|\*+|<!--)/ { next }

# ord-literal: hardcoded Niagara ORD token in a string literal.
# Uses h:/ (not h:) to avoid false positives from CSS shorthand properties
# ending in 'h' (e.g. width:, depth:) — history ORDs always begin h:/path.
/(station:|local:|slot:\/|h:\/)/ {
    print "FAIL  rc-scan  " rel ":" NR "  ord-literal: hardcoded ORD literal"
    next
}

# host-literal: non-namespace http:// URI or bare IPv4.
# Exempt: http://www.w3.org/ — W3C namespace URIs are not network hosts.
/http:\/\// {
    rest = $0
    found = 0
    while (match(rest, /http:\/\/[^ \t"',;)>]+/)) {
        url = substr(rest, RSTART, RLENGTH)
        if (url !~ /^http:\/\/www\.w3\.org\//) {
            print "FAIL  rc-scan  " rel ":" NR "  host-literal: " url
            found = 1
            break
        }
        rest = substr(rest, RSTART + RLENGTH)
    }
    if (found) next
}
# Bare IPv4 on lines that have no http:// (avoid double-reporting).
/[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+/ && !/http:\/\// {
    print "FAIL  rc-scan  " rel ":" NR "  host-literal: bare IPv4 literal"
    next
}

# bare-catch: .catch(() => {}) swallows write errors silently.
/catch[[:space:]]*\([[:space:]]*\(\)[[:space:]]*=>[[:space:]]*\{[[:space:]]*\}[[:space:]]*\)/ {
    v = (strict+0 == 1) ? "FAIL" : "WARN"
    print v "  rc-scan  " rel ":" NR "  bare-catch: bare .catch(()=>{}) swallows errors"
    next
}

# null-branch: ? null : display branch may blank the UI on a null process field.
/\?[[:space:]]*null[[:space:]]*:/ {
    v = (strict+0 == 1) ? "FAIL" : "WARN"
    print v "  rc-scan  " rel ":" NR "  null-branch: ? null : branch may blank UI on null field"
    next
}
AWKEOF

# ---------------------------------------------------------------------------
# File walker: **/rc/** *.html *.js *.css only.
# Excludes: dot-directories (D9b — e.g. .deploy-baseline), srcTest, rc/ext,
#           and *.min.js files.
# ---------------------------------------------------------------------------
_scan_files() {
    find "$1" \
        \( -type d \( -name '.*' -o -name 'srcTest' \) -prune \) \
        -o \( -type d -path '*/rc/ext' -prune \) \
        -o \( -type f -path '*/rc/*' \
              \( -name '*.html' -o -name '*.js' -o -name '*.css' \) \
              -not -name '*.min.js' \
              -print \)
}

# ---------------------------------------------------------------------------
# Per-file scan: run the awk program over every discovered file.
# ---------------------------------------------------------------------------
while IFS= read -r _file; do
    _rel="${_file#"$ARTIFACT_DIR/"}"
    LC_ALL=C awk \
        -v rel="$_rel" \
        -v strict="$STRICT" \
        -f "$_TMP/scan.awk" \
        "$_file"
done < <(_scan_files "$ARTIFACT_DIR" | LC_ALL=C sort) >> "$_ROWS"

LC_ALL=C grep -q '^FAIL' "$_ROWS" && FAILED=1

cat "$_ROWS"
exit $FAILED
