#!/usr/bin/env bash
# lint-private-api-import.sh — WARNs each com.tridium (private API) import or fully qualified use in module code.
#
# The devguide says "com.tridium APIs should never be used by developers, and no compatibility is guaranteed"
# (apiInfo.txt:54): a com.tridium use can break on any Niagara update. Prefer javax.baja; record an unavoidable
# com.tridium use in the module README and pin the vendorVersion floor (types/module-wiring.md).
# Scans every *.java under src/ of each -rt, -ux, -wb and -se part of ONE module root; srcTest/, test/ and build/
# are not read. A row is each `import com.tridium.…;`, `import static com.tridium.…;` and fully qualified
# com.tridium. name in code; Java is read as tokens (go/internal/javasrc), so a comment or a string is never one,
# and com.tridiumx. is another prefix. Default allow-list: the NDriver framework the driver wizard generates
# against, com.tridium.ndriver.** and com.tridium.driver.util.DrUtil (an exact name also allows its members);
# --allow <glob> adds an entry ("*" one name segment, "**" any), --no-default-allow drops the default list.
# Acceptance: the course trafficLightDriver tree gives 0 rows, and 60 with --no-default-allow.
# [ev: retro n4-course-build-deps-permissions-signing Δ3] [ev: retro n4-course-handson-ndriver Δ8]
#
# Implementation: rule package go/internal/lint/rules/privateapiimport on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-private-api-import <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-private-api-import.sh [--strict] [--allow <glob>]... [--no-default-allow] <module-root>
#   Row:  WARN  lint-private-api-import  <file>:<line>  <qualified name> is private API (devguide apiInfo: no compatibility guaranteed)
#   Exit: 0  clean or WARN-only · 1  any WARN under --strict · 3  usage/env, a root with no part src/ *.java
# Mutation: PAI1 -- dropping the default allow-list WARNs every import of the NDriver fixture module
# Mutation: PAI2 -- matching raw lines (comments and strings kept) WARNs the commented-out import
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-private-api-import: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-private-api-import "$@"
