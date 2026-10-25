#!/usr/bin/env bash
# gen-changelog.sh — validates changelog.d/ fragments, or folds them into CHANGELOG.md at release time.
#
# Every kit behaviour change adds one changelog.d/<YYYY-MM-DD>-<slug>.md fragment (the changelog gate,
# scripts/changelog-gate.sh, CONTRIBUTING.md § 6.2), so parallel PRs never edit the same [Unreleased] lines.
# The release (CONTRIBUTING.md § 5) runs --write first: it appends every fragment, in file-name (date then
# slug) order, to the end of CHANGELOG.md's `## [Unreleased]` section and deletes the folded fragments; the
# release then turns [Unreleased] into the version section. CI runs --check, so a malformed fragment (one the
# fold would refuse) fails before it merges.
# Fragment contract: line 1 is `### Added|Changed|Deprecated|Removed|Fixed|Security [— <title>]`, every heading
# has at least one `- ` bullet, and no other heading level appears.
#
# Implementation: go/internal/changelog (fold.go), built on demand by go/go-tool.sh `go_tool_bin`.
# Usage:  gen-changelog.sh --check|--write [<repo-root>]     (default root: this script's ../)
# Exits:  0 valid / folded (or nothing to fold) · 1 a malformed fragment or no [Unreleased] header (nothing
#         written or deleted) · 3 usage/env, including no CHANGELOG.md, no Go toolchain or a failed build
# VCS-free.
set -u

here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'gen-changelog: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" gen-changelog)" || exit 3
GEN_CHANGELOG_DEFAULT_ROOT="$(cd "$here/.." && pwd)" exec "$BIN" "$@"
