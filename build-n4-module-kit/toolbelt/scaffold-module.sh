#!/usr/bin/env bash
# scaffold-module.sh — emit a buildable N4 module skeleton from a bundled fixture.
# Fixture-driven (D1): copies the bundled fixtures/<Type> tree and substitutes
# the fixture type name → <ModuleName> in paths and content (B790/B793 proven tree).
#
# Usage: scaffold-module.sh <ModuleName> <out-dir> [--vendor <v>] [--target-version <x.y>] [--plugin-version <v>] [--type <logic|dashboard|driver>]
#   ModuleName       Java identifier: uppercase-first, alphanumeric only (e.g. ColdRoomPan)
#   out-dir          parent directory; emitted root is <out-dir>/<ModuleName>
#   --vendor <v>     display vendor name (copyright, defaultVendor); default Angeles
#   --target-version <x.y>  baja target version; default 4.14
#   --plugin-version <v>    Niagara Gradle plugin version baked into settings.gradle.kts; default 7.6.17
#   --type <logic|dashboard|driver>  scaffold type (default: logic)
#                    logic     — single-profile -rt skeleton (MinimalPan fixture) + station-test overlay
#                    dashboard — two-profile -rt facade + -ux servlet skeleton (MinimalDash fixture) + station-test overlay
#                    driver    — NDriver -rt (UDP comm config subclass, codec, discovery) + -wb point manager
#                                (MinimalDriver fixture; types/driver-authoring.md § 1.6-§ 1.11, § 5, § 12)
#
# Every type also gets the client version ledger <out-dir>/<ModuleName>/odd/VERSION-LEDGER.md (templates/, its
# sample row dropped; check it with ledger-check.sh).
# The emitted tree is pre-slotomatic. Run build.sh to generate the AUTO region and build the jar.
# The root build.gradle.kts carries the signing block (templates/signing-root.gradle.kts, issue #266):
# set niagaraSigningAlias and niagaraSigningProfile as gradle properties before the first build.
# This script is VCS-free by design. version control is never invoked.
#
# Exit: 0 ok · 2 usage, invalid ModuleName, unknown --type or a refused option value · 3 env (skeleton missing,
#        out-dir not creatable, or <out-dir>/<ModuleName> already exists) · 129, 130, 143 SIGHUP, SIGINT or SIGTERM
#        while the module is emitted (`interrupted (<signal>)`; the partial tree is removed as below)
# A run that fails after it created <out-dir>/<ModuleName> (a failed write, a missing template) removes that
# directory and says so in a second stderr line (`removed the partial <path>`, or `could not remove the partial
# <path> (<error>): its files stay`); the exit code stays the fault's (3). A destination that existed before the run
# is refused and never touched; the out-dir itself is never removed.
#
# Add a concern to an emitted (or any mapped) module:
#   scaffold-module.sh add-concern <Name> --root <module-root> [--part rt|ux|wb|se] [--adapter]
# execs toolbelt/module-map.sh add-concern with the same arguments (its header has the contract and exits).
# [ev: retro modularity-structure Δ1]
#
# Option values are literal text: every character of a value is written as typed (`&` is `&`, `\1` is `\1`, `|` is
# `|`), and inserted text is never rewritten by another substitution. The substituted options are --vendor,
# --target-version and --plugin-version. --vendor must be one Java package segment once lowercased (an ASCII letter
# or _ first, then ASCII letters, digits or _); a --target-version or --plugin-version that holds a newline is
# refused by name. Either refusal is exit 2 with nothing written; a failed write is exit 3.
#
# Implementation: go/internal/scaffoldmodule (command go/cmd/scaffold-module; go/README.md). This wrapper is the
# stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/scaffold-module when it is missing or older than a
# Go source, then this script execs it with SCAFFOLD_MODULE_TOOLBELT = the directory it was invoked from
# (unresolved, so ../fixtures, ../templates and module-map.sh are found beside the invoked path, never from $HOME:
# K8, TC-K8). Needs `go` on PATH to build (exit 3 otherwise).
set -u
# readlink -f: resolve a symlinked copy of this file to the real kit, so ../../go exists.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'scaffold-module: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" scaffold-module)" || exit 3
SCAFFOLD_MODULE_TOOLBELT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export SCAFFOLD_MODULE_TOOLBELT
exec "$BIN" "$@"
