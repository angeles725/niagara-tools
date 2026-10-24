#!/usr/bin/env bash
# kit-cite-resolve.sh — resolves every evidence pointer of kit cards (`[ev: …]` / `[deferred: …]` tags and
# prose citations: retro deltas, corpus blocks and sections, code, devguide/guide lines, memories, kit
# cross-references, course pages, client bitácoras) and prints its status with an excerpt of the cited line,
# so a reviewer only judges whether the source says it. A pre-filter: RESOLVED means the cited place exists,
# never that it is true. A bare file name or partial path found under no root is searched by base name in the
# repository, organized/, the versioned niagara-help source and the client checkouts (one indexed walk per
# tree per run); a line range is checked in every copy of the file (the detail names the copy read).
# A `~/` path expands to $HOME and resolves by existence like an absolute path (RESOLVED, not a separate
# status: the file is there to read; a `kit` path is never expanded). A `code <module> <dir>/` pointer names
# a directory of the module's copies. A javap jar (prose "javap of `x.jar`" or a `javap x.jar[!entry]
# [<ver>]` clause) is looked up in organized/, then in the Gradle cache ($GRADLE_USER_HOME, else ~/.gradle:
# caches/jars-*/*/ and caches/modules-2/files-2.1/*/*/*/*/ only). A `<path>.zip!<entry>` after `code` or
# `course` (the path may hold spaces) resolves the zip; the entry is never opened. A path relative to
# niagara-help (`source/…`, `guides-clean/…`) reads versions/<ver>/ before the unversioned copy. A quoted
# phrase right after a guides-clean line range is an anchor: it must sit on the cited lines, give or take 2.
#
# Implementation: the Go module at <repo>/go (go/internal/citeparse, go/internal/citeresolve). This wrapper
# is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/kit-cite-resolve when the binary is
# missing or older than a Go source, then this script execs it. Needs `go` on PATH to build.
#
# Usage:  kit-cite-resolve.sh [--kit DIR] [--research DIR] [--modules DIR] [--clients DIR] [--memory DIR]
#                             [--version VER] [--format tsv|json] [--strict] <card.md>...
#         --clients defaults to <modules>/Cliente; a <checkout>-worktrees/<tree>/ copy is the same file.
# Output: card line kind target locator status detail (TSV, or a JSON array that adds the scope), then on
#         stderr the per-status counts of all rows, of the LIVE rows and of the HISTORY rows, and the strict
#         findings of each scope. Findings: MISSING_FILE, MISSING_SECTION, MISSING_DELTA, LINE_OUT_OF_RANGE,
#         BARE_ID, UNPARSED, AMBIGUOUS (a searched name matches several files: listed, never guessed),
#         DRIFT (a guides-clean anchor no longer on its cited lines: names where it sits now, if anywhere),
#         TOOL_ERROR. Not findings: RESOLVED; HISTORICAL_DRIFT (a missing, out-of-range or drifted
#         target on a history card: retros/, ledger.d/, BUILD-STATE.md, BUILD-STATE.src.md, CHANGELOG.md;
#         the original status stays in the detail); UNVERIFIABLE_KIND (evidence no tree holds: memory-tool,
#         live, VCS/tracker, command run, decision ID, self-marked, an http(s) URL: never a file lookup);
#         SKIPPED_TEMPLATE (a `<stem>` placeholder, or an elided word: `…`, `B…`, `…Δ1`, `Δk`).
# Exits:  0 no finding, or findings without --strict · 1 any finding under --strict · 3 usage, a missing
#         or unreadable root or card, a tool error row, no Go toolchain or a failed build
# VCS-free by design (kit-links L2).
set -u

# readlink -f: resolve a symlinked copy of this file (report-module test toolbelts) to the real kit, so ../../go exists.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'kit-cite-resolve: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" kit-cite-resolve)" || exit 3
exec "$BIN" "$@"
