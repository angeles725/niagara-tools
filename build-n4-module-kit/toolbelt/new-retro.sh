#!/usr/bin/env bash
# new-retro.sh — Atomic retro stub writer for the build-n4-module kit.
#
# Usage: new-retro.sh <module|kit> <slug> [--date YYYY-MM-DD] [--deltas N]
#   <slug>: [A-Za-z0-9][A-Za-z0-9._-]+, at least 6 chars, checked whole (the bash glob checked only the first 6)
#   --date: a calendar date spelled YYYY-MM-DD (default today) · --deltas: decimal digits (default 0);
#   any other value is a usage error, exit 3 (the bash accepted any value)
#
# Resolves the kit root from $KIT (env) → first positional arg if it is a directory
# with a retros/ dir → this script's ../ (unresolved: a symlinked toolbelt copy names its own parent) → cwd.
# This seam lets the test harness inject the kit root via KIT=<tmpdir> without passing a positional kit-root arg.
#
# Writes (all three in one atomic sequence — none if any step fails):
#   retros/<date>-<slug>.md    stub from template, header block on lines 1-3
#                              (review-status / module / deltas); refuses to overwrite (exit 3)
#   retros/INDEX.md            REGENERATED (go/internal/retroindex, as gen-retro-index.sh) from the retro headers — never
#                              appended to, so no row can be duplicated [ev: retro generated-retro-index Δ2];
#                              generated, not committed (.gitignore), so it may be absent beforehand
#   BUILD-STATE.src.md         sets retro_pending: true (only that key) in the committed
#                              source of the generated BUILD-STATE.md [ev: retro ledger-generated-not-committed]
# If the index generator rejects any retro in the dir (exit 1), the new stub is removed again,
# BUILD-STATE.src.md is not touched and the generator's message names the offending file.
#
# Output rows:
#   retro    PASS  <file>  written
#   index    PASS  <file>  regenerated
#   envelope  PASS  BUILD-STATE.src.md  retro_pending: true
#
# Exit codes: 0 ok · 1 index regeneration failed (nothing landed) · 3 usage/env/file-already-exists (K20),
#   including no Go toolchain or a failed build
#
# Implementation: go/internal/newretro (goldens recorded from the bash script at 15e792d1 by
# go/internal/newretro/testdata/record/record.sh). go/go-tool.sh `go_tool_bin` builds go/bin/new-retro when
# it is missing or older than a Go source, then this script execs it with NEW_RETRO_DEFAULT_ROOT = its ../.
# VCS-free by design; kit-links.bats L2 enforces.
# [ev: retro campaign8-retro-loop]
set -u
# readlink -f: resolve a symlinked copy of this file (report-module test toolbelts) to the real kit, so ../../go exists.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'new-retro: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" new-retro)" || exit 3
# The default kit root stays the UNRESOLVED ../ of this file, as the bash script's BASH_SOURCE parent was.
NEW_RETRO_DEFAULT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)" exec "$BIN" "$@"
