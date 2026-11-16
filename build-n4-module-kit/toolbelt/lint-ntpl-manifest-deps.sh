#!/usr/bin/env bash
# lint-ntpl-manifest-deps.sh — FAILs a template archive (.ntpl) a module ships without template.bog or
# template-manifest.xml, with a manifest that is no well-formed template manifest, or with a <dependency> that
# neither the module's build declares nor the Niagara installation ships.
#
# A .ntpl is a zip holding template.bog and template-manifest.xml; the manifest's <dependencies> name the
# modules the template needs, and an application-template install hard-fails on a missing one
# (types/logic-authoring.md § Shipping a template module). For every *.ntpl under the module root (the
# extension in any letter case; a part's build/ and test/ outputs, dot directories and *-worktrees checkouts
# are not read):
#   ntpl-entries   the archive lacks template.bog or template-manifest.xml (matched by their whole name, at
#                  the top of the archive), holds one of them twice, or is refused whole: not a zip, over
#                  32 MiB on disk, over 10000 entries, or an unsafe entry name (an absolute path, a ".."
#                  element, a backslash, a drive letter, a control character). The archive is untrusted
#                  input: nothing is unpacked, and only the manifest is inflated, up to 4 MiB.
#   ntpl-manifest  template-manifest.xml is not well-formed XML (line:col inside the manifest), cannot be
#                  inflated, has a root element other than <template>, or holds a <dependency> with no name
#                  or outside <dependencies>.
#   ntpl-deps      a <dependencies><dependency name> that no *.gradle.kts under the root declares with
#                  api(":X"), nre(":X") or implementation(":X") (or project(":X")) and that the installation
#                  does not ship. settings.gradle.kts is no build file, and each part (the directory of a
#                  build file, and <part>.gradle.kts) counts as declared for its own module's templates.
#                  Names are listed once, sorted.
#   ntpl-station   a template under a newStations/ directory (the station templates Workbench lists,
#                  types/distribution.md § 8) whose manifest root lacks isStation="true".
#
# Stock modules: a station template lists every module its station uses, most of them shipped by every
# Niagara installation and declared by no build file. --niagara-home <dir> (or NIAGARA_HOME in the
# environment; the option wins) names the installation: each <dir>/modules/<name>.jar is a module it ships
# (part suffix kept: fox-rt), and a dependency it ships is not a finding. Without an installation the lint
# cannot tell a stock module from a missing one: it prints one WARN row saying so and reports the undeclared
# dependencies as WARN ntpl-deps rows, exit 0; it never blocks on what it cannot know. An installation path
# that does not exist, has no modules/ directory or whose modules/ cannot be listed is exit 3.
# [ev: retro gate-followups-and-template-lints Δ12]
#
# Symlinks: a .ntpl that is a symlink is checked through the link, and a symlinked directory inside the
# module is entered (each directory once, so a link back to an ancestor ends); both are reported under the
# link's own path. A .ntpl link that leads nowhere is exit 3 (it could not be read). "Nothing to check" is
# never printed over something that was skipped. A path holding a control character, a double quote or a
# backslash is printed in double quotes with C escapes, so a row is always one line.
# A module root that ships no .ntpl is one PASS row and exit 0. Not checked: whether template.bog uses a
# module the manifest does not list.
# [ev: retro gate-followups-and-template-lints Δ7] [ev: retro gate-followups-and-template-lints Δ8]
#
# Implementation: rule package go/internal/lint/rules/ntplmanifest on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-ntpl-manifest-deps <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-ntpl-manifest-deps.sh [--niagara-home <dir>] <module-root>
#   Row:  FAIL  ntpl-entries|ntpl-manifest|ntpl-deps|ntpl-station  <file.ntpl>: <reason>
#   Row:  WARN  ntpl-deps  <file.ntpl>: <reason>            (no installation given)
#   Row:  WARN  lint-ntpl-manifest-deps  <module-root>: the installation's module list is not known, ...
#   Row:  PASS  lint-ntpl-manifest-deps  <module-root>: no *.ntpl under the root (nothing to check)
#   Exit: 0  no FAIL (WARN rows, or no .ntpl under the module root, are still 0) · 1  any FAIL · 3  usage/env, a root with neither a *.gradle.kts nor a *.ntpl, an installation path that gives no module list, a .ntpl link that leads nowhere, an archive or a build file that cannot be opened, or a directory that cannot be entered
# Mutation: NTPL1 -- dropping the lacks-entry row passes a .ntpl with no template.bog or no manifest
# Mutation: NTPL2 -- reading a manifest that does not parse as one with no dependencies passes it
# Mutation: NTPL3 -- dropping the undeclared-dependency row passes a template that needs a module nobody declares or ships
# Mutation: NTPL4 -- dropping the station row ships a newStations/ template whose manifest is not a station manifest
# Mutation: NTPL6 -- walking without following links reads a symlinked template as "nothing to check"
# Mutation: NTPL7 -- comparing against the build files alone fails a station template on the stock modules it lists
# Mutation: NTPL8 -- failing without an installation blocks a module on stock modules the lint cannot know about
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
