#!/usr/bin/env bash
# lint-ux-version.sh — FAILs a -ux resource reference whose ?v= differs from defaultModuleVersion; WARNs one with none.
#
# A kiosk keeps the file it cached: the servlet serves images with `Cache-Control: private, must-revalidate,
# max-age=2592000` (30 days; kit research block 29), so a -ux-only deploy shows stale JS, CSS and images unless every
# reference changes with the module version. The rule: types/dashboard.md ("Cache-bust every static asset with the
# module version": every <link href>, <script src> and image reference carries ?v=<defaultModuleVersion>),
# types/frontend-standard.md §5 (images under rc/img/ referenced with ?v=) and §11 (bump defaultModuleVersion on any
# rc/ change).
#
# Reads every *.html, *.htm, *.js and *.css under the module's *-ux/src/rc/ (rc/vendor, rc/ext, *.min.js, dot and
# *-worktrees directories are not entered; srcTest/ and any part that is not a -ux part are outside the path):
#   HTML   href/src/poster on <link> <script> <img> <source> <video> <audio> <image> <input>; url(...) in <style>
#          blocks and style attributes; the strings of inline <script> blocks as for JavaScript
#   CSS    url(...) and @import "..."
#   JS     a string that holds url(...) or is one path ending in js css png jpg jpeg gif svg webp avif ico bmp woff
#          woff2 ttf otf eot
# Comments are not read. Not references: an absolute URL (http: https: data: blob: mailto: javascript:), //host,
# #fragment, an empty value, a template (${...}, {{...}}) and a ?v= with no value (the version appended at run
# time). The version is the first uncommented defaultModuleVersion("X") of build.gradle.kts, read as the kit reads it
# everywhere (go/internal/vendorversion SourceVersion), at the root, directly under it, or above it up to the
# gradle root (settings.gradle.kts) or the repository root: either the gradle root or <repo>/<MOD> may be the
# argument. No version found is exit 3, never a pass; so is a root with no *-ux/src/rc file (K20).
#
# Checks (row = the message's first word):
#   stale-version  FAIL  the reference has ?v=<x> and x is not the module version (exact string compare)
#   no-version     WARN  a local reference with no ?v= (the doc says every reference carries it); FAIL under --strict
# Waiver: a trailing `lint-ux-version: allow <check-id> <reason>` comment on the flagged line (the line of the tag
# for a multi-line tag) prints a PASS row carrying the reason instead; a marker with no reason waives nothing.
#
# Known limitations (lexical): a reference built at run time by concatenation is not seen unless it is one
# literal; a JavaScript string is a reference only when it is one path with an asset extension; an attribute
# value holding `>` ends its tag early.
#
# Usage:  lint-ux-version.sh [--strict] <module-root>     (-h, --help: print this usage and exit 0)
#   Row:   FAIL|WARN|PASS  lint-ux-version  <file>:<line>  stale-version|no-version: <reference> ...
#          ERROR  lint-ux-version  <root>  no *-ux/src/rc *.html, *.js or *.css files found (wrong path or no -ux module)
# Exit:   0  no FAIL (WARN-only, PASS and waived rows included, is still 0) · 1  any FAIL, or any WARN under
#         --strict · 3  usage/env (no version found, an empty root, an unreadable file or directory)
#
# Mutation: UXV-stale -- not comparing the suffix with the module version passes a stale ?v= (exit 0)
# Mutation: UXV-missing -- dropping the no-version row passes a local reference with no ?v= at all
# Mutation: UXV-external -- reading an external URL as local WARNs every CDN reference
# Mutation: UXV-waiver -- ignoring the allow marker reports the waived reference again
# Mutation: UXV-noversion -- comparing against an empty version passes a module that declares none
# Mutation: UXV-vendor -- entering rc/vendor reports the vendored library's own references
# Mutation: UXV-strict -- not promoting the WARN under --strict leaves the exit at 0
# Mutation: UXV-empty -- treating a root with no -ux rc files as clean passes a wrong path
#
# Implementation: rule package go/internal/lint/rules/uxversion on the shared engine go/internal/lint (go/README.md).
# This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when it is missing or
# older than a Go source, then this script execs `n4lint lint-ux-version <args>`. Needs `go` on PATH to build
# (exit 3 otherwise). VCS-free by design. LC_ALL=C.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
self="$(readlink -f "${BASH_SOURCE[0]}")"
case "${1:-}" in
  -h | --help)
    awk '/^# Usage:/ { p = 1 } /^# Mutation:/ { p = 0 } p { sub(/^# ?/, ""); print }' "$self"
    exit 0
    ;;
esac
here="$(cd "$(dirname "$self")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-ux-version: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-ux-version "$@"
