#!/usr/bin/env bash
# rc-scan.sh — Browser-resource lint for Niagara rc/ assets (Campaign 8 PR6; WU3 2026-10-02).
#
# Scans **/rc/** *.html *.js *.css under <artifact-dir> for defects the Java-side lints never see.
# Every row names a stable check id:
#   ord-literal        FAIL  hardcoded station:|local:|slot:/ ORD string, or handle ORD h:<hex>
#                            anchored at segment start
#   host-literal       FAIL  http:// non-namespace host OR bare IPv4 literal
#   bare-catch         WARN  .catch(() => {}) swallowing write errors
#   null-branch        WARN  ? null : display branch on a process field
#   browser-floor      WARN  a feature above the Chromium 83 HMI panel floor. CSS (.css files,
#                            <style> blocks, style="" attributes): `inset:`, `gap:`/`row-gap:`/
#                            `column-gap:` in a rule that does not itself declare display:grid
#                            (flex gap = C84), `aspect-ratio:`, `min(`/`max(`/`clamp(`,
#                            `backdrop-filter`. JS (.js files, inline <script>): `??=` `||=` `&&=`
#                            `.replaceAll(` `.at(` `structuredClone(`, and a column-0 `await` in a
#                            .js file (top-level await). FAIL under --strict or --profile hmi|both.
#                            [ev: retro dashboard-frontend-reliability-rules Δ5]
#                            [ev: retro panccadia-commissioning-lessons Δ6]
#                            [ev: retro panccadia-persistent-config-hoa Δ4]
#   disabled-gate      WARN  `.disabled = <not false>`, `setAttribute("disabled"` or a `disabled`
#                            attribute on button/input/select/textarea, in a file that mentions a
#                            login/auth/step-up gate (a disabled control dispatches no click, so the
#                            gate never opens; mark it with a class / aria-disabled instead). `auth`
#                            counts only at a word or camelCase start (auth, Auth, authToken,
#                            isAuthenticated, AUTH_URL, authorize) — never inside author/authority;
#                            a lowercase-glued form (unauthorized) is not seen.
#                            [ev: retro dashboard-frontend-reliability-rules Δ7]
#                            [ev: retro panccadia-persistent-config-hoa Δ4]
#   fetch-no-signal    WARN  `fetch(` whose argument list carries no `signal` (nor do the 3 lines
#                            before it — the fetchT helper idiom); route calls through fetchT.
#                            [ev: retro dashboard-frontend-reliability-rules Δ1]
#   setinterval-async  WARN  `setInterval(async ...` or `setInterval(<name>` where <name> is an
#                            `async function` / `= async` const in the same file.
#                            [ev: retro dashboard-frontend-reliability-rules Δ2]
#   innerhtml-server   WARN  innerHTML/outerHTML assignment or insertAdjacentHTML( concatenating a
#                            member (`+ a.message`, `${a.message}`) on a line with no esc*( call.
#                            [ev: retro dashboard-frontend-standard Δ2]
#   datauri-budget     FAIL  a data: URI over 20 KB (20480 chars); WARN with --legacy (a deployed
#                            module not yet restructured). [ev: retro dashboard-frontend-standard Δ3]
#   orphan-page        WARN  a <section id="page-<id>"> with no data-page="<id>" nav entry, or a
#                            data-page="<id>" with no such section (across all scanned files; only
#                            when the tree uses section#page-* at all). [ev: retro dashboard-frontend-standard Δ5]
#   inline-block-size  WARN  an inline <script> (no src) or <style> block over 300 lines in an
#                            .html file — split per DJS1. [ev: retro dashboard-rc-file-split Δ6]
#   --strict promotes every WARN row to FAIL.
#
# Escape hatch (new checks only): a trailing `rc-scan: allow <check-id>` comment on the flagged line
# (for orphan-page, on the section/data-page line) suppresses that check there. Record why.
# Exclusions: rc/ext/**, *.min.js, srcTest/**, dot-directories (D9b), comment-only lines (// /* * <!--).
# W3C namespace URIs (http://www.w3.org/) are NOT flagged as hosts (not network hosts).
# Handle ORD pattern: h:<hex> matched only when preceded by |, ", ', or ` (segment boundary), so
#   CSS width:100px (h preceded by 't') is never matched.
# Known limitations (advisory heuristics): one-line regexes (a multi-line innerHTML template, a
#   flex/grid display declared by another rule, a feature-guarded `.at(` and a CSS string built in JS
#   are not understood — use the allow marker); naive paren counting for fetch( ignores strings.
#
# Usage:  rc-scan.sh <artifact-dir> [--strict] [--profile hmi|lan|both|unknown] [--legacy]
#   --profile  the module's ui_profile (BUILD-STATE envelope); the caller passes it, rc-scan never
#              reads BUILD-STATE. hmi|both make browser-floor FAIL.
#   Row format:  FAIL|WARN  rc-scan  <file>:<line>  <check-id>: <reason>
#   Exits:       0  no FAIL (WARN-only is still 0) · 1  any FAIL · 3  usage/env (K20)
#
# Mutation: RC11 -- drop the browser-floor CSS token rule so the floor fixture rows vanish
# Mutation: RC13 -- ignore --profile so hmi stays WARN and exit 1 -> 0
# Mutation: RC16 -- drop the .disabled assignment rule so the gated config.js row vanishes
# Mutation: RC18 -- treat a fetch( as signalled unconditionally so RC18 loses its rows
# Mutation: RC20 -- drop the async-name set so setInterval(poll) over an async fn is not flagged
# Mutation: RC22 -- drop the innerhtml-server rule so RC22 loses its rows
# Mutation: RC24 -- raise the data-URI budget so the 21000-char URI passes
# Mutation: RC26 -- skip the orphan-page pass so RC26 loses its rows
# Mutation: RC28 -- drop the inline-block counter so RC28 loses its row
#
# This script is VCS-free by design; version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
# [ev: retro campaign8-rc-scan]
set -u

FAILED=0
STRICT=0
PROFILE=unknown
LEGACY=0

_usage() {
    printf 'usage: rc-scan.sh <artifact-dir> [--strict] [--profile hmi|lan|both|unknown] [--legacy]\n' >&2
    exit 3
}

[ $# -ge 1 ] || _usage
ARTIFACT_DIR="$1"
shift
while [ $# -gt 0 ]; do
    case "$1" in
        --strict) STRICT=1 ;;
        --legacy) LEGACY=1 ;;
        --profile)
            [ $# -ge 2 ] || _usage
            case "$2" in
                hmi|lan|both|unknown) PROFILE="$2" ;;
                *) printf 'rc-scan: unknown profile: %s\n' "$2" >&2; exit 3 ;;
            esac
            shift ;;
        *) printf 'rc-scan: unknown option: %s\n' "$1" >&2; exit 3 ;;
    esac
    shift
done

if [ ! -d "$ARTIFACT_DIR" ]; then
    printf 'rc-scan: not a directory: %s\n' "$ARTIFACT_DIR" >&2
    exit 3
fi

FLOORFAIL=0
{ [ "$STRICT" -eq 1 ] || [ "$PROFILE" = hmi ] || [ "$PROFILE" = both ]; } && FLOORFAIL=1

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT
_ROWS="$_TMP/rows.txt"

# ---------------------------------------------------------------------------
# Per-file awk program (written to a temp file to avoid shell-quoting issues).
# The file is read TWICE: pass 1 (FNR==NR) collects file-wide facts (async
# function names, login-gate mention); pass 2 emits rows.
# ---------------------------------------------------------------------------
cat > "$_TMP/scan.awk" << 'AWKEOF'
BEGIN {
    # sevw (WARN, FAIL under --strict) comes from the shell: one source for this pass and orphan.awk.
    sevfloor = (floorfail + 0 == 1) ? "FAIL" : "WARN"
    sevdata = (legacy + 0 == 1 && strict + 0 != 1) ? "WARN" : "FAIL"
    GAPRE = "(^|[;{[:space:]\"'])(row-|column-)?gap[[:space:]]*:"
    GRIDRE = "display[[:space:]]*:[[:space:]]*(inline-)?grid"
    gated = 0
}

# ---- pass 1: file-wide facts ------------------------------------------------
FNR == NR {
    if (mentions_gate($0)) gated = 1
    t = $0
    while (match(t, /async[[:space:]]+function[[:space:]]*\*?[[:space:]]*[A-Za-z_$][A-Za-z0-9_$]*/)) {
        m = substr(t, RSTART, RLENGTH); sub(/^async[[:space:]]+function[[:space:]]*\*?[[:space:]]*/, "", m)
        isasync[m] = 1; t = substr(t, RSTART + RLENGTH)
    }
    t = $0
    if (match(t, /(const|let|var)[[:space:]]+[A-Za-z_$][A-Za-z0-9_$]*[[:space:]]*=[[:space:]]*async([[:space:](]|$)/)) {
        m = substr(t, RSTART, RLENGTH); sub(/^(const|let|var)[[:space:]]+/, "", m); sub(/[[:space:]]*=.*$/, "", m)
        isasync[m] = 1
    }
    next
}

function allowed(id) { return index($0, "rc-scan: allow " id) > 0 }
# A login/auth/step-up gate mention. `auth` only at a word or camelCase start, and not author/authority
# (authorize/authorization still count).
function mentions_gate(s,    t, pre, nxt, cap) {
    if (tolower(s) ~ /login|step-?up/) return 1
    t = s
    while (match(t, /[Aa][Uu][Tt][Hh]/)) {
        pre = (RSTART > 1) ? substr(t, RSTART - 1, 1) : ""
        nxt = tolower(substr(t, RSTART + 4, 4))
        cap = (substr(t, RSTART, 1) == "A")
        if ((pre !~ /[A-Za-z0-9]/ || (cap && pre ~ /[a-z0-9]/)) && (nxt !~ /^or/ || nxt ~ /^ori[sz]/)) return 1
        t = substr(t, RSTART + 4)
    }
    return 0
}
function emit_at(lno, sev, id, msg) { print sev "  rc-scan  " rel ":" lno "  " id ": " msg }
function emit(sev, id, msg) { emit_at(FNR, sev, id, msg) }

# CSS rule-block feed: flex gap is only decidable per rule (display:grid declared in the rule).
function css_seg(seg) {
    blk = blk " " seg
    if (gapNR == 0 && seg ~ GAPRE && !allowed("browser-floor")) gapNR = FNR
}
function css_close() {
    if (gapNR > 0 && blk !~ GRIDRE)
        emit_at(gapNR, sevfloor, "browser-floor", "gap in a non-grid (flex) rule is above the Chromium 83 panel floor (flex gap = C84)")
    blk = ""; gapNR = 0
}
function css_feed(text,   ch) {
    while (match(text, /[{}]/)) {
        ch = substr(text, RSTART, 1)
        css_seg(substr(text, 1, RSTART - 1))
        if (ch == "{") { blk = ""; gapNR = 0 } else css_close()
        text = substr(text, RSTART + 1)
    }
    css_seg(text)
}
function css_tokens(t,   f) {
    f = ""
    if (t ~ /(^|[;{[:space:]"'])inset[[:space:]]*:/) f = f " inset"
    if (t ~ /aspect-ratio[[:space:]]*:/) f = f " aspect-ratio"
    if (t ~ /(^|[^A-Za-z0-9_-])min\(/) f = f " min()"
    if (t ~ /(^|[^A-Za-z0-9_-])max\(/) f = f " max()"
    if (t ~ /(^|[^A-Za-z0-9_-])clamp\(/) f = f " clamp()"
    if (t ~ /backdrop-filter/) f = f " backdrop-filter"
    return f
}
function js_tokens(t,   f) {
    f = ""
    if (t ~ /\?\?=/) f = f " ??="
    if (t ~ /\|\|=/) f = f " ||="
    if (t ~ /&&=/) f = f " &&="
    if (t ~ /\.replaceAll\(/) f = f " .replaceAll()"
    if (t ~ /\.at\(/) f = f " .at()"
    if (t ~ /structuredClone\(/) f = f " structuredClone()"
    if (ext == "js" && t ~ /^await[[:space:]]/) f = f " top-level-await"
    return f
}
function floor_row(f) {
    if (f != "" && !allowed("browser-floor"))
        emit(sevfloor, "browser-floor", substr(f, 2) " above the Chromium 83 panel floor")
}
function fetch_resolve() {
    if (!fsig) emit_at(fNR, sevw, "fetch-no-signal", "fetch( with no signal/timeout -- route it through the fetchT helper (AbortController + timeout)")
    fpend = 0
}
function fetch_feed(t,   o, c) {
    if (t ~ /signal/) fsig = 1
    o = gsub(/\(/, "(", t); c = gsub(/\)/, ")", t)
    fdepth += o - c; flines++
    if (fdepth <= 0 || flines >= 15) fetch_resolve()
}

# ---- pass 2: block tracking (before the comment skip, so every line counts) --
{
    line = $0
    was_script = inscript; was_style = instyle; opened = 0
    if (ext == "html") {
        if (!inscript && !instyle && match(line, /<script([[:space:]][^>]*)?>/)) {
            tag = substr(line, RSTART, RLENGTH); after = substr(line, RSTART + RLENGTH)
            if (tag !~ /[[:space:]]src[[:space:]]*=/ && after !~ /<\/script>/) { inscript = 1; bstart = FNR; opened = 1 }
            if (tag !~ /[[:space:]]src[[:space:]]*=/) was_script = 1
        } else if (!inscript && !instyle && match(line, /<style([[:space:]][^>]*)?>/)) {
            after = substr(line, RSTART + RLENGTH)
            if (after !~ /<\/style>/) { instyle = 1; bstart = FNR; opened = 1 }
            was_style = 1
        }
        if (!opened && (inscript || instyle) && line ~ /<\/(script|style)>/) {
            if (FNR - bstart - 1 > 300 && index(blkline, "rc-scan: allow inline-block-size") == 0)
                emit_at(bstart, sevw, "inline-block-size", "inline <" (inscript ? "script" : "style") "> block of " (FNR - bstart - 1) " lines (> 300) -- split per DJS1 into rc/js or rc/css files")
            inscript = 0; instyle = 0
        }
        if (opened) blkline = line
    }
    ctx_css = (ext == "css") || was_style
    ctx_js = (ext == "js") || was_script
}

# Skip comment-only lines: // single-line JS/CSS, /* block-comment open,
# * block-comment interior, and <!-- HTML comment opener.
/^[[:space:]]*(\/\/|\/\*|\*+|<!--)/ { next }

# ---- pass 2: new checks (no `next`, so the legacy checks below still run) ----
{
    line = $0

    # datauri-budget (any context)
    rest = line
    while (match(rest, /data:[^"')[:space:]]+/)) {
        if (RLENGTH > 20480) {
            if (!allowed("datauri-budget"))
                emit(sevdata, "datauri-budget", "data: URI of " RLENGTH " chars (> 20480) -- ship it as a separate rc/img file with ?v=")
            break
        }
        rest = substr(rest, RSTART + RLENGTH)
    }

    if (ctx_css) {
        floor_row(css_tokens(line))
        css_feed(line)
    } else if (ext == "html" && !ctx_js) {
        # inline style="" attributes on markup lines
        rest = line; f = ""
        while (match(rest, /style[[:space:]]*=[[:space:]]*("[^"]*"|'[^']*')/)) {
            sv = substr(rest, RSTART, RLENGTH)
            f = f css_tokens(sv)
            if (sv ~ GAPRE && sv !~ GRIDRE && f !~ / gap/) f = f " gap"
            rest = substr(rest, RSTART + RLENGTH)
        }
        floor_row(f)
    }

    if (ctx_js) {
        floor_row(js_tokens(line))

        # fetch-no-signal
        if (fpend) fetch_feed(line)
        else if (match(line, /(^|[^A-Za-z0-9_$])fetch[[:space:]]*\(/) && !allowed("fetch-no-signal")) {
            fpend = 1; fNR = FNR; fdepth = 0; flines = 0
            fsig = (p1 ~ /signal/ || p2 ~ /signal/ || p3 ~ /signal/) ? 1 : 0
            fetch_feed(substr(line, RSTART))
        }

        # setinterval-async
        if (!allowed("setinterval-async")) {
            if (line ~ /setInterval\([[:space:]]*async([[:space:](]|$)/)
                emit(sevw, "setinterval-async", "setInterval over an async function -- self-schedule with setTimeout after the poll settles (+ inFlight guard)")
            else if (match(line, /setInterval\([[:space:]]*[A-Za-z_$][A-Za-z0-9_$]*[[:space:]]*[,)]/)) {
                nm = substr(line, RSTART, RLENGTH); sub(/^setInterval\([[:space:]]*/, "", nm); sub(/[[:space:]]*[,)]$/, "", nm)
                if (nm in isasync)
                    emit(sevw, "setinterval-async", "setInterval(" nm ") where " nm " is async -- self-schedule with setTimeout after the poll settles (+ inFlight guard)")
            }
        }

        # innerhtml-server
        if ((line ~ /(inner|outer)HTML[[:space:]]*\+?=([^=]|$)/ || line ~ /insertAdjacentHTML[[:space:]]*\(/) \
            && (line ~ /\+[[:space:]]*[A-Za-z_$][A-Za-z0-9_$]*\.[A-Za-z_$]/ || line ~ /\$\{[[:space:]]*[A-Za-z_$][A-Za-z0-9_$]*\.[A-Za-z_$]/) \
            && line !~ /esc[A-Za-z]*\(/ && !allowed("innerhtml-server"))
            emit(sevw, "innerhtml-server", "member value concatenated into HTML without esc( -- use textContent or esc()")
    }

    # disabled-gate (login-gated files only; JS assignments + markup attributes)
    if (gated && !ctx_css && !allowed("disabled-gate")) {
        dg = 0
        if (match(line, /\.disabled[[:space:]]*=([^=]|$)/)) {
            rhs = substr(line, RSTART + RLENGTH - 1); sub(/^=?[[:space:]]*/, "", rhs)
            if (rhs !~ /^(false|0)([^A-Za-z0-9_$]|$)/) dg = 1
        }
        if (line ~ /setAttribute\([[:space:]]*["']disabled["']/) dg = 1
        if (line ~ /<(button|input|select|textarea)[^>]*[[:space:]]disabled([[:space:]=>\/]|$)/) dg = 1
        if (dg) emit(sevw, "disabled-gate", "control disabled in a login-gated file -- a disabled control fires no click, so the gate never opens; mark it (class / aria-disabled) and route the click to the gate")
    }

    p3 = p2; p2 = p1; p1 = line
}

# ---- legacy checks (unchanged semantics; `next` after the first hit) --------
# ord-literal: hardcoded Niagara ORD token.
# station:|local:|slot:/ — broad ORD scheme prefixes (always distinctive).
# h:<hex> — the Niagara HANDLE scheme; anchored at ORD segment boundary
#   (preceded by |, ", ', or `) so CSS width:100px (h preceded by 't') is
#   never matched. The handle payload is 1+ hex digits [0-9a-fA-F].
/(station:|local:|slot:\/)|(^|[|"'`])h:[0-9a-fA-F]+/ {
    print "FAIL  rc-scan  " rel ":" FNR "  ord-literal: hardcoded ORD literal"
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
            print "FAIL  rc-scan  " rel ":" FNR "  host-literal: " url
            found = 1
            break
        }
        rest = substr(rest, RSTART + RLENGTH)
    }
    if (found) next
}
# Bare IPv4 on lines that have no http:// (avoid double-reporting).
/[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+/ && !/http:\/\// {
    print "FAIL  rc-scan  " rel ":" FNR "  host-literal: bare IPv4 literal"
    next
}

# bare-catch: .catch(() => {}) swallows write errors silently.
/catch[[:space:]]*\([[:space:]]*\(\)[[:space:]]*=>[[:space:]]*\{[[:space:]]*\}[[:space:]]*\)/ {
    print sevw "  rc-scan  " rel ":" FNR "  bare-catch: bare .catch(()=>{}) swallows errors"
    next
}

# null-branch: ? null : display branch may blank the UI on a null process field.
/\?[[:space:]]*null[[:space:]]*:/ {
    print sevw "  rc-scan  " rel ":" FNR "  null-branch: ? null : branch may blank UI on null field"
    next
}

END {
    if (fpend) fetch_resolve()
    if (gapNR > 0) css_close()
}
AWKEOF

# ---------------------------------------------------------------------------
# orphan-page: cross-file pass over every scanned .html/.js file. Runs only when
# the tree uses the section#page-<id> convention at all.
# ---------------------------------------------------------------------------
cat > "$_TMP/orphan.awk" << 'AWKEOF'
/^[[:space:]]*(\/\/|\/\*|\*+|<!--)/ { next }
index($0, "rc-scan: allow orphan-page") > 0 { next }
{
    rel = FILENAME
    if (substr(rel, 1, length(root) + 1) == root "/") rel = substr(rel, length(root) + 2)
    t = $0
    if (FILENAME ~ /\.html$/) {
        while (match(t, /<section[^>]*[[:space:]]id=["']page-[A-Za-z0-9_-]+["']/)) {
            m = substr(t, RSTART, RLENGTH); t = substr(t, RSTART + RLENGTH)
            sub(/^.*id=["']page-/, "", m); sub(/["']$/, "", m)
            if (!(m in sec)) { sec[m] = rel ":" FNR; order[++ns] = "s" m }
        }
    }
    t = $0
    while (match(t, /data-page=["'][A-Za-z0-9_-]+["']/)) {
        m = substr(t, RSTART + 11, RLENGTH - 12); t = substr(t, RSTART + RLENGTH)
        if (!(m in nav)) { nav[m] = rel ":" FNR; order[++ns] = "n" m }
    }
}
END {
    hassec = 0
    for (k in sec) { hassec = 1; break }
    if (!hassec) exit 0
    for (i = 1; i <= ns; i++) {
        kind = substr(order[i], 1, 1); id = substr(order[i], 2)
        if (kind == "s" && !(id in nav))
            print sev "  rc-scan  " sec[id] "  orphan-page: section#page-" id " has no nav entry data-page=\"" id "\""
        if (kind == "n" && !(id in sec))
            print sev "  rc-scan  " nav[id] "  orphan-page: nav entry data-page=\"" id "\" has no section#page-" id
    }
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

# WARN-row severity, one source for scan.awk (sevw) and orphan.awk (sev): FAIL under --strict.
SEVW=WARN
[ "$STRICT" -eq 1 ] && SEVW=FAIL
_scan_files "$ARTIFACT_DIR" | LC_ALL=C sort > "$_TMP/files.txt"

# ---------------------------------------------------------------------------
# Per-file scan: run the awk program (two passes over the same file).
# ---------------------------------------------------------------------------
while IFS= read -r _file; do
    _rel="${_file#"$ARTIFACT_DIR/"}"
    _ext="${_file##*.}"
    LC_ALL=C awk \
        -v rel="$_rel" \
        -v ext="$_ext" \
        -v strict="$STRICT" \
        -v sevw="$SEVW" \
        -v floorfail="$FLOORFAIL" \
        -v legacy="$LEGACY" \
        -f "$_TMP/scan.awk" \
        "$_file" "$_file"
done < "$_TMP/files.txt" >> "$_ROWS"

_orphan_files=()
while IFS= read -r _file; do
    case "$_file" in *.html|*.js) _orphan_files+=("$_file") ;; esac
done < "$_TMP/files.txt"
if [ "${#_orphan_files[@]}" -gt 0 ]; then
    LC_ALL=C awk -v root="$ARTIFACT_DIR" -v sev="$SEVW" \
        -f "$_TMP/orphan.awk" "${_orphan_files[@]}" >> "$_ROWS"
fi

LC_ALL=C grep -q '^FAIL' "$_ROWS" && FAILED=1

cat "$_ROWS"
exit $FAILED
