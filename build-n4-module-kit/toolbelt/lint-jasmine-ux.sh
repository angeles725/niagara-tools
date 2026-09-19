#!/usr/bin/env bash
# lint-jasmine-ux.sh — a -ux module that ships a browser rc/ tree should also ship JS specs.
#
# A -ux profile serving bajaux widgets or a SPA under src/rc/ can pass the whole kit gate
# with ZERO JavaScript tests. WARN when src/rc exists but there is no srcTest/rc/spec/*.js
# (the Jasmine/Karma/grunt-niagara test seam). Advisory: exit 0, WARN row. See dashboard.md DUX-TEST1.
# [ev: retro 2026-09-19-observability]
#
# Usage:  lint-jasmine-ux.sh <ux-module-root>   (a -ux part root containing src/)
#   Row:  WARN  lint-jasmine-ux  <root>  ux ships src/rc but no srcTest/rc/spec/*.js
#   Exit: 0  always (advisory) · 3  usage/env
# VCS-free by design.
set -u
LC_ALL=C
export LC_ALL

if [ $# -lt 1 ]; then
    printf 'usage: lint-jasmine-ux.sh <ux-module-root>\n' >&2
    exit 3
fi
ROOT="$1"
if [ ! -d "$ROOT" ]; then
    printf 'lint-jasmine-ux: not a directory: %s\n' "$ROOT" >&2
    exit 3
fi

# Only applies when the module actually ships a browser rc/ tree.
if [ ! -d "$ROOT/src/rc" ]; then
    exit 0
fi

specs=$(find "$ROOT/srcTest/rc/spec" -type f -name '*.js' 2>/dev/null | head -1)
if [ -z "$specs" ]; then
    printf 'WARN  lint-jasmine-ux  %s  ux ships src/rc but no srcTest/rc/spec/*.js\n' "$ROOT"
fi

exit 0
