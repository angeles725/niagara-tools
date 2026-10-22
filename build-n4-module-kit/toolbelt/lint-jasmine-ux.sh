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
#   Exit: 0  always (advisory) · 3  usage/env or a spec sub-directory find cannot enter
# VCS-free by design.
# Mutation: JUX2 -- drop the empty-specs check and JUX2 stops warning on a rc/ module without specs
# Mutation: JUX-finderr -- ignoring the find exit status reads an unreadable spec directory as no specs (or hides it)
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/awk-lint.sh"
awk_lint_parse lint-jasmine-ux raw 'usage: lint-jasmine-ux.sh <ux-module-root>' "$@"
ROOT="${AL_ROOTS[0]}"

# Only applies when the module actually ships a browser rc/ tree.
if [ ! -d "$ROOT/src/rc" ]; then
    exit 0
fi

specs=""
if [ -d "$ROOT/srcTest/rc/spec" ]; then
    # A spec sub-directory find cannot enter is an env error, never "no specs" and never a clean pass.
    if ! scan_files "$AL_TMP/files" "$AL_TMP/find.err" "$ROOT/srcTest/rc/spec" -name '*.js'; then
        printf 'lint-jasmine-ux: cannot list every file under %s: %s\n' "$ROOT/srcTest/rc/spec" "$(head -n 1 "$AL_TMP/find.err")" >&2
        exit 3
    fi
    specs=$(head -n 1 "$AL_TMP/files")
fi
if [ -z "$specs" ]; then
    printf 'WARN  lint-jasmine-ux  %s  ux ships src/rc but no srcTest/rc/spec/*.js\n' "$ROOT"
fi

exit 0
