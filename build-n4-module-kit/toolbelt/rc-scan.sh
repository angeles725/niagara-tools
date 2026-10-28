#!/usr/bin/env bash
# rc-scan.sh — Browser-resource lint for Niagara rc/ assets (Campaign 8 PR6; WU3 2026-10-02).
#
# Scans **/rc/** *.html *.js *.css under <artifact-dir> for defects the Java-side lints never see.
# Every row names a stable check id:
#   ord-literal        FAIL  hardcoded station:|local:|slot:/ ORD string, or handle ORD h:<hex>
#                            anchored at segment start
#   host-literal       FAIL  http:// non-namespace host OR bare IPv4 literal. A host in the config of a mode the
#                            page does not select (`base: "http://127.0.0.1/..."` beside `mode: "json"`) still
#                            FAILs (issue #459, decided): a one-word mode edit makes it live, plain http to a fixed
#                            host. Fix the page: a relative base (`/obix/...`), resolved against the station.
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
#                            counts at a word or camelCase start (auth, Auth, authToken,
#                            isAuthenticated, AUTH_URL, authorize) and after the gate prefixes
#                            o / re / un (OAuth, oauth2, reauth, unauthorized) — never inside
#                            author/authority. A line whose every `.disabled = <not false>` element also gets a
#                            `.title` holding a string literal that names no gate on the same line states a
#                            non-gate reason (wiring, interlock, phase) and is no row (issue #459).
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
#                            Decided on the sink's own value (the `=`/`+=` right-hand side or the
#                            insertAdjacentHTML arguments, as jssrc tokens): a member elsewhere on the line, a
#                            `.length`, a `.toFixed(`/`.toPrecision(` number and a `Math.*` value are no row (#459).
#                            Nor is a member of a const array/object literal the same file declares, read through
#                            a lexically bound name (an iterator callback or for-of over it, .find(, [i], a const
#                            arrow returning .find(); a field the page writes into, or push()es to, is not literal.
#                            Nor is a field of a collection computed from such a literal by `.map(u => ({k: e}))`
#                            (or a scalar `.map(u => e)`), reached directly, through a const arrow or a function
#                            declaration with one return, when e uses only literals, operators, Math.*, cleared
#                            members, computed consts, the callback index and toFixed/toString-like methods
#                            (retro go-rc-scan Δ3). A let, a call, a loop-built array or a second return proves
#                            nothing: the row stays.
#                            [ev: retro dashboard-frontend-standard Δ2]
#   status-default-ok  WARN  a status read that falls back to "ok" when the status is missing: `p.st || "ok"`,
#                            `?? 'ok'`, `||= / ??= "ok"` after a status name (st, stat, sts, status, *Status).
#                            An unlinked or uncommissioned point then shows as healthy (fail-open); trust a value only
#                            when `st === "ok"` and render an unknown state otherwise. Advisory: WARN under every
#                            mode, --strict included (audit 2026-10-03 A7 D2).
#                            (origin: the 2026-09-24 phase-2 alarms retro, Δ5)
#   datauri-budget     FAIL  a data: URI over 20 KB (20480 chars); WARN with --legacy (a deployed
#                            module not yet restructured). [ev: retro dashboard-frontend-standard Δ3]
#   orphan-page        WARN  a <section id="page-<id>"> with no data-page="<id>" nav entry, or a
#                            data-page="<id>" with no such section (across all scanned files; only
#                            when the tree uses section#page-* at all). [ev: retro dashboard-frontend-standard Δ5]
#   inline-block-size  WARN  an inline <script> (no src) or <style> block over 300 lines in an
#                            .html file — split per DJS1. [ev: retro dashboard-rc-file-split Δ6]
#   fetch-outside-services  WARN legacy / FAIL strict (module map modes only, see below)  a `fetch(` in
#                            rc JavaScript (a .js file or an inline <script>) outside an rc/js/services/ file:
#                            only the services call the network (types/frontend-standard.md § Component layout).
#                            The exemption is anchored at rc/js/services/ (js/pages/services/ or rc/services/ is
#                            not it). [ev: retro modularity-gaps Δ2]
#   js-file-size       WARN legacy / FAIL strict (module map modes only)  a .js file over 400 lines, reported at
#                            its line 1 (the allow marker goes on line 1). [ev: retro modularity-gaps Δ3]
#   Module map modes: the mode is read with lint.ModuleMode from the nearest <module-root>/MODULE-MAP.md above
#   <artifact-dir> (types/structure.md § Module map). With no map neither check runs, so a map-less module's
#   rows are unchanged; `modularity: legacy` WARNs them (FAIL under --strict), `modularity: strict` FAILs them.
#   Neither check reads third-party code: rc/vendor/** (its HTML included), rc/ext/** and *.min.js.
#   --strict promotes every WARN row to FAIL.
#
# Escape hatch (new checks only): a trailing `rc-scan: allow <check-id>` comment on the flagged line
# (for orphan-page, on the section/data-page line) suppresses that check there. Record why.
# Exclusions: rc/ext/**, *.min.js, rc/vendor/**/*.js (vendored libraries: lint-vendor-floor.sh owns them, issue
#   #459; vendored CSS/HTML stay scanned), srcTest/**, dot-directories (D9b), *-worktrees sibling checkouts,
#   comment-only lines (// /* * <!--). The walk is lib/scan-files.sh's (lint.Walk in Go): a symlinked
#   <artifact-dir> is followed (find -H) and the root itself is never pruned, even when dot-named (RS-walk).
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
# Mutation: RC33 -- scanning rc/vendor/**/*.js again flags the vendored three.js rows (RC33 sees RenderPass.js)
# Mutation: RC34 -- ignoring a stated non-gate title reason WARNs the wiring-disabled control (RC34 sees app.js:2)
# Mutation: RC35 -- matching the member anywhere on the line again WARNs the spec.unit line (RC35 sees app.js:2)
# Mutation: RC36 -- dropping literal provenance WARNs the units.map(u => u.name) line (RC36 sees app.js:2)
# Mutation: RC37 -- reading the module map mode as legacy keeps the strict module at WARN and exit 0
# Mutation: RS-walk -- dropping the *-worktrees prune reports the x-worktrees copy's ORD literal (RS-walk)
# Mutation: RS-unreadable -- skipping a file the scan cannot read (the awk's fatal-and-continue) passes clean
#
# Go: the checks run in go/internal/rcscan (go/cmd/rc-scan), built on demand by go/go-tool.sh; this script is
# the thin wrapper (same arguments, rows and exits). An unreadable rc file is an env error (exit 3), never a
# skipped file. [ev: retro go-rc-scan]
# This script is VCS-free by design; version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
# [ev: retro campaign8-rc-scan]
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'rc-scan: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" rc-scan)" || exit 3
exec "$BIN" "$@"
