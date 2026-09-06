#!/usr/bin/env bash
# lint-servlet.sh — BWebServlet security lint for Niagara N4 Java modules (Campaign 8 PR12, B813).
#
# Scans *.java files under <src-dir> that extend BWebServlet or declare doGet/doPost/service
# handlers. Dot-directories are pruned (D9b). Reports per-check rows.
#
# Checks:
#   auth          FAIL  a doGet/doPost/service handler body contains a write (setDouble/setFloat/
#                       setInt/setLong/setRelTime or .set() on a BComponent) with no
#                       getRemoteUser()/getUser() gate in the same method body.
#   input-400     FAIL  Double.parseDouble( or Integer.parseInt( in the file not enclosed by a
#                       try block (i.e. the enclosing scope has no 'try' within 5 lines above).
#   unbounded-set WARN  setDouble/setFloat/setInt/setLong/setRelTime in a handler body with no
#                       MIN/MAX/getFacets enforcement in the same body.
#   cache-nofinger WARN Cache-Control max-age>0 in the file without a fingerprinted resource
#                       (static detection: presence of max-age=[1-9] header call).
#   log-in-handler WARN LOG.info( inside a doGet/doPost/service handler body.
#   csrf-xrw-only WARN  X-Requested-With check in the file with no CsrfUtil/x-niagara-csrfToken/
#                       csrfToken reference — XHR-only guard is weaker than Niagara CSRF token.
#                       [ev: corpus B813]
#
# WARN -> FAIL under --strict. Deferred: R12.2 step-up re-auth (B803 — no static primitive).
#
# Usage:  lint-servlet.sh <src-dir> [--strict]
#
#   Row format:  <check>  FAIL|WARN  <file>:<line>  <detail>
#   Exits:       0  no FAIL (WARN-only is still 0) · 1  any FAIL · 3  usage/env (K20)
#
# This script is VCS-free by design; version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
# [ev: retro campaign8-lint-servlet]
set -u

FAILED=0
STRICT=0

if [ $# -lt 1 ]; then
    printf 'usage: lint-servlet.sh <src-dir> [--strict]\n' >&2
    exit 3
fi
SRC_DIR="$1"
shift
while [ $# -gt 0 ]; do
    case "$1" in
        --strict) STRICT=1 ;;
        *) printf 'lint-servlet: unknown option: %s\n' "$1" >&2; exit 3 ;;
    esac
    shift
done

if [ ! -d "$SRC_DIR" ]; then
    printf 'lint-servlet: not a directory: %s\n' "$SRC_DIR" >&2
    exit 3
fi

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT
_ROWS="$_TMP/rows.txt"

# ---------------------------------------------------------------------------
# AWK scanner: all checks for one Java file.
# Called once per *.java file that extends BWebServlet or declares a handler.
# Variables passed in:  rel (relative path), strict (0|1)
# ---------------------------------------------------------------------------
cat > "$_TMP/scan.awk" << 'AWKEOF'
BEGIN { n = 0 }
{ lines[++n] = $0 }
END {
    # -----------------------------------------------------------------------
    # Determine if this is a servlet file
    # -----------------------------------------------------------------------
    is_servlet = 0
    for (i = 1; i <= n; i++) {
        if (lines[i] ~ /extends[[:space:]]+BWebServlet/ ||
            lines[i] ~ /void[[:space:]]+(doGet|doPost|service)[[:space:]]*\(/) {
            is_servlet = 1; break
        }
    }
    if (!is_servlet) exit

    # -----------------------------------------------------------------------
    # File-level flags (computed once, on comment-stripped lines)
    # -----------------------------------------------------------------------
    file_has_xhr = 0; file_has_csrf = 0
    for (i = 1; i <= n; i++) {
        sl = lines[i]
        if (match(sl, /\/\//)) sl = substr(sl, 1, RSTART - 1)
        if (sl ~ /X-Requested-With/) file_has_xhr = 1
        if (sl ~ /CsrfUtil|x-niagara-csrfToken|csrfToken/) file_has_csrf = 1
    }

    # -----------------------------------------------------------------------
    # Method-body extraction: handler methods only (doGet / doPost / service)
    # Runs auth + unbounded-set + log-in-handler per handler body.
    # -----------------------------------------------------------------------
    for (start = 1; start <= n; start++) {
        ln = lines[start]
        if (ln !~ /void[[:space:]]+(doGet|doPost|service)[[:space:]]*\(/) continue

        # Extract the handler body by brace counting.
        # Strip // single-line comments to prevent comment text from matching checks.
        depth = 0; body = ""
        for (j = start; j <= n; j++) {
            l = lines[j]
            # Strip single-line comments: remove // and everything after
            stripped = l
            if (match(stripped, /\/\//)) stripped = substr(stripped, 1, RSTART - 1)
            for (c = 1; c <= length(l); c++) {
                ch = substr(l, c, 1)
                if (ch == "{") depth++
                if (ch == "}" && depth > 0) {
                    depth--
                    if (depth == 0) { j = n + 1; break }
                }
            }
            body = body " " stripped
        }

        # --- auth check ---
        # A write (typed BComponent setter or generic .set() call) without auth gate.
        has_write = (body ~ /\.(setDouble|setFloat|setInt|setLong|setRelTime|setString)\(/ \
                  || body ~ /[^a-zA-Z0-9_]set\(/)
        has_auth  = (body ~ /getRemoteUser\(\)/ || body ~ /\.getUser\(\)/)
        if (has_write && !has_auth) {
            v = "FAIL"
            printf "%s  auth  %s:%d  handler writes without getRemoteUser()/getUser() auth gate\n",
                   v, rel, start
        }

        # --- unbounded-set check ---
        has_typed_set = (body ~ /\.(setDouble|setFloat|setInt|setLong|setRelTime)\(/)
        has_clamp     = (body ~ /getMin\(\)|getMax\(\)|getFacets\(\)|MIN|MAX|clamp/)
        if (has_typed_set && !has_clamp) {
            v = (strict+0 == 1) ? "FAIL" : "WARN"
            printf "%s  unbounded-set  %s:%d  typed set* in handler without MIN/MAX facet enforcement\n",
                   v, rel, start
        }

        # --- log-in-handler check ---
        if (body ~ /LOG\.info\(/) {
            v = (strict+0 == 1) ? "FAIL" : "WARN"
            printf "%s  log-in-handler  %s:%d  LOG.info inside request handler (per-request log spam)\n",
                   v, rel, start
        }
    }

    # -----------------------------------------------------------------------
    # input-400: Double.parseDouble( or Integer.parseInt( not inside a try block.
    # Heuristic: check whether any of the 5 lines at or above the call has 'try'.
    # -----------------------------------------------------------------------
    for (i = 1; i <= n; i++) {
        if (lines[i] !~ /Double\.parseDouble\(|Integer\.parseInt\(/) continue
        in_try = 0
        lo = (i - 5 > 1) ? i - 5 : 1
        for (k = lo; k <= i; k++) {
            if (lines[k] ~ /(^|[[:space:]])try[[:space:]]*\{/) { in_try = 1; break }
        }
        if (!in_try) {
            printf "FAIL  input-400  %s:%d  parseDouble/parseInt without enclosing try/catch -> may not return 400 on bad input\n",
                   rel, i
        }
    }

    # -----------------------------------------------------------------------
    # cache-nofinger: Cache-Control max-age > 0 on an unfingerprinted rc asset.
    # Pattern: setHeader("Cache-Control", "...max-age=N...") where N >= 1.
    # -----------------------------------------------------------------------
    for (i = 1; i <= n; i++) {
        if (lines[i] !~ /Cache-Control/) continue
        if (lines[i] !~ /max-age=[1-9]/) continue
        v = (strict+0 == 1) ? "FAIL" : "WARN"
        printf "%s  cache-nofinger  %s:%d  Cache-Control max-age>0 on rc asset without fingerprint\n",
               v, rel, i
    }

    # -----------------------------------------------------------------------
    # csrf-xrw-only: X-Requested-With guard without CsrfUtil/csrfToken.
    # Reports once per file (line 1 as anchor).
    # -----------------------------------------------------------------------
    if (file_has_xhr && !file_has_csrf) {
        v = (strict+0 == 1) ? "FAIL" : "WARN"
        printf "%s  csrf-xrw-only  %s:1  X-Requested-With guard without CsrfUtil/csrfToken (use Niagara CSRF token) [ev: corpus B813]\n",
               v, rel
    }
}
AWKEOF

# ---------------------------------------------------------------------------
# File walker: *.java under <src-dir>, dot-directories pruned (D9b).
# ---------------------------------------------------------------------------
_scan_files() {
    find "$1" \( -type d -name '.*' -prune \) \
         -o \( -type f -name '*.java' -print \)
}

while IFS= read -r _file; do
    _rel="${_file#"$SRC_DIR/"}"
    LC_ALL=C awk \
        -v rel="$_rel" \
        -v strict="$STRICT" \
        -f "$_TMP/scan.awk" \
        "$_file"
done < <(_scan_files "$SRC_DIR" | LC_ALL=C sort) >> "$_ROWS"

LC_ALL=C grep -q '^FAIL' "$_ROWS" && FAILED=1

cat "$_ROWS"
exit $FAILED
