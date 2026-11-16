#!/usr/bin/env bash
# lint-ntpl-manifest-deps.sh — FAILs a template archive (.ntpl) a module ships without template.bog or
# template-manifest.xml, with a malformed manifest or with a <dependency> the module's build never declares.
#
# A .ntpl is a zip holding template.bog and template-manifest.xml; the manifest's <dependencies> name the
# modules the template needs, and an application-template install hard-fails on a missing one
# (types/logic-authoring.md § Shipping a template module). For every *.ntpl under a module root (a part's
# build/ and test/ outputs, dot directories and *-worktrees checkouts are not read):
#   ntpl-entries   the archive lacks template.bog or template-manifest.xml (matched by their whole name, at
#                  the top of the archive), holds one of them twice, or is refused whole: not a zip, over
#                  32 MiB on disk, over 10000 entries, or an unsafe entry name (an absolute path, a ".."
#                  element, a backslash, a drive letter, a control character). The archive is untrusted
#                  input: nothing is unpacked, and only the manifest is inflated, up to 4 MiB.
#   ntpl-manifest  template-manifest.xml is not well-formed XML (line:col inside the manifest), cannot be
#                  inflated, or holds a <dependency> with no name.
#   ntpl-deps      a <dependencies><dependency name> that no *.gradle.kts under the root declares with
#                  api(":X"), nre(":X") or implementation(":X") (or project(":X")); settings.gradle.kts is no
#                  build file, and each part (the directory of a build file, and <part>.gradle.kts) counts as
#                  declared for its own module's templates. Names are listed once, sorted.
#   ntpl-station   a template under a newStations/ directory (the station templates Workbench lists,
#                  types/distribution.md § 8) whose manifest root lacks isStation="true".
# A module root that ships no .ntpl is one PASS row and exit 0. Not checked: whether template.bog uses a
# module the manifest does not list, and whether a listed module is a stock one every install holds.
# [ev: retro gate-followups-and-template-lints Δ7] [ev: retro gate-followups-and-template-lints Δ8]
#
# Implementation: rule package go/internal/lint/rules/ntplmanifest on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-ntpl-manifest-deps <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-ntpl-manifest-deps.sh [--] <module-root>...
#   Row:  FAIL  ntpl-entries|ntpl-manifest|ntpl-deps|ntpl-station  <file.ntpl>: <reason>
#   Row:  PASS  lint-ntpl-manifest-deps  <module-root>: no *.ntpl under the root (nothing to check)
#   Exit: 0  clean, or no .ntpl under a module root · 1  any FAIL · 3  usage/env, a root with neither a *.gradle.kts nor a *.ntpl, an archive or a build file that cannot be opened, or a sub-directory find cannot enter
# Mutation: NTPL1 -- dropping the lacks-entry row passes a .ntpl with no template.bog or no manifest
# Mutation: NTPL2 -- reading a manifest that does not parse as one with no dependencies passes it
# Mutation: NTPL3 -- dropping the undeclared-dependency row passes a template that needs a module the build never declares
# Mutation: NTPL4 -- dropping the station row ships a newStations/ template whose manifest is not a station manifest
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-ntpl-manifest-deps: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-ntpl-manifest-deps "$@"
