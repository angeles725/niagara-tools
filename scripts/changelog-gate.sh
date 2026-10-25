#!/usr/bin/env bash
# changelog-gate.sh — fails a range that changes kit behaviour without a CHANGELOG entry (CONTRIBUTING.md § 6.2).
#
# Behaviour = a changed path under go/, build-n4-module-kit/ or tools/, except the retro gate's own exclusions
# (go/bin/, build-n4-module-kit/BUILD-STATE(.src).md, ledger.d/, retros/). Tests and docs under those trees
# count. An entry is ONE of:
#   - a changelog.d/<YYYY-MM-DD>-<slug>.md fragment added (or edited) in the range: line 1 is a Keep a Changelog
#     heading (### Added|Changed|Deprecated|Removed|Fixed|Security [— <title>]), each heading has a '- ' bullet;
#   - a non-blank line added strictly inside CHANGELOG.md's `## [Unreleased]` section (an older release section
#     does not count);
#   - a `Changelog: none (<reason>)` trailer in any commit of the range (reason: non-empty, one printable line,
#     at most 120 characters).
# A malformed fragment or `Changelog:` line fails the range even when it changes no behaviour (fail closed).
# git IS used (by the Go binary): this is a scripts/ helper, not a toolbelt/ script (kit-links L2).
# Run by .githooks/pre-push over the pushed range and by ci.yml over origin/main..HEAD.
#
# Implementation: go/internal/changelog, built on demand by go/go-tool.sh `go_tool_bin`.
# Usage:  changelog-gate.sh --range <base>..<head>     (run inside the repository)
# Exits:  0 pass / not applicable · 1 no entry, or a malformed fragment or trailer · 3 usage/env (a malformed
#         range, git missing or failing, no Go toolchain or a failed build)
set -u

here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'changelog-gate: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" changelog-gate)" || exit 3
exec "$BIN" "$@"
