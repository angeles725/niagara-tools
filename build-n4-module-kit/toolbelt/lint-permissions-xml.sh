#!/usr/bin/env bash
# lint-permissions-xml.sh — FAILs a module-permissions.xml the security manager does not read as written; WARNs a
# request without <name>/<purposeKey> and an absolute file path in -rt code.
#
# baja reads only a <permissions> root (NModule): any other root, e.g. the old <modulePermissions/>, silently drops
# every request, and a <niagara-permission-groups> type other than station, workbench or all is never granted.
# Each <req-permission>/<opt-permission> needs <name> and <purposeKey>. new File("/…") or Paths.get("/…") in -rt
# code reaches outside Sys.getStationHome()/getNiagaraSharedUserHome() and needs a file grant. Start from
# templates/module-permissions.xml (types/structure.md § module-permissions.xml).
# Scans every module-permissions.xml and every *.java under a -rt part's src/ of each module root (srcTest/, test/
# and build/ are not read). The XML is parsed, so the template's commented-out requests are not read; Java is read
# as tokens (go/internal/javasrc), so a comment or a string is never a path. A file that does not parse is exit 3.
# [ev: retro n4-course-authoring-files-and-libraries Δ2] [ev: retro devguide415-horizons-authoring-security Δ5]
#
# Implementation: rule package go/internal/lint/rules/permissionsxml on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-permissions-xml <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-permissions-xml.sh [--strict] <module-root>...
#   Row:  FAIL  lint-permissions-xml  <file>:<line>  root element <x> is not <permissions>: ... | <niagara-permission-groups type="x"> is not station, workbench or all
#   Row:  WARN  lint-permissions-xml  <file>:<line>  <req-permission> <NAME> has no <purposeKey> ... | new File("/…") is an absolute path ...
#   Exit: 0  clean or WARN-only · 1  any FAIL, or any WARN under --strict · 3  usage/env, a file that does not
#         parse, a root with no module-permissions.xml and no -rt src/ *.java
# Mutation: PX1 -- accepting any root element passes the old <modulePermissions/> fixture root
# Mutation: PX2 -- matching raw Java lines (comments and strings kept) WARNs a commented-out new File("/etc/passwd")
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-permissions-xml: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-permissions-xml "$@"
