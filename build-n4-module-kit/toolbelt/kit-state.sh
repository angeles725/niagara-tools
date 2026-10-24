#!/usr/bin/env bash
# kit-state.sh — the ONE reader command for the kit's generated state.
#
# BUILD-STATE.md (the build-session ledger: every module envelope + the kit block) and retros/INDEX.md
# (the retro promotion registry) are GENERATED and NOT committed (.gitignore): two PRs that each add a
# ledger.d/ fragment or a retro used to conflict on them. Their only sources are committed:
#   BUILD-STATE.src.md   every hand-kept field (module envelopes, kit fields); the kit envelope's
#                        last_change / open_issues items / session log stay empty there
#   ledger.d/*.md        one fragment per change (contract: toolbelt/gen-build-state.sh)
#   retros/*.md          one retro each; the INDEX row comes from its header (toolbelt/gen-retro-index.sh)
# Read the state with this command, never from a possibly stale local file. `--write` materializes
# both files in the kit (scripts/install-skill.sh and the .githooks post-merge/post-checkout/
# post-rewrite hooks run it) and prints each path written. [ev: retro ledger-generated-not-committed]
#
# Implementation: go/internal/kitstate over the go/internal/buildstate and go/internal/retroindex
# generators, built on demand by go/go-tool.sh `go_tool_bin` (as gen-build-state.sh). Needs `go` on PATH.
#
# Usage:  kit-state.sh build-state|retro-index [<kit-root>]            print to stdout
#         kit-state.sh --write [build-state|retro-index] [<kit-root>]  write (default: both)
#         (default kit root: this script's ../)
# Exits:  0 ok · 1 a ledger fragment, BUILD-STATE.src.md or retro-header violation (nothing written)
#         · 3 usage/env, including no Go toolchain or a failed build
# VCS-free by design (kit-links L2).
set -u

# readlink -f: resolve a symlinked copy of this file to the real kit, so ../../go exists.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'kit-state: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" kit-state)" || exit 3
KIT_STATE_DEFAULT_ROOT="$(cd "$here/.." && pwd)" exec "$BIN" "$@"
